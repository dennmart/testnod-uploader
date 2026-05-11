package upload

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/avast/retry-go/v4"

	"testnod-uploader/internal/debug"
)

const retryAttempts = 3

var (
	httpClient = &http.Client{Timeout: 60 * time.Second}
	retryDelay = 1 * time.Second
)

// S3Metadata holds the values for the x-amz-meta-* headers that TestNod's
// presigned URL is signed against. The PUT must send these exact values
// or S3 will reject the request with SignatureDoesNotMatch.
type S3Metadata struct {
	ProjectID string
	TestRunID int
	UploadID  int
}

func UploadJUnitXmlFile(filePath string, uploadURL string, meta S3Metadata) error {
	err := retry.Do(
		func() error {
			// Open the file for each retry attempt
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file %q: %w", filePath, err)
			}
			defer file.Close()

			req, err := http.NewRequest("PUT", uploadURL, file)
			if err != nil {
				return fmt.Errorf("failed to create upload request: %w", err)
			}

			// Need to get the file size to set the Content-Length header,
			// otherwise the server will reject the request since Go's http client
			// will use Transfer-Encoding: chunked without a Content-Length header.
			fileInfo, err := file.Stat()
			if err != nil {
				return fmt.Errorf("failed to stat file: %w", err)
			}

			req.ContentLength = fileInfo.Size()
			req.Header.Set("Content-Type", "application/xml")
			req.Header.Set("x-amz-meta-project_id", meta.ProjectID)
			req.Header.Set("x-amz-meta-test_run_id", strconv.Itoa(meta.TestRunID))
			req.Header.Set("x-amz-meta-upload_id", strconv.Itoa(meta.UploadID))

			debug.Log("file: name=%s size=%d bytes", fileInfo.Name(), fileInfo.Size())
			debug.Log("request: %s content-length=%d project_id=%s test_run_id=%d upload_id=%d", req.Method, req.ContentLength, meta.ProjectID, meta.TestRunID, meta.UploadID)
			resp, err := httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to upload file: %w", err)
			}

			debug.Log("response: status=%d", resp.StatusCode)

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
				resp.Body.Close()
				return fmt.Errorf("failed to upload file: status %d: %s", resp.StatusCode, string(bodyBytes))
			}

			resp.Body.Close()
			return nil
		},
		retry.Delay(retryDelay),
		retry.Attempts(retryAttempts),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(attempt uint, err error) {
			debug.Log("retry attempt %d: %v", attempt, err)
		}),
	)

	return err
}
