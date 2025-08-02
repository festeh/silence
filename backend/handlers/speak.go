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

	// Set SSE headers
	re.Response.Header().Set("Content-Type", "text/event-stream")
	re.Response.Header().Set("Cache-Control", "no-cache")
	re.Response.Header().Set("Connection", "keep-alive")
	re.Response.Header().Set("Access-Control-Allow-Origin", "*")

	// Check if it's multipart form data (frontend) or JSON (CLI via backend)
	contentType := re.Request.Header.Get("Content-Type")

	if contentType == "application/json" {
		return handleJSONRequest(re, elevenlabsAPIKey)
	}

	// Handle multipart form data (frontend)
	logger.Info("Processing multipart form data")
	err := re.Request.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		logger.Error("Failed to parse multipart form", "error", err)
		return sendSSEError(re, "Invalid multipart form")
	}

	// Get the audio file from the form
	file, _, err := re.Request.FormFile("audio")
	if err != nil {
		logger.Error("Failed to get audio file from form", "error", err)
		return sendSSEError(re, "audio file is required")
	}
	defer file.Close()

	// Read the WAV file data
	wavData, err := io.ReadAll(file)
	if err != nil {
		logger.Error("Failed to read audio file", "error", err)
		return sendSSEError(re, "Failed to read audio file")
	}

	if len(wavData) == 0 {
		logger.Error("Audio file is empty")
		return sendSSEError(re, "audio file is empty")
	}

	// Compress the audio data
	logger.Info("Compressing audio data", "original_size", len(wavData))
	compressedData, err := compression.CompressAudio(wavData)
	if err != nil {
		logger.Error("Failed to compress audio", "error", err)
		return sendSSEError(re, fmt.Sprintf("Failed to compress audio: %v", err))
	}

	// Encode compressed audio to base64
	base64Data := base64.StdEncoding.EncodeToString(compressedData)

	// Use shared transcription function for WAV
	logger.Info("Starting WAV transcription")
	result, err := transcription.TranscribeWAV(wavData, elevenlabsAPIKey)
	if err != nil {
		logger.Error("Failed to transcribe WAV audio", "error", err)
		return sendSSEError(re, fmt.Sprintf("Failed to transcribe audio: %v", err))
	}

	// Save compressed audio and transcription to database using PocketBase
	collection, err := app.FindCollectionByNameOrId("silence")
	if err != nil {
		logger.Error("Failed to find silence collection", "error", err)
		return sendSSEError(re, "Database collection not found")
	}

	record := core.NewRecord(collection)
	record.Set("audio", base64Data)
	record.Set("result", result.Text)

	if err := app.Save(record); err != nil {
		logger.Error("Failed to save record to database", "error", err)
		return sendSSEError(re, "Failed to save record to database")
	}

	// Send completion event
	logger.Info("Audio processing completed successfully", "record_id", record.Id)
	return sendSSEEvent(re, "complete", map[string]any{
		"record_id":        record.Id,
		"transcribed_text": result.Text,
		"timestamp":        time.Now().Unix(),
	})
}

func sendSSEEvent(re *core.RequestEvent, event string, data any) error {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(re.Response, "event: %s\n", event)
	fmt.Fprintf(re.Response, "data: %s\n\n", jsonData)
	flushSSE(re.Response)
	return nil
}

func sendSSEError(re *core.RequestEvent, message string) error {
	errorData := map[string]any{
		"error":     message,
		"timestamp": time.Now().Unix(),
	}
	return sendSSEEvent(re, "error", errorData)
}

type AudioTranscriptionRequest struct {
	PCMData []byte `json:"pcm_data"`
}

func handleJSONRequest(re *core.RequestEvent, apiKey string) error {
	var req AudioTranscriptionRequest
	err := json.NewDecoder(re.Request.Body).Decode(&req)
	if err != nil {
		logger.Error("Failed to decode audio transcription request", "error", err)
		return sendSSEError(re, "Failed to decode audio transcription request")
	}

	if len(req.PCMData) == 0 {
		logger.Error("PCM data is empty")
		return sendSSEError(re, "pcm_data is required")
	}

	// Use shared transcription function for PCM
	logger.Info("Starting PCM transcription", "data_size", len(req.PCMData))
	result, err := transcription.TranscribePCM(req.PCMData, apiKey)
	if err != nil {
		logger.Error("Failed to transcribe PCM audio", "error", err)
		return sendSSEError(re, fmt.Sprintf("Failed to transcribe audio: %v", err))
	}

	// Send completion event
	return sendSSEEvent(re, "complete", map[string]any{
		"result":    result,
		"timestamp": time.Now().Unix(),
	})
}

func flushSSE(w http.ResponseWriter) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
