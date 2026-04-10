package grok

// GenerateRequest represents an image-to-video request via xAI Grok
type GenerateRequest struct {
	SourceImage string `json:"source_image,omitempty"` // URL of the source image (image-to-video)
	Prompt      string `json:"prompt,omitempty"`
	Duration    int    `json:"duration,omitempty"`     // seconds, 1–15
	AspectRatio string `json:"aspect_ratio,omitempty"` // "16:9", "9:16", "1:1", etc.
	Resolution  string `json:"resolution,omitempty"`   // "480p" or "720p"
}

// SubmitResponse is returned by the POST /videos/generations endpoint
type SubmitResponse struct {
	RequestID string `json:"request_id"`
}

// GenerateResponse is returned by the GET /videos/{request_id} polling endpoint.
type GenerateResponse struct {
	Status string          `json:"status"` // "pending", "done", "failed", "expired"
	Model  string          `json:"model,omitempty"`
	Video  *GrokVideo      `json:"video,omitempty"`
	Error  *GenerateError  `json:"error,omitempty"`
}

// GenerateError represents the structured error object returned by xAI on failure.
type GenerateError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GrokVideo contains the generated video info
type GrokVideo struct {
	URL      string `json:"url"`
	Duration int    `json:"duration,omitempty"`
}

// ImageURL wraps a URL string into the struct format expected by the xAI API.
type ImageURL struct {
	URL string `json:"url"`
}

// ImageToVideoGenerateRequest represents an image-to-video request where the
// source image becomes the first frame of the generated video.
type ImageToVideoGenerateRequest struct {
	Image       ImageURL `json:"image"`                  // Public URL or base64 data URI of the source image
	Prompt      string   `json:"prompt,omitempty"`
	Duration    int      `json:"duration,omitempty"`     // seconds, 1–15
	AspectRatio string   `json:"aspect_ratio,omitempty"`
	Resolution  string   `json:"resolution,omitempty"`   // "480p" or "720p"
}

// ReferenceImage is a single reference image entry for the reference-image video API.
type ReferenceImage struct {
	URL string `json:"url"` // Public HTTPS URL or base64 data URI
}

// ReferenceVideoGenerateRequest represents a video generation request that uses
// reference images to influence the visual style without locking the first frame.
type ReferenceVideoGenerateRequest struct {
	Prompt          string           `json:"prompt"`
	ReferenceImages []ReferenceImage `json:"reference_images"`
	Duration        int              `json:"duration,omitempty"`
	AspectRatio     string           `json:"aspect_ratio,omitempty"`
	Resolution      string           `json:"resolution,omitempty"`
}

const (
	// DefaultXAIBaseURL is the xAI API base URL
	DefaultXAIBaseURL = "https://api.x.ai/v1"

	// ModelID is the Grok video generation model
	ModelID = "grok-imagine-video"

	DefaultAspectRatio = "16:9"
	DefaultResolution  = "480p"
	DefaultDuration    = 5 // seconds

	StatusPending = "pending"
	StatusDone    = "done"
	StatusFailed  = "failed"
	StatusExpired = "expired"
)
