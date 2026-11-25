# Design

## Context

See `proposal.md` — Why. The repository contains documentation and OpenSpec artifacts but no source module or established architecture. The module retains the Go 1.27.2 language directive and selects Go 1.27.4 through its toolchain directive for current standard-library security fixes. The implementation must create the complete serving path while keeping the distributed binary independent of a Go installation.

The capability deltas define the external contract. In particular, all content access paths must share the same visibility and boundary checks, listener conflict detection must distinguish defaults from explicitly supplied values, and rendering must preserve safe formatting without allowing active-content injection.

## Goals / Non-Goals

**Goals:**

- Use a small, testable Go architecture with clear seams between CLI, configuration, content authorization, rendering, HTTP, and browser presentation.
- Centralize filesystem resolution and policy enforcement so listings, Markdown, and assets cannot bypass one another.
- Render on demand from current source files and embed only application templates/static assets in the binary.
- Prefer server-rendered, semantic HTML and minimal browser JavaScript.
- Make unit and integration tests cover the security and configuration contracts before release packaging.

**Non-Goals:**

- No frontend framework, SPA routing, database, background watcher, remote rendering service, or telemetry.
- No rendered-document cache, plugin interface, or generalized application server abstraction.
- No simultaneous TCP and Unix-socket listeners.
- No guarantee of full GitLab rendering parity or compatibility with the former `serv` command.

## Decisions

### 1. Go module and project boundaries

Use a Go 1.27.2 module with the following package boundaries:

- `cmd/mavis`: process entry point and exit handling.
- `internal/cli`: Cobra command construction, explicit flag tracking, startup output, and signals.
- `internal/config`: TOML decoding, source presence, merging, validation, and wildcard compilation.
- `internal/content`: path resolution, authorization, directory models, file opening, and MIME detection.
- `internal/render`: Markdown AST processing, sanitization, heading models, and URL rewriting.
- `internal/server`: listeners, HTTP routing, handlers, response headers, and error mapping.
- `internal/web`: embedded templates, CSS, and minimal JavaScript.
- `testdata`: representative documentation trees and adversarial fixtures.

Alternatives considered: a single package would reduce initial file count but make policy bypasses and coupled tests more likely; public packages would imply an unsupported Go API, so implementation packages remain internal.

### 2. CLI and configuration layering

Use Cobra's `ExactArgs(1)` for `serve <directory>` and local `--host`, `--port`, `--socket`, `--exclude`, and `--include-hidden` flags. Do not register `serv` or an alias.

Decode `.mavis.conf` with pelletier/go-toml/v2 into typed structs with pointer-based optional fields and `Decoder.DisallowUnknownFields`. Track file presence through nil/non-nil fields and CLI presence through pflag's `Changed` state:

1. Start with defaults (`127.0.0.1:3344`, no exclusions, hidden files disabled).
2. Overlay explicitly defined file values.
3. Overlay explicitly changed CLI scalar values: server flags replace corresponding file values, and an explicitly supplied `--include-hidden` replaces the file/default boolean for the current run, including `--include-hidden=false`.
4. Append file and CLI exclusions into one immutable rule set.
5. Validate the result before opening a listener.

Defaults do not count as explicit TCP settings when validating a socket. Any explicitly supplied host or port combined with a socket is an error, including conflicts created by CLI overrides. Reject malformed patterns, unknown value types, invalid ports, empty socket paths, and unexpected TOML keys with setting-specific errors.

Alternatives considered: Viper would combine flags and files conveniently but adds broader configuration behavior and makes presence/precedence less explicit. BurntSushi/toml would provide metadata-based presence tracking, but pelletier/go-toml/v2 matches the established agent rule and supports strict unknown-field rejection directly.

### 3. Wildcard and hidden-file semantics

Use doublestar's slash-based `PathMatch` behavior for root-relative patterns. Normalize request and directory paths to `/` separators before matching. Support `*`, `?`, character classes, alternatives, and recursive `**`; reject malformed or root-escaping patterns during configuration validation.

Apply patterns to the complete root-relative path and to each parent directory so an excluded directory prunes listings and blocks descendants. Apply the same rules to both a symlink's logical path and its resolved root-relative target to prevent alias-based bypasses.

Define hidden content consistently as any path component whose name begins with `.`, regardless of platform. Do not rely on Windows file attributes for the baseline contract. Always deny the root `.mavis.conf` independently of this setting.

Alternatives considered: `path.Match` lacks recursive `**`; platform-specific hidden attributes would make identical documentation trees behave differently across operating systems.

### 4. Central content resolver and safe file serving

Every directory listing and file request goes through one content resolver. It must:

1. Decode URL path segments and reject NULs, invalid encodings, traversal segments, backslash-based traversal, absolute paths, and Windows volume paths.
2. Normalize to a root-relative logical path without allowing cleaning to hide an earlier traversal attempt.
3. Deny `.mavis.conf`, hidden paths, and exclusions before returning metadata or content.
4. Resolve the served root and candidate with `filepath.EvalSymlinks`, then verify containment using `filepath.Rel` rather than string prefixes.
5. Re-evaluate visibility and exclusions for the resolved target and reject external links, non-regular files, and unsupported special files.
6. Only then return directory entries or an open file to the HTTP layer.

Do not pass request paths directly to `http.ServeFile` or `http.FileServer`. Serve accepted files through `http.ServeContent` after resolution. Determine MIME type from extension with a content-sniffing fallback and `application/octet-stream` as the final fallback. Return `X-Content-Type-Options: nosniff`.

Map missing, hidden, excluded, and outside-root content to a non-revealing not-found response; map unexpected read failures to a generic server error with detailed diagnostics only in local logs. Directory pages use an eligible `README.md` when present, otherwise a listing.

Alternatives considered: `http.FileServer` alone is insufficient for unified hidden/exclusion/symlink policy; checking only lexical paths would allow symlink escape, while checking only resolved targets would allow excluded aliases to bypass policy.

### 5. Markdown pipeline and route model

Use goldmark with its GFM extension for tables, task lists, strikethrough, and linkification, plus automatic heading IDs. Keep goldmark's safe default for raw HTML; do not enable unsafe HTML rendering.

Process the parsed AST to:

- collect current-document heading text, levels, and generated IDs;
- rewrite relative Markdown links to Mavis routes while preserving query/fragment components;
- resolve relative assets against the source document and root-relative paths against the served root;
- leave external absolute URLs unchanged.

Keep content URLs aligned with their root-relative source paths, including the `.md`/`.markdown` suffix. This avoids a second routing map and makes relative-link rewriting deterministic. Directory URLs end in `/`; application assets live under the reserved `/_mavis/assets/` prefix, which cannot access user content.

Render goldmark output through a customized bluemonday UGC allowlist. Preserve heading IDs, required table/list markup, safe images/links, code blocks, and disabled checkbox task items while removing scripts, event handlers, unsafe URL schemes, and active embeds. Add a restrictive Content Security Policy to Mavis pages as defense in depth.

Alternatives considered: rendering raw HTML directly was rejected for injection risk; rewriting links with regular expressions over HTML was rejected in favor of AST-aware processing; full GitLab extension compatibility was rejected as outside baseline scope.

### 6. HTTP lifecycle and freshness

Use Go's `net/http` server and ServeMux with explicit handlers for embedded application assets and the content catch-all. Create the listener before reporting startup success:

- TCP defaults to `127.0.0.1:3344`; print `http://localhost:3344` for that default and the actual address otherwise.
- Unix listeners use platform-specific files, reject unsupported platforms, remove only a verified stale socket, set local-only permissions where supported, and clean up only the socket owned by this process.
- Resolve hostnames when determining loopback status and warn for wildcard/non-loopback listeners.

Use `signal.NotifyContext` for interruption and `http.Server.Shutdown` with a bounded timeout. Serve dynamic HTML with `Cache-Control: no-store`; do not cache parsed documents or directory models. User files are opened per request, while embedded application assets may use normal static caching. No file watcher or reload channel is introduced.

Alternatives considered: a third-party HTTP router is unnecessary for two route classes; automatic live reload would add watcher and websocket complexity contrary to the confirmed manual-refresh behavior.

### 7. Server-rendered accessible browser UI

Use `html/template` and embedded CSS/JavaScript. Render semantic navigation, folder links, headings, document content, and table of contents on the server. Use ordinary document URLs so browser history and heading fragments work without a client router.

The desktop layout uses CSS grid for the left file navigation and right content/TOC. At narrow widths, a keyboard-accessible disclosure controls a modal or dismissible navigation drawer. Folder navigation loads only the needed directory and current ancestry rather than recursively scanning the entire tree for every request. Current-path entries use `aria-current`; error and empty pages retain navigation where possible.

Use visible focus styles, native controls, sufficient contrast, and no pointer-only interactions. Browser verification covers current stable Chrome/Edge, Firefox, and Safari, plus desktop and mobile-width viewports. Accessibility checks combine automated markup/contrast checks with keyboard smoke tests.

Alternatives considered: an SPA would add a build/runtime toolchain and complicate history/accessibility; eagerly constructing a complete recursive tree would perform poorly on large directories.

### 8. Cross-platform build and release

Use `make build`, `make test` (`go test -race ./...`), `make lint` (gofmt, go vet, golangci-lint, staticcheck), and `make coverage` (minimum 85%, excluding test-only support packages) as stable local entry points. GitHub Actions enforces the same checks, then produces:

- macOS `amd64` and `arm64`;
- Linux `amd64` and `arm64`;
- Windows `amd64`.

Unix-socket support targets macOS and Linux. Windows reports an actionable unsupported-listener error. CI runs native smoke checks where runners permit and verifies cross-compiled artifacts for the remaining targets. Release artifacts include the README and configuration/compatibility documentation.

Tests include configuration precedence, listener conflicts, wildcard semantics, heading/link rendering, sanitization, MIME handling, refresh behavior, HTTP integration, and table-driven traversal/hidden/exclusion/symlink cases. Platform-specific symlink and socket tests use build tags or runtime skips with explicit coverage reports.

Alternatives considered: GoReleaser would provide more release automation but is unnecessary for the small initial matrix; package managers and containers remain deferred.

### 9. Documentation and comprehensive quality suite

Use the Hugo project under `docs/` with public content in `docs/content/` and unpublished maintainer documents in `docs/internal/`. The Mavis theme is a self-contained adaptation of Mailfire's theme, including vendored UIkit, responsive layouts, search, dark mode, copy buttons, and strict link validation. Do not commit generated Hugo output or Mailfire artwork/content.

`make docs` performs the production build and `make docs-serve` starts local preview. A pinned-Hugo GitHub Pages workflow builds documentation changes and deploys only from main.

Use `run-all-tests.sh` for comprehensive local verification: Go formatting/tests/vet, pinned golangci-lint and staticcheck, race tests, 85% coverage, jscpd marker validation and scanning, optional govulncheck, the Hugo build when Hugo is installed, and Trivy with failure on detected vulnerabilities. CI enforces the required Go, duplication, coverage, and Trivy checks with pinned actions and tool versions; the separate pinned-Hugo documentation workflow verifies documentation changes. `govulncheck` remains an optional local supplement rather than a second mandatory CI vulnerability gate.

Alternatives considered: an external Hugo theme or CDN assets would add supply-chain and version drift; relying only on Makefile checks would omit the duplication, documentation, and filesystem vulnerability checks required for this project.

## Risks / Trade-offs

- [Filesystem entries can change between symlink validation and file open] → Re-resolve on every request, avoid elevated privileges, centralize checks, and document the residual local-adversary TOCTOU limitation rather than claiming protection against concurrent hostile filesystem mutation.
- [Goldmark GFM differs from GitLab-specific rendering] → Publish a compatibility matrix and add representative fixtures for every claimed core feature; do not claim specialized extension support.
- [Sanitization can remove useful task-list or heading markup] → Use focused rendering tests to tune a narrow allowlist instead of enabling broad HTML or unsafe renderer options.
- [Cross-platform path and symlink behavior varies] → Normalize URL and glob paths separately from filesystem paths and run platform-specific security tests on all target operating systems.
- [Directory authorization can be repeated across many entries] → Check parent rules while walking listings, avoid whole-tree scans, and benchmark representative directory sizes before tuning without weakening policy.
- [Unix socket cleanup can remove another process's file] → Remove only verified stale sockets and paths created by the current process; never delete an arbitrary existing path.
- [No authentication on non-loopback listeners can expose documents] → Require explicit host configuration, print a prominent warning, and keep loopback as the default; the user confirmed this trade-off.
- [On-demand rendering costs more than caching] → Keep the baseline simple and correct, measure typical workloads, and defer any cache until freshness and invalidation requirements are explicitly designed.

## Migration Plan

1. Scaffold the Go module and configuration/CLI contract, including rejection of `serv`.
2. Add the central content resolver and security tests before exposing HTTP file serving.
3. Add Markdown rendering and the server-rendered browser interface.
4. Update the README quick start and publish configuration, security, compatibility, and platform documentation.
5. Build and smoke-test all release artifacts before publishing them.

There is no stored data or deployed service to migrate. Rollback consists of withdrawing the new binary and documentation; no compatibility alias or configuration conversion is required.
