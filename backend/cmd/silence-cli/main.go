package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"silence-backend/database"
	"silence-backend/env"
	"silence-backend/handlers"
	"silence-backend/logger"
)

type AudioTranscriptionRequest struct {
	PCMData []byte `json:"pcm_data"`
}

func main() {
	// Initialize logger
	logger.Init()
	
	// Load environment
	environment, err := env.NewEnvironment()
	if err != nil {
		log.Fatal("Failed to load environment:", err)
	}

	// Initialize database client
	db, err := database.NewClient(environment)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	fmt.Println("Starting audio recording test CLI...")
	fmt.Println("Press any key to stop recording and process audio")
	
	// Start audio recording
	pcmData, err := recordAudio()
	if err != nil {
		log.Fatal("Failed to record audio:", err)
	}

	if len(pcmData) == 0 {
		fmt.Println("No audio data recorded")
		return
	}

	fmt.Printf("Recorded %d bytes of PCM data\n", len(pcmData))
	
	// Test HandleSpeak function directly
	err = testHandleSpeak(pcmData, db, environment)
	if err != nil {
		log.Fatal("Failed to test HandleSpeak:", err)
	}
}

func recordAudio() ([]byte, error) {
	fmt.Println("Recording audio... Press any key to stop")
	
	// Channel to receive PCM data
	dataChan := make(chan []byte)
	
	// Start recording in a goroutine
	go func() {
		// Generate dummy 16-bit PCM data continuously
		sampleRate := 16000
		samplesPerSecond := sampleRate
		pcmData := make([]byte, 0)
		
		start := time.Now()
		for {
			// Generate 100ms worth of samples at a time
			chunkSamples := samplesPerSecond / 10
			chunk := make([]byte, chunkSamples*2) // 16-bit = 2 bytes per sample
			
			for i := 0; i < chunkSamples; i++ {
				// Simple sine wave at 440Hz
				elapsed := time.Since(start).Seconds()
				sampleIndex := elapsed*float64(sampleRate) + float64(i)
				value := int16(32767 * 0.1 * math.Sin(2*math.Pi*440*sampleIndex/float64(sampleRate)))
				chunk[i*2] = byte(value & 0xFF)
				chunk[i*2+1] = byte((value >> 8) & 0xFF)
			}
			
			pcmData = append(pcmData, chunk...)
			time.Sleep(100 * time.Millisecond)
			
			// Check if we should stop (non-blocking)
			select {
			case dataChan <- pcmData:
				return
			default:
				// Continue recording
			}
		}
	}()
	
	// Wait for any key press
	go func() {
		reader := bufio.NewReader(os.Stdin)
		reader.ReadByte()
		// Signal to stop recording by requesting data
		<-dataChan
	}()
	
	// Wait for the recording to finish
	pcmData := <-dataChan
	fmt.Printf("Recording stopped. Captured %d bytes\n", len(pcmData))
	
	return pcmData, nil
}

func testHandleSpeak(pcmData []byte, db *database.Client, env *env.Environment) error {
	// Create request payload
	request := AudioTranscriptionRequest{
		PCMData: pcmData,
	}
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create a mock HTTP request
	req, err := http.NewRequest("POST", "/speak", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	
	// Create a response recorder to capture the output
	rr := httptest.NewRecorder()
	
	fmt.Println("Calling HandleSpeak function directly...")
	
	// Call the HandleSpeak function directly
	handlers.HandleSpeak(rr, req, db, env)
	
	fmt.Printf("Response status: %d\n", rr.Code)
	fmt.Println("Response body:")
	
	// Parse and display the SSE response
	responseBody := rr.Body.String()
	lines := strings.Split(responseBody, "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		if strings.HasPrefix(line, "event:") {
			fmt.Printf("Event: %s\n", strings.TrimSpace(strings.TrimPrefix(line, "event:")))
		} else if strings.HasPrefix(line, "data:") {
			fmt.Printf("Data: %s\n", strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	
	return nil
}