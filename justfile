# Load .env file and create dart-define arguments
_dart_defines := `if [ -f .env ]; then grep -v '^#' .env | grep -v '^$' | sed 's/^/--dart-define=/' | tr '\n' ' '; fi`

# Check if BACKEND_URL is set in .env
_check_backend_url := `if [ -f .env ] && grep -q "^BACKEND_URL=" .env; then echo "ok"; else echo "missing"; fi`

# Validate required environment variables
_validate_env:
    #!/usr/bin/env bash
    if [ "{{_check_backend_url}}" = "missing" ]; then
        echo "❌ Error: BACKEND_URL is not set in .env file"
        echo "Please add BACKEND_URL=your_backend_url to .env file"
        exit 1
    fi
    echo "✅ Environment variables validated"

# Show available recipes
default:
    @just --list

# Run Flutter frontend app on Linux desktop
run-frontend: _validate_env
    cd frontend && flutter run -d linux {{_dart_defines}}

# Install Flutter dependencies  
install-frontend:
    cd frontend && flutter pub get

# Build Flutter app for Linux
build-frontend: _validate_env
    cd frontend && flutter build linux {{_dart_defines}}

# Build Flutter app for Linux (debug)
build-frontend-debug: _validate_env
    cd frontend && flutter build linux --debug {{_dart_defines}}

# Run Flutter tests
test-frontend: _validate_env
    cd frontend && flutter test {{_dart_defines}}

# Analyze Flutter code
analyze-frontend:
    cd frontend && flutter analyze

# Backend commands
# ================

# Kill all backend instances
kill-backend:
    #!/usr/bin/env bash
    # Kill by process pattern
    pkill -f "go run main.go serve" || true
    pkill -f "silence-backend" || true
    pkill -f "/tmp/silence" || true
    pkill -f "main.go serve" || true
    # Kill anything on port 8090
    lsof -ti :8090 | xargs -r kill -9 || true
    echo "✓ All backend instances killed"

# Run backend development server (default port 8090)
run-backend: kill-backend
    cd backend && go run main.go serve

# Run backend on custom port
run-backend-port PORT: kill-backend
    cd backend && go run main.go serve --http=127.0.0.1:{{PORT}}

# Build backend binary
build-backend:
    cd backend && go build -o silence-backend main.go

# Build backend with optimizations for production
build-backend-prod:
    cd backend && go build -ldflags="-s -w" -o silence-backend main.go

# Build and run backend with debug logging
run-backend-debug: kill-backend
    cd backend && go build -o /tmp/silence-debug main.go && /tmp/silence-debug serve

# Test GET route
test-get:
    curl -s http://localhost:8090/test

# Test POST route
test-post:
    curl -s -X POST http://localhost:8090/speak -H "Content-Type: application/json" -d '{}'

# Check router debug logs
check-logs:
    grep "🔍" /tmp/debug.log | tail -20

# Run backend tests
test-backend:
    cd backend && go test ./...

# Run backend tests with coverage
test-backend-coverage:
    cd backend && go test -cover ./...

# Run backend tests with verbose output
test-backend-verbose:
    cd backend && go test -v ./...

# Install/update Go dependencies
deps-backend:
    cd backend && go mod tidy && go mod download

# Generate Swagger documentation
swagger:
    cd backend && swag init -g main.go

# Open Swagger UI in browser (regenerates docs first)
swagger-open: swagger
    xdg-open http://localhost:8090/swagger/index.html

# Start backend and open Swagger UI (all-in-one development command)
swagger-dev: kill-backend swagger
    #!/usr/bin/env bash
    cd backend && go run main.go serve &
    BACKEND_PID=$!
    echo "🚀 Backend started (PID: $BACKEND_PID)"
    echo "⏳ Waiting for server to start..."
    sleep 2
    xdg-open http://localhost:8090/swagger/index.html
    echo "📖 Swagger UI opened in browser"
    echo "Press Ctrl+C to stop the backend server"
    wait $BACKEND_PID

# Format Go code
fmt-backend:
    cd backend && go fmt ./...

# Run Go linter (requires golangci-lint)
lint-backend:
    cd backend && golangci-lint run

# Clean backend build artifacts
clean-backend:
    cd backend && rm -f silence-backend

# Full backend rebuild (deps, swagger, build)
rebuild-backend: deps-backend swagger build-backend

# Development workflow: format, swagger, and run backend
dev-backend: fmt-backend swagger run-backend

# Combined commands
# =================

# Install all dependencies (frontend + backend)
install-all: install-frontend deps-backend

# Run all tests (frontend + backend)
test-all: test-frontend test-backend

# Format all code (frontend + backend)
fmt-all: fmt-backend
    cd frontend && dart format .

# Clean all build artifacts
clean-all: clean-backend
    cd frontend && flutter clean