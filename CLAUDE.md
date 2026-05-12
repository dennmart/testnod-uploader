# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Build the binary
go build -o testnod-uploader ./cmd/testnod-uploader

# Build with debug logging (prints [DEBUG] lines to stderr)
go build -tags debug -o testnod-uploader ./cmd/testnod-uploader

# Run with debug logging (without building)
go run -tags debug ./cmd/testnod-uploader

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/validation
go test ./internal/upload
go test ./internal/testnod
go test ./cmd/testnod-uploader

# Run a specific test
go test ./cmd/testnod-uploader -run TestParseFlags

# Run tests with verbose output
go test -v ./...
```

## Project Architecture

This is a CLI tool for uploading JUnit XML test results to TestNod (testnod.com). The tool validates JUnit XML files and uploads them via a two-step process.

### Package Structure

- `cmd/testnod-uploader/` - CLI entry point with flag parsing and orchestration
- `internal/debug/` - Build-tag-based debug logging (`-tags debug` enables output, no-op otherwise)
- `internal/testnod/` - TestNod API client for creating test runs (returns presigned upload URL)
- `internal/upload/` - Handles file upload to the presigned S3 URL
- `internal/validation/` - JUnit XML validation (checks for valid XML with `<testsuite>` element)

### Upload Flow

1. Parse CLI flags and validate inputs (`-build-id` is required outside of `-validate` mode — it groups parallel/matrix shards into one logical test run on the server)
2. Call TestNod API to create a test run; the response includes `project_id`, `test_run_id`, `upload_id`, and a presigned S3 URL
3. PUT the JUnit XML file to the presigned URL with `Content-Type: application/xml`. The S3 object metadata (`project_id`, `test_run_id`, `upload_id`) is hoisted into the URL's query string by the presigner — no extra request headers are needed.
4. On upload failure, notify TestNod via `POST /integrations/test_runs/uploads/{upload_id}/failed` (per-upload, not per-run) with body `{project_id, test_run_id, failure_message}` and the `Testnod-Auth` header

Both API calls and file uploads use retry logic (3 attempts with 1 second delay) via `github.com/avast/retry-go/v4`.

This binary owns per-upload state only. Run-level finalization is the webapp's job — CI calls `/integrations/test_runs/finalize` separately to aggregate results across all uploads.

### CLI Usage

```bash
./testnod-uploader -token=<project-token> -build-id=<build-id> [-branch=<branch>] [-commit-sha=<sha>] [-tag=<tag>]... <file.xml>
./testnod-uploader -validate <file.xml>  # Validate only, no upload (no -build-id needed)
```
