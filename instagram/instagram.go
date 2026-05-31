package instagram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

	// Step 2: Wait for the media container to finish processing.
	statusURL := fmt.Sprintf("https://graph.instagram.com/v22.0/%s?fields=status_code,status", containerResp.ID)
	maxStatusChecks := 30
	statusDelay := 10 * time.Second

	for i := 0; i < maxStatusChecks; i++ {
		statusReq, err := http.NewRequest("GET", statusURL, nil)
		if err != nil {
			return fmt.Errorf("error creating status request: %v", err)
		}
		statusReq.Header.Set("Authorization", "Bearer "+bearerToken)

		statusResp, err := client.Do(statusReq)
		if err != nil {
			return fmt.Errorf("error checking container status: %v", err)
		}

		statusRespBytes, err := ioutil.ReadAll(statusResp.Body)
		statusResp.Body.Close()
		if err != nil {
			return fmt.Errorf("error reading status response: %v", err)
		}

		fmt.Printf("Status check %d - response (status %d): %s\n", i+1, statusResp.StatusCode, string(statusRespBytes))

		var statusResult struct {
			StatusCode string `json:"status_code"`
			Status     string `json:"status"`
		}
		if err := json.Unmarshal(statusRespBytes, &statusResult); err != nil {
			return fmt.Errorf("error parsing status response: %v", err)
		}

		if statusResult.StatusCode == "FINISHED" {
			fmt.Println("Media container processing finished.")
			break
		}
		if statusResult.StatusCode == "ERROR" {
			return fmt.Errorf("media container processing failed: %s", statusResult.Status)
		}
		if i == maxStatusChecks-1 {
			return fmt.Errorf("media container still not ready after %d status checks", maxStatusChecks)
		}

		time.Sleep(statusDelay)
	}

	// Step 3: Publish the media container.
	publishURL := fmt.Sprintf("https://graph.instagram.com/v22.0/%s/media_publish", igUserID)
	publishPayload := map[string]string{
		"creation_id": containerResp.ID,
	}
	publishBody, err := json.Marshal(publishPayload)
	if err != nil {
		return fmt.Errorf("error marshalling publish payload: %v", err)
	}

	pubReq, err := http.NewRequest("POST", publishURL, bytes.NewBuffer(publishBody))
	if err != nil {
		return fmt.Errorf("error creating publish request: %v", err)
	}
	pubReq.Header.Set("Content-Type", "application/json")
	pubReq.Header.Set("Authorization", "Bearer "+bearerToken)

	pubResp, err := client.Do(pubReq)
	if err != nil {
		return fmt.Errorf("error publishing media container: %v", err)
	}
	defer pubResp.Body.Close()

	publishRespBytes, err := ioutil.ReadAll(pubResp.Body)
	if err != nil {
		return fmt.Errorf("error reading publish response: %v", err)
	}

	fmt.Printf("Media publish response (status %d): %s\n", pubResp.StatusCode, string(publishRespBytes))

	if pubResp.StatusCode != http.StatusOK {
		return fmt.Errorf("error publishing media container: status %d, response: %s", pubResp.StatusCode, string(publishRespBytes))
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
