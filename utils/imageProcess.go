package utils

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/fogleman/gg"
)

// BrandColor defines the brand's primary color (deep blue-purple)
var BrandColor = color.RGBA{R: 25, G: 45, B: 85, A: 255} // #192D55

// BrandAccentColor defines the brand's accent color (vibrant coral/orange)
var BrandAccentColor = color.RGBA{R: 255, G: 87, B: 34, A: 255} // #FF5722 (Deep Orange)

// CreateReelImage creates a branded 9:16 reel image with consistent styling.
// The original image is centered with a consistent brand background and overlay.
// text is the main German text. englishText is optional (can be empty).
func CreateReelImage(inputPath, outputPath, text, fontPath string) (string, error) {
	return CreateReelImageWithTranslation(inputPath, outputPath, text, "", fontPath)
}

// CreateReelImageWithTranslation creates a branded 9:16 reel with German and English text.
func CreateReelImageWithTranslation(inputPath, outputPath, germanText, englishText, fontPath string) (string, error) {
	// Load input image
	im, err := gg.LoadImage(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to load input image: %w", err)
	}
	bounds := im.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	_ = origW // May be used for future layout calculations
	_ = origH

	// Instagram Reel standard: 1080x1920 (9:16)
	const canvasW, canvasH = 1080, 1920

	// Create canvas with brand color background
	dc := gg.NewContext(canvasW, canvasH)
	// Fill with brand primary color
	dc.SetColor(BrandColor)
	dc.Clear()

	// Calculate scaled image to fit within the canvas (with margins)
	marginTop := 120
	marginBottom := 300
	marginSides := 40
	availableW := canvasW - (marginSides * 2)
	availableH := canvasH - marginTop - marginBottom

	origAspect := float64(origW) / float64(origH)
	targetAspect := float64(availableW) / float64(availableH)

	var scaledW, scaledH int
	if origAspect > targetAspect {
		// Image is wider, fit by width
		scaledW = availableW
		scaledH = int(float64(availableW) / origAspect)
	} else {
		// Image is taller, fit by height
		scaledH = availableH
		scaledW = int(float64(availableH) * origAspect)
	}

	// Draw accent border/frame around image area
	frameX := float64(canvasW-scaledW) / 2
	frameY := float64(marginTop + (availableH-scaledH)/2)
	dc.SetColor(BrandAccentColor)
	dc.DrawRectangle(frameX-8, frameY-8, float64(scaledW+16), float64(scaledH+16))
	dc.Stroke()

	// Scale and draw the original image (use resizeImage for better quality)
	scaledIm := resizeImage(im, scaledW, scaledH)
	offsetX := (canvasW - scaledW) / 2
	offsetY := marginTop + (availableH-scaledH)/2
	dc.DrawImage(scaledIm, offsetX, offsetY)

	// Draw semi-transparent overlay at bottom for text readability
	dc.SetColor(color.RGBA{R: 0, G: 0, B: 0, A: 180})
	overlayHeight := 280.0
	dc.DrawRectangle(0, float64(canvasH)-overlayHeight, float64(canvasW), overlayHeight)
	dc.Fill()

	// Draw accent line above text overlay
	dc.SetColor(BrandAccentColor)
	dc.SetLineWidth(4)
	dc.DrawLine(100, float64(canvasH)-overlayHeight, float64(canvasW)-100, float64(canvasH)-overlayHeight)
	dc.Stroke()

	// Load font and draw text
	fontSize := 64.0
	if err := dc.LoadFontFace(fontPath, fontSize); err != nil {
		return "", fmt.Errorf("failed to load font: %w", err)
	}

	// Draw main German text in white (larger)
	dc.SetColor(color.White)
	x := float64(canvasW) / 2
	y := float64(canvasH) - 150
	dc.DrawStringAnchored(germanText, x, y, 0.5, 0.5)

	// Draw English meaning below German text in accent color (if provided) with wrapping and adaptive font size
	if englishText != "" {
		maxWidth := float64(canvasW) - 80 // 40px margin on each side
		availableHeight := 200.0 // Space available in overlay for English text

		// Try font sizes from 28 down to 12, finding the best fit
		bestFontSize := 28.0
		var wrappedLines []string

		for tryFontSize := 28.0; tryFontSize >= 12.0; tryFontSize -= 2.0 {
			if err := dc.LoadFontFace(fontPath, tryFontSize); err != nil {
				continue
			}

			lines := wrapText(dc, englishText, maxWidth)
			lineHeight := tryFontSize * 1.2 // Line spacing multiplier
			totalHeight := float64(len(lines)) * lineHeight

			if totalHeight <= availableHeight {
				bestFontSize = tryFontSize
				wrappedLines = lines
				break
			}
		}

		// Load best font size and draw wrapped text
		if err := dc.LoadFontFace(fontPath, bestFontSize); err == nil {
			dc.SetColor(BrandAccentColor)

			lineHeight := bestFontSize * 1.2
			startY := float64(canvasH) - 70.0 // Start below the German text

			for i, line := range wrappedLines {
				yPos := startY + float64(i)*lineHeight
				dc.DrawStringAnchored(line, x, yPos, 0.5, 0.5)
			}
		}
	}

	// Save
	if err := dc.SavePNG(outputPath); err != nil {
		return "", fmt.Errorf("failed to save output: %w", err)
	}

	return outputPath, nil
}

// resizeImage resizes an image to fit within the given width and height while preserving aspect ratio
func resizeImage(im image.Image, maxW, maxH int) image.Image {
	bounds := im.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	// Calculate scale factor to fit image in maxW x maxH
	scaleX := float64(maxW) / float64(origW)
	scaleY := float64(maxH) / float64(origH)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newW := int(float64(origW) * scale)
	newH := int(float64(origH) * scale)

	// Create canvas with exact dimensions needed
	dc := gg.NewContext(newW, newH)
	// Draw image scaled to fit
	dc.DrawImage(im, 0, 0)
	return dc.Image()
}

// wrapText wraps text to fit within maxWidth, breaking on word boundaries
func wrapText(dc *gg.Context, text string, maxWidth float64) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		bounds, _ := dc.MeasureString(testLine)
		if bounds <= maxWidth {
			currentLine = testLine
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
