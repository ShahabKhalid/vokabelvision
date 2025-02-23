package instagram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// PublishVideo uploads and publishes a video as a Reel using the Instagram Graph API.
// videoURL must be publicly accessible. igUserID and bearerToken are required for authentication.
// caption is optional.
func PublishVideo(igUserID, bearerToken, videoURL, caption string) error {
	client := &http.Client{}

	// Step 1: Create a media container for the video.
	containerURL := fmt.Sprintf("https://graph.instagram.com/v22.0/%s/media", igUserID)
	containerPayload := map[string]string{
		"media_type": "REELS", // Use "REELS" instead of "VIDEO"
		"video_url":  videoURL,
		"caption":    caption,
	}
	containerBody, err := json.Marshal(containerPayload)
	if err != nil {
		return fmt.Errorf("error marshalling container payload: %v", err)
	}

	req, err := http.NewRequest("POST", containerURL, bytes.NewBuffer(containerBody))
	if err != nil {
		return fmt.Errorf("error creating container request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error creating media container: %v", err)
	}
	defer resp.Body.Close()

	containerRespBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading container response: %v", err)
	}

	fmt.Printf("Container creation response (status %d): %s\n", resp.StatusCode, string(containerRespBytes))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error creating media container: status %d, response: %s", resp.StatusCode, string(containerRespBytes))
	}

	var containerResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(containerRespBytes, &containerResp); err != nil {
		return fmt.Errorf("error parsing container response: %v", err)
	}

	// Step 2: Publish the media container with retry.
	publishURL := fmt.Sprintf("https://graph.instagram.com/v22.0/%s/media_publish", igUserID)
	publishPayload := map[string]string{
		"creation_id": containerResp.ID,
	}
	publishBody, err := json.Marshal(publishPayload)
	if err != nil {
		return fmt.Errorf("error marshalling publish payload: %v", err)
	}

	var publishRespBytes []byte
	maxRetries := 10
	delay := 5 * time.Second
	var pubResp *http.Response

	for i := 0; i < maxRetries; i++ {
		req2, err := http.NewRequest("POST", publishURL, bytes.NewBuffer(publishBody))
		if err != nil {
			return fmt.Errorf("error creating publish request: %v", err)
		}
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "Bearer "+bearerToken)

		pubResp, err = client.Do(req2)
		if err != nil {
			return fmt.Errorf("error publishing media container: %v", err)
		}

		publishRespBytes, err = ioutil.ReadAll(pubResp.Body)
		pubResp.Body.Close()
		if err != nil {
			return fmt.Errorf("error reading publish response: %v", err)
		}

		fmt.Printf("Attempt %d - Media publish response (status %d): %s\n", i+1, pubResp.StatusCode, string(publishRespBytes))

		// If the response status is OK and it doesn't indicate that Media ID is not available, break.
		if pubResp.StatusCode == http.StatusOK && !strings.Contains(string(publishRespBytes), "Media ID is not available") {
			break
		}

		// Wait before the next retry.
		time.Sleep(delay)
	}

	// Final check after retry loop.
	if pubResp.StatusCode != http.StatusOK || strings.Contains(string(publishRespBytes), "Media ID is not available") {
		return fmt.Errorf("error publishing media container after retries: status %d, response: %s", pubResp.StatusCode, string(publishRespBytes))
	}

	var publishResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(publishRespBytes, &publishResp); err != nil {
		return fmt.Errorf("error parsing publish response: %v", err)
	}

	fmt.Printf("Video published with ID: %s\n", publishResp.ID)
	return nil
}
