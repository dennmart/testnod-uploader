package testnod

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/avast/retry-go/v5"

	"testnod-uploader/internal/debug"
)

type CreateTestRunRequest struct {
	Tags    []Tag   `json:"tags"`
	TestRun TestRun `json:"test_run"`
	Retry   bool    `json:"retry"`
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
	ProjectID    string `json:"project_id"`
	TestRunID    int    `json:"test_run_id"`
	UploadID     int    `json:"upload_id"`
	TestRunURL   string `json:"test_run_url"`
	PresignedURL string `json:"presigned_url"`
}

const retryAttempts = 3

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	retryDelay = 1 * time.Second
	// TestNod's rate limit is per minute, so a 1s wait rarely clears it. The
	// submission API docs recommend waiting a few seconds before retrying.
	rateLimitRetryDelay = 5 * time.Second
)

// StatusError is a non-success HTTP response from the TestNod API. Message is
// the server's error_message, when the body had one.
type StatusError struct {
	StatusCode int
	Status     string
	Message    string
}

func (e *StatusError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("received non-OK response: %s", e.Status)
}

// newStatusError reads the TestNod error body ({"error_message": "..."}) from
// resp so the reason (e.g. a paused organization or project) reaches the CI log.
// The caller still owns closing resp.Body.
func newStatusError(resp *http.Response) *StatusError {
	statusErr := &StatusError{StatusCode: resp.StatusCode, Status: resp.Status}

	var body struct {
		ErrorMessage string `json:"error_message"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&body); err == nil {
		statusErr.Message = body.ErrorMessage
	}

	return statusErr
}

// retryDelayFor waits longer after a 429 than after other retryable errors.
func retryDelayFor(_ uint, err error, _ retry.DelayContext) time.Duration {
	var statusErr *StatusError
	if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusTooManyRequests {
		return rateLimitRetryDelay
	}
	return retryDelay
}

func CreateTestRun(uploadURL string, projectToken string, requestBody CreateTestRunRequest) (SuccessfulServerResponse, error) {
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return SuccessfulServerResponse{}, fmt.Errorf("failed to marshal request body: %w", err)
	}

	var resp *http.Response

	err = retry.New(
		retry.DelayType(retryDelayFor),
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
				statusErr := newStatusError(resp)
				resp.Body.Close()

				// 4xx responses won't change on retry (bad token, missing build ID,
				// paused organization or project), except a rate limit and a 409,
				// which TestNod returns when two shards of the same build race to
				// create the test run.
				if resp.StatusCode >= 400 && resp.StatusCode < 500 &&
					resp.StatusCode != http.StatusTooManyRequests &&
					resp.StatusCode != http.StatusConflict {
					return retry.Unrecoverable(statusErr)
				}

				return statusErr
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

	debug.Log("response body: id=%d project_id=%s test_run_id=%d upload_id=%d test_run_url=%s", successfulServerResponse.ID, successfulServerResponse.ProjectID, successfulServerResponse.TestRunID, successfulServerResponse.UploadID, successfulServerResponse.TestRunURL)

	// The upload itself only needs the presigned URL, but the failure callback
	// later is keyed on the IDs.
	if successfulServerResponse.PresignedURL == "" || successfulServerResponse.UploadID == 0 || successfulServerResponse.TestRunID == 0 {
		return SuccessfulServerResponse{}, fmt.Errorf("incomplete response from server: missing presigned URL, upload ID, or test run ID")
	}

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
		retry.DelayType(retryDelayFor),
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
				statusErr := newStatusError(resp)

				if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
					return retry.Unrecoverable(statusErr)
				}

				return statusErr
			}

			return nil
		},
	)

	return err
}
