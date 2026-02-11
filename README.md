# testnod-uploader

A CLI tool for uploading JUnit XML test results to [TestNod](https://testnod.com). It validates JUnit XML files and uploads them via a two-step process: creating a test run through the TestNod API, then uploading the XML file directly to S3 using a presigned URL.

## Installation

Build from source (requires Go 1.25.5+):

```bash
go build -o testnod-uploader ./cmd/testnod-uploader
```

Pre-built binaries for Linux, macOS, and Windows (amd64 and arm64) are available from the release workflow.

## Usage

```bash
# Upload test results
./testnod-uploader -token=<project-token> [options] <file.xml>

# Validate a JUnit XML file without uploading
./testnod-uploader -validate <file.xml>
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `-token` | Yes (unless `-validate`) | TestNod project token |
| `-validate` | No | Validate the XML file only, skip upload |
| `-branch` | No | Branch name to associate with the test run |
| `-commit-sha` | No | Commit SHA to associate with the test run |
| `-run-url` | No | URL to the CI/CD run |
| `-build-id` | No | Build identifier for the CI/CD run |
| `-tag` | No | Tag for the test run (repeatable) |
| `-ignore-failures` | No | Always exit 0, even if upload fails |

### Examples

```bash
# Validate only
./testnod-uploader -validate junit_results.xml

# Basic upload
./testnod-uploader -token=abc123 junit_results.xml

# Upload with CI metadata and tags
./testnod-uploader \
  -token=abc123 \
  -branch=main \
  -commit-sha=abc1234567890 \
  -run-url=https://ci.example.com/run/123 \
  -build-id=build-456 \
  -tag=integration \
  -tag=backend \
  junit_results.xml

# Don't fail the CI build if upload has issues
./testnod-uploader -token=abc123 -ignore-failures junit_results.xml
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `TESTNOD_BASE_URL` | Override the TestNod API base URL (defaults to `https://testnod.com`) |

## Supported JUnit XML Formats

The validator accepts XML files with either a `<testsuite>` or `<testsuites>` root element, covering output from most test frameworks including JUnit, Gradle, Maven Surefire, and pytest.

## Project Structure

```
cmd/testnod-uploader/   CLI entry point, flag parsing, orchestration
internal/testnod/       TestNod API client (creates test runs, gets presigned URLs)
internal/upload/        File upload to presigned S3 URLs
internal/validation/    JUnit XML validation
testdata/               Test fixture XML files
```

## Testing

```bash
# Run all tests
go test ./...

# Verbose output
go test -v ./...

# Run tests for a specific package
go test ./internal/validation

# Run a specific test
go test ./cmd/testnod-uploader -run TestParseFlags
```

## How It Works

1. Parse CLI flags and validate inputs
2. Validate the JUnit XML file (check for well-formed XML with a `<testsuite>` or `<testsuites>` element)
3. POST to the TestNod API to create a test run — the response includes a presigned S3 upload URL
4. PUT the XML file to the presigned URL

Both API and upload steps retry up to 3 times with a 1-second delay between attempts.
