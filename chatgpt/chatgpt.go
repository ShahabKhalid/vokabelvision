package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// PostedItem represents one entry in the posted history.
type PostedItem struct {
	Type     string `json:"type"`
	Key      string `json:"key"`
	PostedAt string `json:"posted_at"`
}

// ContentResult is the normalized output every content-type generator returns.
type ContentResult struct {
	Type        string
	OverlayText string // Main German text displayed on image
	EnglishText string // English translation/meaning (optional, shown below)
	AudioText   string
	Caption     string
	ImagePrompt string
	DedupeKey   string
}

// Vocab holds the vocabulary word, its translation, a reel caption, and a sample sentence.
type Vocab struct {
	English  string `json:"english"`
	German   string `json:"german"`
	Caption  string `json:"caption"`
	Sentence string `json:"sentence"`
}

// LoadPostedItems loads posted history from the given filename.
// Tries new format first, falls back to old []string format with automatic migration.
func LoadPostedItems(filename string) ([]PostedItem, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []PostedItem{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return []PostedItem{}, nil
	}

	// Try new format first
	var items []PostedItem
	if err := json.Unmarshal(data, &items); err == nil {
		return items, nil
	}

	// Fall back to old flat string array format
	var words []string
	if err := json.Unmarshal(data, &words); err != nil {
		return nil, fmt.Errorf("postedvocabs.json is in an unrecognized format: %w", err)
	}

	// Migrate: treat all old entries as type "vocab"
	migrated := make([]PostedItem, len(words))
	for i, w := range words {
		migrated[i] = PostedItem{Type: "vocab", Key: w, PostedAt: ""}
	}
	return migrated, nil
}

// SavePostedItems saves posted history to the given filename.
func SavePostedItems(filename string, items []PostedItem) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

// UpdatePostedItems appends a new item to the posted history and saves.
func UpdatePostedItems(filename, contentType, key string) error {
	items, err := LoadPostedItems(filename)
	if err != nil {
		return err
	}

	// Check if key is already in the list (case-insensitive).
	for _, item := range items {
		if item.Type == contentType && strings.EqualFold(item.Key, key) {
			return nil // Already posted, skip.
		}
	}

	// Append new item.
	items = append(items, PostedItem{
		Type:     contentType,
		Key:      key,
		PostedAt: time.Now().UTC().Format(time.RFC3339),
	})

	return SavePostedItems(filename, items)
}

// buildExcludeList builds an exclude string filtering by content type.
func buildExcludeList(items []PostedItem, contentType string) string {
	var keys []string
	for _, item := range items {
		if item.Type == contentType {
			keys = append(keys, item.Key)
		}
	}
	return strings.Join(keys, ", ")
}

// isDuplicate checks if a key is already in the posted items (case-insensitive, normalized).
func isDuplicate(items []PostedItem, key string) bool {
	normalizedKey := strings.ToLower(strings.TrimSpace(key))
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Key), normalizedKey) {
			return true
		}
	}
	return false
}

// callGPT calls the OpenAI API and returns the JSON response content.
func callGPT(apiKey, prompt string) (string, error) {
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
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
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
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from API")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// GetVocab generates a vocabulary word.
func GetVocab(apiKey, postedFile string) (ContentResult, error) {
	items, err := LoadPostedItems(postedFile)
	if err != nil {
		return ContentResult{}, fmt.Errorf("error loading posted items: %v", err)
	}

	excludeList := buildExcludeList(items, "vocab")
	prompt := "Give me a German vocabulary word with its English translation. " +
		"Provide a reel caption that includes the German word (with its article when possible) " +
		"and its English translation, along with hashtags related to German learning. " +
		"Also provide exactly one sample sentence in German using the word (max 10 words), " +
		"and a creative image prompt that visually represents this word. " +
		fmt.Sprintf("Do not use the following words: %s. ", excludeList) +
		"Always include the article with the German word when possible. " +
		"Return the result in JSON format with keys 'english', 'german', 'caption', 'sentence', and 'image_prompt'. " +
		"IMPORTANT: The 'sentence' value must be a single string, not an array."

	content, err := callGPT(apiKey, prompt)
	if err != nil {
		return ContentResult{}, err
	}

	var resp struct {
		English     string `json:"english"`
		German      string `json:"german"`
		Caption     string `json:"caption"`
		Sentence    string `json:"sentence"`
		ImagePrompt string `json:"image_prompt"`
	}
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return ContentResult{}, fmt.Errorf("error parsing vocab response: %v", err)
	}

	if isDuplicate(items, resp.English) {
		return ContentResult{}, fmt.Errorf("GPT returned a duplicate word: %q", resp.English)
	}

	// Use AI-generated image prompt, with fallback
	imagePrompt := resp.ImagePrompt
	if imagePrompt == "" {
		imagePrompt = fmt.Sprintf("A %s, centered on a solid background.", resp.English)
	}

	return ContentResult{
		Type:        "vocab",
		OverlayText: resp.German,
		EnglishText: resp.English,
		AudioText:   resp.German,
		Caption:     resp.Caption,
		ImagePrompt: imagePrompt,
		DedupeKey:   resp.English,
	}, nil
}

// GetVerb generates a verb conjugation.
func GetVerb(apiKey, postedFile string) (ContentResult, error) {
	items, err := LoadPostedItems(postedFile)
	if err != nil {
		return ContentResult{}, fmt.Errorf("error loading posted items: %v", err)
	}

	excludeList := buildExcludeList(items, "verb")
	prompt := "Give me a common German verb. Provide: the infinitive, English translation, and all 6 Präsens conjugations (ich, du, er, wir, ihr, sie). " +
		"Also provide a reel caption mentioning the infinitive and English meaning, one short example sentence (max 10 words) in German, and a creative image prompt. " +
		"The image prompt should visually represent this verb (can be concrete like 'person running' or abstract like 'visualization of thinking with colorful thoughts'). " +
		fmt.Sprintf("Do not use: %s. ", excludeList) +
		"Return JSON with keys: \"infinitive\",\"english\",\"ich\",\"du\",\"er\",\"wir\",\"ihr\",\"sie\",\"caption\",\"sentence\",\"image_prompt\"."

	content, err := callGPT(apiKey, prompt)
	if err != nil {
		return ContentResult{}, err
	}

	var resp struct {
		Infinitive string `json:"infinitive"`
		English    string `json:"english"`
		Ich        string `json:"ich"`
		Du         string `json:"du"`
		Er         string `json:"er"`
		Wir        string `json:"wir"`
		Ihr        string `json:"ihr"`
		Sie        string `json:"sie"`
		Caption    string `json:"caption"`
		Sentence   string `json:"sentence"`
		ImagePrompt string `json:"image_prompt"`
	}
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return ContentResult{}, fmt.Errorf("error parsing verb response: %v", err)
	}

	if isDuplicate(items, resp.English) {
		return ContentResult{}, fmt.Errorf("GPT returned a duplicate verb: %q", resp.English)
	}

	audioText := fmt.Sprintf("%s. Ich %s. Du %s. Er %s.", resp.Infinitive, resp.Ich, resp.Du, resp.Er)

	// Use the AI-generated image prompt directly
	imagePrompt := resp.ImagePrompt
	if imagePrompt == "" {
		// Fallback if AI doesn't provide one
		verbForm := resp.English
		if strings.HasPrefix(strings.ToLower(verbForm), "to ") {
			verbForm = strings.TrimPrefix(verbForm, "to ") + "ing"
		}
		imagePrompt = fmt.Sprintf("A person %s, centered on a solid background.", verbForm)
	}

	return ContentResult{
		Type:        "verb",
		OverlayText: resp.Infinitive,
		EnglishText: resp.English,
		AudioText:   audioText,
		Caption:     resp.Caption,
		ImagePrompt: imagePrompt,
		DedupeKey:   resp.English,
	}, nil
}

// GetAdjectivePair generates a pair of opposite adjectives.
func GetAdjectivePair(apiKey, postedFile string) (ContentResult, error) {
	items, err := LoadPostedItems(postedFile)
	if err != nil {
		return ContentResult{}, fmt.Errorf("error loading posted items: %v", err)
	}

	excludeList := buildExcludeList(items, "adjective_pair")
	prompt := "Give me a common German adjective and its opposite. Provide: the German adjective, its opposite, their English translations, " +
		"a reel caption showing the contrast, one short sentence in German using both, and a creative image prompt showing the visual contrast. " +
		fmt.Sprintf("Do not use: %s. ", excludeList) +
		"Return JSON with keys: \"german1\",\"german2\",\"english1\",\"english2\",\"caption\",\"sentence\",\"image_prompt\"."

	content, err := callGPT(apiKey, prompt)
	if err != nil {
		return ContentResult{}, err
	}

	var resp struct {
		German1     string `json:"german1"`
		German2     string `json:"german2"`
		English1    string `json:"english1"`
		English2    string `json:"english2"`
		Caption     string `json:"caption"`
		Sentence    string `json:"sentence"`
		ImagePrompt string `json:"image_prompt"`
	}
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return ContentResult{}, fmt.Errorf("error parsing adjective_pair response: %v", err)
	}

	// Dedup key: sorted pair of English words (lowercase for consistency)
	sorted := []string{strings.ToLower(resp.English1), strings.ToLower(resp.English2)}
	sort.Strings(sorted)
	dedupeKey := strings.Join(sorted, "/")

	if isDuplicate(items, dedupeKey) {
		return ContentResult{}, fmt.Errorf("GPT returned a duplicate adjective pair: %q", dedupeKey)
	}

	overlayText := fmt.Sprintf("%s / %s", resp.German1, resp.German2)
	audioText := fmt.Sprintf("%s... %s", resp.German1, resp.German2)
	englishPair := fmt.Sprintf("%s / %s", resp.English1, resp.English2)

	// Use AI-generated image prompt, with fallback
	imagePrompt := resp.ImagePrompt
	if imagePrompt == "" {
		imagePrompt = fmt.Sprintf("Split image showing %s and %s in dramatic contrast, visual comparison, centered on a solid background.", resp.English1, resp.English2)
	}

	return ContentResult{
		Type:        "adjective_pair",
		OverlayText: overlayText,
		EnglishText: englishPair,
		AudioText:   audioText,
		Caption:     resp.Caption,
		ImagePrompt: imagePrompt,
		DedupeKey:   dedupeKey,
	}, nil
}

// GetPhrase generates a common German phrase or idiom.
func GetPhrase(apiKey, postedFile string) (ContentResult, error) {
	items, err := LoadPostedItems(postedFile)
	if err != nil {
		return ContentResult{}, fmt.Errorf("error loading posted items: %v", err)
	}

	excludeList := buildExcludeList(items, "phrase")
	prompt := "Give me a common German idiom or everyday phrase. Provide: the German phrase, its English equivalent meaning, " +
		"a one-sentence explanation, a reel caption, and a creative image prompt that visually represents this phrase or its meaning. " +
		fmt.Sprintf("Do not use: %s. ", excludeList) +
		"Return JSON with keys: \"german\",\"english\",\"explanation\",\"caption\",\"image_prompt\"."

	content, err := callGPT(apiKey, prompt)
	if err != nil {
		return ContentResult{}, err
	}

	var resp struct {
		German      string `json:"german"`
		English     string `json:"english"`
		Explanation string `json:"explanation"`
		Caption     string `json:"caption"`
		ImagePrompt string `json:"image_prompt"`
	}
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return ContentResult{}, fmt.Errorf("error parsing phrase response: %v", err)
	}

	if isDuplicate(items, resp.English) {
		return ContentResult{}, fmt.Errorf("GPT returned a duplicate phrase: %q", resp.English)
	}

	// Use AI-generated image prompt, with fallback
	imagePrompt := resp.ImagePrompt
	if imagePrompt == "" {
		imagePrompt = fmt.Sprintf("Scene depicting '%s', German culture, everyday life, vibrant and authentic, centered on a solid background.", resp.Explanation)
	}

	return ContentResult{
		Type:        "phrase",
		OverlayText: resp.German,
		EnglishText: resp.Explanation,
		AudioText:   resp.German,
		Caption:     resp.Caption,
		ImagePrompt: imagePrompt,
		DedupeKey:   resp.English,
	}, nil
}

// GetFalseFriend generates a German "false friend" word.
func GetFalseFriend(apiKey, postedFile string) (ContentResult, error) {
	items, err := LoadPostedItems(postedFile)
	if err != nil {
		return ContentResult{}, fmt.Errorf("error loading posted items: %v", err)
	}

	excludeList := buildExcludeList(items, "false_friend")
	prompt := "Give me a German \"false friend\" — a German word that looks like an English word but means something completely different. " +
		"Provide: the German word, what it actually means in German, the English word it resembles, " +
		"an English explanation of the false friend trap, a reel caption, and a creative image prompt showing the confusion or contrast. " +
		fmt.Sprintf("Do not use: %s. ", excludeList) +
		"Return JSON with keys: \"german\",\"german_meaning\",\"english_false\",\"english_actual\",\"caption\",\"image_prompt\"."

	content, err := callGPT(apiKey, prompt)
	if err != nil {
		return ContentResult{}, err
	}

	var resp struct {
		German       string `json:"german"`
		GermanMeaning string `json:"german_meaning"`
		EnglishFalse string `json:"english_false"`
		EnglishActual string `json:"english_actual"`
		Caption      string `json:"caption"`
		ImagePrompt  string `json:"image_prompt"`
	}
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return ContentResult{}, fmt.Errorf("error parsing false_friend response: %v", err)
	}

	dedupeKey := strings.ToLower(strings.TrimSpace(resp.German))
	if isDuplicate(items, dedupeKey) {
		return ContentResult{}, fmt.Errorf("GPT returned a duplicate false friend: %q", resp.German)
	}

	// Use AI-generated image prompt, with fallback
	imagePrompt := resp.ImagePrompt
	if imagePrompt == "" {
		imagePrompt = fmt.Sprintf("Warning symbol and visual comparison showing '%s' (German) vs '%s' (English) language trap, educational poster style, centered on a solid background.", resp.German, resp.EnglishFalse)
	}

	return ContentResult{
		Type:        "false_friend",
		OverlayText: resp.German,
		EnglishText: resp.GermanMeaning,
		AudioText:   resp.German,
		Caption:     resp.Caption,
		ImagePrompt: imagePrompt,
		DedupeKey:   dedupeKey,
	}, nil
}

// ===== DEPRECATED (kept for backward compatibility) =====

// LoadPostedVocabs is deprecated. Use LoadPostedItems instead.
func LoadPostedVocabs(filename string) ([]string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return []string{}, nil
	}
	var vocabs []string
	if err := json.Unmarshal(data, &vocabs); err != nil {
		return nil, err
	}
	return vocabs, nil
}

// SavePostedVocabs is deprecated. Use savePostedItems instead.
func SavePostedVocabs(filename string, vocabs []string) error {
	data, err := json.Marshal(vocabs)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

// UpdatePostedVocabs is deprecated. Use UpdatePostedItems instead.
func UpdatePostedVocabs(filename, newWord string) error {
	vocabs, err := LoadPostedVocabs(filename)
	if err != nil {
		return err
	}
	for _, v := range vocabs {
		if strings.EqualFold(v, newWord) {
			return nil
		}
	}
	vocabs = append(vocabs, newWord)
	return SavePostedVocabs(filename, vocabs)
}
