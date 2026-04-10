package providers

// Registry holds all registered providers by category and name.
type Registry struct {
	textToImage       map[string]TextToImageProvider
	textToVideo       map[string]TextToVideoProvider
	imageToVideo      map[string]ImageToVideoProvider
	refImageToVideo   map[string]ReferenceImageVideoProvider
	textToAudio       map[string]TextToAudioProvider
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		textToImage:     make(map[string]TextToImageProvider),
		textToVideo:     make(map[string]TextToVideoProvider),
		imageToVideo:    make(map[string]ImageToVideoProvider),
		refImageToVideo: make(map[string]ReferenceImageVideoProvider),
		textToAudio:     make(map[string]TextToAudioProvider),
	}
}

// RegisterTextToImage adds a TextToImageProvider.
func (r *Registry) RegisterTextToImage(p TextToImageProvider) {
	r.textToImage[p.ProviderName()] = p
}

// RegisterTextToVideo adds a TextToVideoProvider.
func (r *Registry) RegisterTextToVideo(p TextToVideoProvider) {
	r.textToVideo[p.ProviderName()] = p
}

// RegisterImageToVideo adds an ImageToVideoProvider.
func (r *Registry) RegisterImageToVideo(p ImageToVideoProvider) {
	r.imageToVideo[p.ProviderName()] = p
}

// RegisterTextToAudio adds a TextToAudioProvider.
func (r *Registry) RegisterTextToAudio(p TextToAudioProvider) {
	r.textToAudio[p.ProviderName()] = p
}

// GetTextToImageProvider returns the named provider, or false if not found.
func (r *Registry) GetTextToImageProvider(name string) (TextToImageProvider, bool) {
	p, ok := r.textToImage[name]
	return p, ok
}

// GetTextToVideoProvider returns the named provider, or false if not found.
func (r *Registry) GetTextToVideoProvider(name string) (TextToVideoProvider, bool) {
	p, ok := r.textToVideo[name]
	return p, ok
}

// GetImageToVideoProvider returns the named provider, or false if not found.
func (r *Registry) GetImageToVideoProvider(name string) (ImageToVideoProvider, bool) {
	p, ok := r.imageToVideo[name]
	return p, ok
}

// RegisterReferenceImageVideo adds a ReferenceImageVideoProvider.
func (r *Registry) RegisterReferenceImageVideo(p ReferenceImageVideoProvider) {
	r.refImageToVideo[p.ProviderName()] = p
}

// GetReferenceImageVideoProvider returns the named provider, or false if not found.
func (r *Registry) GetReferenceImageVideoProvider(name string) (ReferenceImageVideoProvider, bool) {
	p, ok := r.refImageToVideo[name]
	return p, ok
}

// GetTextToAudioProvider returns the named provider, or false if not found.
func (r *Registry) GetTextToAudioProvider(name string) (TextToAudioProvider, bool) {
	p, ok := r.textToAudio[name]
	return p, ok
}
