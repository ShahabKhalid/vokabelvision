package leonardo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// GeneratePrompt creates a Leonardo.ai prompt using the English and German words.
func GeneratePrompt(english, german, sentence string) string {
	promptTemplate := fmt.Sprintf(`Create an image
		Display a picture of an %s set against a solid background. Ensure that the background color contrasts with the typical color of an %s (i.e. do not use a red background if the %s is red). Picture should in square with white border.
		Below %s picture, Show the word "%s" in a prominent font.`, english, english, english, english, german)
	prompt := strings.ReplaceAll(promptTemplate, "[word]", english)
	prompt = strings.ReplaceAll(prompt, "[german_translation]", german)
	return prompt
}

// GetImage calls the Leonardo.ai API using the prompt and downloads the generated image.
func GetImage(apiKey, prompt string) (string, error) {
	apiURL := "https://cloud.leonardo.ai/api/rest/v1/generations" // Correct endpoint
	// Replace "prompt" with "textPrompts" as required by the API:
	payload := map[string]interface{}{
		"modelId":       "6b645e3a-d64f-4341-a6d8-7a3690fbf042",
		"contrast":      3.5,
		"prompt":        prompt,
		"num_images":    1,
		"width":         1080,
		"height":        1920,
		"alchemy":       true,
		"styleUUID":     "111dc692-d470-4eec-b791-3475abac4c46",
		"enhancePrompt": false,
		"seed":          "8933646694",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	log.Println(string(body))
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("authorization", "Bearer "+apiKey)
	req.Header.Set("content-type", "application/json")
	log.Println("authorization", "Bearer "+apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	log.Println(resp)
	// Check for error status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("Leonardo API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Assume the API returns JSON with an sdGenerationJob field containing the generationId.
	var res struct {
		SDGenerationJob struct {
			GenerationId  string `json:"generationId"`
			APICreditCost int    `json:"apiCreditCost"`
		} `json:"sdGenerationJob"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	log.Println(res)

	// Now poll the API using the generationId to get the image URL.
	generationId := res.SDGenerationJob.GenerationId
	log.Printf("Generation ID: %s", generationId)
	imageURL, err := pollForImage(apiKey, generationId)
	if err != nil {
		return "", err
	}

	// Download the image.
	imageResp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer imageResp.Body.Close()

	imagePath := "vocab_image.jpg"
	out, err := os.Create(imagePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, imageResp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println("Image saved to", imagePath)
	return imagePath, nil
}

func pollForImage(apiKey, generationId string) (string, error) {
	apiURL := fmt.Sprintf("https://cloud.leonardo.ai/api/rest/v1/generations/%s", generationId)
	client := &http.Client{}

	maxRetries := 10
	delay := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("accept", "application/json")
		req.Header.Set("authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}

		var pollRes struct {
			GenerationsByPK struct {
				GeneratedImages []struct {
					URL string `json:"url"`
				} `json:"generated_images"`
				Status string `json:"status"`
			} `json:"generations_by_pk"`
		}

		// Decode the response and close the body
		if err := json.NewDecoder(resp.Body).Decode(&pollRes); err != nil {
			resp.Body.Close()
			return "", err
		}
		resp.Body.Close()

		// Check if the job is complete and at least one image is available
		if pollRes.GenerationsByPK.Status == "COMPLETE" && len(pollRes.GenerationsByPK.GeneratedImages) > 0 {
			imageUrl := pollRes.GenerationsByPK.GeneratedImages[0].URL
			if imageUrl != "" {
				return imageUrl, nil
			}
		}

		// Wait before the next poll
		time.Sleep(delay)
	}

	return "", fmt.Errorf("timed out waiting for image generation")
}
