package video

import "os/exec"

// GenerateVideo uses FFmpeg to create a video reel from the image and audio.
func GenerateVideo(imagePath, audioPath, outputVideoPath string) error {
	// Example FFmpeg command: create a video using a static image and overlaying the audio.
	// Adjust parameters as needed for looping audio or adding pauses.
	cmd := exec.Command("ffmpeg",
		"-loop", "1",
		"-i", imagePath,
		"-i", audioPath,
		"-c:v", "libx264",
		"-t", "10", // Duration of the video (seconds); adjust as needed.
		"-pix_fmt", "yuv420p",
		"-vf", "scale=720:960", // For a 2:3 aspect ratio.
		outputVideoPath,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
