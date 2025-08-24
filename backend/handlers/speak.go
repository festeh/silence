package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"silence-backend/compression"
	"silence-backend/logger"
	"silence-backend/transcription"
)

func HandleSpeak(re *core.RequestEvent, app core.App, elevenlabsAPIKey string) error {
	logger.Info("Starting audio processing request")

	// Set JSON response headers
	re.Response.Header().Set("Content-Type", "application/json")
	re.Response.Header().Set("Access-Control-Allow-Origin", "*")

	// Check if it's multipart form data (frontend) or JSON (CLI via backend)
	contentType := re.Request.Header.Get("Content-Type")

	if contentType == "application/json" {
		return handleJSONRequest(re, app, elevenlabsAPIKey)
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

	// Use shared transcription function for WAV
	logger.Info("Starting WAV transcription")
	result, err := transcription.TranscribeWAV(wavData, elevenlabsAPIKey)
	if err != nil {
		logger.Error("Failed to transcribe WAV audio", "error", err)
		return sendJSONError(re, fmt.Sprintf("Failed to transcribe audio: %v", err))
	}

	// Send JSON response immediately after transcription
	response := map[string]any{
		"transcribed_text": result.Text,
		"timestamp":        time.Now().Unix(),
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

type AudioTranscriptionRequest struct {
	PCMData []byte `json:"pcm_data"`
}

func handleJSONRequest(re *core.RequestEvent, app core.App, apiKey string) error {
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

	// Use shared transcription function for PCM
	logger.Info("Starting PCM transcription", "data_size", len(req.PCMData))
	result, err := transcription.TranscribePCM(req.PCMData, apiKey)
	if err != nil {
		logger.Error("Failed to transcribe PCM audio", "error", err)
		return sendJSONError(re, fmt.Sprintf("Failed to transcribe audio: %v", err))
	}

	// Send JSON response
	response := map[string]any{
		"result":    result,
		"timestamp": time.Now().Unix(),
	}
	
	jsonData, err := json.Marshal(response)
	if err != nil {
		return sendJSONError(re, "Failed to encode response")
	}
	
	re.Response.WriteHeader(http.StatusOK)
	re.Response.Write(jsonData)
	return nil
}

