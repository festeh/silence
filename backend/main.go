package main

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"silence-backend/env"
	"silence-backend/handlers"
	"silence-backend/logger"
)

func main() {
	logger.Init()
	
	enviroment, err := env.NewEnvironment()
	if err != nil {
		logger.Error("Failed to initialize environment", "error", err)
		panic(err)
	}

	logger.Info("Starting PocketBase application")
	app := pocketbase.New()

	// Add custom routes
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Register the /speak endpoint
		se.Router.POST("/speak", func(e *core.RequestEvent) error {
			// Set CORS headers
			e.Response.Header().Set("Access-Control-Allow-Origin", "*")
			e.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			e.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			
			handlers.HandleSpeak(e.Response, e.Request, se.App.(*pocketbase.PocketBase), enviroment)
			return nil
		})

		return se.Next()
	})

	logger.Info("Starting server")
	if err := app.Start(); err != nil {
		logger.Error("Failed to start server", "error", err)
		panic(err)
	}
}