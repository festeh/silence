# Run Flutter frontend app on Linux desktop
run-frontend:
    cd frontend && flutter run -d linux

# Install Flutter dependencies  
install-frontend:
    cd frontend && flutter pub get

# Build Flutter app for Linux
build-frontend:
    cd frontend && flutter build linux

# Run Flutter tests
test-frontend:
    cd frontend && flutter test