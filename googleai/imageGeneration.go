package googleai

import (
	"context"
	"fmt"
	"os"

	genai "github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GenerateImage takes a prompt and an output file path, generates an image,
// saves it to outputFile, and returns the output file path or an error.
func GenerateImage(prompt, outputFile string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("create genai client: %w", err)
	}
	defer client.Close()

	// Use an image-capable model. Adjust to the image model available to your project,
	// e.g. "gemini-2.5-flash-image-preview" or "imagen-3.0".
	model := client.GenerativeModel("gemini-2.5-flash-image-preview")

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("generate content: %w", err)
	}

	// Find the first image blob in the response and write it.
	for _, cand := range resp.Candidates {
		if cand == nil || cand.Content == nil {
			continue
		}
		for _, part := range cand.Content.Parts {
			switch p := part.(type) {
			case genai.Blob:
				// p.MIMEType will be like "image/png" or "image/jpeg"
				if err := os.WriteFile(outputFile, p.Data, 0o644); err != nil {
					return "", fmt.Errorf("write file: %w", err)
				}
				return outputFile, nil
			case *genai.Blob: // handle pointer form defensively
				if p != nil {
					if err := os.WriteFile(outputFile, p.Data, 0o644); err != nil {
						return "", fmt.Errorf("write file: %w", err)
					}
					return outputFile, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no image blob found in model response")
}

func main() {
	out, err := GenerateImage("Generate an image of a banana wearing a costume.", "banana.png")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Saved:", out)
}
