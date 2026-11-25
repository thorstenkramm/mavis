# Spec Delta

## Purpose

Delivers Mavis as documented, prebuilt executables that users can run on supported desktop operating systems without installing a language runtime.

## ADDED Requirements

### Requirement: Provide prebuilt operating-system binaries
Mavis SHALL provide prebuilt binaries for macOS, Linux, and Windows that run without separately installing Go or another language runtime.

#### Scenario: Run a downloaded binary
- **WHEN** a user downloads the binary for a supported platform and runs `mavis serve .`
- **THEN** Mavis starts without requiring a language runtime or a project dependency install

### Requirement: Document the compatibility matrix
Mavis SHALL document supported operating systems, CPU architectures, browser versions, and known platform limitations before release.

#### Scenario: Select a release artifact
- **WHEN** a user consults the release documentation
- **THEN** the user can identify the correct artifact and supported browser/environment for their system

### Requirement: Define platform behavior for Unix sockets
Mavis SHALL support Unix-socket listeners on supported Unix-like systems and SHALL document Windows behavior, returning an actionable error where the requested socket mode is unsupported.

#### Scenario: Request a socket on an unsupported platform
- **WHEN** a user configures a Unix socket on a platform where Mavis does not support it
- **THEN** Mavis reports the unsupported listener clearly instead of silently using TCP

### Requirement: Ship accurate usage and compatibility documentation
Mavis documentation SHALL describe `serve`, TCP and socket configuration, `.mavis.conf`, exclusions, hidden files, network warnings, core GLFM support, and unsupported extensions.

#### Scenario: Follow the quick start
- **WHEN** a new user follows the documented quick start
- **THEN** the user runs `mavis serve .`, opens the printed local URL manually, and reaches the documentation interface

#### Scenario: Review Markdown compatibility
- **WHEN** a user consults the rendering documentation
- **THEN** the user can distinguish supported core GLFM from unsupported specialized GitLab extensions

### Requirement: Separate public and maintainer documentation
Mavis SHALL maintain public user documentation separately from maintainer-only documents and SHALL exclude maintainer-only content from the published documentation site.

#### Scenario: Build the public documentation site
- **WHEN** the project documentation site is built
- **THEN** public user documentation is available and maintainer-only documents are absent from the generated output

### Requirement: Verify and deploy documentation changes
Mavis SHALL verify documentation changes before publication and SHALL deploy the public documentation site to GitHub Pages only from `main`.

#### Scenario: Review a documentation change
- **WHEN** a documentation change is submitted for review
- **THEN** the documentation site builds successfully without deploying to GitHub Pages

#### Scenario: Publish documentation from main
- **WHEN** a verified documentation change is present on `main`
- **THEN** the documentation workflow deploys the updated public site

### Requirement: Verify release artifacts per platform
Each release SHALL pass startup, document rendering, configuration, and content-boundary smoke checks with its intended macOS, Linux, and Windows binaries.

#### Scenario: Validate a platform artifact
- **WHEN** a release candidate is built for a supported operating system
- **THEN** its binary completes the documented smoke checks before publication
