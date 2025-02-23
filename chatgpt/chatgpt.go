package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Vocab holds the vocabulary word, its translation, a reel caption, and a sample sentence.
type Vocab struct {
	English  string `json:"english"`
	German   string `json:"german"`
	Caption  string `json:"caption"`
	Sentence string `json:"sentence"`
}

// GetVocab calls the ChatGPT API to get a random German vocabulary word with its English translation,
// a reel caption (which includes the German word with its article when possible, its English translation,
// and relevant hashtags for German learning), and a sample sentence (each sentence should not exceed 10 words).
// Return the result in JSON format with keys 'english', 'german', 'caption', and 'sentence'.
func GetVocab(apiKey string) (Vocab, error) {
	apiURL := "https://api.openai.com/v1/chat/completions"
	payload := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{
				"role": "user",
				"content": "Give me a random German vocabulary word with its English translation. " +
					"Provide a reel caption that includes the German word (with its article when possible) and its English translation, " +
					"as well as hashtags related to German learning to increase the reel's reach. " +
					"Also provide one sample sentence in German using the word, with each sentence not exceeding 10 words. " +
					"Return the result in JSON format with keys 'english', 'german', 'caption', and 'sentence'.",
			},
		},
		"temperature": 0.7,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return Vocab{}, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return Vocab{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Vocab{}, err
	}
	defer resp.Body.Close()

	// Define the structure of the ChatGPT response.
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return Vocab{}, err
	}

	if len(chatResp.Choices) == 0 {
		return Vocab{}, fmt.Errorf("no choices returned from API")
	}

	// The assistant's message content is expected to be a JSON string.
	var vocab Vocab
	if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &vocab); err != nil {
		return Vocab{}, err
	}

	return vocab, nil
}
