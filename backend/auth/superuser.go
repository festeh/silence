package auth

import (
	"silence-backend/logger"

	"github.com/pocketbase/pocketbase/core"
)

func EnsureSuperuser(app core.App, email, password string) error {
	if email == "" || password == "" {
		logger.Info("SILENCE_EMAIL or SILENCE_PASSWORD not provided, skipping superuser creation")
		return nil
	}

	// Check if admin with this email already exists
	record, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, email)
	if err == nil && record != nil {
		logger.Info("Superuser already exists", "email", email)
		return nil
	}

	// Get the superusers collection
	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		logger.Error("Failed to find superusers collection", "error", err)
		return err
	}

	// Create new superuser record
	record = core.NewRecord(superusers)
	record.Set("email", email)
	record.Set("password", password)

	// Save the record
	if err := app.Save(record); err != nil {
		logger.Error("Failed to create superuser", "email", email, "error", err)
		return err
	}

	logger.Info("Superuser created successfully", "email", email)
	return nil
}
