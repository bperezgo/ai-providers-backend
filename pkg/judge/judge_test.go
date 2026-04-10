package judge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func newTestHTTPClient() *common.HTTPClient {
	return common.NewHTTPClient(10*time.Second, 0, time.Second)
}

func TestEvaluate_highQualityImage(t *testing.T) {
	expectedScore := ImageEvalScore{
		PromptAdherence:  5,
		ArtifactScore:    5,
		CompositionScore: 4,
		OverallScore:     5,
		Reasoning:        "The image matches the prompt perfectly with no artifacts.",
		SuggestedRetry:   false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "test-anthropic-key", r.Header.Get("x-api-key"))
		assert.Equal(t, anthropicVersion, r.Header.Get("anthropic-version"))

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, judgeModel, body["model"])
		assert.Equal(t, float64(0), body["temperature"])

		scoreJSON, _ := json.Marshal(expectedScore)
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(scoreJSON)},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	j := NewJudgeWithURL("test-anthropic-key", newTestHTTPClient(), server.URL)

	score, err := j.Evaluate(context.Background(), ImageEvalRequest{
		OriginalPrompt: "a beautiful sunset over the ocean",
		ImagePath:      testdataPath("fixture_good.png"),
		UseCase:        "social-post",
	})
	require.NoError(t, err)

	assert.Equal(t, 5, score.PromptAdherence)
	assert.Equal(t, 5, score.ArtifactScore)
	assert.Equal(t, 4, score.CompositionScore)
	assert.Equal(t, 5, score.OverallScore)
	assert.False(t, score.SuggestedRetry)
}

func TestEvaluate_artifactImage(t *testing.T) {
	expectedScore := ImageEvalScore{
		PromptAdherence:  2,
		ArtifactScore:    1,
		CompositionScore: 1,
		OverallScore:     1,
		Reasoning:        "The image has severe artifacts and does not match the prompt.",
		SuggestedRetry:   true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scoreJSON, _ := json.Marshal(expectedScore)
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(scoreJSON)},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	j := NewJudgeWithURL("test-key", newTestHTTPClient(), server.URL)

	score, err := j.Evaluate(context.Background(), ImageEvalRequest{
		OriginalPrompt: "a corporate office meeting",
		ImagePath:      testdataPath("fixture_bad.png"),
		UseCase:        "marketing-banner",
	})
	require.NoError(t, err)

	assert.Equal(t, 2, score.PromptAdherence)
	assert.Equal(t, 1, score.ArtifactScore)
	assert.Equal(t, 1, score.OverallScore)
	assert.True(t, score.SuggestedRetry)
}

func TestEvaluate_apiError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	j := NewJudgeWithURL("test-key", newTestHTTPClient(), server.URL)

	_, err := j.Evaluate(context.Background(), ImageEvalRequest{
		OriginalPrompt: "test",
		ImagePath:      testdataPath("fixture_good.png"),
		UseCase:        "social-post",
	})
	require.Error(t, err)

	var apiErr *common.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
}

func TestEvaluate_invalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "not valid json at all"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	j := NewJudgeWithURL("test-key", newTestHTTPClient(), server.URL)

	_, err := j.Evaluate(context.Background(), ImageEvalRequest{
		OriginalPrompt: "test",
		ImagePath:      testdataPath("fixture_good.png"),
		UseCase:        "social-post",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse judge score")
}

func TestEvaluate_missingImage(t *testing.T) {
	j := NewJudge("test-key", newTestHTTPClient(), "")

	_, err := j.Evaluate(context.Background(), ImageEvalRequest{
		OriginalPrompt: "test",
		ImagePath:      "/nonexistent/path.png",
		UseCase:        "social-post",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to encode image")
}

func TestBuildJudgePrompt(t *testing.T) {
	prompt := buildJudgePrompt("a red fish", "social-post")
	assert.Contains(t, prompt, "a red fish")
	assert.Contains(t, prompt, "social-post")
	assert.Contains(t, prompt, "prompt_adherence")
	assert.Contains(t, prompt, "artifact_score")
	assert.Contains(t, prompt, "composition_score")
	assert.Contains(t, prompt, "overall_score")
	assert.Contains(t, prompt, "suggested_retry")
}

func TestBuildJudgePrompt_emptyUseCase(t *testing.T) {
	prompt := buildJudgePrompt("test prompt", "")
	assert.Contains(t, prompt, "general marketing")
}

func TestEncodeImage(t *testing.T) {
	encoded, mediaType, err := encodeImage(testdataPath("fixture_good.png"))
	require.NoError(t, err)
	assert.Equal(t, "image/png", mediaType)
	assert.NotEmpty(t, encoded)
}
