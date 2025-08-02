package routes

import (
	"silence-backend/handlers"

	"github.com/pocketbase/pocketbase/core"
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
}
