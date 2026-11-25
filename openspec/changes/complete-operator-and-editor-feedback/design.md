# Design

## Context

The renderer returns article HTML also used by exports; the viewer template owns controls. `serveMarkdown` already receives the actual README resource separately from the navigation path. `config.Load` strictly decodes and validates TOML before `config.Merge` applies CLI flags. Unix startup currently creates a listener, records its filesystem identity, applies `0600`, and closes/unlinks only its own socket. The Windows listener rejects socket mode. See `proposal.md` for motivation.

## Goals / Non-Goals

Use existing HTML, authorization, listener lifecycle and documentation packaging. Editing means following a link; Mavis does not clone repositories, infer remotes or branches, write files or handle Git authentication. Service deployment is operator-managed.

## Decisions

1. **One provider-neutral template.** Use top-level `edit_url_template`, e.g. `https://github.com/thorstenkramm/mavis/edit/main/docs/content/{path}`. An operator can express a branch and repository prefix directly. Validate once: exactly one literal placeholder occupying a whole path segment, fixed HTTPS origin with a hostname, no userinfo, control characters, backslashes or other unexpanded braces. Parse with a sentinel in place of the token. Preserve fixed query/fragment text; substitute the slash-separated logical path with each segment escaped exactly once. Reject explicit empty strings; omission is valid. This avoids a provider registry and URL inference. Never echo a credential-bearing template in diagnostics.

2. **Page control, separate from document contents.** Thread the validated setting through server/web construction. Use the already authorized resource's `RelativePath`; for README indexes use the README resource, for internal symlink aliases use the logical URL name. Populate an optional edit URL only on successful Markdown or text viewer responses. Use a normal same-tab anchor below the article body/viewer, labelled “Edit page”, with an inline decorative Git icon and visible focus. Rely on HTML template URL/attribute escaping, without marking user data trusted. A link performs no background requests; readers authenticate at the destination if necessary.

3. **Group name in file configuration, mode determined by policy.** Add `[server].socket_group` as a presence-aware string; accept a name rather than a general permission mask. Validate type and nonblank syntax when reading the file; check that socket mode is selected *after* CLI overrides so a group can accompany `--socket`. Preserve existing socket/TCP conflict checks. On Unix resolve the local group before touching a socket, retaining standalone `CGO_ENABLED=0` support (document local group database limitations). After binding, restrict mode to `0600`, change only the GID, then set `0660`. Complete setup before serving or announcing success; close with the existing identity-aware cleanup on failure. No group means no ownership change. Continue to reject Unix sockets on Windows before any group lookup. Chown alone would leave Caddy unable to connect on Linux; world-writable modes grant more access than requested.

4. **Four reproducible public guides.** Put examples in `docs/content/guides/` so Hugo and `scripts/package-release.go` ship the same instructions. Link them from the README and configuration/platform/security references. Use absolute executable and content paths, a service-readable but non-writable content tree, service-specific PATH for optional pandoc, and explicit start/stop, logging, verification and removal steps.
   - Linux: create a non-login `mavis` account, install a systemd unit for loopback TCP, use `KillSignal=SIGINT` for the existing graceful shutdown path, and verify service identity and HTTP response.
   - macOS: provision or verify an unprivileged `mavis` account without reusing an occupied UID/GID; supply a system LaunchDaemon plist with `UserName`, discrete `ProgramArguments`, logging paths and permissions. Use `plutil`, `launchctl bootstrap/print/bootout` and an HTTP probe.
   - Windows: use the [Servy documentation](https://servy-win.github.io/) and a scoped service identity. Specify the binary, arguments, quoted content path, startup directory, logs and loopback TCP. Validate registration and stop/removal on Windows; do not substitute Unix examples.
   - Linux proxy: dedicate a local group (e.g. `mavis-proxy`) to the socket. Run Mavis as `User=mavis`, `Group=mavis-proxy`; give Caddy supplementary membership and restart it. Use a service-owned `RuntimeDirectory=mavis` with `RuntimeDirectoryMode=0750`, `UMask=0077`, `/run/mavis/mavis.sock`, and the matching `socket_group`. Provide a complete hardened unit (`NoNewPrivileges`, `ProtectSystem=strict`, `ProtectHome`, `PrivateTmp`, appropriate runtime writes); keep content outside protected home directories. Caddy terminates TLS at `mavis.example.com` and proxies to `unix//run/mavis/mavis.sock`. Cover DNS, certificate issuance/ports, Caddy validation, local socket smoke as the proxy user and public HTTPS checks. TLS alone is not authentication.

5. **Badges mirror Mailfire's actual set.** Link the native `ci.yml` main-branch badge to Mavis Actions and Shields release, Go and license badges to Mavis releases/latest, go.mod and license. Use alt text and absolute GitHub targets that also make sense from a packaged README. Keep remote badge imagery confined to the repository README rather than application chrome.

## Risks / Trade-offs

- Misconfigured repository prefix → show a nested README example and test special characters; no server-side existence check or disclosure to the edit host.
- Group assignment can fail without suitable membership or directory access → fail closed and document service identities, non-group-writable parent directories, group traversal and diagnostics. Do not require a root-running Mavis service.
- Init systems, TLS and service wrappers vary by host → record OS/tool versions and actual native smoke results separately from syntax checks. Use disposable services for verification.

## Migration Plan

Both settings are opt-in. Install a version supporting the keys before adding them. To roll back, remove the edit template and group setting, switch the proxy to another supported connection or stop it, and restart Mavis to recreate an owner-only socket. Remove added service registrations and memberships according to the platform guide.
