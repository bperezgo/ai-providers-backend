package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/config"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AIChatService provides a context-aware AI chat assistant for course chapter editing.
type AIChatService struct {
	client anthropic.Client
	db     *gorm.DB
	logger *zap.Logger
}

// ChatMessage represents a single message in the conversation.
type ChatMessage struct {
	Role    string `json:"role" binding:"required"`    // "user" or "assistant"
	Content string `json:"content" binding:"required"` // message text
}

// ChatRequest is the request body for the AI chat endpoint.
type ChatRequest struct {
	Messages []ChatMessage `json:"messages" binding:"required"`
	Context  ChatContext   `json:"context"`
}

// ChatContext provides chapter/section awareness to the AI.
type ChatContext struct {
	ChapterID string `json:"chapter_id"`
	SectionID string `json:"section_id,omitempty"`
}

// ChatStreamEvent is sent to the client during streaming.
type ChatStreamEvent struct {
	Type    string `json:"type"`              // "token", "done", "error"
	Content string `json:"content,omitempty"` // text token
}

// NewAIChatService creates a new AI chat service.
func NewAIChatService(cfg *config.Config, db *gorm.DB, logger *zap.Logger) (*AIChatService, error) {
	if cfg.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is required for AI chat")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.AnthropicAPIKey),
	}
	if cfg.AnthropicBaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.AnthropicBaseURL))
	}

	return &AIChatService{
		client: anthropic.NewClient(opts...),
		db:     db,
		logger: logger,
	}, nil
}

// Chat streams a Claude response to the events channel.
func (s *AIChatService) Chat(ctx context.Context, req *ChatRequest, events chan<- ChatStreamEvent) error {
	defer close(events)

	systemPrompt := s.buildSystemPrompt(ctx, req.Context)

	// Convert messages to Anthropic format
	var messages []anthropic.MessageParam
	for _, msg := range req.Messages {
		switch msg.Role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		}
	}

	stream := s.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 2048,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: messages,
	})

	for stream.Next() {
		event := stream.Current()

		switch variant := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			if variant.Delta.Type == "text_delta" && variant.Delta.Text != "" {
				events <- ChatStreamEvent{Type: "token", Content: variant.Delta.Text}
			}
		case anthropic.MessageStopEvent:
			events <- ChatStreamEvent{Type: "done"}
		}
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("streaming error: %w", err)
	}

	return nil
}

// buildSystemPrompt creates a context-aware system prompt including chapter data.
func (s *AIChatService) buildSystemPrompt(ctx context.Context, chatCtx ChatContext) string {
	base := `You are an AI assistant helping build video production plans for educational courses. You help with:
- Writing narrative scripts (voice-over text in Spanish)
- Writing visual direction descriptions (camera shots, lighting, mood)
- Suggesting production settings (voice style, music style, visual style, color palettes)
- Improving existing content based on feedback
- Answering questions about video production for online courses

Be concise, creative, and specific. When writing narratives, use Spanish. When writing visual directions, include specific camera angles and lighting descriptions. Always keep the prenatal/maternal wellness context in mind when relevant.`

	if chatCtx.ChapterID == "" {
		return base
	}

	// Load chapter context
	var chapter models.CourseChapter
	if err := s.db.WithContext(ctx).First(&chapter, "id = ?", chatCtx.ChapterID).Error; err != nil {
		s.logger.Warn("Failed to load chapter context for chat", zap.Error(err))
		return base
	}

	contextInfo := fmt.Sprintf("\n\n## Current Chapter Context\n- Title: %s\n- Number: %d\n- Status: %s", chapter.Title, chapter.Number, chapter.Status)

	if chapter.Content != nil {
		contentJSON, _ := json.Marshal(chapter.Content)
		// Truncate if too large
		content := string(contentJSON)
		if len(content) > 4000 {
			content = content[:4000] + "...(truncated)"
		}
		contextInfo += fmt.Sprintf("\n\nChapter content (sections and rows):\n```json\n%s\n```", content)
	}

	if chatCtx.SectionID != "" {
		contextInfo += fmt.Sprintf("\n\nThe user is currently editing section: %s", chatCtx.SectionID)
	}

	return base + contextInfo
}
