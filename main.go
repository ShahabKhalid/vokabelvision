package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"vokabelvision/chatgpt"
	"vokabelvision/cloudinary"
	"vokabelvision/config"
	"vokabelvision/elevenlabs"
	"vokabelvision/instagram"
	"vokabelvision/leonardo"
	"vokabelvision/video"

	"github.com/robfig/cron/v3"
)

func main() {

	// Define the --once flag. It defaults to false.
	once := flag.Bool("once", false, "Run the task once instead of scheduling it")

	// Parse command-line flags.
	flag.Parse()

	// Check if the --once flag is provided.
	if *once {
		GenerateAndPost()
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
			GenerateAndPost()
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

func GenerateAndPost() {
	// Load configuration.
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	// Step 1: Get vocabulary word from ChatGPT.
	vocab, err := chatgpt.GetVocab(cfg.ChatGPTAPIKey)
	if err != nil {
		log.Fatalf("Error getting vocab: %v", err)
	}
	fmt.Printf("Got vocab: %+v\n", vocab)

	// // // Step 2: Generate Leonardo.ai prompt.
	prompt := leonardo.GeneratePrompt(vocab.English, vocab.German, vocab.Sentence)
	fmt.Println("Generated Leonardo prompt:", prompt)

	// Step 3: Get image from Leonardo.ai.
	imagePath, err := leonardo.GetImage(cfg.LeonardoAPIKey, prompt)
	if err != nil {
		log.Fatalf("Error getting image: %v", err)
	}
	fmt.Println("Image saved at:", imagePath)
	// os.Exit(1)
	// Step 4: Get audio from ElevenLabs.
	audioPath, err := elevenlabs.GetAudio(cfg.ElevenLabsAPIKey, vocab.German, vocab.Sentence, cfg.ElevenLabsVoiceID)
	if err != nil {
		log.Fatalf("Error getting audio: %v", err)
	}
	fmt.Println("Audio saved at:", audioPath)

	// Step 5: Generate video reel.
	// imagePath := "vocab_image.jpg"
	// audioPath := "vocab_audio.mp3"
	outputVideoPath := "vocab_reel.mp4"

	if err := video.GenerateVideo(imagePath, audioPath, outputVideoPath); err != nil {
		log.Fatalf("Error generating video: %v", err)
	}
	fmt.Println("Video generated at:", outputVideoPath)

	videoURL, publicID := cloudinary.UploadVideo(cfg.CloudinaryURL, outputVideoPath)

	// // Step 6: Upload video to Instagram.
	captionWithTags := fmt.Sprintf("%s #love #instagood #instagram #art #happy #travel #repost #german #germanlanguage", vocab.Caption)
	if err := instagram.PublishVideo(cfg.InstagramUserID, cfg.InstagramAccessToken, videoURL, captionWithTags); err != nil {
		log.Fatalf("Error uploading video: %v", err)
	}
	fmt.Println("Reel uploaded successfully!")
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
