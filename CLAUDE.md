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

- Both API calls and the S3 PUT retry 3× with a 1s delay.
- `-build-id` is required outside `-validate` mode. It groups parallel/matrix shards into one logical test run server-side.
- This binary owns per-upload state only. Run-level finalization (`/integrations/test_runs/finalize`) is the webapp's job and is called separately from CI — never add it here.

## Compatibility

CLI flag names are a public interface: the GitHub Action (`testnod/testnod-uploader`) and CircleCI orb documented in `docs/ci-integrations.md` pass them through. Renaming or removing a flag breaks those consumers — add new flags, keep old ones working.

## Releasing

Pushing a `v*` tag triggers `.github/workflows/release.yml`, which cross-compiles six static binaries (`CGO_ENABLED=0`) and uploads them to Cloudflare R2 under both `<version>/` and `latest/`. There is no GitHub Release. `dist/` is gitignored local build output.
