package database

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
	"silence-backend/env"
	"silence-backend/logger"
)

type Client struct {
	app *core.App
}

func NewClient(env *env.Environment) (*Client, error) {
	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       "./pb_data",
		EncryptionEnv: "PB_ENCRYPTION_KEY",
		IsDebug:       false,
	})

	if err := app.Bootstrap(); err != nil {
		return nil, fmt.Errorf("failed to bootstrap PocketBase app: %w", err)
	}

	client := &Client{app: app}
	
	if err := client.ensureSchema(); err != nil {
		return nil, fmt.Errorf("failed to ensure schema: %w", err)
	}

	return client, nil
}

func (c *Client) ensureSchema() error {
	if err := c.ensureAudioCollection(); err != nil {
		return fmt.Errorf("failed to ensure audio collection: %w", err)
	}

	if err := c.ensureNotesCollection(); err != nil {
		return fmt.Errorf("failed to ensure notes collection: %w", err)
	}

	return nil
}

func (c *Client) ensureAudioCollection() error {
	collection, err := c.app.Dao().FindCollectionByNameOrId("audio")
	if err != nil {
		logger.Info("Creating audio collection")
		collection = &models.Collection{
			Name:       "audio",
			Type:       models.CollectionTypeBase,
			ListRule:   types.Pointer(""),
			ViewRule:   types.Pointer(""),
			CreateRule: types.Pointer(""),
			UpdateRule: types.Pointer(""),
			DeleteRule: types.Pointer(""),
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name:     "data",
					Type:     schema.FieldTypeText,
					Required: true,
				},
				&schema.SchemaField{
					Name:     "original_size",
					Type:     schema.FieldTypeNumber,
					Required: true,
				},
				&schema.SchemaField{
					Name:     "compressed_size",
					Type:     schema.FieldTypeNumber,
					Required: true,
				},
			),
		}

		if err := c.app.Dao().SaveCollection(collection); err != nil {
			return fmt.Errorf("failed to create audio collection: %w", err)
		}
	}

	return nil
}

func (c *Client) ensureNotesCollection() error {
	collection, err := c.app.Dao().FindCollectionByNameOrId("notes")
	if err != nil {
		logger.Info("Creating notes collection")
		collection = &models.Collection{
			Name:       "notes",
			Type:       models.CollectionTypeBase,
			ListRule:   types.Pointer(""),
			ViewRule:   types.Pointer(""),
			CreateRule: types.Pointer(""),
			UpdateRule: types.Pointer(""),
			DeleteRule: types.Pointer(""),
			Schema: schema.NewSchema(
				&schema.SchemaField{
					Name:     "title",
					Type:     schema.FieldTypeText,
					Required: true,
				},
				&schema.SchemaField{
					Name:     "content",
					Type:     schema.FieldTypeText,
					Required: false,
				},
				&schema.SchemaField{
					Name:     "audio_id",
					Type:     schema.FieldTypeText,
					Required: true,
				},
			),
		}

		if err := c.app.Dao().SaveCollection(collection); err != nil {
			return fmt.Errorf("failed to create notes collection: %w", err)
		}
	}

	return nil
}

func (c *Client) UpsertAudio(data string, originalSize, compressedSize int) (*models.Record, error) {
	collection, err := c.app.Dao().FindCollectionByNameOrId("audio")
	if err != nil {
		return nil, fmt.Errorf("failed to find audio collection: %w", err)
	}

	record := models.NewRecord(collection)
	record.Set("data", data)
	record.Set("original_size", originalSize)
	record.Set("compressed_size", compressedSize)

	if err := c.app.Dao().SaveRecord(record); err != nil {
		return nil, fmt.Errorf("failed to save audio record: %w", err)
	}

	return record, nil
}

func (c *Client) CreateNote(title, content, audioId string) (*models.Record, error) {
	collection, err := c.app.Dao().FindCollectionByNameOrId("notes")
	if err != nil {
		return nil, fmt.Errorf("failed to find notes collection: %w", err)
	}

	record := models.NewRecord(collection)
	record.Set("title", title)
	record.Set("content", content)
	record.Set("audio_id", audioId)

	if err := c.app.Dao().SaveRecord(record); err != nil {
		return nil, fmt.Errorf("failed to save note record: %w", err)
	}

	return record, nil
}

func (c *Client) Close() error {
	return c.app.OnTerminate().Trigger(&core.TerminateEvent{
		App: c.app,
	})
}