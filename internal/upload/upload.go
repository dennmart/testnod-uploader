package upload

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/avast/retry-go/v4"
)

const retryAttempts = 3

var (
	httpClient = &http.Client{Timeout: 60 * time.Second}
	retryDelay = 1 * time.Second
)

func UploadJUnitXmlFile(filePath string, uploadURL string) error {
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

			resp, err := httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to upload file: %w", err)
			}

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
	)

	return err
}
