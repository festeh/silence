package env

import (
	"fmt"
	"os"
)

type Environment struct {
	ElevenLabsAPIKey string
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
	
	if e.ElevenLabsAPIKey == "" {
		return fmt.Errorf("ELEVENLABS_API_KEY environment variable is required")
	}
	
	return nil
}