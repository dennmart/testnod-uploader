package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type UploadRequest struct {
	Tags    []Tags  `json:"tags"`
	TestRun TestRun `json:"test_run"`
}

type TestRun struct {
	Metadata TestRunMetadata `json:"metadata"`
}

type Tags struct {
	Value string `json:"value"`
}

type TestRunMetadata struct {
	Branch    string `json:"branch"`
	CommitSHA string `json:"commit_sha"`
	RunURL    string `json:"run_url"`
}

type FailedServerResponse struct {
	ErrorMsg string `json:"error_message"`
}

type SuccessfulServerResponse struct {
	ID           int    `json:"id"`
	Project      string `json:"project"`
	TestRunURL   string `json:"test_run_url"`
	PresignedURL string `json:"presigned_url"`
}

func UploadJUnitXmlFile(filePath string, uploadURL string, projectToken string, requestBody UploadRequest) (int, string, error) {
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return 0, "", fmt.Errorf("failed to marshal request body: %w", err)
	}

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

		return resp.StatusCode, "", fmt.Errorf("received non-OK response: %s", failedServerResponse.ErrorMsg)
	}
	// TODO: Handle other potential errors

	var successfulServerResponse SuccessfulServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&successfulServerResponse); err != nil {
		return 0, "", fmt.Errorf("failed to decode response body: %w", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return 0, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	uploadReq, err := http.NewRequest("PUT", successfulServerResponse.PresignedURL, file)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create upload request: %w", err)
	}

	// Need to get the file size to set the Content-Length header,
	// otherwise the server will reject the request since Go's http client
	// will use Transfer-Encoding: chunked without a Content-Length header.
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, "", fmt.Errorf("failed to stat file: %w", err)
	}

	uploadReq.ContentLength = fileInfo.Size()
	uploadReq.Header.Set("Content-Type", "application/xml")

	uploadResp, err := client.Do(uploadReq)
	if err != nil {
		return 0, "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusOK {
		return uploadResp.StatusCode, "", fmt.Errorf("failed to upload file")
	}

	return resp.StatusCode, successfulServerResponse.TestRunURL, nil
}
