package transcription

import (
	"bytes"
	"encoding/binary"
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

// Transcribe processes WAV audio data using the ElevenLabs API.
// Returns transcribed text and detected language code.
func (p *ElevenLabsProvider) Transcribe(wavData []byte) (*TranscriptionResult, error) {
	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add model_id field
	err := writer.WriteField("model_id", "scribe_v1")
	if err != nil {
		return nil, fmt.Errorf("failed to write model_id field: %v", err)
	}

	// Add audio file
	fileWriter, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	_, err = fileWriter.Write(wavData)
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

// PcmToWav converts PCM S16LE data to WAV format.
// Assumes 16kHz sample rate, 1 channel (mono), and 16-bit depth.
func PcmToWav(pcmData []byte, sampleRate, channels, bitsPerSample int) ([]byte, error) {
	var buf bytes.Buffer

	// WAV header
	dataSize := len(pcmData)
	fileSize := 36 + dataSize

	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(fileSize))
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16)) // fmt chunk size
	binary.Write(&buf, binary.LittleEndian, uint16(1))  // PCM format
	binary.Write(&buf, binary.LittleEndian, uint16(channels))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))

	byteRate := sampleRate * channels * bitsPerSample / 8
	binary.Write(&buf, binary.LittleEndian, uint32(byteRate))

	blockAlign := channels * bitsPerSample / 8
	binary.Write(&buf, binary.LittleEndian, uint16(blockAlign))
	binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample))

	// data chunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataSize))
	buf.Write(pcmData)

	return buf.Bytes(), nil
}
