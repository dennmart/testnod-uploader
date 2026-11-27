# CLAUDE.md

CLI that uploads JUnit XML test results to TestNod (testnod.com). Go, stdlib + `github.com/avast/retry-go/v5` only. The README covers flags, usage, and CI setup — don't duplicate it here.

## Commands

```bash
go build -o testnod-uploader ./cmd/testnod-uploader
go build -tags debug -o testnod-uploader ./cmd/testnod-uploader   # enables [DEBUG] logging to stderr
go test ./...
go test ./cmd/testnod-uploader -run TestParseFlags
```

- `internal/debug` is build-tag gated: `debug.Log` is a no-op unless built with `-tags debug`. Its tests must be run both with and without the tag.
- Tests use `httptest` servers; no network or credentials needed. Run `go test ./...` after every change.
- `TESTNOD_BASE_URL` overrides the API host (default `https://testnod.com`) for manual testing against a local server.

## Upload contract (non-obvious, easy to break)

1. `POST /integrations/test_runs/upload` with `Project-Token` header → response includes `project_id`, `test_run_id`, `upload_id`, and a presigned S3 URL.
2. `PUT` the file to the presigned URL with `Content-Type: application/xml` and **no other headers**. The object metadata is already hoisted into the URL query string by the presigner; adding headers (e.g. `x-amz-meta-*`) breaks the signature.
3. On upload failure, `POST /integrations/test_runs/upload_failed` with `{test_run_id, upload_id, failure_message}` and the same `Project-Token`.

- Error responses carry `{"error_message": "..."}`; the client prints it. `upload` returns `403` when the organization or project is paused by plan limits. Never retry it or match on its text, and never PUT or call `upload_failed` after it (there are no IDs).
- `project_id` in the `upload` response is a UUID string; `id`, `test_run_id`, and `upload_id` are integers.
- The request body is JSON. The server also accepts multipart, which is what the public docs show for curl.
- Both API calls and the S3 PUT make 3 attempts with a 1s delay (5s after a `429`). API 4xx responses are not retried except `409` (concurrent shards racing to create the run) and `429`.
- `-build-id` is required outside `-validate` mode. It groups parallel/matrix shards into one logical test run server-side.
- This binary owns per-upload state only. Run-level finalization (`/integrations/test_runs/finalize`) is the webapp's job and is called separately from CI — never add it here.

## Compatibility

CLI flag names are a public interface: the CI integrations documented in `docs/ci-integrations.md` (GitHub Action `testnod/testnod-uploader`, CircleCI orb, Jenkins step, GitLab CI/CD component `gitlab.com/testnod/testnod-uploader`) pass them through. Renaming or removing a flag breaks those consumers — add new flags, keep old ones working.

## Releasing

Pushing a `v*` tag triggers `.github/workflows/release.yml`, which cross-compiles six static binaries (`CGO_ENABLED=0`) and uploads them to Cloudflare R2 under both `<version>/` and `latest/`, then creates a draft GitHub Release with the binaries and `checksums.txt` attached. Once the workflow passes, the `/release` skill adds the notes to that draft, publishes it, and marks it latest. `dist/` is gitignored local build output.
