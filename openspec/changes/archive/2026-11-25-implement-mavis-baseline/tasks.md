# Tasks

## 1. Go scaffolding and configuration

- [x] 1.1 Create the Go 1.27.2 module, package layout, and test fixtures; add Cobra, pelletier/go-toml/v2, doublestar, goldmark, and bluemonday dependencies, then verify the scaffold with `go mod tidy`, `go test ./...`, and the adopted Makefile quality entry points.
- [x] 1.2 Implement typed `.mavis.conf` loading, safe defaults, strict TOML/key/type validation, and setting-specific errors; verify absent, malformed, unknown, and invalid configurations with unit tests.
- [x] 1.3 Implement source-aware file/CLI merging, CLI server overrides, exclusion union, and explicit socket-versus-TCP conflict detection; verify precedence and conflict tables with unit tests.
- [x] 1.4 Implement root-relative doublestar compilation/matching and cross-platform dotfile semantics; verify `*`, `**`, parent-directory exclusion, malformed patterns, and path normalization with unit tests and document the syntax in `docs/content/reference/configuration.md`.

## 2. CLI and listener lifecycle

- [x] 2.1 Build the Cobra root and `serve <directory>` command with exact argument validation, local serving flags, content-root validation, and no `serv` alias; verify command behavior and nonzero failures with CLI tests.
- [x] 2.2 Implement TCP listener creation, default/custom startup output, hostname loopback detection, non-loopback warnings, and bind diagnostics; verify output and errors with listener and command tests.
- [x] 2.3 Implement platform-specific Unix-socket listeners with stale-socket safeguards, local permissions, owned cleanup, and an actionable Windows unsupported error; verify supported and unsupported behavior with platform-tagged tests.
- [x] 2.4 Add signal-driven graceful shutdown and bounded HTTP shutdown; verify interruption releases TCP and socket listeners and allows an immediate restart in integration tests.
- [x] 2.5 Update the README quick start and complete `docs/content/reference/configuration.md` with CLI flags, TOML examples, precedence, listener conflicts, hidden files, and network warnings; verify every documented command and configuration example against the implementation.

## 3. Authorized content serving

- [x] 3.1 Implement the central URL/filesystem resolver with segment validation, traversal rejection, symlink resolution, `filepath.Rel` containment, and regular-file checks; verify encoded, platform-specific, and adversarial paths with table-driven tests.
- [x] 3.2 Enforce `.mavis.conf`, hidden-file, wildcard, parent-directory, logical-path, and resolved-target authorization in one shared policy; verify listing and direct-request bypass attempts with unit and HTTP tests.
- [x] 3.3 Implement per-request directory models, eligible folder/file listings, `README.md` indexes, current-ancestry navigation data, and no whole-tree scans; verify nested navigation, README fallback, empty directories, and source edits with tests.
- [x] 3.4 Implement safe non-Markdown file responses using resolved open files, extension/sniffed MIME detection, `nosniff`, and non-revealing error mapping; verify images, text, PDFs, unknown types, missing files, and unreadable files with `httptest` coverage.
- [x] 3.5 Document content visibility, symlinks, exclusion behavior, privacy, and the residual local-adversary filesystem race in `docs/content/reference/security.md`; verify the documented behavior against the content-policy test matrix.

## 4. Markdown rendering

- [x] 4.1 Implement case-insensitive Markdown detection and goldmark core GFM rendering with tables, task lists, strikethrough, linkification, fenced code, and safe raw-HTML defaults; verify each claimed feature with golden/rendering tests.
- [x] 4.2 Add automatic heading IDs and current-document hierarchical TOC extraction; verify duplicate headings, fragments, and heading levels with rendering tests.
- [x] 4.3 Implement AST-aware relative Markdown-link rewriting, fragment/query preservation, relative assets, root-relative paths, and unchanged external URLs; verify nested documents and excluded/out-of-root targets with tests.
- [x] 4.4 Add the bluemonday allowlist for safe formatting, heading IDs, tables, images, links, code, and disabled task checkboxes; verify script/event/unsafe-scheme removal and safe-markup preservation with security tests.
- [x] 4.5 Add isolated rendering-error fallback and write `docs/content/reference/markdown.md` covering supported core GLFM and unsupported GitLab extensions; verify one bad document does not affect other requests and review documentation against fixtures.

## 5. HTTP and browser experience

- [x] 5.1 Connect embedded assets, directory, Markdown, and file handlers to explicit HTTP routes with security headers, dynamic `no-store`, and on-demand reads; verify manual-refresh freshness, reserved asset routing, CSP, and isolation with end-to-end HTTP tests.
- [x] 5.2 Build semantic server-rendered templates for two-pane desktop reading, folder navigation, current-document state, TOC, README pages, empty states, and errors; verify markup and navigation states with template and `httptest` tests.
- [x] 5.3 Add responsive CSS and minimal JavaScript for a dismissible narrow-screen file navigator without a client router; verify desktop/mobile layouts, browser Back/Forward, heading URLs, and no horizontal page scrolling in browser smoke tests.
- [x] 5.4 Add keyboard-operable controls, visible focus, `aria-current`, semantic landmarks, and WCAG AA contrast; verify keyboard-only operation, automated accessibility checks, and current stable Chrome/Edge, Firefox, and Safari behavior, recording results in `docs/content/reference/browser-compatibility.md`.

## 6. Cross-platform distribution

- [x] 6.1 Complete CI for `make lint`, `make test`, `make coverage` (minimum 85%), jscpd, Trivy, and builds across macOS amd64/arm64, Linux amd64/arm64, and Windows amd64; verify the quality jobs enforce the same required Go, duplication, coverage, and Trivy checks as `./run-all-tests.sh`, with cross-platform tests and standalone-binary builds for the documented targets.
- [x] 6.2 Add release packaging and native smoke checks for startup, rendering, configuration, and content boundaries; verify artifacts on available macOS, Linux, and Windows runners and confirm unsupported Windows socket behavior.
- [x] 6.3 Finalize installation/download instructions and the OS, CPU, browser, Unix-socket, and GLFM compatibility documentation; verify each release artifact includes accurate documentation and no Go/runtime prerequisite.
- [x] 6.4 Complete and verify the Hugo documentation site and Mavis theme, keeping public content in `docs/content/` and maintainer documents in `docs/internal/`; run `make docs`, verify maintainer content is absent from generated output, and verify the documentation workflow builds pull requests without deployment and deploys only from `main`.

## 7. Baseline integration verification

- [x] 7.1 Assemble a representative documentation tree and run an end-to-end checklist covering all 20 PRD acceptance criteria, recording any failure and returning it to the owning task group rather than deferring it.
- [x] 7.2 Run `./run-all-tests.sh`, security/traversal tests, and release-binary smoke checks, then verify `openspec validate implement-mavis-baseline --strict` passes and report the final results.
