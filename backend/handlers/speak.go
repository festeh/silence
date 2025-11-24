package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"silence-backend/compression"
	"silence-backend/logger"
	"silence-backend/transcription"
)

// SuccessResponse represents a successful transcription response
type SuccessResponse struct {
	Text         string `json:"text" example:"Hello world, this is a transcription"`
	LanguageCode string `json:"language_code" example:"en"`
	AudioLength  int    `json:"audio_length" example:"15"`
	Timestamp    int64  `json:"timestamp" example:"1629840000"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string `json:"error" example:"Invalid audio format"`
	Timestamp int64  `json:"timestamp" example:"1629840000"`
}

// HandleSpeak godoc
// @Summary Transcribe audio
// @Description Accepts audio in multipart/form-data format and returns transcribed text using the configured transcription provider. Supports both PCM and WAV formats.
// @Tags Audio
// @Accept multipart/form-data
// @Produce json
// @Param audio formData file true "Audio file (PCM or WAV format, max 32MB)"
// @Param file_format formData string false "Audio format: 'pcm_s16le_16' or 'wav'. Defaults to 'pcm_s16le_16' for lower latency. Use pcm_s16le_16 for 16-bit PCM at 16kHz, mono, little-endian."
// @Param language_code formData string false "ISO-639-1 or ISO-639-3 language code. Use 'auto' or omit for auto-detection. Examples: 'en', 'es', 'fr'"
// @Success 200 {object} SuccessResponse "Transcription successful"
// @Failure 400 {object} ErrorResponse "Bad request (invalid format, empty audio, etc.)"
// @Router /speak [post]
func HandleSpeak(re *core.RequestEvent, app core.App, provider transcription.TranscriptionProvider) error {
	logger.Info("Starting audio processing request")

	// Set JSON response headers
	re.Response.Header().Set("Content-Type", "application/json")
	re.Response.Header().Set("Access-Control-Allow-Origin", "*")

	// Handle multipart form data
	logger.Info("Processing multipart form data")
	err := re.Request.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		logger.Error("Failed to parse multipart form", "error", err)
		return sendJSONError(re, "Invalid multipart form")
	}

	// Get the audio file from the form
	file, _, err := re.Request.FormFile("audio")
	if err != nil {
		logger.Error("Failed to get audio file from form", "error", err)
		return sendJSONError(re, "audio file is required")
	}
	defer file.Close()

	// Get optional language_code from form
	languageCode := re.Request.FormValue("language_code")
	if languageCode == "" {
		languageCode = "auto"
	}

	// Get optional file_format from form (default to pcm_s16le_16)
	fileFormat := re.Request.FormValue("file_format")
	if fileFormat == "" {
		fileFormat = "pcm_s16le_16"
	}

	// Read the audio file data
	audioData, err := io.ReadAll(file)
	if err != nil {
		logger.Error("Failed to read audio file", "error", err)
		return sendJSONError(re, "Failed to read audio file")
	}

	if len(audioData) == 0 {
		logger.Error("Audio file is empty")
		return sendJSONError(re, "audio file is empty")
	}

	// Calculate audio length (assuming 16kHz, 1 channel, 16-bit PCM)
	audioLength := calculateAudioLength(len(audioData))

	// Use provider to transcribe audio
	logger.Info("Starting audio transcription", "language_code", languageCode, "file_format", fileFormat)
	result, err := provider.Transcribe(audioData, transcription.TranscriptionOptions{
		LanguageCode: languageCode,
		Metadata: transcription.AudioMetadata{
			Format:        transcription.AudioFormat(fileFormat),
			SampleRate:    16000,
			Channels:      1,
			BitsPerSample: 16,
		},
	})
	if err != nil {
		logger.Error("Failed to transcribe audio", "error", err)
		return sendJSONError(re, fmt.Sprintf("Failed to transcribe audio: %v", err))
	}

	// Send JSON response immediately after transcription
	response := map[string]any{
		"text":          result.Text,
		"language_code": result.LanguageCode,
		"audio_length":  audioLength,
		"timestamp":     time.Now().Unix(),
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return sendJSONError(re, "Failed to encode response")
	}

	re.Response.WriteHeader(http.StatusOK)
	re.Response.Write(jsonData)

	// Handle compression and database storage asynchronously
	go saveAudioToDatabase(app, audioData, result.Text)

	return nil
}

// saveAudioToDatabase compresses audio data and stores it in the PocketBase database.
// This function runs asynchronously in a goroutine to avoid blocking the response.
// The audio is compressed and base64-encoded before storage in the 'silence' collection.
func saveAudioToDatabase(app core.App, audioData []byte, transcriptionText string) {
	logger.Info("Starting background compression and database storage")

	// Compress the audio data
	logger.Info("Compressing audio data", "original_size", len(audioData))
	compressedData, err := compression.CompressAudio(audioData)
	if err != nil {
		logger.Error("Failed to compress audio in background", "error", err)
		return
	}

	// Encode compressed audio to base64
	base64Data := base64.StdEncoding.EncodeToString(compressedData)

	// Save compressed audio and transcription to database using PocketBase
	collection, err := app.FindCollectionByNameOrId("silence")
	if err != nil {
		logger.Error("Failed to find silence collection in background", "error", err)
		return
	}

	record := core.NewRecord(collection)
	record.Set("audio", base64Data)
	record.Set("result", transcriptionText)

	if err := app.Save(record); err != nil {
		logger.Error("Failed to save record to database in background", "error", err)
		return
	}

	logger.Info("Background processing completed successfully", "record_id", record.Id)
}

// sendJSONError sends a JSON-formatted error response with a 400 status code.
// The response includes the error message and current timestamp.
func sendJSONError(re *core.RequestEvent, message string) error {
	errorData := map[string]any{
		"error":     message,
		"timestamp": time.Now().Unix(),
	}

	jsonData, err := json.Marshal(errorData)
	if err != nil {
		re.Response.WriteHeader(http.StatusInternalServerError)
		re.Response.Write([]byte(`{"error": "Internal server error"}`))
		return nil
	}

	re.Response.WriteHeader(http.StatusBadRequest)
	re.Response.Write(jsonData)
	return nil
}

// calculateAudioLength calculates audio duration in seconds from raw audio data size.
// Assumes 16kHz sample rate, 1 channel (mono), and 16-bit depth.
func calculateAudioLength(dataSize int) int {
	sampleRate := 16000
	channels := 1
	bitsPerSample := 16
	bytesPerSample := bitsPerSample / 8
	totalSamples := dataSize / (bytesPerSample * channels)
	return int(math.Ceil(float64(totalSamples) / float64(sampleRate)))
}
