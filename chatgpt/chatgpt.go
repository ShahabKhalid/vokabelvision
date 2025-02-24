package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// Vocab holds the vocabulary word, its translation, a reel caption, and a sample sentence.
type Vocab struct {
	English  string `json:"english"`
	German   string `json:"german"`
	Caption  string `json:"caption"`
	Sentence string `json:"sentence"`
}

// LoadPostedVocabs loads the list of posted vocab words from the given filename.
// The file is expected to contain a JSON array of strings. If the file is empty,
// it returns an empty slice.
func LoadPostedVocabs(filename string) ([]string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	// If file is empty, return an empty slice.
	if len(data) == 0 {
		return []string{}, nil
	}
	var vocabs []string
	if err := json.Unmarshal(data, &vocabs); err != nil {
		return nil, err
	}
	return vocabs, nil
}

// SavePostedVocabs saves the list of posted vocab words to the given filename.
func SavePostedVocabs(filename string, vocabs []string) error {
	data, err := json.Marshal(vocabs)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

// UpdatePostedVocabs appends the new word to the list and keeps only the last 50 words.
func UpdatePostedVocabs(filename, newWord string) error {
	vocabs, err := LoadPostedVocabs(filename)
	if err != nil {
		return err
	}
	// Append new word (assuming uniqueness by English word)
	vocabs = append(vocabs, newWord)
	// If the list grows beyond 50, trim the oldest words.
	if len(vocabs) > 50 {
		vocabs = vocabs[len(vocabs)-50:]
	}
	return SavePostedVocabs(filename, vocabs)
}

// GetVocab calls the ChatGPT API to get a new German vocab, its English translation, a reel caption, and a short sentence.
// It also instructs ChatGPT to avoid words that are already in the posted list.
func GetVocab(apiKey, postedFile string) (Vocab, error) {
	// Load the list of posted vocabulary words.
	postedWords, err := LoadPostedVocabs(postedFile)
	if err != nil {
		return Vocab{}, fmt.Errorf("error loading posted vocabs: %v", err)
	}
	excludeList := strings.Join(postedWords, ", ")

	// Build the prompt with instructions:
	prompt := "Give me a random German vocabulary word with its English translation. " +
		"Provide a reel caption that includes the German word (with its article when possible) " +
		"and its English translation, along with hashtags related to German learning. " +
		"Also provide one sample sentence in German using the word, with each sentence not exceeding 10 words. " +
		fmt.Sprintf("Do not use the following words: %s. ", excludeList) +
		"Always include the article with the German word when possible. " +
		"Return the result in JSON format with keys 'english', 'german', 'caption', and 'sentence'."

	apiURL := "https://api.openai.com/v1/chat/completions"
	payload := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
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

	// The assistant's message content should be a JSON string.
	var vocab Vocab
	if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &vocab); err != nil {
		return Vocab{}, err
	}

	return vocab, nil
}
