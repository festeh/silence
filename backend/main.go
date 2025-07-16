package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"silence-backend/env"
	"silence-backend/handlers"
)

func main() {
	enviroment, err := env.NewEnvironment()
	if err != nil {
		log.Fatal(err)
	}

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

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}