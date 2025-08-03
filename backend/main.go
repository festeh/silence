package main

import (
	"log"
	"silence-backend/auth"
	"silence-backend/database"
	"silence-backend/env"
	"silence-backend/logger"
	"silence-backend/routes"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func logServerStart(addr string) {
	if strings.Contains(addr, ":") {
		parts := strings.Split(addr, ":")
		port := parts[len(parts)-1]
		logger.Info("Server started successfully", "port", port, "address", addr)
	} else {
		logger.Info("Server started successfully", "address", addr)
	}
}

func main() {
	logger.Init()

	envVars := env.Load()

	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := auth.EnsureSuperuser(se.App, envVars.SilenceEmail, envVars.SilencePassword); err != nil {
			logger.Error("Failed to ensure superuser", "error", err)
			return err
		}

		if err := database.EnsureSilenceCollection(se.App); err != nil {
			logger.Error("Failed to ensure silence collection", "error", err)
			return err
		}

		if err := database.EnsureAppsCollection(se.App); err != nil {
			logger.Error("Failed to ensure apps collection", "error", err)
			return err
		}

		logServerStart(se.Server.Addr)
		routes.Setup(se, app, envVars.ElevenlabsAPIKey, envVars.OpenRouterAPIKey)

		return se.Next()
	})

	logger.Info("Silence!")
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
