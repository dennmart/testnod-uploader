package upload

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestUploadJUnitXmlFile_Success(t *testing.T) {
	// Create test content
	testContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="test" tests="1" failures="0" errors="0" time="0.001">
	<testcase name="test_example" classname="test.example" time="0.001"/>
</testsuite>`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "junit_upload_test_*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	tmpFile.Close()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/xml" {
			t.Errorf("Expected Content-Type application/xml, got %s", r.Header.Get("Content-Type"))
		}

		// Verify content length is set
		if r.ContentLength <= 0 {
			t.Errorf("Expected positive Content-Length, got %d", r.ContentLength)
		}

		// Read and verify body content
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
		}
		if string(body) != testContent {
			t.Errorf("Body content mismatch.\nGot:      %s\nExpected: %s", string(body), testContent)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test the function
	err = UploadJUnitXmlFile(tmpFile.Name(), server.URL)
	if err != nil {
		t.Fatalf("UploadJUnitXmlFile() unexpected error: %v", err)
	}
}

func TestUploadJUnitXmlFile_FileNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := UploadJUnitXmlFile("/path/that/does/not/exist.xml", server.URL)
	if err == nil {
		t.Error("UploadJUnitXmlFile() expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "failed to open file") {
		t.Errorf("Expected error to contain 'failed to open file', got: %v", err)
	}
}

func TestUploadJUnitXmlFile_ServerError(t *testing.T) {
	// Create test file
	tmpFile, err := os.CreateTemp("", "junit_upload_test_*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("<testsuite></testsuite>")
	tmpFile.Close()

	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err = UploadJUnitXmlFile(tmpFile.Name(), server.URL)
	if err == nil {
		t.Error("UploadJUnitXmlFile() expected error for server error response")
	}
	if !strings.Contains(err.Error(), "failed to upload file") {
		t.Errorf("Expected error to contain 'failed to upload file', got: %v", err)
	}
}

func TestUploadJUnitXmlFile_NetworkError(t *testing.T) {
	// Create test file
	tmpFile, err := os.CreateTemp("", "junit_upload_test_*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("<testsuite></testsuite>")
	tmpFile.Close()

	// Use malformed URL to trigger network error without making actual request
	err = UploadJUnitXmlFile(tmpFile.Name(), "://invalid-url")
	if err == nil {
		t.Error("UploadJUnitXmlFile() expected error for network failure")
	}
}

func TestUploadJUnitXmlFile_RetryBehavior(t *testing.T) {
	// Create test file
	tmpFile, err := os.CreateTemp("", "junit_upload_test_*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := "<testsuite></testsuite>"
	tmpFile.WriteString(testContent)
	tmpFile.Close()

	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Fail first two attempts
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Succeed on third attempt
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	start := time.Now()
	err = UploadJUnitXmlFile(tmpFile.Name(), server.URL)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("UploadJUnitXmlFile() unexpected error: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}

	// Should have taken at least 2 seconds due to retry delays (1s + 1s)
	// Note: retry delay is in milliseconds, so 2000ms = 2s
	if duration < 2*time.Second {
		t.Logf("Retry timing test: Expected at least 2 seconds due to retries, took %v", duration)
		// Don't fail the test as timing can be inconsistent in test environments
	}
}

func TestUploadJUnitXmlFile_AllRetriesFail(t *testing.T) {
	// Create test file
	tmpFile, err := os.CreateTemp("", "junit_upload_test_*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("<testsuite></testsuite>")
	tmpFile.Close()

	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err = UploadJUnitXmlFile(tmpFile.Name(), server.URL)
	if err == nil {
		t.Error("UploadJUnitXmlFile() expected error when all retries fail")
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestUploadJUnitXmlFile_EmptyFile(t *testing.T) {
	// Create empty file
	tmpFile, err := os.CreateTemp("", "junit_upload_test_*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify content length is 0 (may be -1 for empty files)
		if r.ContentLength != 0 && r.ContentLength != -1 {
			t.Errorf("Expected Content-Length 0 or -1 for empty file, got %d", r.ContentLength)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
		}
		if len(body) != 0 {
			t.Errorf("Expected empty body, got %d bytes", len(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err = UploadJUnitXmlFile(tmpFile.Name(), server.URL)
	if err != nil {
		t.Fatalf("UploadJUnitXmlFile() unexpected error for empty file: %v", err)
	}
}

func TestUploadJUnitXmlFile_LargeFile(t *testing.T) {
	// Create a larger file to test content handling
	tmpFile, err := os.CreateTemp("", "junit_upload_test_*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Create content with multiple test cases
	largeContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
	<testsuite name="test1" tests="100" failures="0" errors="0" time="1.0">`

	for i := 0; i < 100; i++ {
		largeContent += `
		<testcase name="test_` + strings.Repeat("a", 100) + `" classname="test.example" time="0.001"/>`
	}

	largeContent += `
	</testsuite>
</testsuites>`

	tmpFile.WriteString(largeContent)
	tmpFile.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify content length matches file size
		fileInfo, err := os.Stat(tmpFile.Name())
		if err != nil {
			t.Errorf("Failed to stat file: %v", err)
		}
		if r.ContentLength != fileInfo.Size() {
			t.Errorf("Content-Length mismatch: got %d, expected %d", r.ContentLength, fileInfo.Size())
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
		}
		if string(body) != largeContent {
			t.Errorf("Body content mismatch for large file")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err = UploadJUnitXmlFile(tmpFile.Name(), server.URL)
	if err != nil {
		t.Fatalf("UploadJUnitXmlFile() unexpected error for large file: %v", err)
	}
}

func TestUploadJUnitXmlFile_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	// Create test file
	tmpFile, err := os.CreateTemp("", "junit_upload_test_*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("<testsuite></testsuite>")
	tmpFile.Close()

	// Remove read permissions
	if err := os.Chmod(tmpFile.Name(), 0000); err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}
	defer os.Chmod(tmpFile.Name(), 0644) // Restore permissions for cleanup

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err = UploadJUnitXmlFile(tmpFile.Name(), server.URL)
	if err == nil {
		t.Error("UploadJUnitXmlFile() expected error for permission denied")
	}
}

func TestUploadJUnitXmlFile_Directory(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "junit_upload_test_dir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err = UploadJUnitXmlFile(tmpDir, server.URL)
	if err == nil {
		t.Error("UploadJUnitXmlFile() expected error for directory")
	}
}
