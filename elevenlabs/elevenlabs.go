package elevenlabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// GetAudio calls the ElevenLabs text-to-speech API to generate German pronunciation audio.
// It uses the voice endpoint and logs errors if the response isn't OK.
func GetAudio(apiKey, text string, sentence string, voiceID string) (string, error) {
	// Update the endpoint to match ElevenLabs' TTS API.
	// Replace "german_voice" with your actual voice ID if different.
	apiURL := "https://api.elevenlabs.io/v1/text-to-speech/" + voiceID

	pausedText := fmt.Sprintf(`
			<speak>
				<prosody rate="slow">%s</prosody>
				<break time="2s"/>
				<prosody rate="x-slow">%s</prosody>
				<break time="2s"/>
				<prosody rate="x-slow">%s</prosody>
			</speak>
		`, text, text, text)

	// Prepare the payload.
	// Some endpoints might require additional fields such as a model_id.
	payload := map[string]interface{}{
		"text":     pausedText,
		"model_id": "eleven_multilingual_v2", // Uncomment if needed per documentation.
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	// Use the required header key for ElevenLabs.
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check for errors.
	if resp.StatusCode != http.StatusOK {
		responseBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ElevenLabs API error: status %d, response: %s", resp.StatusCode, string(responseBytes))
	}

	// This endpoint typically streams audio directly.
	audioPath := "vocab_audio.mp3"
	out, err := os.Create(audioPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return audioPath, nil
}
