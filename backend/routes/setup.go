package routes

import (
	"silence-backend/handlers"
	_ "silence-backend/docs" // Swagger docs

	"github.com/pocketbase/pocketbase/core"
	httpSwagger "github.com/swaggo/http-swagger"
)

func SetCORSHeaders(re *core.RequestEvent) {
	re.Response.Header().Set("Access-Control-Allow-Origin", "*")
	re.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	re.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func Setup(se *core.ServeEvent, app core.App, elevenlabsAPIKey string) {
	se.Router.POST("/speak", func(re *core.RequestEvent) error {
		SetCORSHeaders(re)
		return handlers.HandleSpeak(re, app, elevenlabsAPIKey)
	})

	se.Router.OPTIONS("/speak", func(re *core.RequestEvent) error {
		SetCORSHeaders(re)
		return re.NoContent(200)
	})

	// Swagger UI endpoint
	se.Router.GET("/swagger/*", func(re *core.RequestEvent) error {
		httpSwagger.WrapHandler(re.Response, re.Request)
		return nil
	})

}
