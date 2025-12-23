package testnod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/avast/retry-go/v4"
)

type CreateTestRunRequest struct {
	Tags    []Tag   `json:"tags"`
	TestRun TestRun `json:"test_run"`
}

type TestRun struct {
	Metadata TestRunMetadata `json:"metadata"`
}

type Tag struct {
	Value string `json:"value"`
}

type TestRunMetadata struct {
	Branch    string `json:"branch"`
	CommitSHA string `json:"commit_sha"`
	RunURL    string `json:"run_url"`
	BuildID   string `json:"build_id"`
}

type SuccessfulServerResponse struct {
	ID           int    `json:"id"`
	Project      string `json:"project"`
	TestRunURL   string `json:"test_run_url"`
	PresignedURL string `json:"presigned_url"`
}

func CreateTestRun(uploadURL string, projectToken string, requestBody CreateTestRunRequest) (SuccessfulServerResponse, error) {
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return SuccessfulServerResponse{}, fmt.Errorf("failed to marshal request body: %w", err)
	}

	var resp *http.Response

	err = retry.Do(
		func() error {
			req, err := http.NewRequest("POST", uploadURL, bytes.NewBuffer(requestBodyBytes))
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Project-Token", projectToken)

			client := &http.Client{Timeout: 30 * time.Second}

			resp, err = client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to perform request: %w", err)
			}

			if resp.StatusCode != http.StatusCreated {
				resp.Body.Close()
				return fmt.Errorf("received non-OK response: %s", resp.Status)
			}

			return nil
		},
		retry.Delay(1000),
		retry.Attempts(3),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(attempt uint, err error) {
			fmt.Println("Could not create test run, retrying...")
		}),
	)

	if err != nil {
		return SuccessfulServerResponse{}, err
	}

	defer resp.Body.Close()

	var successfulServerResponse SuccessfulServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&successfulServerResponse); err != nil {
		return SuccessfulServerResponse{}, fmt.Errorf("failed to decode response body: %w", err)
	}

	return successfulServerResponse, nil
}
