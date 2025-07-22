package main

import (
	"log"
	"os"
	"silence-backend/handlers"
	"silence-backend/logger"
	
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	logger.Init()
	
	// Get ElevenLabs API key from environment
	elevenlabsAPIKey := os.Getenv("ELEVENLABS_API_KEY")
	if elevenlabsAPIKey == "" {
		log.Fatal("ELEVENLABS_API_KEY environment variable is required")
	}
	
	app := pocketbase.New()

	// Add custom routes
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
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