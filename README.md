# testnod-uploader

A CLI tool for uploading JUnit XML test results to [TestNod](https://testnod.com). It validates JUnit XML files and uploads them via a two-step process: creating a test run through the TestNod API, then uploading the XML file directly to S3 using a presigned URL.

## Installation

Build from source (requires Go 1.27.3+):

```bash
go build -o testnod-uploader ./cmd/testnod-uploader
```

Pre-built binaries for Linux, macOS, and Windows (amd64 and arm64) are attached to each [GitHub Release](https://github.com/dennmart/testnod-uploader/releases).

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
| `-build-id` | Yes (unless `-validate`) | Build identifier for the CI/CD run. Shards of one build (parallel runners, matrix jobs) that share a build ID are grouped into one logical test run. Each build ID counts as one test run toward your plan's monthly limit, so give shards their own build IDs only if you want them as separate runs. See [Plans and billing](https://testnod.com/docs/administration/plans-and-billing). |
| `-tag` | No | Tag for the test run (repeatable) |
| `-retry` | No | Mark this file as a rerun of tests that failed earlier in the same build. Tests that pass in it replace their earlier failures and are recorded as passed on retry. Upload it with the same `-build-id` as the original results. |
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

# Upload a rerun of failed tests (e.g. RSpec --only-failures) to the same build
./testnod-uploader -token=abc123 -build-id=build-456 results.xml
./testnod-uploader -token=abc123 -build-id=build-456 -retry results-retry.xml

# Don't fail the CI build if upload has issues
./testnod-uploader -token=abc123 -build-id=build-456 -ignore-failures junit_results.xml
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `TESTNOD_BASE_URL` | Override the TestNod API base URL (defaults to `https://testnod.com`) |

## Errors and Exit Codes

The uploader exits with `1` when validation or the upload fails, and `0` on success. With `-ignore-failures` it always exits `0`, so a TestNod problem never fails your CI job.

When the TestNod API rejects a request, the uploader prints the reason TestNod gives, for example:

```
Error creating test run on TestNod: 403 Forbidden: This project is paused because the Solo plan includes 1 project and another project in your organization is active. ...
```

| Status | Meaning | Retried? |
|--------|---------|----------|
| `403` | Uploads are paused for this organization (well over the plan's monthly limit) or this project (inactive project on the Solo plan). Nothing is recorded. Uploads resume when the monthly count resets or an admin upgrades or makes the project active. In sharded builds, every shard gets the `403`. See [troubleshooting](https://testnod.com/docs/sending-test-results/troubleshooting#i-get-a-403-from-the-upload-endpoint). | No |
| `404` | The project token is wrong or doesn't match a project. | No |
| `409` | Two shards of the same build raced to create the test run. | Yes |
| `422` | The request was invalid, for example a missing build ID. | No |
| `429` | Rate limited (120 requests a minute per project token). | Yes, after 5 seconds |
| `5xx` | TestNod server error. | Yes |

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
3. POST to the TestNod API to create a test run (including the `retry` flag, so TestNod knows whether this file is a rerun of earlier failures) — the response includes the presigned S3 upload URL and the identifiers (`project_id`, `test_run_id`, `upload_id`) needed for the failure callback
4. PUT the XML file to the presigned URL with `Content-Type: application/xml` — the object metadata is encoded in the URL's query string by the presigner, so no extra headers are needed
5. If the PUT fails, notify TestNod via the per-upload failure callback (`/integrations/test_runs/upload_failed`) so the upload row is marked failed without poisoning the whole run

Both API and upload steps make up to 3 attempts with a 1-second delay between them (5 seconds after a `429` rate limit response). API responses in the 4xx range other than `409` and `429` are not retried, since they won't succeed until something changes, and the uploader stops without uploading the file. See [Errors and Exit Codes](#errors-and-exit-codes).

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
| macOS (13 Ventura or later) | amd64, arm64 |
| Windows | amd64, arm64 |

The workflow generates SHA-256 checksums and a VERSION file, then uploads everything to a Cloudflare R2 bucket. Binaries are stored both under the version path (`testnod-uploader/<version>/`) and under `testnod-uploader/latest/` so the most recent release is always accessible at a stable URL. Finally, it creates a draft GitHub Release for the tag with the binaries and `checksums.txt` attached, which is published once release notes are added.

**Required repository secrets for releases:**

- `CLOUDFLARE_API_TOKEN`
- `CLOUDFLARE_ACCOUNT_ID`
- `R2_BUCKET`
