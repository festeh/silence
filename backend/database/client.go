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

type Record struct {
	ID      string `json:"id"`
	Data    string `json:"data"`
	Note    string `json:"note"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

func NewClient(env *env.Environment) (*Client, error) {
	client := pocketbase.NewClient(env.PocketBaseURL, 
		pocketbase.WithAdminEmailPassword(env.PocketBaseEmail, env.PocketBasePass))
	
	logger.Info("Successfully authenticated with PocketBase")
	
	return &Client{client: client}, nil
}

func (c *Client) CreateRecord(data, note string) (*Record, error) {
	logger.Info("Creating record")
	
	recordData := map[string]any{
		"data": data,
		"note": note,
	}
	
	response, err := c.client.Create("records", recordData)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}
	
	record := Record{
		ID:      response.ID,
		Data:    data,
		Note:    note,
		Created: response.Created,
		Updated: response.Updated,
	}
	
	logger.Info("Record created successfully", "id", record.ID)
	return &record, nil
}


func (c *Client) Close() error {
	// The pluja/pocketbase client doesn't require explicit closing
	return nil
}