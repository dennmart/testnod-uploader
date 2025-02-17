package upload

import (
	"fmt"
	"net/http"
	"os"
)

func UploadJUnitXmlFile(filePath string, uploadURL string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	uploadReq, err := http.NewRequest("PUT", uploadURL, file)
	if err != nil {
		return 0, fmt.Errorf("failed to create upload request: %w", err)
	}

	// Need to get the file size to set the Content-Length header,
	// otherwise the server will reject the request since Go's http client
	// will use Transfer-Encoding: chunked without a Content-Length header.
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}

	uploadReq.ContentLength = fileInfo.Size()
	uploadReq.Header.Set("Content-Type", "application/xml")

	client := &http.Client{}
	uploadResp, err := client.Do(uploadReq)
	if err != nil {
		return 0, fmt.Errorf("failed to upload file: %w", err)
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusOK {
		return uploadResp.StatusCode, fmt.Errorf("failed to upload file")
	}

	return uploadResp.StatusCode, nil
}
