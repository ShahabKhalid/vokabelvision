package utils

import (
	"fmt"

	"github.com/fogleman/gg"
)

// CreateReelImage creates a 9:16 image with the full input image preserved.
// The original image is scaled to fit inside the reel canvas, no cropping.
func CreateReelImage(inputPath, outputPath, text, fontPath string) (string, error) {
	// Load input image
	im, err := gg.LoadImage(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to load input image: %w", err)
	}
	bounds := im.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	// Target Reel aspect ratio
	targetAspect := 9.0 / 16.0
	origAspect := float64(origW) / float64(origH)

	var canvasW, canvasH int

	if origAspect > targetAspect {
		// Image is too wide → set width, adjust height to 16:9
		canvasW = origW
		canvasH = int(float64(origW) / targetAspect)
	} else {
		// Image is too tall/narrow → set height, adjust width to 9:16
		canvasH = origH
		canvasW = int(float64(origH) * targetAspect)
	}

	// Create canvas with white background
	dc := gg.NewContext(canvasW, canvasH)
	dc.SetRGB(1, 1, 1) // white
	dc.Clear()

	// Center the original image
	offsetX := (canvasW - origW) / 2
	offsetY := (canvasH - origH) / 2
	dc.DrawImage(im, offsetX, offsetY)

	// Load font
	fontSize := float64(canvasW) * 0.05 // text ~10% of width
	if err := dc.LoadFontFace(fontPath, fontSize); err != nil {
		return "", fmt.Errorf("failed to load font: %w", err)
	}

	// Draw text at bottom center
	dc.SetRGB(0, 0, 0) // black
	x := float64(canvasW) / 2
	y := float64(canvasH) - fontSize - 50
	dc.DrawStringAnchored(text, x, y, 0.5, 0.5)

	// Save
	if err := dc.SavePNG(outputPath); err != nil {
		return "", fmt.Errorf("failed to save output: %w", err)
	}

	return outputPath, nil
}
