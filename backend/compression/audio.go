package compression

import (
	"bytes"
	"fmt"
	"os/exec"
)

// CompressWAVToOGG compresses WAV audio data to OGG format using ffmpeg
// OGG provides good compression ratio with reasonable quality loss
func CompressWAVToOGG(wavData []byte) ([]byte, error) {
	// Create ffmpeg command to convert WAV to OGG with aggressive compression
	cmd := exec.Command("ffmpeg", 
		"-f", "wav",           // Input format: WAV
		"-i", "pipe:0",        // Input from stdin
		"-f", "ogg",           // Output format: OGG
		"-c:a", "libvorbis",   // Audio codec: Vorbis
		"-q:a", "2",           // Quality: 2 (lower = more compression, less quality)
		"-ac", "1",            // Mono audio (reduces size)
		"-ar", "16000",        // Sample rate: 16kHz (reduces size)
		"pipe:1",              // Output to stdout
	)
	
	// Set up pipes
	cmd.Stdin = bytes.NewReader(wavData)
	
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	
	// Execute the command
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg compression failed: %w, stderr: %s", err, errBuf.String())
	}
	
	return outBuf.Bytes(), nil
}

// CompressWAVToMP3 compresses WAV audio data to MP3 format using ffmpeg
// Alternative compression method if OGG is not preferred
func CompressWAVToMP3(wavData []byte) ([]byte, error) {
	// Create ffmpeg command to convert WAV to MP3 with aggressive compression
	cmd := exec.Command("ffmpeg", 
		"-f", "wav",           // Input format: WAV
		"-i", "pipe:0",        // Input from stdin
		"-f", "mp3",           // Output format: MP3
		"-c:a", "libmp3lame",  // Audio codec: LAME MP3
		"-b:a", "32k",         // Bitrate: 32kbps (very compressed)
		"-ac", "1",            // Mono audio
		"-ar", "16000",        // Sample rate: 16kHz
		"pipe:1",              // Output to stdout
	)
	
	// Set up pipes
	cmd.Stdin = bytes.NewReader(wavData)
	
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	
	// Execute the command
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg compression failed: %w, stderr: %s", err, errBuf.String())
	}
	
	return outBuf.Bytes(), nil
}

// CompressAudio compresses audio data using the default compression method (OGG)
func CompressAudio(wavData []byte) ([]byte, error) {
	return CompressWAVToOGG(wavData)
}