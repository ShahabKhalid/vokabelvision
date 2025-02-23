package cloudinary

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

func UploadVideo(cloudinaryURL, videoPath string) (string, string) {
	ctx := context.Background()

	// Initialize Cloudinary from the CLOUDINARY_URL environment variable.
	// The CLOUDINARY_URL format is: cloudinary://<api_key>:<api_secret>@<cloud_name>
	cld, err := cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		log.Fatalf("failed to create Cloudinary instance: %v", err)
	}

	// Open the video file to upload.
	file, err := os.Open(videoPath)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	// Upload the video using the Cloudinary Go SDK.
	// Set ResourceType to "video" to indicate that this is a video upload.
	resp, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		ResourceType: "video",
		// Optionally, you can set PublicID or other parameters here.
	})
	if err != nil {
		log.Fatalf("failed to upload video: %v", err)
	}

	// Print the secure URL returned from Cloudinary.
	fmt.Printf("Upload successful! Secure URL: %s\n", resp.SecureURL)
	return resp.SecureURL, resp.PublicID
}

func DeleteVideo(cloudinaryURL, publicID string) {
	ctx := context.Background()

	// Initialize Cloudinary using the CLOUDINARY_URL environment variable.
	cld, err := cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		log.Fatalf("failed to create Cloudinary instance: %v", err)
	}

	// Call the Destroy function specifying the ResourceType as "video".
	result, err := cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: "video",
	})
	if err != nil {
		log.Fatalf("failed to delete video: %v", err)
	}

	fmt.Printf("Delete result: %#v\n", result)
}
