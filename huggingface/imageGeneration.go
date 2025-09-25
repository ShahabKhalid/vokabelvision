package huggingface

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

// GenerateImage calls Hugging Face Inference API and saves the result as a PNG.
// Returns the path to the saved image.
func GenerateImage(prompt, model, outputFile string) (string, error) {
	apiKey := os.Getenv("HF_TOKEN")
	if apiKey == "" {
		return "", fmt.Errorf("HF_TOKEN environment variable not set")
	}

	url := fmt.Sprintf("https://api-inference.huggingface.co/models/%s", model)
	payload := []byte(fmt.Sprintf(`{"inputs": %q}`, prompt))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
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

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s\n%s", resp.Status, string(body))
	}

	out, err := os.Create(outputFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return outputFile, nil
}

func main() {
	// Example usage
	path, err := GenerateImage(
		"A pencil centered on a solid background, with a white square border.",
		"black-forest-labs/FLUX.1-schnell",
		"flux_output.png",
	)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("✅ Image saved at:", path)
}
