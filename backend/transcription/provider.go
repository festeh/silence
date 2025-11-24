package transcription

import (
	"fmt"
)

// TranscriptionResult represents a generic transcription response from any provider.
type TranscriptionResult struct {
	Text         string // Transcribed text
	LanguageCode string // Detected language code (e.g., "en", "es")
}

// AudioFormat represents the format of audio data.
type AudioFormat string

const (
	// AudioFormatPCMLE16 represents 16-bit PCM audio, little-endian, 16kHz, mono
	AudioFormatPCMLE16 AudioFormat = "pcm_s16le_16"
	// AudioFormatWAV represents WAV format audio
	AudioFormatWAV AudioFormat = "wav"
)

// AudioMetadata contains information about audio format and encoding.
type AudioMetadata struct {
	Format        AudioFormat // Audio format (pcm_s16le_16 or wav)
	SampleRate    int         // Sample rate in Hz (e.g., 16000)
	Channels      int         // Number of channels (1 = mono, 2 = stereo)
	BitsPerSample int         // Bits per sample (e.g., 16)
}

// TranscriptionOptions contains optional parameters for transcription.
type TranscriptionOptions struct {
	LanguageCode string        // ISO-639-1 or ISO-639-3 language code. Use "auto" or empty string for auto-detection.
	Metadata     AudioMetadata // Audio format metadata
}

// TranscriptionProvider defines the interface for audio transcription providers.
// Providers accept raw audio data with format metadata and return transcribed text with language detection.
type TranscriptionProvider interface {
	// Transcribe processes audio data and returns transcription result.
	// Audio format is specified in opts.Metadata.
	// If opts.LanguageCode is "auto" or empty, language is auto-detected.
	// Returns an error if transcription fails.
	Transcribe(audioData []byte, opts TranscriptionOptions) (*TranscriptionResult, error)
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
func (pc *ProviderChain) Transcribe(audioData []byte, opts TranscriptionOptions) (*TranscriptionResult, error) {
	if len(pc.providers) == 0 {
		return nil, fmt.Errorf("no transcription providers configured")
	}

	var lastErr error
	for i, provider := range pc.providers {
		result, err := provider.Transcribe(audioData, opts)
		if err == nil {
			return result, nil
		}
		lastErr = fmt.Errorf("provider %d failed: %w", i+1, err)
	}

	return nil, fmt.Errorf("all providers failed, last error: %w", lastErr)
}
