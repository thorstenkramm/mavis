# Proposal

## Why

Mavis currently has a product vision and confirmed PRD but no implementation. The baseline release is needed to turn an existing directory of Markdown and supporting files into a secure, navigable documentation site without a build step, hosted service, or language runtime installed by the user.

## What Changes

- Add a Go-based `mavis serve <directory>` command that serves the selected directory at `localhost:3344` by default, prints its address and root, supports graceful shutdown, and never opens a browser automatically.
- **BREAKING:** Establish `serve` as the only serving command spelling; do not retain the README's former `serv` spelling as an alias.
- Add TCP host/port configuration and Unix-socket operation, including listener validation, actionable startup failures, and an explicit warning for non-loopback binding without authentication.
- Add `.mavis.conf` TOML configuration for exclusions, hidden-file inclusion, and server defaults, with CLI overrides for server settings and combined CLI/file exclusion lists.
- Serve directory content dynamically so manual browser refresh reflects source changes without automatic reloads or a server restart.
- Render `.md` and `.markdown` files with core GLFM, heading anchors, a current-document table of contents, and rewritten relative document links.
- Serve eligible non-Markdown files with appropriate content types and display a directory's `README.md` as its default page.
- Enforce hidden-file, exclusion, served-root, symbolic-link, and `.mavis.conf` access rules for both listings and direct requests.
- Add a responsive, accessible browser interface with a file browser, two-pane desktop reading view, document table of contents, browser history, and clear empty/error states.
- Document core GLFM compatibility, configuration, supported platforms, and provide prebuilt binaries for macOS, Linux, and Windows.
- Publish a Hugo documentation site using the Mavis-adapted Mailfire theme, with public content under `docs/content/`, maintainer documents under `docs/internal/`, and GitHub Pages deployment.
- Enforce the shared `run-all-tests.sh` quality suite with formatting, tests, race checks, linting, 85% coverage, jscpd duplication checks, Hugo verification, and Trivy vulnerability scanning.
- Defer full-text search of served content, automatic browser reload, static-site export of served directories, custom themes for the Mavis browser interface, specialized GitLab extensions, and package-manager/container distribution.

## Capabilities

### New Capabilities

- `server-cli`: Command-line serving, listener selection, startup/shutdown behavior, diagnostics, and network-binding warnings.
- `server-configuration`: `.mavis.conf` schema, defaults, validation, CLI precedence, exclusion merging, and hidden-file settings.
- `content-serving`: Directory and file access, content types, directory indexes, refresh behavior, exclusion enforcement, and filesystem boundary protection.
- `markdown-rendering`: Markdown detection, core GLFM rendering, heading anchors, document-link rewriting, and table-of-contents data.
- `documentation-browser`: File-tree navigation, two-pane reading experience, responsive behavior, accessibility, history, and user-facing states.
- `binary-distribution`: Prebuilt platform binaries and accompanying compatibility and usage documentation.

### Modified Capabilities

None. The repository has no existing capability specs.

## Impact

- Creates a new Go module, CLI entry point, internal server/configuration/content/rendering packages, embedded browser assets, and automated tests.
- Introduces the public `mavis serve` CLI, `.mavis.conf` format, HTTP routes, rendered HTML behavior, and release artifacts.
- Requires Go dependencies for CLI parsing, TOML, Markdown/GLFM rendering, and sanitization; exact selections belong to the technical design.
- Adds cross-platform build, test, security, accessibility, and release verification for macOS, Linux, and Windows.
- Adds a self-contained Hugo/UIkit documentation theme, public/internal documentation separation, pinned documentation deployment, jscpd configuration, and Trivy filesystem scanning.
- Updates the README's `serv` example and expands usage/configuration documentation.
- Does not add accounts, hosted publishing, a database, remote synchronization, or transmission of document contents to external services.
