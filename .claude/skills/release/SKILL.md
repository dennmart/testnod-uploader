---
name: release
description: Cut a new testnod-uploader version — pick the next semver tag, run pre-release gates, create a signed tag, push it only when every gate passes and the user confirms, then watch the release workflow and publish the GitHub Release. Use when the user asks to release, bump the version, tag, or ship a new build.
argument-hint: "[patch|minor|major|vX.Y.Z]"
disable-model-invocation: true
---

# Release testnod-uploader

The version lives in the git tag — there is no version constant or changelog to update. Each version also gets a GitHub Release carrying its notes (step 7); the binaries live on R2, not on the release. Pushing a `v*` tag runs `.github/workflows/release.yml`, which builds six binaries and uploads them to R2 under both `<version>/` **and `latest/`**. Every CI pipeline pulling `latest/` gets the new binary on its next run, and there's no rollback short of cutting another release. Treat the push as a production deploy.

Requested bump: `$ARGUMENTS` (if empty, recommend one in step 3).

## 1. Gather state

Run these and keep the output. Don't make any changes yet.

```bash
git fetch origin --tags
git status --porcelain
git rev-parse --abbrev-ref HEAD
git rev-list --left-right --count origin/main...HEAD      # behind / ahead
LAST=$(git describe --tags --abbrev=0 --match 'v[0-9]*' --exclude '*-*')
git log --oneline "$LAST"..HEAD
git diff --stat "$LAST"..HEAD
```

Ignore `v0.0.0-test` and any other pre-release tags when working out the last version.

## 2. Gates: when NOT to release

Go through every gate. A **STOP** means: say why, suggest the fix, and end without creating a tag. A **CONFIRM** means: explain the risk, then ask the user with AskUserQuestion before going on. Don't try to fix a STOP yourself unless the user asks.

**Repository state: STOP if any of these fail**
- The current branch isn't `main`.
- The working tree has uncommitted or untracked changes (a tag records a commit, so local edits won't ship, and that's misleading).
- `HEAD` is ahead of `origin/main`. Unpushed commits would go out in a tag no one reviewed on main. Ask the user to push main first and wait for CI.
- `HEAD` is behind `origin/main`. You'd be tagging a stale commit.
- The target tag already exists, locally or on `origin` (`git ls-remote --tags origin vX.Y.Z`). **Never** delete, move, or force-push an existing version tag. R2 already has those binaries and consumers may have pinned them. Choose the next version.

**Nothing to ship: STOP**
- There are no commits since `$LAST`.
- Nothing that affects the binary changed since `$LAST`:
  ```bash
  git diff --quiet "$LAST"..HEAD -- cmd internal go.mod go.sum
  ```
  If that exits 0, only docs, CI config, CLAUDE.md, or `.claude/` changed. A new tag would republish the same binary under a new number, so say that and stop. (A Go toolchain bump in `go.mod` *does* count: it rebuilds with a new compiler and stdlib.)

**Verification: STOP if any of these fail.** This mirrors `test.yml`:
```bash
go vet ./... && go vet -tags debug ./...
go test ./... && go test -tags debug ./...
for t in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  CGO_ENABLED=0 GOOS=${t%/*} GOARCH=${t#*/} go build -trimpath -o /dev/null ./cmd/testnod-uploader || echo "FAIL $t"
done
```
- Run CI on the exact commit when `gh` is available: `gh run list --workflow test.yml --commit "$(git rev-parse HEAD)" --limit 1`. STOP if it failed or is still running. If there's no run for this commit, or `gh` isn't available, treat it as a CONFIRM.

**Risky content: CONFIRM each one that applies**
- **CLI flag compatibility.** Diff the flag definitions: `git diff "$LAST"..HEAD -- cmd/testnod-uploader | grep -E '^[-+].*flag\.'`. The GitHub Action and CircleCI orb (`docs/ci-integrations.md`) pass flags by name. A removed or renamed flag, a changed default, or a newly required flag breaks consumers pulling `latest/`. List what changed and ask whether the Action/orb is ready. Recommend not releasing until it is.
- **Upload contract changes.** If the diff touches `internal/upload` or the request/response shapes, check it against the contract in CLAUDE.md: no extra headers on the S3 PUT, the same endpoints, and no `/finalize` call. If the change relies on a TestNod webapp change, ask whether that change is **deployed to production**. A binary that ships before its server side is ready will fail every upload.
- **Release pipeline changed.** If `git diff --quiet "$LAST"..HEAD -- .github/workflows/release.yml` shows changes, this is the first run of the edited pipeline, and it writes to `latest/`. Summarize what changed and confirm.
- **Toolchain bump.** If the `go` directive in `go.mod` changed, mention it. The release job uses `go-version-file`, so it needs `setup-go` to support that version.

## 3. Choose the version

Tags are `vMAJOR.MINOR.PATCH` with no pre-release suffix. The project is pre-1.0 and so far every release has been a patch bump. Recommend a bump from the commits since `$LAST`:
- **patch**: bug fixes, internal changes, dependency or toolchain bumps.
- **minor**: new flags or new user-visible behavior.
- **major**: only if the user asks for it explicitly. Never go to 1.0.0 on your own.

If `$ARGUMENTS` is an explicit `vX.Y.Z`, check that it's valid semver and strictly greater than `$LAST`. If it isn't, STOP.

## 4. Summarize and confirm

Show the user one summary:
- `$LAST` → new version, and the commit SHA to be tagged
- The commits included (one line each)
- The result of every gate, with any CONFIRM items called out
- Draft release notes. They become the GitHub Release body in step 7. Keep them short and user-facing: lead with anything a CI user has to change in how they invoke the binary, say compatibility breaks plainly, leave out server-side and implementation details, and describe doc changes only as "documentation updates".

Then ask with AskUserQuestion: **Create and push tag** / **Create tag locally only** / **Cancel**. Prior approval in the conversation doesn't count. Ask every time.

## 5. Tag and push

Existing tags are signed and annotated, with the version as the message:
```bash
git tag -s vX.Y.Z -m vX.Y.Z
git tag -v vX.Y.Z
```
If signing fails (for example, no pinentry in a non-interactive shell), don't fall back to an unsigned tag. Give the user the command to run themselves: `! git tag -s vX.Y.Z -m vX.Y.Z`.

Only if the user chose to push:
```bash
git push origin vX.Y.Z
```
Push **only that tag**. Never use `--tags` (it would push stray local tags) and never `--force`.

## 6. After pushing

This step and step 7 need `gh`, installed and authenticated (`gh auth status`). Without it, give the user the Actions URL (`https://github.com/<owner>/<repo>/actions/workflows/release.yml`), the release notes, and the step 7 command to run themselves, then stop.

- Find the run for the tag and link it: `gh run list --workflow release.yml --branch vX.Y.Z --limit 1`. It can take a few seconds to show up after the push.
- Watch it to completion in the background: `gh run watch <run-id> --exit-status`, then `gh run view <run-id>`. Tell the user the result either way.
- Tell the user the release writes to `latest/`. If the run fails partway, `latest/` may contain a mix of old and new binaries. Recover by fixing forward with a new patch release, never by re-pushing the same tag.
- Don't download or smoke-test the published binaries yourself. Hand verification back to the user.

## 7. GitHub Release

Only once the release workflow has **passed**. If it failed, don't create a release for that tag; fix forward instead. Skip this step if the tag was only created locally.

1. Read the current latest release as the template: `gh release view --json tagName,name,body`. Match its format: the title is the bare version (`vX.Y.Z`), the body is a short bullet list, and no assets are attached.
2. Draft the notes in that format from the step 4 draft, following the same rules (user-facing only, compatibility breaks stated plainly, doc changes only as "documentation updates"). Write them to a file in the scratchpad, not the repo.
3. Show the notes and ask with AskUserQuestion: **Publish release** / **Edit notes** / **Skip**. A GitHub Release is public, so ask even though the tag push was already approved.
4. Publish it and mark it latest:
   ```bash
   gh release create vX.Y.Z --verify-tag --title vX.Y.Z --notes-file <notes.md> --latest
   gh release list --limit 3
   ```
   Confirm `vX.Y.Z` shows as `Latest`, and link the release URL in the final report.
