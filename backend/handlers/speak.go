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
	Text        string `json:"text" example:"Hello world, this is a transcription"`
	AudioLength int    `json:"audio_length" example:"15"`
	Timestamp   int64  `json:"timestamp" example:"1629840000"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string `json:"error" example:"Invalid audio format"`
	Timestamp int64  `json:"timestamp" example:"1629840000"`
}

// HandleSpeak godoc
// @Summary Transcribe audio
// @Description Accepts audio in multipart/form-data (WAV file) or application/json (PCM data) format and returns transcribed text using the configured transcription provider
// @Tags Audio
// @Accept multipart/form-data
// @Accept json
// @Produce json
// @Param audio formData file false "WAV audio file (multipart/form-data only, max 32MB)"
// @Param body body AudioTranscriptionRequest false "PCM audio data (application/json only)"
// @Success 200 {object} SuccessResponse "Transcription successful"
// @Failure 400 {object} ErrorResponse "Bad request (invalid format, empty audio, etc.)"
// @Router /speak [post]
func HandleSpeak(re *core.RequestEvent, app core.App, provider transcription.TranscriptionProvider) error {
	logger.Info("Starting audio processing request")

	// Set JSON response headers
	re.Response.Header().Set("Content-Type", "application/json")
	re.Response.Header().Set("Access-Control-Allow-Origin", "*")

	// Check if it's multipart form data (frontend) or JSON (CLI via backend)
	contentType := re.Request.Header.Get("Content-Type")

	if contentType == "application/json" {
		return handleJSONRequest(re, app, provider)
	}

	// Handle multipart form data (frontend)
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

	// Read the WAV file data
	wavData, err := io.ReadAll(file)
	if err != nil {
		logger.Error("Failed to read audio file", "error", err)
		return sendJSONError(re, "Failed to read audio file")
	}

	if len(wavData) == 0 {
		logger.Error("Audio file is empty")
		return sendJSONError(re, "audio file is empty")
	}

	// Calculate audio length from WAV data (assuming 16kHz, 1 channel, 16-bit)
	// WAV header is 44 bytes, so subtract that from total size
	audioDataSize := len(wavData) - 44
	if audioDataSize < 0 {
		audioDataSize = 0
	}
	audioLength := calculateAudioLength(audioDataSize)

	// Use provider to transcribe WAV
	logger.Info("Starting WAV transcription")
	result, err := provider.Transcribe(wavData)
	if err != nil {
		logger.Error("Failed to transcribe WAV audio", "error", err)
		return sendJSONError(re, fmt.Sprintf("Failed to transcribe audio: %v", err))
	}

	// Send JSON response immediately after transcription
	response := map[string]any{
		"text":         result.Text,
		"audio_length": audioLength,
		"timestamp":    time.Now().Unix(),
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return sendJSONError(re, "Failed to encode response")
	}

	re.Response.WriteHeader(http.StatusOK)
	re.Response.Write(jsonData)

	// Handle compression and database storage asynchronously
	go saveAudioToDatabase(app, wavData, result.Text)

	return nil
}

// saveAudioToDatabase compresses WAV audio and stores it in the PocketBase database.
// This function runs asynchronously in a goroutine to avoid blocking the response.
// The audio is compressed and base64-encoded before storage in the 'silence' collection.
func saveAudioToDatabase(app core.App, wavData []byte, transcriptionText string) {
	logger.Info("Starting background compression and database storage")

	// Compress the audio data
	logger.Info("Compressing audio data", "original_size", len(wavData))
	compressedData, err := compression.CompressAudio(wavData)
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

// AudioTranscriptionRequest represents a JSON request for audio transcription.
// Used by CLI tools to send raw PCM audio data for transcription.
type AudioTranscriptionRequest struct {
	PCMData []byte `json:"pcm_data"`
}

// handleJSONRequest processes JSON-based audio transcription requests.
// Accepts raw PCM data and returns transcription without database storage.
// This is primarily used by CLI tools that send PCM data directly.
func handleJSONRequest(re *core.RequestEvent, app core.App, provider transcription.TranscriptionProvider) error {
	var req AudioTranscriptionRequest
	err := json.NewDecoder(re.Request.Body).Decode(&req)
	if err != nil {
		logger.Error("Failed to decode audio transcription request", "error", err)
		return sendJSONError(re, "Failed to decode audio transcription request")
	}

	if len(req.PCMData) == 0 {
		logger.Error("PCM data is empty")
		return sendJSONError(re, "pcm_data is required")
	}

	// Calculate audio length from PCM data (16kHz, 1 channel, 16-bit)
	audioLength := calculateAudioLength(len(req.PCMData))

	// Convert PCM to WAV
	logger.Info("Converting PCM to WAV", "data_size", len(req.PCMData))
	wavData, err := transcription.PcmToWav(req.PCMData, 16000, 1, 16)
	if err != nil {
		logger.Error("Failed to convert PCM to WAV", "error", err)
		return sendJSONError(re, fmt.Sprintf("Failed to convert PCM to WAV: %v", err))
	}

	// Use provider to transcribe WAV
	logger.Info("Starting PCM transcription")
	result, err := provider.Transcribe(wavData)
	if err != nil {
		logger.Error("Failed to transcribe PCM audio", "error", err)
		return sendJSONError(re, fmt.Sprintf("Failed to transcribe audio: %v", err))
	}

	// Send JSON response
	response := map[string]any{
		"text":         result.Text,
		"audio_length": audioLength,
		"timestamp":    time.Now().Unix(),
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return sendJSONError(re, "Failed to encode response")
	}

	re.Response.WriteHeader(http.StatusOK)
	re.Response.Write(jsonData)
	return nil
}
