package main

import (
	"log"
	"os"
	"silence-backend/handlers"
	"silence-backend/logger"
	"strings"
	
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func ensureSuperuser(app core.App, email, password string) error {
	if email == "" || password == "" {
		logger.Info("SILENCE_EMAIL or SILENCE_PASSWORD not provided, skipping superuser creation")
		return nil
	}

	// Check if admin with this email already exists
	record, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, email)
	if err == nil && record != nil {
		logger.Info("Superuser already exists", "email", email)
		return nil
	}

	// Get the superusers collection
	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		logger.Error("Failed to find superusers collection", "error", err)
		return err
	}

	// Create new superuser record
	record = core.NewRecord(superusers)
	record.Set("email", email)
	record.Set("password", password)

	// Save the record
	if err := app.Save(record); err != nil {
		logger.Error("Failed to create superuser", "email", email, "error", err)
		return err
	}

	logger.Info("Superuser created successfully", "email", email)
	return nil
}

func main() {
	logger.Init()
	
	// Get ElevenLabs API key from environment
	elevenlabsAPIKey := os.Getenv("ELEVENLABS_API_KEY")
	if elevenlabsAPIKey == "" {
		log.Fatal("ELEVENLABS_API_KEY environment variable is required")
	}
	
	// Get superuser credentials from environment
	silenceEmail := os.Getenv("SILENCE_EMAIL")
	silencePassword := os.Getenv("SILENCE_PASSWORD")
	
	app := pocketbase.New()

	// Check and create superuser if needed
	app.OnBootstrap().BindFunc(func(be *core.BootstrapEvent) error {
		return ensureSuperuser(be.App, silenceEmail, silencePassword)
	})

	// Add custom routes
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		addr := se.Server.Addr
		if strings.Contains(addr, ":") {
			parts := strings.Split(addr, ":")
			port := parts[len(parts)-1]
			logger.Info("Server started successfully", "port", port, "address", addr)
		} else {
			logger.Info("Server started successfully", "address", addr)
		}
		
		// Custom /speak endpoint with CORS
		se.Router.POST("/speak", func(re *core.RequestEvent) error {
			// Set CORS headers
			re.Response.Header().Set("Access-Control-Allow-Origin", "*")
			re.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			re.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			
			// Handle the speak logic
			return handlers.HandleSpeak(re, app, elevenlabsAPIKey)
		})
		
		// Handle CORS preflight
		se.Router.OPTIONS("/speak", func(re *core.RequestEvent) error {
			re.Response.Header().Set("Access-Control-Allow-Origin", "*")
			re.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			re.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			return re.NoContent(200)
		})

		return se.Next()
	})

	logger.Info("Starting PocketBase application")
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}