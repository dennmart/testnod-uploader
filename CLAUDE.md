# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Build the binary
go build -o testnod-uploader ./cmd/testnod-uploader

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
- `internal/testnod/` - TestNod API client for creating test runs (returns presigned upload URL)
- `internal/upload/` - Handles file upload to the presigned S3 URL
- `internal/validation/` - JUnit XML validation (checks for valid XML with `<testsuite>` element)

### Upload Flow

1. Parse CLI flags and validate inputs
2. Call TestNod API to create a test run (receives presigned S3 URL in response)
3. Upload the JUnit XML file directly to S3 using the presigned URL

Both API calls and file uploads use retry logic (3 attempts with 1 second delay) via `github.com/avast/retry-go/v4`.

### CLI Usage

```bash
./testnod-uploader -token=<project-token> [-branch=<branch>] [-commit-sha=<sha>] [-tag=<tag>]... <file.xml>
./testnod-uploader -validate <file.xml>  # Validate only, no upload
```
