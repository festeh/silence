package transcription

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// elevenLabsResponse represents the API response from ElevenLabs speech-to-text.
type elevenLabsResponse struct {
	LanguageCode        string  `json:"language_code"`
	LanguageProbability float64 `json:"language_probability"`
	Text                string  `json:"text"`
	Words               []word  `json:"words"`
}

// word represents timestamped word data from ElevenLabs response.
type word struct {
	Word      string  `json:"word"`
	Start     float64 `json:"start"`
	End       float64 `json:"end"`
	Punctuate bool    `json:"punctuate"`
}

// ElevenLabsProvider implements transcription using the ElevenLabs API.
type ElevenLabsProvider struct {
	apiKey string
}

// NewElevenLabsProvider creates a new ElevenLabs transcription provider.
func NewElevenLabsProvider(apiKey string) *ElevenLabsProvider {
	return &ElevenLabsProvider{
		apiKey: apiKey,
	}
}

// Transcribe processes audio data using the ElevenLabs API.
// Returns transcribed text and detected language code.
func (p *ElevenLabsProvider) Transcribe(audioData []byte, opts TranscriptionOptions) (*TranscriptionResult, error) {
	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add model_id field
	err := writer.WriteField("model_id", "scribe_v1")
	if err != nil {
		return nil, fmt.Errorf("failed to write model_id field: %v", err)
	}

	// Determine file_format and filename based on metadata
	var fileFormat, filename string
	switch opts.Metadata.Format {
	case AudioFormatPCMLE16:
		fileFormat = "pcm_s16le_16"
		filename = "audio.pcm"
	case AudioFormatWAV:
		fileFormat = "other"
		filename = "audio.wav"
	default:
		return nil, fmt.Errorf("unsupported audio format: %s", opts.Metadata.Format)
	}

	// Add file_format field
	err = writer.WriteField("file_format", fileFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to write file_format field: %v", err)
	}

	// Add language_code field if specified (not "auto" or empty)
	if opts.LanguageCode != "" && opts.LanguageCode != "auto" {
		err = writer.WriteField("language_code", opts.LanguageCode)
		if err != nil {
			return nil, fmt.Errorf("failed to write language_code field: %v", err)
		}
	}

	// Add audio file
	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	_, err = fileWriter.Write(audioData)
	if err != nil {
		return nil, fmt.Errorf("failed to write audio data: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %v", err)
	}

	// Create request to ElevenLabs API
	req, err := http.NewRequest("POST", "https://api.elevenlabs.io/v1/speech-to-text", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create ElevenLabs request: %v", err)
	}

	req.Header.Set("xi-api-key", p.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Make request to ElevenLabs
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call ElevenLabs API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ElevenLabs API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse ElevenLabs response
	var elevenLabsResp elevenLabsResponse
	err = json.NewDecoder(resp.Body).Decode(&elevenLabsResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ElevenLabs response: %v", err)
	}

	// Map to generic result
	return &TranscriptionResult{
		Text:         elevenLabsResp.Text,
		LanguageCode: elevenLabsResp.LanguageCode,
	}, nil
}
