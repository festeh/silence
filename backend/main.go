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

func validateEnvironment() (string, string, string) {
	elevenlabsAPIKey := os.Getenv("ELEVENLABS_API_KEY")
	if elevenlabsAPIKey == "" {
		log.Fatal("ELEVENLABS_API_KEY environment variable is required")
	}
	
	silenceEmail := os.Getenv("SILENCE_EMAIL")
	silencePassword := os.Getenv("SILENCE_PASSWORD")
	
	return elevenlabsAPIKey, silenceEmail, silencePassword
}

func setCORSHeaders(re *core.RequestEvent) {
	re.Response.Header().Set("Access-Control-Allow-Origin", "*")
	re.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	re.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func logServerStart(addr string) {
	if strings.Contains(addr, ":") {
		parts := strings.Split(addr, ":")
		port := parts[len(parts)-1]
		logger.Info("Server started successfully", "port", port, "address", addr)
	} else {
		logger.Info("Server started successfully", "address", addr)
	}
}

func setupRoutes(se *core.ServeEvent, app core.App, elevenlabsAPIKey string) {
	se.Router.POST("/speak", func(re *core.RequestEvent) error {
		setCORSHeaders(re)
		return handlers.HandleSpeak(re, app, elevenlabsAPIKey)
	})
	
	se.Router.OPTIONS("/speak", func(re *core.RequestEvent) error {
		setCORSHeaders(re)
		return re.NoContent(200)
	})
}

func main() {
	logger.Init()
	
	elevenlabsAPIKey, silenceEmail, silencePassword := validateEnvironment()
	
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := ensureSuperuser(se.App, silenceEmail, silencePassword); err != nil {
			logger.Error("Failed to ensure superuser", "error", err)
			return err
		}
		
		logServerStart(se.Server.Addr)
		setupRoutes(se, app, elevenlabsAPIKey)
		
		return se.Next()
	})

	logger.Info("Starting PocketBase application")
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}