# Proposal

## Why

The current tag workflow implements cross-compilation, archives, checksums, and changelog publication in bespoke scripts, making routine release maintenance harder than necessary. Mailfire's GoReleaser setup provides a proven model, but Mavis must retain its native per-platform verification before publishing.

## What Changes

- Replace the custom release packaging script and its use in tag-driven CI with a pinned GoReleaser configuration for the existing five build targets.
- Preserve archive names, wrapped directory layouts, embedded documentation, individual SHA-256 files, prerelease detection, and the native smoke-test gate.
- Build release artifacts without publishing, verify them on native macOS, Linux, and Windows runners, then publish the same verified artifacts with generated release notes.
- Document local GoReleaser validation and the tag-driven release process, and ignore generated `dist/` output.

## Capabilities

### New Capabilities

None. This changes release tooling rather than product behavior.

### Modified Capabilities

None. Existing `binary-distribution` requirements for supported binaries, documentation, checksums, and native pre-publication verification remain unchanged.

## Impact

Adds `.goreleaser.yaml` and a split-checksum compatibility helper; updates `.github/workflows/release.yml`, `.gitignore`, `run-all-tests.sh`, and the public installation instructions; removes the superseded `scripts/package-release.go`. Existing native smoke and packaged-document checks remain in use.
