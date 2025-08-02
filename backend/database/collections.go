package database

import (
	"silence-backend/logger"

	"github.com/pocketbase/pocketbase/core"
)

func EnsureSilenceCollection(app core.App) error {
	// Check if silence collection already exists
	_, err := app.FindCollectionByNameOrId("silence")
	if err == nil {
		logger.Info("Silence collection already exists")
		return nil
	}

	// Create the silence collection
	collection := core.NewBaseCollection("silence")
	
	// Add data field for storing base64 compressed audio
	dataField := &core.TextField{
		Name:     "audio",
		Required: true,
		Max:      1000000, // 1MB limit for base64 data
	}
	
	// Add note field for storing transcribed text
	resultField := &core.TextField{
		Name:     "result",
		Required: false,
		Max:      10000, // 10k characters limit
	}
	
	collection.Fields.Add(dataField)
	collection.Fields.Add(resultField)

	// Save the collection
	if err := app.Save(collection); err != nil {
		logger.Error("Failed to create silence collection", "error", err)
		return err
	}

	logger.Info("Silence collection created successfully")
	return nil
}
