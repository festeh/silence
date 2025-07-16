# Silence Audio Web Server

An audio web server that responds with SSE (Server-Sent Events) events. It provides a `/speak` endpoint for audio processing and transcription using ElevenLabs API.

## Features

- **Audio Processing**: Supports both multipart form data (WAV files) and JSON (PCM data)
- **Audio Compression**: Automatically compresses audio using FFmpeg to OGG format
- **Speech Transcription**: Uses ElevenLabs API for audio transcription
- **SSE Events**: Real-time progress updates via Server-Sent Events
- **PocketBase Integration**: Uses PocketBase as the database backend
- **Auto-save**: Automatically saves compressed audio and creates notes with transcribed text

## Setup

1. **Install dependencies**:
   ```bash
   go mod tidy
   ```

2. **Set up environment variables**:
   ```bash
   cp .env.example .env
   # Edit .env and add your ElevenLabs API key
   ```

3. **Install FFmpeg** (required for audio compression):
   ```bash
   # Ubuntu/Debian
   sudo apt install ffmpeg
   
   # macOS
   brew install ffmpeg
   ```

4. **Import database schema**:
   - Start the server: `./silence-backend serve`
   - Open PocketBase admin UI (usually http://localhost:8090/_/)
   - Go to Settings > Import collections
   - Import the `pb_schema.json` file

## Usage

### Start the server:
```bash
./silence-backend serve
```

### API Endpoints

#### POST /api/custom/speak

Processes audio and returns transcription via SSE events.

**For multipart form data (WAV files):**
```bash
curl -X POST http://localhost:8090/api/custom/speak \
  -F "audio=@your_audio.wav" \
  -H "Accept: text/event-stream"
```

**For JSON (PCM data):**
```bash
curl -X POST http://localhost:8090/api/custom/speak \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"pcm_data": "base64_encoded_pcm_data"}'
```

### SSE Events

The endpoint emits the following events:

- `start`: Processing started
- `processing`: Reading/processing audio data
- `compressing`: Compressing audio (multipart only)
- `saving`: Saving to database (multipart only)
- `transcribing`: Transcribing audio
- `transcribed`: Transcription completed (JSON only)
- `complete`: Processing completed with results
- `error`: Error occurred

### Example SSE Response

```
event: start
data: {"message":"Starting audio processing","timestamp":1642694400}

event: processing
data: {"message":"Reading audio file","timestamp":1642694401}

event: compressing
data: {"message":"Compressing audio data","original_size":48000,"timestamp":1642694402}

event: saving
data: {"message":"Saving compressed audio to database","compressed_size":12000,"base64_size":16000,"timestamp":1642694403}

event: transcribing
data: {"message":"Transcribing audio","audio_id":"record_id","timestamp":1642694404}

event: complete
data: {"note_id":"note_record_id","audio_id":"audio_record_id","transcribed_text":"Hello world","result":{"id":"note_record_id","title":"Hello world","content":"","audio_id":"audio_record_id"},"timestamp":1642694405}
```

## Database Schema

The server uses two PocketBase collections:

### Audio Collection
- `data` (text): Base64-encoded compressed audio data
- `original_size` (number): Size of original WAV file
- `compressed_size` (number): Size of compressed audio

### Notes Collection
- `title` (text): Transcribed text from audio
- `content` (text): Additional note content
- `audio_id` (relation): Reference to audio record

## Dependencies

- [PocketBase](https://github.com/pocketbase/pocketbase) - Database and backend framework
- [ElevenLabs API](https://elevenlabs.io/) - Speech-to-text transcription
- [FFmpeg](https://ffmpeg.org/) - Audio compression

## Development

Build the project:
```bash
go build -o silence-backend
```

Run tests:
```bash
go test ./...
```