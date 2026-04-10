package judge

// ImageEvalRequest contains the inputs needed to evaluate a generated image.
type ImageEvalRequest struct {
	OriginalPrompt string // the prompt sent to NanoBanana
	ImagePath      string // local path to the generated image
	UseCase        string // "marketing-banner", "social-post", "product-shot"
}

// ImageEvalScore contains the VLM judge's evaluation of a generated image.
type ImageEvalScore struct {
	PromptAdherence  int    `json:"prompt_adherence"`  // 1-5: does it match the prompt?
	ArtifactScore    int    `json:"artifact_score"`    // 1-5: 5 = no artifacts
	CompositionScore int    `json:"composition_score"` // 1-5: usable for the stated use case?
	OverallScore     int    `json:"overall_score"`     // 1-5: would you use this image?
	Reasoning        string `json:"reasoning"`
	SuggestedRetry   bool   `json:"suggested_retry"` // true if overall < 3
}
