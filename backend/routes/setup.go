// Package routes configures HTTP routes and middleware for the Silence backend API.
package routes

import (
	"silence-backend/handlers"

	"github.com/pocketbase/pocketbase/core"
)

// SetCORSHeaders configures Cross-Origin Resource Sharing (CORS) headers for API responses.
// Allows all origins, POST and OPTIONS methods, and Content-Type header.
func SetCORSHeaders(re *core.RequestEvent) {
	re.Response.Header().Set("Access-Control-Allow-Origin", "*")
	re.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	re.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// Setup registers all HTTP routes for the Silence backend API.
// Configures the following endpoints:
//   - POST /speak: Audio transcription (multipart or JSON)
//   - OPTIONS /speak: CORS preflight handling
func Setup(se *core.ServeEvent, app core.App, elevenlabsAPIKey string) {
	se.Router.POST("/speak", func(re *core.RequestEvent) error {
		SetCORSHeaders(re)
		return handlers.HandleSpeak(re, app, elevenlabsAPIKey)
	})

	se.Router.OPTIONS("/speak", func(re *core.RequestEvent) error {
		SetCORSHeaders(re)
		return re.NoContent(200)
	})

}
