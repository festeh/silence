package transcription

import (
	"fmt"
)

// TranscriptionResult represents a generic transcription response from any provider.
type TranscriptionResult struct {
	Text         string // Transcribed text
	LanguageCode string // Detected language code (e.g., "en", "es")
}

// TranscriptionProvider defines the interface for audio transcription providers.
// Providers accept WAV audio data and return transcribed text with language detection.
type TranscriptionProvider interface {
	// Transcribe processes WAV audio data and returns transcription result.
	// Returns an error if transcription fails.
	Transcribe(wavData []byte) (*TranscriptionResult, error)
}

// ProviderChain implements a fallback mechanism for multiple transcription providers.
// It attempts transcription with each provider in sequence until one succeeds.
type ProviderChain struct {
	providers []TranscriptionProvider
}

// NewProviderChain creates a new provider chain with the given providers.
// Providers are tried in the order they are provided.
func NewProviderChain(providers ...TranscriptionProvider) *ProviderChain {
	return &ProviderChain{
		providers: providers,
	}
}

// Transcribe attempts transcription with each provider until one succeeds.
// Returns the result from the first successful provider.
// Returns an error only if all providers fail.
func (pc *ProviderChain) Transcribe(wavData []byte) (*TranscriptionResult, error) {
	if len(pc.providers) == 0 {
		return nil, fmt.Errorf("no transcription providers configured")
	}

	var lastErr error
	for i, provider := range pc.providers {
		result, err := provider.Transcribe(wavData)
		if err == nil {
			return result, nil
		}
		lastErr = fmt.Errorf("provider %d failed: %w", i+1, err)
	}

	return nil, fmt.Errorf("all providers failed, last error: %w", lastErr)
}
