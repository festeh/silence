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