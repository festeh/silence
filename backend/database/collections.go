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

func EnsureAppsCollection(app core.App) error {
	_, err := app.FindCollectionByNameOrId("apps")
	if err == nil {
		logger.Info("Apps collection already exists")
		return nil
	}

	collection := core.NewBaseCollection("apps")
	
	nameField := &core.TextField{
		Name:     "name",
		Required: true,
		Max:      255,
	}
	
	descriptionField := &core.JSONField{
		Name:     "description",
		Required: false,
	}
	
	collection.Fields.Add(nameField)
	collection.Fields.Add(descriptionField)

	if err := app.Save(collection); err != nil {
		logger.Error("Failed to create apps collection", "error", err)
		return err
	}

	logger.Info("Apps collection created successfully")
	return nil
}
