package transcription

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// chutesRequest represents the request body for Chutes AI transcription API.
type chutesRequest struct {
	AudioB64 string  `json:"audio_b64"`
	Language *string `json:"language,omitempty"`
}

// chutesSegment represents a transcription segment from Chutes AI response.
type chutesSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// ChutesProvider implements transcription using the Chutes AI API.
type ChutesProvider struct {
	apiKey string
}

// NewChutesProvider creates a new Chutes AI transcription provider.
func NewChutesProvider(apiKey string) *ChutesProvider {
	return &ChutesProvider{
		apiKey: apiKey,
	}
}

// Transcribe processes audio data using the Chutes AI API.
// Returns transcribed text. Language detection is not supported by this provider.
func (p *ChutesProvider) Transcribe(audioData []byte, opts TranscriptionOptions) (*TranscriptionResult, error) {
	// Convert PCM to WAV if needed (Chutes API expects WAV format)
	if opts.Metadata.Format == AudioFormatPCMLE16 {
		wavData, err := PcmToWav(audioData, opts.Metadata.SampleRate, opts.Metadata.Channels, opts.Metadata.BitsPerSample)
		if err != nil {
			return nil, fmt.Errorf("failed to convert PCM to WAV: %v", err)
		}
		audioData = wavData
	}

	// Base64 encode the audio data
	audioB64 := base64.StdEncoding.EncodeToString(audioData)

	// Build request body (language omitted if nil due to omitempty)
	reqBody := chutesRequest{
		AudioB64: audioB64,
	}

	// Set language if specified (not "auto" or empty)
	if opts.LanguageCode != "" && opts.LanguageCode != "auto" {
		reqBody.Language = &opts.LanguageCode
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Create request to Chutes AI API
	req, err := http.NewRequest("POST", "https://chutes-whisper-large-v3.chutes.ai/transcribe", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create Chutes request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Make request to Chutes AI
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Chutes API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Chutes response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Chutes API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse JSON array response
	var segments []chutesSegment
	if err := json.Unmarshal(body, &segments); err != nil {
		return nil, fmt.Errorf("failed to parse Chutes response: %v", err)
	}

	// Concatenate all segment texts
	var fullText string
	for _, seg := range segments {
		fullText += seg.Text
	}

	return &TranscriptionResult{
		Text:         strings.TrimSpace(fullText),
		LanguageCode: "", // Chutes API doesn't return language code
	}, nil
}
