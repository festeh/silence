// Package routes configures HTTP routes and middleware for the Silence backend API.
package routes

import (
	"net/http"

	_ "silence-backend/docs" // Swagger docs
	"silence-backend/handlers"
	"silence-backend/transcription"

	"github.com/pocketbase/pocketbase/core"
	httpSwagger "github.com/swaggo/http-swagger"
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
func Setup(se *core.ServeEvent, app core.App, provider transcription.TranscriptionProvider) {
	se.Router.POST("/speak", func(re *core.RequestEvent) error {
		SetCORSHeaders(re)
		return handlers.HandleSpeak(re, app, provider)
	})

	se.Router.OPTIONS("/speak", func(re *core.RequestEvent) error {
		SetCORSHeaders(re)
		return re.NoContent(200)
	})

	// Swagger UI - redirect /swagger to /swagger/index.html
	se.Router.GET("/swagger", func(re *core.RequestEvent) error {
		http.Redirect(re.Response, re.Request, "/swagger/index.html", http.StatusMovedPermanently)
		return nil
	})

	// Swagger UI endpoint - serves at /swagger/index.html and other swagger assets
	se.Router.GET("/swagger/{path...}", func(re *core.RequestEvent) error {
		// Get the wildcard path
		path := re.Request.PathValue("path")

		// Reconstruct the full path for Swagger handler
		// Swagger expects paths like /swagger/index.html
		re.Request.URL.Path = "/swagger/" + path

		httpSwagger.WrapHandler(re.Response, re.Request)
		return nil
	})

}
