package config

import (
	"encoding/json"
	"os"
)

// Config holds the API keys and other configuration settings.
type Config struct {
	ChatGPTAPIKey        string `json:"chatgpt_api_key"`
	LeonardoAPIKey       string `json:"leonardo_api_key"`
	ElevenLabsAPIKey     string `json:"elevenlabs_api_key"`
	ElevenLabsVoiceID    string `json:"elevenlabs_voice_id"`
	InstagramUserID      string `json:"instagram_user_id"`
	InstagramAccessToken string `json:"instagram_access_token"`
	CloudinaryURL        string `json:"cloudinary_url"`
}

// LoadConfig reads the configuration from the given file.
func LoadConfig(filename string) (Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
