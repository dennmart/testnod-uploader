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
| `-build-id` | Yes (unless `-validate`) | Build identifier for the CI/CD run. Shards of one build (parallel runners, matrix jobs) that share a build ID are grouped into one logical test run. |
| `-tag` | No | Tag for the test run (repeatable) |
| `-ignore-failures` | No | Always exit 0, even if upload fails |

### Examples

```bash
# Validate only
./testnod-uploader -validate junit_results.xml

# Basic upload
./testnod-uploader -token=abc123 -build-id=build-456 junit_results.xml

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
./testnod-uploader -token=abc123 -build-id=build-456 -ignore-failures junit_results.xml
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

1. Parse CLI flags and validate inputs (`-build-id` is required for uploads — shards with the same build ID are aggregated into one test run server-side)
2. Validate the JUnit XML file (check for well-formed XML with a `<testsuite>` or `<testsuites>` element)
3. POST to the TestNod API to create a test run — the response includes the presigned S3 upload URL and the identifiers (`project_id`, `test_run_id`, `upload_id`) needed for the failure callback
4. PUT the XML file to the presigned URL with `Content-Type: application/xml` — the object metadata is encoded in the URL's query string by the presigner, so no extra headers are needed
5. If the PUT fails, notify TestNod via the per-upload failure callback (`/integrations/test_runs/upload_failed`) so the upload row is marked failed without poisoning the whole run

Both API and upload steps retry up to 3 times with a 1-second delay between attempts.

## CI/CD

### Tests

Tests run automatically on every push to `main` via GitHub Actions (`.github/workflows/test.yml`).

### Releasing

The release workflow (`.github/workflows/release.yml`) triggers when a version tag is pushed:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This builds binaries for six platforms:

| OS | Architectures |
|----|---------------|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64, arm64 |

The workflow generates SHA-256 checksums and a VERSION file, then uploads everything to a Cloudflare R2 bucket. Binaries are stored both under the version path (`testnod-uploader/<version>/`) and under `testnod-uploader/latest/` so the most recent release is always accessible at a stable URL.

**Required repository secrets for releases:**

- `CLOUDFLARE_API_TOKEN`
- `CLOUDFLARE_ACCOUNT_ID`
- `R2_BUCKET`
