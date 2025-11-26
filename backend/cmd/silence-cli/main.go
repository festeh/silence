package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"time"

	"silence-backend/logger"
)

func main() {
	// Initialize logger
	logger.Init()

	// Define command-line flags
	backendURL := flag.String("url", "http://localhost:8090", "Backend URL to send audio to")
	provider := flag.String("provider", "", "Transcription provider: 'elevenlabs' or 'chutes' (empty for default chain)")
	flag.Parse()

	var pcmData []byte
	var err error

	// Check if a file path was provided as argument
	args := flag.Args()
	if len(args) > 0 {
		filePath := args[0]
		fmt.Printf("Reading audio from file: %s\n", filePath)

		pcmData, err = readWAVFile(filePath)
		if err != nil {
			log.Fatal("Failed to read WAV file:", err)
		}

		fmt.Printf("Read %d bytes of PCM data from file\n", len(pcmData))
	} else {
		fmt.Println("Starting audio recording test CLI...")
		fmt.Println("Press any key to stop recording and process audio")

		// Start audio recording
		pcmData, err = recordAudio()
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
	}

	// Send to backend API
	err = sendToBackend(pcmData, *backendURL, *provider)
	if err != nil {
		log.Fatal("Failed to send to backend:", err)
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

func readWAVFile(filePath string) ([]byte, error) {
	// Read the entire WAV file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// WAV files have a 44-byte header, extract PCM data after header
	if len(data) < 44 {
		return nil, fmt.Errorf("file too small to be a valid WAV file")
	}

	// Verify it's a WAV file by checking RIFF header
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a valid WAV file")
	}

	// Extract PCM data (skip 44-byte header)
	pcmData := data[44:]

	return pcmData, nil
}

func sendToBackend(pcmData []byte, backendURL string, provider string) error {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add audio file field
	fileWriter, err := writer.CreateFormFile("audio", "recording.pcm")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = fileWriter.Write(pcmData)
	if err != nil {
		return fmt.Errorf("failed to write audio data: %w", err)
	}

	// Add file_format field
	err = writer.WriteField("file_format", "pcm_s16le_16")
	if err != nil {
		return fmt.Errorf("failed to write file_format field: %w", err)
	}

	// Add language_code field (optional, defaults to "auto")
	err = writer.WriteField("language_code", "auto")
	if err != nil {
		return fmt.Errorf("failed to write language_code field: %w", err)
	}

	// Add provider field if specified
	if provider != "" {
		err = writer.WriteField("provider", provider)
		if err != nil {
			return fmt.Errorf("failed to write provider field: %w", err)
		}
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	fmt.Printf("Sending audio to backend: %s\n", backendURL)
	req, err := http.NewRequest("POST", backendURL+"/speak", &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("Response status: %d\n", resp.StatusCode)
	fmt.Println("Response body:")

	// Parse and pretty-print JSON response
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(responseBody, &jsonResponse); err != nil {
		// If not JSON, just print raw response
		fmt.Println(string(responseBody))
	} else {
		prettyJSON, _ := json.MarshalIndent(jsonResponse, "", "  ")
		fmt.Println(string(prettyJSON))
	}

	return nil
}
