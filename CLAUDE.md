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
  - `handlers/speak.go`: Processes audio uploads, handles both multipart form data and JSON
  - `transcription/`: Audio transcription using ElevenLabs
  - `compression/`: Audio compression utilities
  - `database/collections.go`: Auto-creates required database collections
  - `routes/setup.go`: Route registration and CORS configuration
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
- `SILENCE_EMAIL`: Optional superuser email for PocketBase
- `SILENCE_PASSWORD`: Optional superuser password for PocketBase

## API Endpoints
- `POST /speak`: Audio upload and transcription (supports multipart form and JSON with PCM data)
- `GET /swagger/*`: Interactive Swagger API documentation (access at `/swagger/index.html`)
- Standard PocketBase admin UI and API endpoints

## API Documentation
The project uses Swagger/OpenAPI for API documentation:
- **Swagger UI**: Available at `http://localhost:8090/swagger/index.html` when server is running
- **Generation**: Run `swag init -g main.go` from the `backend/` directory to regenerate docs
- **Annotations**: API documentation is maintained via code comments in `main.go` and `handlers/speak.go`
- **Spec files**: Generated OpenAPI specs are in `backend/docs/` (swagger.json, swagger.yaml)

## Data Flow
1. Frontend records WAV audio
2. Audio uploaded to `/speak` endpoint
3. Backend transcribes audio via ElevenLabs API
4. Transcription result returned immediately as JSON response
5. Audio compression and database storage happens asynchronously in background

## Testing
No specific test commands found - use standard Go testing (`go test ./...`) and Flutter testing (`flutter test`).

## CLI Tool
The project includes a CLI tool at `backend/cmd/silence-cli/main.go` for backend interaction.