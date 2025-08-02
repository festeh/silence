package env

import (
	"fmt"
	"os"
)

type Environment struct {
	ElevenLabsAPIKey string
	PocketBaseURL    string
	PocketBaseEmail  string
	PocketBasePass   string
}

func NewEnvironment() (*Environment, error) {
	env := &Environment{}

	if err := env.load(); err != nil {
		return nil, err
	}

	return env, nil
}

func (e *Environment) load() error {
	e.ElevenLabsAPIKey = os.Getenv("ELEVENLABS_API_KEY")
	e.PocketBaseURL = os.Getenv("POCKETBASE_URL")
	e.PocketBaseEmail = os.Getenv("POCKETBASE_EMAIL")
	e.PocketBasePass = os.Getenv("POCKETBASE_PASS")

	if e.ElevenLabsAPIKey == "" {
		return fmt.Errorf("ELEVENLABS_API_KEY environment variable is required")
	}

	if e.PocketBaseURL == "" {
		return fmt.Errorf("POCKETBASE_URL environment variable is required")
	}

	if e.PocketBaseEmail == "" {
		return fmt.Errorf("POCKETBASE_EMAIL environment variable is required")
	}

	if e.PocketBasePass == "" {
		return fmt.Errorf("POCKETBASE_PASS environment variable is required")
	}

	return nil
}
