# Design

## Context

The existing tag workflow packages one target per native runner, uploads the artifacts, and creates a GitHub release after every runner passes. `scripts/package-release.go` duplicates standard Go release plumbing, while `scripts/smoke-release.go` and `scripts/check-packaged-docs.py` enforce Mavis-specific behavior worth retaining. Mailfire's `.goreleaser.yaml` supplies a compatible GoReleaser v2 configuration style and pinned action version; Mavis has no build-time version variables or external service secrets.

## Goals / Non-Goals

**Goals:**

- Use standard GoReleaser builds, archives, checksums, and git-derived release notes.
- Keep all five existing targets, filenames, archive layouts, and checksum commands unchanged.
- Do not create or upload a GitHub release until native verification of every target succeeds.
- Permit a complete local snapshot build without GitHub credentials.

**Non-Goals:**

- No Linux distribution packages, Homebrew, Cloudsmith, signing, notarization, containers, or additional targets.
- No publication of a partial release and no migration of already published assets.
- No changes to Mavis CLI behavior or the native smoke-test contract.

## Decisions

1. **One GoReleaser build stage on Linux.** A single `.goreleaser.yaml` declares `./cmd/mavis`, `CGO_ENABLED=0`, `-trimpath`, and `-s -w`, with Windows ARM64 excluded. Archives retain `mavis-vVERSION-OS-ARCH` names, wrap their contents in that directory, use ZIP only on Windows, and include README, LICENSE, `docs/content`, and `testdata/docs`. Split SHA-256 files preserve the current installation commands; `scripts/format-release-checksums.py` verifies GoReleaser's digest-only output and adds the archive filename required by `shasum -c` and `sha256sum -c`. Git history produces `dist/CHANGELOG.md` for tagged releases in commit order with short hashes (snapshot mode skips notes); the first tag, which has no predecessor, gets a short first-release note instead of the full repository history. GoReleaser has no `release` section because it never publishes; Mailfire's nFPM and secret-specific settings do not apply.

2. **Package, verify, then publish.** The packaging job runs `goreleaser release --clean --skip=publish` against a full checkout of the version tag and uploads the archives, checksums, and notes as workflow artifacts. Native macOS Intel, macOS ARM64, Linux AMD64, Linux ARM64, and Windows AMD64 jobs download their corresponding archive, verify its SHA-256 file, extract it, validate the packaged links, and run the native smoke tool. The final job downloads those exact files and invokes `gh release create --verify-tag --notes-file dist/CHANGELOG.md`, adding `--prerelease` when the tag contains `-` before any `+` build metadata. Each native job verifies only its own archive's checksum.

3. **Use GoReleaser for preparation, GitHub CLI for the gated final upload.** GoReleaser v2.13.2 has no portable resume-from-workflow-artifacts command. Re-running `goreleaser release` after verification would rebuild rather than publish the already checked files, and publication before native tests would violate the existing requirement. The final `gh release create` publishes only the artifacts produced by the first GoReleaser run. Alternatives of dropping native checks, publishing directly from GoReleaser before testing, or upgrading to an unverified newer tool were rejected.

4. **Fail closed and keep write permission narrow.** Global workflow permission remains `contents: read`; only the final publication job receives `contents: write`. Packaging validates the tag and native verification must pass before that job can start. The `release-<ref>` concurrency group never cancels an active release. Workflow and action versions are pinned by SHA, and GoReleaser itself is pinned to v2.13.2, matching the locally tested Mailfire tool.

5. **Local dry runs never publish.** `goreleaser check` validates configuration, while `goreleaser release --snapshot --clean --skip=publish` generates local artifacts under ignored `dist/`. Running the checksum formatter after either command produces the same standard checksum files as CI. The installation guide replaces references to the deleted custom packaging script with these commands and explains the pre-publication native checks.

## Risks / Trade-offs

- [An archive name or path differs from the existing tooling] → Inspect every snapshot archive, checksum, and extracted directory against the documented names; run `scripts/check-packaged-docs.py`.
- [A cross-built binary differs in native behavior] → Retain execution on all five native runner types before publication.
- [Two tools participate in publication] → Limit GitHub CLI to the final upload of GoReleaser-produced files; do not rebuild or regenerate notes after native verification.
- [An interrupted workflow leaves a tag without a release] → Re-run the same tag workflow after resolving the failed check; the publish job never runs after a verification failure.

## Migration Plan

1. Add and locally validate the GoReleaser configuration and snapshot artifacts.
2. Replace the workflow and remove the custom packaging script after the snapshot checks pass.
3. Merge through normal CI, then validate an actual tag with native runners before approving any subsequent release automation changes.
4. Roll back by restoring the previous workflow and packaging script; no runtime data or published archive format migration is needed.
