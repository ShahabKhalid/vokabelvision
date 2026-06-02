package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"vokabelvision/chatgpt"
	"vokabelvision/cloudinary"
	"vokabelvision/config"
	"vokabelvision/edgetts"
	"vokabelvision/googleai"
	"vokabelvision/instagram"
	"vokabelvision/leonardo"
	"vokabelvision/utils"
	"vokabelvision/video"

	"github.com/robfig/cron/v3"
)

var contentTypes = []string{"vocab", "verb", "adjective_pair", "phrase", "false_friend"}

func main() {
	// Define flags.
	once := flag.Bool("once", false, "Run the task once instead of scheduling it")
	testMode := flag.Bool("test", false, "Generate content without uploading or publishing")
	contentType := flag.String("type", "", "Specific content type to test (vocab, verb, adjective_pair, phrase, false_friend)")

	// Parse command-line flags.
	flag.Parse()

	// Check if the --once flag is provided.
	if *once {
		GenerateAndPost(*testMode, *contentType)
	} else if *testMode {
		log.Fatalf("--test flag requires --once flag")
	} else if *contentType != "" {
		log.Fatalf("--type flag requires --once flag")
	} else {
		// Load the Berlin location.
		berlin, err := time.LoadLocation("Europe/Berlin")
		if err != nil {
			log.Fatalf("Failed to load Berlin timezone: %v", err)
		}

		// Create a new cron scheduler that runs in the Berlin timezone.
		c := cron.New(cron.WithLocation(berlin))

		// Schedule the job to run at 6 AM and 6 PM every day.
		// Cron spec (minute hour day month day-of-week): "0 6,18 * * *"
		_, err = c.AddFunc("0 7,13,19 * * *", func() {
			GenerateAndPost(false, "")
		})

		if err != nil {
			log.Fatalf("Failed to add cron job: %v", err)
		}

		// Start the cron scheduler.
		c.Start()
		log.Println("Scheduler started. Waiting for scheduled tasks...")

		// Block forever to keep the application running.
		select {}
	}
}

func pickContentType() string {
	return contentTypes[rand.Intn(len(contentTypes))]
}

func GenerateAndPost(testMode bool, forcedType string) {
	// Load configuration.
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Step 1: Pick a random content type and generate content (with retries on duplicates).
	postedVocabFilePath := "postedvocabs.json"
	maxRetries := 5
	var result chatgpt.ContentResult
	var contentErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		var pickedType string
		if forcedType != "" {
			// Use the forced type if provided
			pickedType = forcedType
		} else {
			// Pick a random type
			pickedType = pickContentType()
		}
		fmt.Printf("Attempt %d - Picked content type: %s\n", attempt, pickedType)
		contentType := pickedType

		switch contentType {
		case "vocab":
			result, contentErr = chatgpt.GetVocab(cfg.ChatGPTAPIKey, postedVocabFilePath)
		case "verb":
			result, contentErr = chatgpt.GetVerb(cfg.ChatGPTAPIKey, postedVocabFilePath)
		case "adjective_pair":
			result, contentErr = chatgpt.GetAdjectivePair(cfg.ChatGPTAPIKey, postedVocabFilePath)
		case "phrase":
			result, contentErr = chatgpt.GetPhrase(cfg.ChatGPTAPIKey, postedVocabFilePath)
		case "false_friend":
			result, contentErr = chatgpt.GetFalseFriend(cfg.ChatGPTAPIKey, postedVocabFilePath)
		}

		// If it's a duplicate error, retry
		if contentErr != nil && attempt < maxRetries {
			fmt.Printf("Attempt %d failed: %v (retrying...)\n", attempt, contentErr)
			continue
		}

		// Either success or max retries reached
		if contentErr != nil {
			log.Fatalf("Error getting content after %d attempts: %v", maxRetries, contentErr)
		}
		break
	}

	fmt.Printf("Got content: type=%s, overlayText=%s\n", result.Type, result.OverlayText)

	// Step 2: Generate image prompt.
	prompt := leonardo.GeneratePrompt(result.ImagePrompt, "", "")
	fmt.Println("Generated image prompt:", prompt)

	// Step 3: Generate image.
	imagePath, err := googleai.GenerateImage(prompt, "vocab_image.jpg")
	if err != nil {
		fmt.Println("Error generating image:", err)
		return
	}
	fmt.Println("Saved:", imagePath)

	// Step 4: Post-process image with English translation.
	imagePath, err = utils.CreateReelImageWithTranslation(
		imagePath,
		imagePath,
		result.OverlayText,
		result.EnglishText,
		"font/din1451alt.ttf",
	)
	if err != nil {
		fmt.Println("Error post-processing image:", err)
		return
	}
	fmt.Println("✅ Saved updated image at:", imagePath)

	// Step 5: Generate audio.
	var audioPath string
	switch result.Type {
	case "verb", "phrase", "false_friend":
		// These types read their content once
		audioPath, err = edgetts.GetAudioSequence([]string{result.AudioText}, "de-DE-ConradNeural")
	default:
		// vocab and adjective_pair: repeat 3x
		audioPath, err = edgetts.GetAudio(result.AudioText, "de-DE-ConradNeural")
	}

	if err != nil {
		log.Fatalf("Error generating audio: %v", err)
	}
	fmt.Println("Audio saved at:", audioPath)

	// Step 6: Generate video.
	outputVideoPath := "vocab_reel.mp4"
	if err := video.GenerateVideo(imagePath, audioPath, outputVideoPath); err != nil {
		log.Fatalf("Error generating video: %v", err)
	}
	fmt.Println("Video generated at:", outputVideoPath)

	if testMode {
		fmt.Println("\n✅ TEST MODE: Content generated successfully. Skipping upload and publish.")
		fmt.Printf("Content type: %s\n", result.Type)
		fmt.Printf("Overlay text: %s\n", result.OverlayText)
		fmt.Printf("Audio text: %s\n", result.AudioText)
		fmt.Printf("Caption: %s\n", result.Caption)
		fmt.Println("\nGenerated files (NOT cleaned up):")
		fmt.Println("  - vocab_image.jpg")
		fmt.Println("  - vocab_audio.mp3")
		fmt.Println("  - vocab_reel.mp4")
		return
	}

	// Step 7: Upload to Cloudinary.
	videoURL, publicID := cloudinary.UploadVideo(cfg.CloudinaryURL, outputVideoPath)

	// Step 8: Publish to Instagram.
	captionWithTags := fmt.Sprintf("%s #german #germanlanguage #deutschlernen #languagelearning", result.Caption)
	if err := instagram.PublishVideo(cfg.InstagramUserID, cfg.InstagramAccessToken, videoURL, captionWithTags); err != nil {
		log.Fatalf("Error uploading video: %v", err)
	}
	fmt.Println("Reel uploaded successfully!")

	// Step 9: Update posted history.
	if err := chatgpt.UpdatePostedItems(postedVocabFilePath, result.Type, result.DedupeKey); err != nil {
		log.Printf("Warning: failed to update posted items: %v", err)
	}
	cloudinary.DeleteVideo(cfg.CloudinaryURL, publicID)
	DeleteFileIfExists(outputVideoPath)
	DeleteFileIfExists("vocab_audio.mp3")
	DeleteFileIfExists("vocab_image.jpg")
}

// DeleteFileIfExists deletes the specified file if it exists.
func DeleteFileIfExists(filename string) error {
	// Check if the file exists.
	if _, err := os.Stat(filename); err == nil {
		// File exists, attempt deletion.
		err = os.Remove(filename)
		if err != nil {
			return fmt.Errorf("failed to delete file: %v", err)
		}
		fmt.Printf("File %s deleted successfully.\n", filename)
	} else if os.IsNotExist(err) {
		// File does not exist.
		fmt.Printf("File %s does not exist.\n", filename)
	} else {
		// Some other error occurred.
		return fmt.Errorf("error checking file: %v", err)
	}
	return nil
}
