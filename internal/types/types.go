package types

// ProviderName identifies the AI provider for a job.
type ProviderName string

const (
	ProviderElevenLabs ProviderName = "elevenlabs"
	ProviderGrok       ProviderName = "grok"
	ProviderGrokI2V    ProviderName = "grok-i2v"
	ProviderGrokRef    ProviderName = "grok-ref"
	ProviderNanaBanana ProviderName = "nanobanana"
)
