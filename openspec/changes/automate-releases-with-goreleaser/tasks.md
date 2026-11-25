# Tasks

## 1. GoReleaser packaging

- [x] 1.1 Add `.goreleaser.yaml` for the existing five targets, archive names and contents, split SHA-256 files, and git release notes; verify it with `goreleaser check`.
- [x] 1.2 Run `goreleaser release --snapshot --clean --skip=publish` and inspect all archives to confirm the documented filenames, wrapped directories, executable, documentation, LICENSE, and checksum files; confirm `dist/CHANGELOG.md` with a non-publishing tagged dry run.
- [x] 1.3 Ignore `/dist/`, extract the native snapshot archive, run `scripts/check-packaged-docs.py` and `scripts/smoke-release.go` on it, and remove `scripts/package-release.go` after these checks pass.

## 2. Tag-driven workflow

- [x] 2.1 Replace custom per-platform packaging in `.github/workflows/release.yml` with the pinned GoReleaser action, artifact staging, five native verification jobs, and the final notes-based GitHub publication job; verify dependency ordering, artifact names, tag validation, prerelease handling, and job-level permissions.
- [x] 2.2 Parse the workflow and inspect every referenced action and local helper to confirm all inputs and files exist, native smoke remains mandatory before the sole `contents: write` job, and no packaging or verification step publishes.

## 3. Documentation and integration

- [x] 3.1 Update `docs/content/getting-started/installation.md` with local GoReleaser snapshot commands and the native pre-publication release process, then verify `make docs` succeeds.
- [x] 3.2 Run `./run-all-tests.sh`, the native snapshot smoke checks, `openspec validate automate-releases-with-goreleaser --strict`, and `git diff --check`; confirm generated release output is ignored and no GitHub release was created.

## Workflow follow-up

- After merging, create a release tag and supervise all native verification and publication jobs; verify the release assets and notes on GitHub.
- Archive the change after release verification and confirm the archived result.
