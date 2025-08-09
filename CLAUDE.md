# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Silence is an AI-powered audio transcription application with a Go backend and Flutter frontend. The system captures audio, transcribes it using ElevenLabs API, and stores results in a PocketBase database.

## Architecture

### Backend (Go + PocketBase)
- **Framework**: PocketBase (SQLite-based backend-as-a-service)
- **Entry point**: `backend/main.go` - initializes PocketBase app, ensures collections exist, sets up routes
- **Database**: Two main collections:
  - `silence`: Stores compressed audio (base64) and transcription results
  - `apps`: Stores application metadata
- **Key modules**:
  - `handlers/speak.go`: Processes audio uploads via SSE (Server-Sent Events), handles both multipart form data and JSON
  - `handlers/ai.go`: AI chat completion using OpenRouter API
  - `transcription/`: Audio transcription using ElevenLabs
  - `compression/`: Audio compression utilities
  - `database/collections.go`: Auto-creates required database collections
  - `env/`: Environment variable management

### Frontend (Flutter)
- **Entry point**: `frontend/lib/main.dart` - audio recording and real-time SSE event display
- **Core functionality**: 
  - Records WAV audio using `record` package
  - Uploads to backend via multipart form
  - Real-time SSE event streaming with selectable text logs
- **Platform**: Primarily Linux desktop support

## Development Commands

### Backend
```bash
cd backend
go run main.go                    # Start development server
go build -o silence-backend main.go  # Build binary
go mod tidy                       # Update dependencies
```

### Frontend  
```bash
cd frontend
flutter run -d linux             # Run on Linux desktop
flutter build linux              # Build for Linux
flutter pub get                   # Install dependencies
flutter test                     # Run tests
```

## Environment Variables (Backend)
- `ELEVENLABS_API_KEY`: Required for audio transcription
- `OPENROUTER_API_KEY`: Required for AI chat completion
- `SILENCE_EMAIL`: Optional superuser email for PocketBase
- `SILENCE_PASSWORD`: Optional superuser password for PocketBase

## API Endpoints
- `POST /speak`: Audio upload and transcription (supports multipart form and JSON with PCM data)
- `POST /ai`: Chat completion via OpenRouter
- Standard PocketBase admin UI and API endpoints

## Data Flow
1. Frontend records WAV audio
2. Audio uploaded to `/speak` endpoint
3. Backend compresses audio, transcribes via ElevenLabs
4. Results stored in PocketBase `silence` collection
5. Real-time progress sent to frontend via SSE

## Testing
No specific test commands found - use standard Go testing (`go test ./...`) and Flutter testing (`flutter test`).

## CLI Tool
The project includes a CLI tool at `backend/cmd/silence-cli/main.go` for backend interaction.