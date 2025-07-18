package database

import (
	"fmt"

	"github.com/pluja/pocketbase"
	"silence-backend/env"
	"silence-backend/logger"
)

type Client struct {
	client *pocketbase.Client
}

type AudioRecord struct {
	ID             string `json:"id"`
	Data           string `json:"data"`
	OriginalSize   int    `json:"original_size"`
	CompressedSize int    `json:"compressed_size"`
	Created        string `json:"created"`
	Updated        string `json:"updated"`
}

type NoteRecord struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	AudioID string `json:"audio_id"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

func NewClient(env *env.Environment) (*Client, error) {
	client := pocketbase.NewClient(env.PocketBaseURL, 
		pocketbase.WithAdminEmailPassword(env.PocketBaseEmail, env.PocketBasePass))
	
	logger.Info("Successfully authenticated with PocketBase")
	
	return &Client{client: client}, nil
}

func (c *Client) UpsertAudio(data string, originalSize, compressedSize int) (*AudioRecord, error) {
	logger.Info("Creating audio record", "original_size", originalSize, "compressed_size", compressedSize)
	
	recordData := map[string]any{
		"data":            data,
		"original_size":   originalSize,
		"compressed_size": compressedSize,
	}
	
	response, err := c.client.Create("audio", recordData)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio record: %w", err)
	}
	
	record := AudioRecord{
		ID:             response.ID,
		Data:           data,
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		Created:        response.Created,
		Updated:        response.Updated,
	}
	
	logger.Info("Audio record created successfully", "id", record.ID)
	return &record, nil
}

func (c *Client) CreateNote(title, content, audioID string) (*NoteRecord, error) {
	logger.Info("Creating note record", "title", title, "audio_id", audioID)
	
	recordData := map[string]any{
		"title":    title,
		"content":  content,
		"audio_id": audioID,
	}
	
	response, err := c.client.Create("notes", recordData)
	if err != nil {
		return nil, fmt.Errorf("failed to create note record: %w", err)
	}
	
	record := NoteRecord{
		ID:      response.ID,
		Title:   title,
		Content: content,
		AudioID: audioID,
		Created: response.Created,
		Updated: response.Updated,
	}
	
	logger.Info("Note record created successfully", "id", record.ID)
	return &record, nil
}

func (c *Client) Close() error {
	// The pluja/pocketbase client doesn't require explicit closing
	return nil
}