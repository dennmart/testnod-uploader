package testnod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/avast/retry-go/v5"

	"testnod-uploader/internal/debug"
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
	TestRunID    int    `json:"test_run_id"`
	UploadID     int    `json:"upload_id"`
	TestRunURL   string `json:"test_run_url"`
	PresignedURL string `json:"presigned_url"`
}

const retryAttempts = 3

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	retryDelay = 1 * time.Second
)

func CreateTestRun(uploadURL string, projectToken string, requestBody CreateTestRunRequest) (SuccessfulServerResponse, error) {
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return SuccessfulServerResponse{}, fmt.Errorf("failed to marshal request body: %w", err)
	}

	var resp *http.Response

	err = retry.New(
		retry.Delay(retryDelay),
		retry.Attempts(retryAttempts),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(attempt uint, err error) {
			debug.Log("retry attempt %d: %v", attempt, err)
			fmt.Println("Could not create test run, retrying...")
		}),
	).Do(
		func() error {
			req, err := http.NewRequest("POST", uploadURL, bytes.NewBuffer(requestBodyBytes))
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Project-Token", projectToken)

			debug.Log("request: %s %s content-type=%s", req.Method, req.URL, req.Header.Get("Content-Type"))
			resp, err = httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to perform request: %w", err)
			}
			debug.Log("response: status=%d", resp.StatusCode)

			if resp.StatusCode != http.StatusCreated {
				resp.Body.Close()
				return fmt.Errorf("received non-OK response: %s", resp.Status)
			}

			return nil
		},
	)

	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return SuccessfulServerResponse{}, err
	}

	defer resp.Body.Close()

	var successfulServerResponse SuccessfulServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&successfulServerResponse); err != nil {
		return SuccessfulServerResponse{}, fmt.Errorf("failed to decode response body: %w", err)
	}

	debug.Log("response body: id=%d project=%s test_run_id=%d upload_id=%d test_run_url=%s", successfulServerResponse.ID, successfulServerResponse.Project, successfulServerResponse.TestRunID, successfulServerResponse.UploadID, successfulServerResponse.TestRunURL)
	return successfulServerResponse, nil
}

type UploadFailureRequest struct {
	TestRunID      int    `json:"test_run_id"`
	UploadID       int    `json:"upload_id"`
	FailureMessage string `json:"failure_message"`
}

func NotifyUploadFailure(baseURL string, projectToken string, uploadID int, testRunID int, failureMessage string) error {
	failureURL := baseURL + "/integrations/test_runs/upload_failed"
	debug.Log("NotifyUploadFailure URL: %s", failureURL)

	requestBodyBytes, err := json.Marshal(UploadFailureRequest{
		TestRunID:      testRunID,
		UploadID:       uploadID,
		FailureMessage: failureMessage,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	err = retry.New(
		retry.Delay(retryDelay),
		retry.Attempts(retryAttempts),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(attempt uint, err error) {
			debug.Log("retry attempt %d: %v", attempt, err)
			fmt.Println("Could not notify TestNod of upload failure, retrying...")
		}),
	).Do(
		func() error {
			req, err := http.NewRequest("POST", failureURL, bytes.NewBuffer(requestBodyBytes))
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Project-Token", projectToken)

			debug.Log("request: %s %s", req.Method, req.URL)
			resp, err := httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to perform request: %w", err)
			}
			defer resp.Body.Close()

			debug.Log("response: status=%d", resp.StatusCode)

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("received non-OK response: %s", resp.Status)
			}

			return nil
		},
	)

	return err
}
