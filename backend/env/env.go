package env

import (
	"log"
	"os"
)

type Env struct {
	ElevenlabsAPIKey string
	ChutesAPIToken   string
	SilenceEmail     string
	SilencePassword  string
}

func Load() *Env {
	elevenlabsAPIKey := os.Getenv("ELEVENLABS_API_KEY")
	if elevenlabsAPIKey == "" {
		log.Fatal("ELEVENLABS_API_KEY environment variable is required")
	}

	chutesAPIToken := os.Getenv("CHUTES_API_TOKEN")
	if chutesAPIToken == "" {
		log.Fatal("CHUTES_API_TOKEN environment variable is required")
	}

	silenceEmail := os.Getenv("SILENCE_EMAIL")
	silencePassword := os.Getenv("SILENCE_PASSWORD")

	return &Env{
		ElevenlabsAPIKey: elevenlabsAPIKey,
		ChutesAPIToken:   chutesAPIToken,
		SilenceEmail:     silenceEmail,
		SilencePassword:  silencePassword,
	}
}
