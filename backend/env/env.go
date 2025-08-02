package env

import (
	"log"
	"os"
)

type Env struct {
	ElevenlabsAPIKey string
	SilenceEmail     string
	SilencePassword  string
}

func Load() *Env {
	elevenlabsAPIKey := os.Getenv("ELEVENLABS_API_KEY")
	if elevenlabsAPIKey == "" {
		log.Fatal("ELEVENLABS_API_KEY environment variable is required")
	}
	
	silenceEmail := os.Getenv("SILENCE_EMAIL")
	silencePassword := os.Getenv("SILENCE_PASSWORD")
	
	return &Env{
		ElevenlabsAPIKey: elevenlabsAPIKey,
		SilenceEmail:     silenceEmail,
		SilencePassword:  silencePassword,
	}
}