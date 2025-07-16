package main

import (
	"net/http"

	"silence-backend/database"
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

	logger.Info("Initializing database client")
	dbClient, err := database.NewClient(enviroment)
	if err != nil {
		logger.Error("Failed to initialize database client", "error", err)
		panic(err)
	}
	defer dbClient.Close()

	logger.Info("Setting up HTTP routes")
	mux := http.NewServeMux()
	
	// Add CORS middleware
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
	
	mux.Handle("/speak", corsHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleSpeak(w, r, dbClient, enviroment)
	})))

	logger.Info("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		logger.Error("Failed to start HTTP server", "error", err)
		panic(err)
	}
}