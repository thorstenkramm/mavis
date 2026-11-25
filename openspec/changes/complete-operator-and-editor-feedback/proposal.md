# Proposal

## Why

Readers need a direct way to edit a page in its source repository. Operators need a supported way to share a Unix socket with a proxy and instructions for running Mavis as a service on each supported OS.

## What Changes

- Accept an optional HTTPS edit URL template. Put a Git icon and an “Edit page” link below rendered Markdown, directory READMEs and text/code viewers.
- Accept an optional Unix socket group. Grant that group access on request, retaining owner-only access by default.
- Publish four deployment guides: systemd and launchd as `mavis`, Windows with Servy, and Linux HTTPS with Caddy and a hardened systemd unit.
- Add the four badges used by Mailfire (CI, latest release, Go version, license) to the Mavis repository README, with Mavis-specific sources and destinations.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `server-configuration`: optional edit URL template and socket group, with strict validation.
- `documentation-browser`: accessible repository edit links for eligible text documents.
- `server-cli`: optional group ownership and permissions on Unix listeners.
- `binary-distribution`: discoverable deployment guides and repository status badges.

## Impact

Changes touch configuration, listener setup, page models and embedded templates, public Hugo documentation and README. Existing configurations keep their behavior. No Git credentials, working copy, service installer, or new runtime dependency is required. Search and downloads are already implemented and outside this change.
