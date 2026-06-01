package edgetts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// GetAudio generates audio using Microsoft Edge TTS via edge-tts CLI.
// Requires edge-tts to be installed: pip install edge-tts
func GetAudio(text string, voiceID string) (string, error) {
	audioPath := "vocab_audio.mp3"

	// Create temp directory for intermediate files
	tempDir, err := os.MkdirTemp("", "edgetts")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate audio for: text repeated 3 times
	var audioFiles []string
	for i := 0; i < 3; i++ {
		tempFile := filepath.Join(tempDir, fmt.Sprintf("part%d.mp3", i))

		// Run edge-tts command with retries
		var output []byte
		maxRetries := 3
		for attempt := 0; attempt < maxRetries; attempt++ {
			cmd := exec.Command("edge-tts",
				"--voice", voiceID,
				"--rate=-10%",
				"--text", text,
				"--write-media", tempFile,
			)
			output, err = cmd.CombinedOutput()
			if err == nil {
				break
			}
			if attempt < maxRetries-1 {
				fmt.Printf("edge-tts attempt %d failed, retrying in 5s...\n", attempt+1)
				time.Sleep(5 * time.Second)
			}
		}
		if err != nil {
			return "", fmt.Errorf("edge-tts failed after %d attempts: %w, output: %s", maxRetries, err, string(output))
		}

		audioFiles = append(audioFiles, tempFile)
	}

	// Concatenate all audio files with silence between them
	outFile, err := os.Create(audioPath)
	if err != nil {
		return "", fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	silence := generateSilence(2000) // 2 seconds of silence

	for i, audioFile := range audioFiles {
		data, err := os.ReadFile(audioFile)
		if err != nil {
			return "", fmt.Errorf("read audio part: %w", err)
		}
		outFile.Write(data)

		// Add silence between repetitions (except after last)
		if i < len(audioFiles)-1 {
			outFile.Write(silence)
		}
	}

	return audioPath, nil
}

// generateSilence creates silent MP3 frames for the given duration in milliseconds
func generateSilence(ms int) []byte {
	// MP3 frame for silence (48kbps, 24000Hz) - approximately 26ms per frame
	silentFrame := []byte{
		0xFF, 0xF3, 0x84, 0xC4, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	framesNeeded := ms / 26
	var result []byte
	for i := 0; i < framesNeeded; i++ {
		result = append(result, silentFrame...)
	}
	return result
}

// ListVoices returns available German voice options
func ListVoices() []string {
	return []string{
		"de-DE-ConradNeural",
		"de-DE-KatjaNeural",
		"de-DE-AmalaNeural",
		"de-DE-BerndNeural",
		"de-DE-ChristophNeural",
		"de-DE-ElkeNeural",
		"de-DE-GiselaNeural",
		"de-DE-KasperNeural",
		"de-DE-KillianNeural",
		"de-DE-KlarissaNeural",
		"de-DE-KlausNeural",
		"de-DE-LouisaNeural",
		"de-DE-MajaNeural",
		"de-DE-RalfNeural",
		"de-DE-TanjaNeural",
	}
}
