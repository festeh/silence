package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"silence-backend/compression"
	"silence-backend/database"
	"silence-backend/env"
	"silence-backend/logger"
	"silence-backend/transcription"
)

func HandleSpeak(w http.ResponseWriter, r *http.Request, db *database.Client, env *env.Environment) {
	logger.Info("Starting audio processing request")
	
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get ElevenLabs API key from environment
	apiKey := env.ElevenLabsAPIKey

	// Send initial event
	sendSSEEvent(w, "start", map[string]interface{}{
		"message": "Starting audio processing",
		"timestamp": time.Now().Unix(),
	})

	// Check if it's multipart form data (frontend) or JSON (CLI via backend)
	contentType := r.Header.Get("Content-Type")

	if contentType == "application/json" {
		handleJSONRequest(w, r, apiKey)
		return
	}

	// Handle multipart form data (frontend)
	logger.Info("Processing multipart form data")
	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		logger.Error("Failed to parse multipart form", "error", err)
		sendSSEError(w, "Invalid multipart form")
		return
	}

	// Get the audio file from the form
	file, _, err := r.FormFile("audio")
	if err != nil {
		logger.Error("Failed to get audio file from form", "error", err)
		sendSSEError(w, "audio file is required")
		return
	}
	defer file.Close()

	// Send processing event
	sendSSEEvent(w, "processing", map[string]interface{}{
		"message": "Reading audio file",
		"timestamp": time.Now().Unix(),
	})

	// Read the WAV file data
	wavData, err := io.ReadAll(file)
	if err != nil {
		logger.Error("Failed to read audio file", "error", err)
		sendSSEError(w, "Failed to read audio file")
		return
	}

	if len(wavData) == 0 {
		logger.Error("Audio file is empty")
		sendSSEError(w, "audio file is empty")
		return
	}

	// Send compression event
	sendSSEEvent(w, "compressing", map[string]interface{}{
		"message": "Compressing audio data",
		"original_size": len(wavData),
		"timestamp": time.Now().Unix(),
	})

	// Compress the audio data
	logger.Info("Compressing audio data", "original_size", len(wavData))
	compressedData, err := compression.CompressAudio(wavData)
	if err != nil {
		logger.Error("Failed to compress audio", "error", err)
		sendSSEError(w, fmt.Sprintf("Failed to compress audio: %v", err))
		return
	}

	// Encode compressed audio to base64
	base64Data := base64.StdEncoding.EncodeToString(compressedData)

	// Send database save event
	sendSSEEvent(w, "saving", map[string]interface{}{
		"message": "Saving compressed audio to database",
		"compressed_size": len(compressedData),
		"base64_size": len(base64Data),
		"timestamp": time.Now().Unix(),
	})

	// Send transcription event
	sendSSEEvent(w, "transcribing", map[string]interface{}{
		"message": "Transcribing audio",
		"timestamp": time.Now().Unix(),
	})

	// Use shared transcription function for WAV
	logger.Info("Starting WAV transcription")
	result, err := transcription.TranscribeWAV(wavData, apiKey)
	if err != nil {
		logger.Error("Failed to transcribe WAV audio", "error", err)
		sendSSEError(w, fmt.Sprintf("Failed to transcribe audio: %v", err))
		return
	}

	// Save compressed audio and transcription to database
	record, err := db.CreateRecord(base64Data, result.Text)
	if err != nil {
		logger.Error("Failed to save record to database", "error", err)
		sendSSEError(w, "Failed to save record to database")
		return
	}

	// Send completion event
	logger.Info("Audio processing completed successfully", "record_id", record.ID)
	sendSSEEvent(w, "complete", map[string]any{
		"record_id": record.ID,
		"transcribed_text": result.Text,
		"result": record,
		"timestamp": time.Now().Unix(),
	})
}

func sendSSEEvent(w http.ResponseWriter, event string, data any) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flushSSE(w)
}

func sendSSEError(w http.ResponseWriter, message string) {
	errorData := map[string]any{
		"error": message,
		"timestamp": time.Now().Unix(),
	}
	sendSSEEvent(w, "error", errorData)
}

type AudioTranscriptionRequest struct {
	PCMData []byte `json:"pcm_data"`
}

func handleJSONRequest(w http.ResponseWriter, r *http.Request, apiKey string) {
	var req AudioTranscriptionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.Error("Failed to decode audio transcription request", "error", err)
		sendSSEError(w, "Failed to decode audio transcription request")
		return
	}

	if len(req.PCMData) == 0 {
		logger.Error("PCM data is empty")
		sendSSEError(w, "pcm_data is required")
		return
	}

	// Send processing event
	sendSSEEvent(w, "processing", map[string]interface{}{
		"message": "Processing PCM audio data",
		"timestamp": time.Now().Unix(),
	})

	// Use shared transcription function for PCM
	logger.Info("Starting PCM transcription", "data_size", len(req.PCMData))
	result, err := transcription.TranscribePCM(req.PCMData, apiKey)
	if err != nil {
		logger.Error("Failed to transcribe PCM audio", "error", err)
		sendSSEError(w, fmt.Sprintf("Failed to transcribe audio: %v", err))
		return
	}

	// Send transcription complete event
	sendSSEEvent(w, "transcribed", map[string]interface{}{
		"text": result.Text,
		"timestamp": time.Now().Unix(),
	})

	// Send completion event
	sendSSEEvent(w, "complete", map[string]interface{}{
		"result": result,
		"timestamp": time.Now().Unix(),
	})
}

func flushSSE(w http.ResponseWriter) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
