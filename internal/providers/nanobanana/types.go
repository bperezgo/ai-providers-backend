package nanobanana

// GenerateRequest represents a text-to-image request via Nano Banana Pro on Fal.ai
type GenerateRequest struct {
	Prompt            string  `json:"prompt" binding:"required"`
	NegativePrompt    string  `json:"negative_prompt,omitempty"`
	ImageSize         string  `json:"image_size,omitempty"`          // "square", "square_hd", "portrait_4_3", "portrait_16_9", "landscape_4_3", "landscape_16_9"
	NumImages         int     `json:"num_images,omitempty"`          // 1-4, default 1
	NumInferenceSteps int     `json:"num_inference_steps,omitempty"` // default 28
	GuidanceScale     float64 `json:"guidance_scale,omitempty"`      // default 3.5
	Seed              int64   `json:"seed,omitempty"`
}

// FalSubmitResponse is the response from the Fal.ai queue submission
type FalSubmitResponse struct {
	RequestID   string `json:"request_id"`
	ResponseURL string `json:"response_url"`
	StatusURL   string `json:"status_url"`
	CancelURL   string `json:"cancel_url"`
}

// FalStatusResponse is the response from polling the Fal.ai status endpoint
type FalStatusResponse struct {
	Status string `json:"status"` // "IN_QUEUE", "IN_PROGRESS", "COMPLETED", "FAILED"
	Logs   []struct {
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
	} `json:"logs,omitempty"`
}

// FalResultResponse is the response from the Fal.ai result endpoint
type FalResultResponse struct {
	Images []FalImage `json:"images,omitempty"`
	Error  string     `json:"error,omitempty"`
}

// FalImage contains a generated image info
type FalImage struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	FileSize    int64  `json:"file_size"`
}

// ImageResult is stored in the job result
type ImageResult struct {
	ImageURL  string `json:"image_url"`
	ImagePath string `json:"image_path"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

const (
	// DefaultFalBaseURL is the Fal.ai queue API base URL
	DefaultFalBaseURL = "https://queue.fal.run"

	// ModelID is the Nano Banana Pro model on Fal.ai
	ModelID = "fal-ai/nano-banana-pro"

	DefaultImageSize         = "square_hd"
	DefaultNumImages         = 1
	DefaultNumInferenceSteps = 28
	DefaultGuidanceScale     = 3.5

	StatusInQueue    = "IN_QUEUE"
	StatusInProgress = "IN_PROGRESS"
	StatusCompleted  = "COMPLETED"
	StatusFailed     = "FAILED"
)
