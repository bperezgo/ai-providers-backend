package judge

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
)

const (
	defaultAnthropicAPIURL = "https://api.anthropic.com/v1/messages"
	anthropicVersion       = "2023-06-01"
	judgeModel             = "claude-sonnet-4-6"
	maxTokensResponse      = 1024
)

// Judge evaluates AI-generated images using Claude as a VLM judge.
type Judge struct {
	anthropicKey string
	httpClient   *common.HTTPClient
	apiURL       string
}

// NewJudge creates a new VLM judge backed by Claude.
// If baseURL is empty, the default Anthropic API URL is used.
func NewJudge(anthropicKey string, httpClient *common.HTTPClient, baseURL string) *Judge {
	apiURL := defaultAnthropicAPIURL
	if baseURL != "" {
		apiURL = baseURL + "/v1/messages"
	}
	return &Judge{
		anthropicKey: anthropicKey,
		httpClient:   httpClient,
		apiURL:       apiURL,
	}
}

// NewJudgeWithURL creates a judge pointing at a full API URL (for testing).
func NewJudgeWithURL(anthropicKey string, httpClient *common.HTTPClient, apiURL string) *Judge {
	return &Judge{
		anthropicKey: anthropicKey,
		httpClient:   httpClient,
		apiURL:       apiURL,
	}
}

// Evaluate scores a generated image against the original prompt using Claude's vision.
func (j *Judge) Evaluate(ctx context.Context, req ImageEvalRequest) (*ImageEvalScore, error) {
	encoded, mediaType, err := encodeImage(req.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	prompt := buildJudgePrompt(req.OriginalPrompt, req.UseCase)

	body := map[string]any{
		"model":       judgeModel,
		"max_tokens":  maxTokensResponse,
		"temperature": 0,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "image",
						"source": map[string]any{
							"type":       "base64",
							"media_type": mediaType,
							"data":       encoded,
						},
					},
					{
						"type": "text",
						"text": prompt,
					},
				},
			},
		},
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal judge request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", j.apiURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create judge request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", j.anthropicKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := j.httpClient.DoWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call judge API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, &common.APIError{
			Provider:   "anthropic-judge",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	return parseJudgeResponse(resp.Body)
}

// encodeImage reads and base64-encodes an image file.
func encodeImage(imagePath string) (encoded string, mediaType string, err error) {
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read image: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(imagePath))
	switch ext {
	case ".png":
		mediaType = "image/png"
	case ".jpg", ".jpeg":
		mediaType = "image/jpeg"
	case ".gif":
		mediaType = "image/gif"
	case ".webp":
		mediaType = "image/webp"
	default:
		mediaType = "image/png"
	}

	encoded = base64.StdEncoding.EncodeToString(imageBytes)
	return encoded, mediaType, nil
}

// buildJudgePrompt constructs the evaluation prompt for the VLM judge.
func buildJudgePrompt(originalPrompt, useCase string) string {
	if useCase == "" {
		useCase = "general marketing"
	}

	return fmt.Sprintf(`You are evaluating an AI-generated image for marketing use. Be conservative: only score what you actually see in the image.

Original prompt: %s
Use case: %s

Look at the provided image and score it on these criteria (1-5 each):
- prompt_adherence: Does the image match what was requested? (1 = completely wrong, 5 = perfect match)
- artifact_score: Are there visual artifacts, distortions, or errors? (1 = many artifacts, 5 = none)
- composition_score: Is the image suitable for %s? (1 = unusable, 5 = production-ready)
- overall_score: Would you use this image as-is? (1 = reject, 5 = ship it)

Set suggested_retry to true if overall_score < 3.

Respond with ONLY valid JSON matching this schema:
{"prompt_adherence": int, "artifact_score": int, "composition_score": int, "overall_score": int, "reasoning": "string", "suggested_retry": bool}`, originalPrompt, useCase, useCase)
}

// anthropicResponse represents the Claude API message response.
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// parseJudgeResponse extracts and decodes the ImageEvalScore from Claude's response.
func parseJudgeResponse(body io.Reader) (*ImageEvalScore, error) {
	var resp anthropicResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode judge response: %w", err)
	}

	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("empty judge response")
	}

	// Find the text content block
	var text string
	for _, block := range resp.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}

	if text == "" {
		return nil, fmt.Errorf("no text content in judge response")
	}

	text = stripCodeFences(text)

	var score ImageEvalScore
	if err := json.Unmarshal([]byte(text), &score); err != nil {
		return nil, fmt.Errorf("failed to parse judge score (raw: %s): %w", text, err)
	}

	return &score, nil
}

// stripCodeFences removes markdown code fences (```json ... ```) that LLMs sometimes wrap around JSON.
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence line
		if i := strings.Index(s, "\n"); i != -1 {
			s = s[i+1:]
		}
		// Remove closing fence
		if i := strings.LastIndex(s, "```"); i != -1 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
	}
	return s
}
