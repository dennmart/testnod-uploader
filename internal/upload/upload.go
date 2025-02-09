package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type UploadRequest struct {
	TestRun TestRun `json:"test_run"`
}

type TestRun struct {
	Metadata TestRunMetadata `json:"metadata"`
}

type TestRunMetadata struct {
	Branch    string   `json:"branch"`
	CommitSHA string   `json:"commit_sha"`
	RunURL    string   `json:"run_url"`
	Tags      []string `json:"tags"`
}

type FailedServerResponse struct {
	ErrorMsg string `json:"error_message"`
}

type SuccessfulServerResponse struct {
	ID           int    `json:"id"`
	Project      string `json:"project"`
	PresignedURL string `json:"presigned_url"`
}

func UploadJUnitXmlFile(filePath string, uploadURL string, projectToken string, requestBody UploadRequest) (int, string, error) {
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return 0, "", fmt.Errorf("failed to marshal request body: %w", err)
	}
	fmt.Println("requestBodyBytes", string(requestBodyBytes))

	req, err := http.NewRequest("POST", uploadURL, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Project-Token", projectToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var failedServerResponse FailedServerResponse
		if err := json.NewDecoder(resp.Body).Decode(&failedServerResponse); err != nil {
			return 0, "", fmt.Errorf("failed to decode response body: %w", err)
		}

		return resp.StatusCode, failedServerResponse.ErrorMsg, fmt.Errorf("received non-OK response: %s", resp.Status)
	}
	// TODO: Handle other potential errors

	var successfulServerResponse SuccessfulServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&successfulServerResponse); err != nil {
		return 0, "", fmt.Errorf("failed to decode response body: %w", err)
	}

	// TODO: Do upload using presigned URL
	return resp.StatusCode, successfulServerResponse.PresignedURL, nil
}
