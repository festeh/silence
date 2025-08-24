# Load .env file and create dart-define arguments
_dart_defines := `if [ -f .env ]; then grep -v '^#' .env | grep -v '^$' | sed 's/^/--dart-define=/' | tr '\n' ' '; fi`

# Show available recipes
default:
    @just --list

# Run Flutter frontend app on Linux desktop
run-frontend:
    cd frontend && flutter run -d linux {{_dart_defines}}

# Install Flutter dependencies  
install-frontend:
    cd frontend && flutter pub get

# Build Flutter app for Linux
build-frontend:
    cd frontend && flutter build linux {{_dart_defines}}

# Build Flutter app for Linux (debug)
build-frontend-debug:
    cd frontend && flutter build linux --debug {{_dart_defines}}

# Run Flutter tests
test-frontend:
    cd frontend && flutter test {{_dart_defines}}

# Analyze Flutter code
analyze-frontend:
    cd frontend && flutter analyze