package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"silence-backend/compression"
	"silence-backend/env"
	"silence-backend/transcription"
)

func main() {
	enviroment, err := env.NewEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	app := pocketbase.New()

	// Add custom routes
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Register the /speak endpoint
		se.Router.POST("/speak", func(e *core.RequestEvent) error {
			// Set CORS headers
			e.Response.Header().Set("Access-Control-Allow-Origin", "*")
			e.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			e.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			
			handleSpeak(e.Response, e.Request, se.App.(*pocketbase.PocketBase), enviroment)
			return nil
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func handleSpeak(w http.ResponseWriter, r *http.Request, app *pocketbase.PocketBase, env *env.Environment) {
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
		// Handle JSON request for PCM data
		type AudioTranscriptionRequest struct {
			PCMData []byte `json:"pcm_data"`
		}

		var req AudioTranscriptionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			sendSSEError(w, "Failed to decode audio transcription request")
			return
		}

		if len(req.PCMData) == 0 {
			sendSSEError(w, "pcm_data is required")
			return
		}

		// Send processing event
		sendSSEEvent(w, "processing", map[string]interface{}{
			"message": "Processing PCM audio data",
			"timestamp": time.Now().Unix(),
		})

		// Use shared transcription function for PCM
		result, err := transcription.TranscribePCM(req.PCMData, apiKey)
		if err != nil {
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
		return
	}

	// Handle multipart form data (frontend)
	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		sendSSEError(w, "Invalid multipart form")
		return
	}

	// Get the audio file from the form
	file, _, err := r.FormFile("audio")
	if err != nil {
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
		sendSSEError(w, "Failed to read audio file")
		return
	}

	if len(wavData) == 0 {
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
	compressedData, err := compression.CompressAudio(wavData)
	if err != nil {
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

	// Save base64 encoded audio to PocketBase
	collection, err := app.FindCollectionByNameOrId("audio")
	if err != nil {
		sendSSEError(w, "Failed to find audio collection")
		return
	}

	record := core.NewRecord(collection)
	record.Set("data", base64Data)
	record.Set("original_size", len(wavData))
	record.Set("compressed_size", len(compressedData))

	if err := app.Save(record); err != nil {
		sendSSEError(w, "Failed to save audio to database")
		return
	}

	// Send transcription event
	sendSSEEvent(w, "transcribing", map[string]interface{}{
		"message": "Transcribing audio",
		"audio_id": record.Id,
		"timestamp": time.Now().Unix(),
	})

	// Use shared transcription function for WAV
	result, err := transcription.TranscribeWAV(wavData, apiKey)
	if err != nil {
		sendSSEError(w, fmt.Sprintf("Failed to transcribe audio: %v", err))
		return
	}

	// Create a new note with the transcribed text and audio reference
	noteCollection, err := app.FindCollectionByNameOrId("notes")
	if err != nil {
		sendSSEError(w, "Failed to find notes collection")
		return
	}

	noteRecord := core.NewRecord(noteCollection)
	noteRecord.Set("title", result.Text)
	noteRecord.Set("content", "")
	noteRecord.Set("audio_id", record.Id)

	if err := app.Save(noteRecord); err != nil {
		sendSSEError(w, "Failed to create note")
		return
	}

	// Send completion event
	sendSSEEvent(w, "complete", map[string]interface{}{
		"note_id": noteRecord.Id,
		"audio_id": record.Id,
		"transcribed_text": result.Text,
		"result": noteRecord,
		"timestamp": time.Now().Unix(),
	})
}

func sendSSEEvent(w http.ResponseWriter, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flushSSE(w)
}

func sendSSEError(w http.ResponseWriter, message string) {
	errorData := map[string]interface{}{
		"error": message,
		"timestamp": time.Now().Unix(),
	}
	sendSSEEvent(w, "error", errorData)
}

func flushSSE(w http.ResponseWriter) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
