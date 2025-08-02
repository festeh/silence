package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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

	// Save recording as WAV file for investigation
	err = saveAsWAV(pcmData, "/tmp/record.wav")
	if err != nil {
		log.Printf("Failed to save WAV file: %v", err)
	} else {
		fmt.Println("Recording saved to /tmp/record.wav")
	}

	// Test HandleSpeak function directly
	err = testHandleSpeak(pcmData, db, environment)
	if err != nil {
		log.Fatal("Failed to test HandleSpeak:", err)
	}
}

func recordAudio() ([]byte, error) {
	fmt.Println("Recording audio... Press any key to stop")

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel to receive PCM data
	dataChan := make(chan []byte)
	errChan := make(chan error)

	// Start recording with ffmpeg in a goroutine
	go func() {
		// ffmpeg command to record from default microphone
		// Output: 16-bit PCM, 16kHz, mono, raw format
		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-f", "pulse", // Use PulseAudio (Linux default)
			"-i", "default", // Default microphone
			"-ar", "16000", // Sample rate 16kHz
			"-ac", "1", // Mono channel
			"-f", "s16le", // 16-bit little endian PCM
			"-", // Output to stdout
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errChan <- fmt.Errorf("failed to create stdout pipe: %w", err)
			return
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			errChan <- fmt.Errorf("failed to start ffmpeg: %w", err)
			return
		}

		// Read all PCM data from stdout
		pcmData, err := io.ReadAll(stdout)
		if err != nil && ctx.Err() == nil {
			// Only report error if context wasn't cancelled
			errChan <- fmt.Errorf("failed to read PCM data: %w", err)
			return
		}

		// Wait for command to finish
		cmd.Wait()

		dataChan <- pcmData
	}()

	// Wait for any key press
	go func() {
		reader := bufio.NewReader(os.Stdin)
		reader.ReadByte()
		// Cancel the context to stop ffmpeg
		cancel()
	}()

	// Wait for either the recording to finish or an error
	select {
	case pcmData := <-dataChan:
		fmt.Printf("Recording stopped. Captured %d bytes\n", len(pcmData))
		return pcmData, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(30 * time.Second):
		cancel()
		return nil, fmt.Errorf("recording timeout after 30 seconds")
	}
}

func saveAsWAV(pcmData []byte, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// WAV file parameters
	sampleRate := uint32(16000)
	bitsPerSample := uint16(16)
	channels := uint16(1)

	dataSize := uint32(len(pcmData))
	fileSize := 36 + dataSize

	// Write WAV header
	// RIFF header
	file.Write([]byte("RIFF"))
	binary.Write(file, binary.LittleEndian, fileSize)
	file.Write([]byte("WAVE"))

	// fmt chunk
	file.Write([]byte("fmt "))
	binary.Write(file, binary.LittleEndian, uint32(16)) // chunk size
	binary.Write(file, binary.LittleEndian, uint16(1))  // audio format (PCM)
	binary.Write(file, binary.LittleEndian, channels)
	binary.Write(file, binary.LittleEndian, sampleRate)
	binary.Write(file, binary.LittleEndian, sampleRate*uint32(channels)*uint32(bitsPerSample)/8) // byte rate
	binary.Write(file, binary.LittleEndian, channels*bitsPerSample/8)                            // block align
	binary.Write(file, binary.LittleEndian, bitsPerSample)

	// data chunk
	file.Write([]byte("data"))
	binary.Write(file, binary.LittleEndian, dataSize)
	file.Write(pcmData)

	return nil
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
