package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/config"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PlanGeneratorService generates production plans via Claude API.
type PlanGeneratorService struct {
	client       anthropic.Client
	db           *gorm.DB
	logger       *zap.Logger
	systemPrompt string

}

// PlanStreamEvent represents a streaming event sent to the client during plan generation.
type PlanStreamEvent struct {
	Type    string `json:"type"`              // "progress", "section", "complete", "error"
	Message string `json:"message,omitempty"` // Human-readable progress message
	Data    any    `json:"data,omitempty"`    // Structured data (plan JSON, section info, etc.)
}

// GeneratePlanRequest is the request body for full plan generation.
type GeneratePlanRequest struct {
	Feedback string `json:"feedback,omitempty"` // Optional user feedback/guidance
}

// GenerateSectionRequest is the request body for single section regeneration.
type GenerateSectionRequest struct {
	SectionID string `json:"section_id" binding:"required"`
	Feedback  string `json:"feedback,omitempty"`
}

// RefineSegmentRequest is the request body for single segment refinement.
type RefineSegmentRequest struct {
	SegmentID string `json:"segment_id" binding:"required"`
	Feedback  string `json:"feedback" binding:"required"`
}

// NewPlanGeneratorService creates a new plan generation service.
func NewPlanGeneratorService(cfg *config.Config, db *gorm.DB, logger *zap.Logger, promptPath string) (*PlanGeneratorService, error) {
	if cfg.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is required for plan generation")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.AnthropicAPIKey),
	}
	if cfg.AnthropicBaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.AnthropicBaseURL))
	}

	client := anthropic.NewClient(opts...)

	prompt, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read system prompt from %s: %w", promptPath, err)
	}

	return &PlanGeneratorService{
		client:       client,
		db:           db,
		logger:       logger,
		systemPrompt: string(prompt),
	}, nil
}

// GeneratePlan generates a full production plan for a chapter via Claude API.
// It streams progress events to the provided channel and saves the final plan to the database.
func (s *PlanGeneratorService) GeneratePlan(ctx context.Context, chapterID string, req *GeneratePlanRequest, events chan<- PlanStreamEvent) error {
	defer close(events)

	chapter, err := s.getChapter(ctx, chapterID)
	if err != nil {
		return err
	}

	if chapter.Content == nil {
		return fmt.Errorf("chapter has no content — add sections and rows first")
	}

	events <- PlanStreamEvent{Type: "progress", Message: "Starting plan generation..."}

	contentJSON, err := json.Marshal(chapter.Content)
	if err != nil {
		return fmt.Errorf("failed to marshal chapter content: %w", err)
	}

	userMessage := string(contentJSON)
	if req != nil && req.Feedback != "" {
		userMessage = fmt.Sprintf("Chapter content:\n%s\n\nAdditional guidance: %s", contentJSON, req.Feedback)
	}

	plan, err := s.callClaude(ctx, s.systemPrompt, userMessage, productionPlanTool(), events)
	if err != nil {
		return fmt.Errorf("Claude API call failed: %w", err)
	}

	// Save plan to database
	if err := s.savePlan(ctx, chapter, plan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	events <- PlanStreamEvent{Type: "complete", Message: "Plan generation complete", Data: plan}
	return nil
}

// GenerateSection regenerates a single section within an existing plan.
func (s *PlanGeneratorService) GenerateSection(ctx context.Context, chapterID string, req *GenerateSectionRequest, events chan<- PlanStreamEvent) error {
	defer close(events)

	chapter, err := s.getChapter(ctx, chapterID)
	if err != nil {
		return err
	}

	if chapter.Content == nil {
		return fmt.Errorf("chapter has no content")
	}
	if chapter.Plan == nil {
		return fmt.Errorf("chapter has no existing plan — generate a full plan first")
	}

	events <- PlanStreamEvent{Type: "progress", Message: fmt.Sprintf("Regenerating section %s...", req.SectionID)}

	// Build a focused prompt with just this section's data + production settings
	sectionPrompt := s.buildSectionPrompt(chapter, req.SectionID, req.Feedback)

	sectionSystemPrompt := s.systemPrompt + "\n\n## CONTEXT: Section Regeneration\n\nYou are regenerating ONLY one section of an existing plan. Return segments for this section only. Keep segment IDs and ordering consistent with the existing plan context provided."

	result, err := s.callClaude(ctx, sectionSystemPrompt, sectionPrompt, sectionSegmentsTool(), events)
	if err != nil {
		return fmt.Errorf("Claude API call failed: %w", err)
	}

	events <- PlanStreamEvent{Type: "complete", Message: fmt.Sprintf("Section %s regenerated", req.SectionID), Data: result}
	return nil
}

// RefineSegment refines a single segment using AI feedback.
func (s *PlanGeneratorService) RefineSegment(ctx context.Context, chapterID string, req *RefineSegmentRequest, events chan<- PlanStreamEvent) error {
	defer close(events)

	chapter, err := s.getChapter(ctx, chapterID)
	if err != nil {
		return err
	}

	if chapter.Plan == nil {
		return fmt.Errorf("chapter has no plan")
	}

	events <- PlanStreamEvent{Type: "progress", Message: fmt.Sprintf("Refining segment %s...", req.SegmentID)}

	segment := s.findSegment(chapter.Plan, req.SegmentID)
	if segment == nil {
		return fmt.Errorf("segment %s not found in plan", req.SegmentID)
	}

	segmentJSON, _ := json.Marshal(segment)
	userMessage := fmt.Sprintf("Current segment:\n```json\n%s\n```\n\nUser feedback: %s\n\nReturn the improved segment with the feedback applied.", segmentJSON, req.Feedback)

	refinementSystem := "You are refining a single video production segment. Apply the user's feedback to improve the segment. Keep the same structure and ID. Only change what the feedback requests."

	result, err := s.callClaude(ctx, refinementSystem, userMessage, refinedSegmentTool(), events)
	if err != nil {
		return fmt.Errorf("Claude API call failed: %w", err)
	}

	events <- PlanStreamEvent{Type: "complete", Message: fmt.Sprintf("Segment %s refined", req.SegmentID), Data: result}
	return nil
}

// callClaude makes a streaming call to Claude and extracts the tool_use result.
func (s *PlanGeneratorService) callClaude(ctx context.Context, systemPrompt string, userMessage string, tool anthropic.ToolUnionParam, events chan<- PlanStreamEvent) (map[string]any, error) {
	stream := s.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 16384,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(userMessage),
			),
		},
		Tools: []anthropic.ToolUnionParam{tool},
		ToolChoice: anthropic.ToolChoiceUnionParam{
			OfAny: &anthropic.ToolChoiceAnyParam{
				DisableParallelToolUse: anthropic.Bool(true),
			},
		},
	})

	// Accumulate the tool input JSON from streaming deltas
	var toolInputJSON string
	var currentBlockType string

	for stream.Next() {
		event := stream.Current()

		switch variant := event.AsAny().(type) {
		case anthropic.ContentBlockStartEvent:
			if variant.ContentBlock.Type == "tool_use" {
				currentBlockType = "tool_use"
				events <- PlanStreamEvent{Type: "progress", Message: "Claude is generating the plan..."}
			} else if variant.ContentBlock.Type == "text" {
				currentBlockType = "text"
			}

		case anthropic.ContentBlockDeltaEvent:
			if currentBlockType == "tool_use" {
				if variant.Delta.Type == "input_json_delta" {
					toolInputJSON += variant.Delta.PartialJSON
				}
			} else if currentBlockType == "text" {
				if variant.Delta.Type == "text_delta" {
					// Stream text progress to client
					if variant.Delta.Text != "" {
						events <- PlanStreamEvent{Type: "progress", Message: variant.Delta.Text}
					}
				}
			}

		case anthropic.MessageDeltaEvent:
			s.logger.Info("Claude generation complete",
				zap.String("stop_reason", string(variant.Delta.StopReason)),
			)
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("streaming error: %w", err)
	}

	if toolInputJSON == "" {
		return nil, fmt.Errorf("Claude did not return a tool call — no structured output received")
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(toolInputJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse Claude tool output: %w", err)
	}

	return result, nil
}

// getChapter fetches a chapter from the database.
func (s *PlanGeneratorService) getChapter(ctx context.Context, id string) (*models.CourseChapter, error) {
	var chapter models.CourseChapter
	if err := s.db.WithContext(ctx).First(&chapter, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("chapter not found: %w", err)
	}
	return &chapter, nil
}

// savePlan saves the generated plan to the chapter.
func (s *PlanGeneratorService) savePlan(ctx context.Context, chapter *models.CourseChapter, plan map[string]any) error {
	chapter.Plan = models.JSONB(plan)
	chapter.Status = models.ChapterStatusPlanned
	return s.db.WithContext(ctx).Save(chapter).Error
}

// buildSectionPrompt constructs a focused prompt for section-level regeneration.
func (s *PlanGeneratorService) buildSectionPrompt(chapter *models.CourseChapter, sectionID string, feedback string) string {
	// Extract the specific section from content
	sections, _ := chapter.Content["sections"].([]any)
	var targetSection any
	for _, sec := range sections {
		secMap, ok := sec.(map[string]any)
		if !ok {
			continue
		}
		if secMap["id"] == sectionID {
			targetSection = secMap
			break
		}
	}

	// Extract production settings
	chapterMeta, _ := chapter.Content["chapter"].(map[string]any)
	production, _ := chapterMeta["production"].(map[string]any)

	sectionJSON, _ := json.Marshal(targetSection)
	productionJSON, _ := json.Marshal(production)

	// Include existing plan context for consistency
	existingPlanJSON, _ := json.Marshal(chapter.Plan)

	prompt := fmt.Sprintf(`Production settings:
%s

Section to regenerate:
%s

Existing plan (for context and consistency):
%s`, productionJSON, sectionJSON, existingPlanJSON)

	if feedback != "" {
		prompt += fmt.Sprintf("\n\nUser feedback: %s", feedback)
	}

	return prompt
}

// findSegment finds a segment by ID in the plan.
func (s *PlanGeneratorService) findSegment(plan map[string]any, segmentID string) map[string]any {
	segments, ok := plan["segments"].([]any)
	if !ok {
		// Try nested under production_plan
		pp, ok := plan["production_plan"].(map[string]any)
		if ok {
			segments, _ = pp["segments"].([]any)
		}
	}

	for _, seg := range segments {
		segMap, ok := seg.(map[string]any)
		if !ok {
			continue
		}
		if segMap["id"] == segmentID {
			return segMap
		}
	}
	return nil
}

// --- Tool Definitions ---

// productionPlanTool defines the tool for full plan generation output.
func productionPlanTool() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        "save_production_plan",
			Description: anthropic.String("Save the complete production plan with all segments, shared assets, execution groups, and asset checklist."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"production_plan": map[string]any{
						"type":        "object",
						"description": "The complete production plan",
						"properties": map[string]any{
							"chapter_id":                       map[string]any{"type": "string"},
							"chapter_title":                    map[string]any{"type": "string"},
							"version":                          map[string]any{"type": "string"},
							"generated_at":                     map[string]any{"type": "string"},
							"estimated_total_duration_seconds": map[string]any{"type": "number"},
							"segments": map[string]any{
								"type":  "array",
								"items": map[string]any{"type": "object"},
							},
							"shared_assets": map[string]any{
								"type":  "array",
								"items": map[string]any{"type": "object"},
							},
							"api_calls_summary": map[string]any{"type": "object"},
							"execution_groups": map[string]any{
								"type":  "array",
								"items": map[string]any{"type": "object"},
							},
							"asset_checklist": map[string]any{
								"type":  "array",
								"items": map[string]any{"type": "object"},
							},
						},
						"required": []string{"chapter_id", "chapter_title", "segments", "shared_assets", "execution_groups", "asset_checklist"},
					},
				},
				Required: []string{"production_plan"},
			},
		},
	}
}

// sectionSegmentsTool defines the tool for section-level regeneration output.
func sectionSegmentsTool() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        "save_section_segments",
			Description: anthropic.String("Save the regenerated segments for a single section."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"section_id":   map[string]any{"type": "string"},
					"section_name": map[string]any{"type": "string"},
					"segments": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "object"},
					},
				},
				Required: []string{"section_id", "segments"},
			},
		},
	}
}

// refinedSegmentTool defines the tool for single segment refinement output.
func refinedSegmentTool() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        "save_refined_segment",
			Description: anthropic.String("Save the refined segment with user feedback applied."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"segment": map[string]any{
						"type":        "object",
						"description": "The refined segment object",
					},
				},
				Required: []string{"segment"},
			},
		},
	}
}
