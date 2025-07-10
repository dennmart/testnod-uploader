package upload

import (
	"fmt"
	"net/http"
	"os"

	"github.com/avast/retry-go/v4"
)

func UploadJUnitXmlFile(filePath string, uploadURL string) error {
	err := retry.Do(
		func() error {
			// Open the file for each retry attempt
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
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

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to upload file: %w", err)
			}

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return fmt.Errorf("failed to upload file")
			}

			resp.Body.Close()
			return nil
		},
		retry.Delay(1000),
		retry.Attempts(3),
		retry.LastErrorOnly(true),
	)

	if err != nil {
		return err
	}

	return nil
}
