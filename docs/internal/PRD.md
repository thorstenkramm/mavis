# Mavis Product Requirements Document

| Field | Value |
| --- | --- |
| Product | Mavis — Markdown View Server |
| Status | Draft for review |
| Date | 2026-11-24 |
| Source material | Repository `README.md` |

## 1. Overview

Mavis is a Markdown View Server that turns a local directory of Markdown and other files into a browsable HTML documentation site. A user starts Mavis from a documentation directory, opens the local site in a browser, and navigates through a file browser and rendered document content.

The initial product experience is:

```sh
cd ~/my-project-docs
mavis serve .
```

The user then visits `http://localhost:3344`.

This PRD combines the behavior stated in the README with confirmed product decisions gathered through stakeholder questions. It defines the baseline release scope rather than describing an already-complete implementation.

## 2. Problem Statement

Teams and individuals often keep documentation as plain files in a directory or repository. Reading those files through an editor or raw file viewer makes it harder to:

- read Markdown as formatted documentation;
- discover related documents and folders;
- navigate long documents through a table of contents;
- preview documentation locally before publishing or sharing it; and
- control which files should be exposed without reorganizing the source directory.

Existing documentation platforms can require a build pipeline, a hosted service, a repository convention, or a dedicated site configuration. Mavis should provide a lightweight alternative for serving an existing directory directly.

## 3. Product Vision

Make local Markdown documentation readable and navigable in a browser with a single command, without requiring users to convert, move, or build their source files.

### Confirmed Product Decisions

- Mavis is a **local-first documentation preview tool**. It binds locally by default; users may deliberately bind to other interfaces with a clear security warning. Public hosted publishing remains outside the product scope.
- The host and port must be configurable, and Mavis must also support serving over a Unix socket.
- Mavis is general-purpose: developers, technical writers, and general Markdown users are equally important audiences.
- The CLI command will be renamed from `serv` to `serve`; backward compatibility with `serv` is not required.
- Mavis never opens a browser automatically; it prints the address for the user to open manually.
- Source changes are reflected when the user manually refreshes; automatic browser reload is not required.
- Mavis must support macOS, Linux, and Windows; Unix-socket support may vary where the operating system does not support it.
- Mavis is distributed as prebuilt binaries; package-manager and container distributions are not required.
- Markdown detection uses `.md` and `.markdown`, matched case-insensitively.
- Non-Markdown files are served with the appropriate `Content-Type`; the browser decides whether to display them inline or download them.
- Mavis targets core GLFM—tables, task lists, strikethrough, fenced code, heading anchors, and related common features—with unsupported extensions documented.
- Relative links to Markdown documents are rewritten to rendered Mavis routes, preserving anchors and asset links.
- Opening a directory displays its `README.md` as the default page when one exists.
- The table of contents represents the headings in the current document; the file browser handles collection navigation.
- The table of contents appears inside the right-hand content pane.
- The initial release must provide a usable responsive layout on mobile and narrow screens.
- Full-text search is deferred to a later release.
- `.mavis.conf` supports both content exclusions and server settings, including host, port, and Unix-socket configuration.
- For server settings, CLI flags override `.mavis.conf` values.
- CLI and file-based exclusion lists are combined; content matching either list is excluded.
- Hidden files and folders remain excluded by default but may be explicitly included; exclusion rules still apply.
- Symbolic links are followed only when their resolved targets remain inside the served root.
- `.mavis.conf` is never served, even when hidden files are explicitly included.
- Non-loopback binding requires explicit configuration and a warning; authentication is not required.
- External absolute URLs remain unchanged; root-relative paths resolve against the Mavis served root.
- `.mavis.conf` uses top-level `exclude` and `include_hidden` keys, with `host`, `port`, and optional `socket` under `[server]`.
- Configuring a Unix socket together with host or port settings is an error; users must select one listener type.

## 4. Target Users

### 4.1 Software developers

Developers who maintain Markdown documentation alongside code and want to preview it locally with formatting and navigation.

### 4.2 Technical writers

Writers who work with collections of Markdown files and need a convenient way to review document structure and rendered content.

### 4.3 Engineering teams

Teams that keep project knowledge in repositories and want a low-configuration documentation viewer for local use.

### 4.4 General Markdown users

People with directories of notes or documents who prefer a browser-based reading experience.

## 5. Goals

- Serve an existing directory as rendered HTML with one CLI command.
- Render GitLab Flavored Markdown (GLFM).
- Make folders and documents discoverable through a browsable file tree.
- Provide a table of contents for rendered documents.
- Present navigation and content in a clear two-pane layout.
- Serve non-hidden files by default.
- Let users exclude files and folders using wildcard patterns.
- Support exclusions through both CLI flags and a root-level `.mavis.conf` TOML file.
- Keep setup and configuration lightweight.

## 6. Non-Goals

The following are non-goals for the initial release:

- A hosted multi-user documentation platform.
- A Markdown editor or content management system.
- User accounts, authentication, or organization management.
- Full static-site generation and deployment pipelines.
- Automatic synchronization with remote repositories.
- A plugin ecosystem or theme marketplace.
- Collaborative editing, comments, or review workflows.
- Full-text search, which is deferred to a later release.
- Automatic browser reload when source files change.
- Package-manager or container distribution.

## 7. User Stories

### 7.1 Serving documentation

> As a documentation author, I want to run one command in a directory so that I can read its documents in my browser without a separate build step.

### 7.2 Browsing files

> As a reader, I want to browse the served directory hierarchy so that I can discover and open related documents.

### 7.3 Reading rendered Markdown

> As a reader, I want Markdown rendered as HTML so that I can read formatted headings, lists, links, code, and other supported content.

### 7.4 Navigating long documents

> As a reader, I want a table of contents so that I can understand a document’s structure and jump to a section.

### 7.5 Controlling exposed content

> As a documentation author, I want to exclude selected files and folders so that I can serve only the intended content.

### 7.6 Reusing configuration

> As a repeated user, I want to store exclusions and server defaults in `.mavis.conf` so that I do not need to repeat CLI flags each time I start Mavis.

## 8. Functional Requirements

### 8.1 Start a documentation server

**Priority: Must have**

1. The CLI shall provide a `serve` command; the former `serv` spelling is not retained as an alias.
2. The command shall accept a directory argument, including `.` for the current directory.
3. Mavis shall use the supplied directory as the root of the browsable content.
4. The default local address shall be `http://localhost:3344`.
5. Users shall be able to configure the listening host and port through CLI flags or `.mavis.conf`.
6. Users shall be able to serve through a Unix socket instead of TCP where supported by the operating system.
7. Configuring a Unix socket together with host or port settings shall produce an actionable configuration error.
8. After startup, Mavis shall print the serving address or socket path and the served root directory; it shall not open a browser automatically.
9. Startup failures shall produce a clear error and a non-zero exit status.
10. Binding to a non-loopback interface shall require deliberate configuration and display a clear warning that anyone with network access may read the served files.
11. Mavis shall not require authentication for non-loopback binding.

### 8.2 Render Markdown

**Priority: Must have**

1. Mavis shall recognize `.md` and `.markdown` files case-insensitively and render them as HTML.
2. Mavis shall support core GLFM, including tables, task lists, strikethrough, fenced code blocks, and heading anchors.
3. Unsupported GLFM extensions shall be documented; full GitLab parity, including specialized diagrams, is not required.
4. Relative links between Markdown documents shall be rewritten to rendered Mavis routes while preserving heading anchors.
5. External absolute URLs shall remain unchanged, and root-relative paths shall resolve against the served root.
6. Relative asset links shall continue to resolve to the corresponding served assets.
7. Invalid or unusual Markdown shall not prevent the rest of the site from being browsed.

### 8.3 Browse directories and files

**Priority: Must have**

1. The interface shall display the served folder hierarchy in a file browser.
2. Users shall be able to expand, collapse, or otherwise navigate folders.
3. Selecting a Markdown document shall display its rendered content.
4. Opening a directory shall render its `README.md` as the default page when one exists.
5. Non-Markdown files that are in scope shall be served with the appropriate `Content-Type`; the browser determines whether to display or download them.
6. Hidden files and folders shall be excluded by default, with an explicit option to include them.
7. Exclusion rules shall continue to apply when hidden-file inclusion is enabled.
8. Symbolic links shall be followed only when their resolved targets remain inside the served root.
9. `.mavis.conf` shall never be served, even when hidden-file inclusion is enabled.

### 8.4 Display document structure

**Priority: Must have**

1. Mavis shall provide a table of contents based on the current document’s headings.
2. The table of contents shall appear inside the right-hand content pane.
3. Table-of-contents entries shall link to the corresponding document sections.
4. The selected section or heading behavior shall be consistent across supported browsers.

### 8.5 Provide a two-pane reading experience

**Priority: Must have**

1. The primary desktop interface shall use two panes: file navigation on the left and rendered content, including its table of contents, on the right.
2. Navigation shall remain available while reading a document.
3. The interface shall remain usable at common desktop viewport sizes.
4. The initial release shall provide a usable responsive layout on mobile and narrow screens, including a way to reach the file browser when space is limited.

### 8.6 Exclude files and folders

**Priority: Must have**

1. Users shall be able to exclude files and folders from being served.
2. Exclusion rules shall support wildcard patterns.
3. Exclusions shall be configurable through CLI flags.
4. Exclusions shall also be configurable in a `.mavis.conf` file in the served root directory.
5. `.mavis.conf` shall use TOML syntax.
6. CLI and file-based exclusion lists shall be combined; content matching either list shall be excluded.
7. For server settings supplied in both places, CLI flags shall override `.mavis.conf` values.
8. Excluded content shall not be listed or accessible through direct URL requests.

### 8.7 Stop and restart the server

**Priority: Must have**

1. Users shall be able to stop the server through normal CLI interruption, such as `Ctrl+C`.
2. Restarting Mavis shall use the current files and configuration.
3. Source-file changes shall be reflected on manual browser refresh without requiring a server restart.
4. Mavis shall not automatically reload the browser when files change.
5. Port conflicts shall produce an actionable error message.

## 9. User Experience Requirements

### 9.1 First-run experience

A new user should be able to move from installation to reading a document by following three steps:

1. Open a terminal in the documentation directory.
2. Run `mavis serve .`.
3. Open `http://localhost:3344`.

### 9.2 Navigation

- The current document should be identifiable in the file browser.
- Folder and file names should be readable without horizontal scrolling where practical.
- Browser back and forward navigation should work when moving between documents.
- Heading anchors should produce shareable document URLs.

### 9.3 Empty and error states

The following states should have clear, actionable presentation:

- The served directory contains no visible documents.
- The requested file does not exist.
- The requested content is excluded.
- A file cannot be read.
- The configuration file is invalid.
- The configured port is unavailable.

## 10. Non-Functional Requirements

The following quality requirements apply to the baseline release. Measurable performance targets remain implementation-planning work.

### 10.1 Performance

- The initial page should become usable quickly for a typical documentation directory.
- Opening a document should not require restarting the server.
- Large directories and documents should not make navigation unresponsive.
- Performance benchmarks should cover directory size, document size, and number of files.

### 10.2 Reliability

- Mavis should continue serving accessible documents if one document cannot be rendered.
- Errors should identify the affected file or configuration entry without exposing unnecessary system details.
- Normal shutdown should release the listening port.

### 10.3 Security and privacy

- Mavis shall serve only content permitted by the selected root, hidden-file setting, and exclusion rules.
- Requests must not escape the served root through path traversal or symbolic links.
- Hidden and excluded files must not be retrievable through direct requests unless hidden files have been explicitly included and not excluded.
- `.mavis.conf` must never be retrievable through the server.
- Binding beyond localhost requires deliberate configuration and a clear warning; Mavis does not provide authentication.
- Mavis should not transmit document contents to external services as part of basic local serving.

### 10.4 Compatibility

- Mavis shall provide prebuilt binaries for macOS, Linux, and Windows.
- Unix-socket support is required on supported Unix-like systems; Windows behavior must be documented where Unix sockets are unavailable or differ.
- Supported CPU architectures and browser versions must be documented before release.
- Core GLFM rendering differences from GitLab shall be documented.

### 10.5 Accessibility

Requirements:

- Navigation and document controls should be keyboard accessible.
- Focus states should be visible.
- Text and controls should meet WCAG AA color-contrast targets.
- The interface should use semantic HTML for headings, navigation, and document content.

### 10.6 Observability and diagnostics

Requirements:

- CLI output should identify the serving address and root directory.
- Errors should include enough context to diagnose invalid configuration, unreadable files, and port conflicts.
- Verbose logging, if provided, should be opt-in.

## 11. Configuration Requirements

Mavis reads `.mavis.conf` from the served root. The file uses TOML and supports content rules and server defaults.

TCP example:

```toml
exclude = ["drafts/**", "*.tmp"]
include_hidden = false

[server]
host = "127.0.0.1"
port = 3344
```

Unix-socket alternative:

```toml
exclude = ["drafts/**", "*.tmp"]
include_hidden = false

[server]
socket = "/tmp/mavis.sock"
```

Configuration behavior:

- `exclude` is an array of wildcard patterns applied to files and folders.
- `include_hidden` defaults to `false`; exclusion rules still apply when it is `true`.
- `[server]` supports `host`, `port`, and optional `socket` values.
- `socket` is an alternative to TCP, not an additional listener. Configuring it together with host or port settings is an error.
- CLI flags override file values for server settings.
- CLI and file exclusion lists are combined rather than replaced.
- `.mavis.conf` itself is never served.

Before implementation is complete, the configuration reference shall define wildcard syntax, path normalization, matching rules, defaults, value types, validation errors, and common examples.

## 12. Success Metrics

No baseline metrics are available. Proposed initial measures are:

### 12.1 Activation

- Percentage of new users who successfully serve a directory and open a rendered document on their first attempt.
- Time from starting the CLI to viewing the first rendered document.

### 12.2 Reliability

- Frequency of startup failures caused by Mavis rather than invalid user input.
- Frequency of rendering or file-serving errors during normal use.

### 12.3 Usability

- Percentage of usability-test participants who can open a document, navigate using the table of contents, and configure an exclusion without assistance.
- User-reported satisfaction with file discovery and reading.

### 12.4 Adoption

- Number of active installations or repeated CLI uses, measured only if a privacy-preserving measurement method is explicitly approved.
- Qualitative feedback from developers and technical writers.

## 13. Baseline Acceptance Criteria

The baseline release is acceptable when:

1. A user can run `mavis serve .` from a documentation directory and access the site at `http://localhost:3344` without Mavis opening a browser.
2. `.md` and `.markdown` files render as HTML with the defined core GLFM feature set.
3. The desktop interface displays a left-side file browser and right-side rendered content, and narrow screens retain usable navigation and reading.
4. A user can browse folders, open served documents, and use browser back and forward navigation.
5. A directory with `README.md` displays that document by default.
6. The current document exposes a navigable table of contents inside the content pane.
7. Relative Markdown links open rendered documents, anchors remain functional, and assets load from their intended paths.
8. Non-Markdown files are served with appropriate content types.
9. Non-hidden files are accessible by default, subject to exclusion rules.
10. Hidden files remain unserved by default and become available only through explicit inclusion.
11. CLI and file-based wildcard exclusions combine and prevent matching content from being listed or retrieved.
12. `.mavis.conf` accepts the documented TOML schema and is never served.
13. CLI server settings override file values.
14. Users can configure TCP host and port or select a Unix socket; conflicting listener settings fail with an actionable error.
15. Non-loopback binding emits the documented security warning.
16. Source changes appear after manual browser refresh without an automatic page reload or server restart.
17. Invalid paths, invalid configuration, unavailable ports, and unreadable content produce actionable errors.
18. Requests cannot retrieve content outside the served root, including through symbolic links.
19. Keyboard navigation, visible focus, semantic structure, and WCAG AA contrast requirements are met.
20. Prebuilt macOS, Linux, and Windows binaries and basic usage/configuration documentation are available.

## 14. Release Scope

### Must have

- `mavis serve` directory-serving command.
- Default localhost address on port `3344`.
- Configurable TCP host and port, plus Unix-socket support.
- Explicit warning for non-loopback binding.
- Core GLFM rendering and rendered-document link rewriting.
- Appropriate content types for non-Markdown files.
- File and folder browser with `README.md` directory indexes.
- Current-document table of contents inside the content pane.
- Responsive two-pane desktop and narrow-screen layouts.
- Default hidden-file behavior and explicit hidden-file inclusion.
- Combined wildcard exclusions through CLI and `.mavis.conf`.
- Path-boundary, symbolic-link, configuration-file, and exclusion enforcement.
- Manual-refresh updates without automatic browser reload.
- Clear empty and error states and actionable startup diagnostics.
- Browser history and keyboard-accessible navigation.
- Prebuilt macOS, Linux, and Windows binaries.
- Essential usage and configuration documentation.

### Later releases

- Full-text search.
- Custom themes.
- Additional Markdown dialects or specialized GitLab extensions.
- Static-site export.
- Package-manager or container distribution.

## 15. Decision Log

All initial product questions have been resolved and incorporated into the requirements. The questions and answers are retained here for traceability.

### Product and audience

1.  Is Mavis strictly a local preview tool, or is publishing to other users a supported goal? **Answered: local-first by default; deliberate non-loopback binding is allowed with a warning, but hosted public publishing is out of scope.**
2.  Is the primary audience developers, technical writers, or general Markdown users? **Answered: all audiences equally.**
3.  Does “publish documentation” mean local viewing only, LAN sharing, or deployment to a public server? **Answered: local viewing by default, with user-configured network exposure possible.**

### CLI and server

4.  What does `serv` stand for in the public CLI design, and should `serve` also be accepted? **Answered: rename to `serve`; no compatibility alias required.**
5.  Should host and port be configurable? **Answered: host and port are configurable, Unix sockets are supported, and non-loopback binding requires a warning.**
6.  Should Mavis open the browser automatically? **Answered: never; print the address instead.**
7.  Should file changes trigger reloads, or is restart required? **Answered: update on manual browser refresh; no automatic reload or restart required.**
8.  Which installation methods and operating systems must be supported? **Answered: prebuilt binaries for macOS, Linux, and Windows.**

### Content and rendering

9.  Which file extensions should be treated as Markdown? **Answered: `.md` and `.markdown`, case-insensitively.**
10.  Which non-Markdown file types should render inline, download, or display as raw text? **Answered: serve with the appropriate `Content-Type` and let the browser decide.**
11.  Is full GLFM parity required, including diagrams and other extended features? **Answered: core GLFM is sufficient; document unsupported extensions.**
12.  How should relative links, absolute links, images, and heading anchors behave? **Answered: rewrite relative Markdown links to rendered routes, preserve external absolute URLs and anchors, resolve root-relative paths against the served root, and serve assets with their appropriate content types.**
13.  Should a directory have a default index document, such as `README.md`? **Answered: yes, display `README.md` when present.**

### Navigation and interface

14.  Is the table of contents per document, for the whole directory, or both? **Answered: current document only.**
15.  Where should the table of contents appear in the two-pane layout? **Answered: inside the right-hand content pane.**
16.  Is mobile support required for the baseline release? **Answered: yes, a responsive baseline is required.**
17.  Is full-text search in scope? **Answered: deferred to a later release.**

### Configuration and security

18.  What is the exact `.mavis.conf` schema? **Answered: top-level `exclude` and `include_hidden`, plus `host`, `port`, and optional `socket` under `[server]`.**
19.  How are CLI and file-based exclusions combined? **Answered: combine both exclusion lists; CLI overrides file values for server settings.**
20.  Can hidden files be explicitly included? **Answered: yes, through explicit opt-in; exclusion rules still apply.**
21.  Should symbolic links be followed? **Answered: only when resolved targets remain inside the served root.**
22.  Is `.mavis.conf` itself hidden from the served site? **Answered: always; it is never served.**
23.  Is external network binding supported, and if so, what security controls are required? **Answered: supported with explicit configuration and a warning; no authentication required.**

## 16. Assumptions

- The README is the authoritative source for currently stated product behavior.
- The baseline product is local-first and does not require accounts or a hosted service.
- Existing source documents remain in place; Mavis serves them rather than importing them into a separate content store.
- The initial interface is browser based.
- Product decisions in this document have been confirmed; measurable performance targets, supported CPU architectures, supported browser versions, and detailed wildcard semantics remain implementation-planning work.
