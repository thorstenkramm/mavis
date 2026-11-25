# Tasks

## 1. Repository edit links

- [x] 1.1 Add `edit_url_template` parsing, validation and per-segment path expansion. Document GitHub and another HTTPS template in the configuration reference. Test omission, invalid types, blank values, missing/duplicate/misplaced placeholders, malformed schemes/authorities, credentials and special filenames; ensure errors do not echo credentials.
- [x] 1.2 Pass the validated template to the web handler and render the Git-icon “Edit page” anchor below successful Markdown, README and text/code views. Verify direct, nested, aliased and README paths, including spaces, `#`, `%`, Unicode and HTML-special characters; verify absence on all inapplicable responses and exports. Check keyboard/no-JavaScript operation, visible focus and mobile layout; add a usage example to the serving guide.

## 2. Socket group ownership

- [x] 2.1 Parse `[server].socket_group` and validate effective listener selection after CLI merge. Add config/CLI tests for socket from file or flag, absent/blank/wrong-type group, TCP-only and conflicting listener settings. Update configuration reference with default and opt-in examples.
- [x] 2.2 Resolve the named group on Unix and apply the GID and `0660` before serving, preserving default `0600`, user owner, stale/active socket handling, identity-safe cleanup and Windows rejection. Test real socket GID/mode with a local group, unknown group and injected ownership/mode failures, path replacement and immediate restart. Update platform/security guidance and verify builds with `CGO_ENABLED=0`.

## 3. Deployment guides

- [x] 3.1 Write the Linux systemd guide with `mavis` user creation, content and executable permissions, complete unit, lifecycle, journal and removal commands. Verify the unit with `systemd-analyze verify` and, on a disposable Linux system, confirm the running UID and loopback HTTP response.
- [ ] 3.2 Write the macOS launchd guide with a dedicated `mavis` account, LaunchDaemon plist, paths, ownership, logs and lifecycle. Validate with `plutil -lint`; verify identity, page retrieval and bootout on a disposable macOS installation.
- [ ] 3.3 Write the Windows Servy guide with upstream documentation, `mavis.exe` path, quoted arguments, startup directory, least-privilege service identity, logging, status and removal. Check against the named Servy version and verify service start, loopback HTTP, stop and uninstall on Windows.
- [x] 3.4 Write the Linux-only Caddy guide with a complete hardened `mavis.service`, matching `.mavis.conf`, group memberships, runtime directory, Caddyfile for `https://mavis.example.com`, DNS/TLS prerequisites and permission diagnostics. Verify with `systemd-analyze verify`, `caddy validate`, a socket HTTP request as the proxy user, denial for an unrelated user, no Mavis TCP listener and an HTTPS response through Caddy in a test environment.
- [x] 3.5 Link all four guides from public docs and README, and cover them in the extracted-release document check. Run `make docs`, package and extract a native archive, then run `scripts/check-packaged-docs.py` on its root; record which service examples were run on which platforms.

## 4. README and integrated verification

- [x] 4.1 Add labelled CI, latest release, Go version and license badges below the README title; verify GitHub renders the images and each link resolves to the correct Mavis resource.
- [x] 4.2 Run `./run-all-tests.sh`, fresh native release smoke, and strict OpenSpec validation. Verify race/build checks, coverage of at least 85%, and continued search/download operation alongside the opt-in controls.
