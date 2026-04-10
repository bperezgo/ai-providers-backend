package providers

import "context"

// EvalScore contains the VLM judge's evaluation of a generated image.
type EvalScore struct {
	PromptAdherence  int    `json:"prompt_adherence"`
	ArtifactScore    int    `json:"artifact_score"`
	CompositionScore int    `json:"composition_score"`
	OverallScore     int    `json:"overall_score"`
	Reasoning        string `json:"reasoning"`
	SuggestedRetry   bool   `json:"suggested_retry"`
}

// ImageResult is the output of any image-generating provider.
type ImageResult struct {
	LocalPath   string     // Absolute path to file on disk: ./outputs/dalle/xxx.png
	ProviderURL string     // Original CDN URL from the provider (may expire)
	Width       int
	Height      int
	MimeType    string     // "image/png", "image/jpeg"
	EvalScore   *EvalScore // Optional VLM judge evaluation (nil if not evaluated)
}

// VideoResult is the output of any video-generating provider.
type VideoResult struct {
	LocalPath   string  // Absolute path to file on disk: ./outputs/kling/xxx.mp4
	ProviderURL string  // Original CDN URL from the provider (may expire)
	Width       int
	Height      int
	Duration    float64 // seconds
	MimeType    string  // "video/mp4"
}

// AudioResult is the output of any audio-generating provider.
type AudioResult struct {
	LocalPath string  // Absolute path to file on disk: ./outputs/elevenlabs/xxx.mp3
	Duration  float64 // seconds
	MimeType  string  // "audio/mpeg"
}

// TextToImageOptions configures a text-to-image generation request.
type TextToImageOptions struct {
	Size    string // "1024x1024", "1792x1024", etc.
	Quality string // "standard", "hd"
	Style   string // "vivid", "natural"
	N       int    // number of images
}

// TextToVideoOptions configures a text-to-video generation request.
type TextToVideoOptions struct {
	AspectRatio string // "16:9", "9:16", "1:1"
	Duration    int    // seconds
	Loop        bool
}

// ImageToVideoOptions configures an image-to-video generation request.
type ImageToVideoOptions struct {
	AspectRatio string // "16:9", "9:16", "1:1"
	Duration    string // "5" or "10"
	Mode        string // "standard" or "pro"
}

// TextToAudioOptions configures a text-to-audio generation request.
type TextToAudioOptions struct {
	VoiceID         string
	ModelID         string
	Stability       float64
	SimilarityBoost float64
}

// ImageToImageOptions configures an image-to-image transformation request.
type ImageToImageOptions struct {
	Strength float64 // 0.0–1.0, how much to change the original image
	Width    int
	Height   int
}

// VideoToVideoOptions configures a video-to-video transformation request.
type VideoToVideoOptions struct {
	AspectRatio string
	Duration    string
}

// TextToImageProvider generates images from text prompts.
// Implemented by: NanoBanana
type TextToImageProvider interface {
	GenerateImage(ctx context.Context, prompt string, opts TextToImageOptions) (*ImageResult, error)
	ProviderName() string
}

// TextToVideoProvider generates videos from text prompts.
type TextToVideoProvider interface {
	GenerateVideo(ctx context.Context, prompt string, opts TextToVideoOptions) (*VideoResult, error)
	ProviderName() string
}

// ImageToVideoProvider animates a still image into a video.
// Implemented by: Grok (grok, grok-i2v)
type ImageToVideoProvider interface {
	GenerateVideo(ctx context.Context, imageURL string, prompt string, opts ImageToVideoOptions) (*VideoResult, error)
	ProviderName() string
}

// TextToAudioProvider generates audio from text.
// Implemented by: ElevenLabs
type TextToAudioProvider interface {
	GenerateAudio(ctx context.Context, text string, opts TextToAudioOptions) (*AudioResult, error)
	ProviderName() string
}

// ReferenceImageVideoOptions configures a reference-image video generation request.
type ReferenceImageVideoOptions struct {
	AspectRatio string // "16:9", "9:16", "1:1"
	Duration    string // "5" or "10"
	Mode        string // resolution: "480p" or "720p"
}

// ReferenceImageVideoProvider generates videos influenced by reference images.
// Reference images affect the visual style without locking the first frame.
// Implemented by: Grok (grok-ref)
type ReferenceImageVideoProvider interface {
	GenerateVideoFromReferences(ctx context.Context, prompt string, referenceImageURLs []string, opts ReferenceImageVideoOptions) (*VideoResult, error)
	ProviderName() string
}

// ImageToImageProvider transforms an image based on a prompt.
// Not yet implemented — see Plan 04.
type ImageToImageProvider interface {
	TransformImage(ctx context.Context, imageURL string, prompt string, opts ImageToImageOptions) (*ImageResult, error)
	ProviderName() string
}

// VideoToVideoProvider transforms a video based on a prompt.
// Not yet implemented — see Plan 04.
type VideoToVideoProvider interface {
	TransformVideo(ctx context.Context, videoURL string, prompt string, opts VideoToVideoOptions) (*VideoResult, error)
	ProviderName() string
}
