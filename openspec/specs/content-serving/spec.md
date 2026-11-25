# content-serving Specification

## Purpose

Provides safe, current browser access to eligible files within a selected documentation directory and its folder hierarchy.

## Requirements

### Requirement: Expose the eligible directory hierarchy
Mavis SHALL make the served root's eligible folders and files discoverable and accessible without copying them into another content store.

#### Scenario: Browse a nested documentation tree
- **WHEN** the served root contains nested folders and eligible files
- **THEN** users can browse that hierarchy and request its eligible content

### Requirement: Serve non-Markdown files with appropriate content types
Mavis SHALL serve original non-Markdown bytes with their appropriate `Content-Type` through explicit raw and download representations. Normal navigation to eligible UTF-8 text or code, images, and PDFs SHALL return an HTML viewer; other files SHALL retain raw responses.

#### Scenario: Request an image
- **WHEN** a user requests the raw representation of an eligible PNG image
- **THEN** Mavis returns the image with an image content type rather than an HTML viewer

#### Scenario: Request another file type
- **WHEN** a user requests the raw representation of an eligible PDF, text file, or downloadable asset
- **THEN** Mavis returns the original bytes with an appropriate content type and leaves inline-versus-download behavior to the browser

#### Scenario: Request a download
- **WHEN** a user requests the explicit download representation of an eligible non-Markdown file
- **THEN** Mavis returns the original bytes with an appropriate content type and a safe `Content-Disposition: attachment` filename

#### Scenario: Open a viewer-supported file
- **WHEN** a user navigates normally to an eligible UTF-8 text or code file, image, or PDF
- **THEN** Mavis returns the corresponding HTML viewer and provides a link to its raw representation

#### Scenario: Request an unsupported viewer format
- **WHEN** a user navigates to an eligible file for which Mavis has no built-in viewer
- **THEN** Mavis returns the raw file response with an appropriate content type

### Requirement: Render a directory README as its index
When a requested directory contains `README.md`, Mavis SHALL display that document as the directory's default page; otherwise it SHALL present the directory listing.

#### Scenario: README exists
- **WHEN** a user opens a directory containing an eligible `README.md`
- **THEN** Mavis displays the rendered README with navigation still available

#### Scenario: README does not exist
- **WHEN** a user opens a directory without an eligible `README.md`
- **THEN** Mavis presents the eligible folder and file listing

### Requirement: Control hidden-file visibility
Mavis SHALL exclude hidden files and folders by default, allow explicit inclusion, and continue applying exclusions when hidden content is included.

#### Scenario: Default behavior
- **WHEN** hidden-file inclusion is not enabled
- **THEN** hidden content is absent from listings and unavailable through direct requests

#### Scenario: Inclusion with an exclusion
- **WHEN** hidden-file inclusion is enabled and an excluded hidden file is requested
- **THEN** Mavis still refuses to list or serve that file

### Requirement: Enforce exclusions at every access path
Mavis SHALL prevent wildcard-excluded files and folders from appearing in listings or being retrieved through viewer pages, raw representations, direct requests, or crafted requests.

#### Scenario: Request excluded content directly
- **WHEN** a user requests either the viewer or raw URL corresponding to an excluded file or a file below an excluded folder
- **THEN** Mavis does not return the content through either representation

### Requirement: Confine content to the served root
Mavis SHALL prevent path traversal and SHALL follow symbolic links only when their resolved targets remain inside the served root.

#### Scenario: Traversal attempt
- **WHEN** a request attempts to address content outside the served root
- **THEN** Mavis refuses the request without exposing filesystem content

#### Scenario: Internal symbolic link
- **WHEN** an eligible symbolic link resolves to a target inside the served root
- **THEN** Mavis serves the target subject to all visibility and exclusion rules

#### Scenario: External symbolic link
- **WHEN** a symbolic link resolves outside the served root
- **THEN** Mavis does not list or serve the linked target

### Requirement: Keep server configuration private
Mavis SHALL NEVER list or serve the root `.mavis.conf`, including when hidden-file inclusion is enabled.

#### Scenario: Request the configuration file
- **WHEN** a user directly requests `.mavis.conf`
- **THEN** Mavis refuses to return it regardless of other visibility settings

### Requirement: Reflect source changes on manual refresh
Mavis SHALL read current source content so a manual browser refresh reflects changes without a server restart and SHALL NOT automatically reload the browser.

#### Scenario: Edit a served document
- **WHEN** a user edits a served document and manually refreshes its page
- **THEN** Mavis displays the updated content without an automatic reload or restart

### Requirement: Isolate content failures
Mavis SHALL return clear errors for missing, excluded, or unreadable content without exposing unnecessary filesystem details or preventing unrelated eligible content from being served.

#### Scenario: One document is unreadable
- **WHEN** a requested document cannot be read
- **THEN** Mavis reports the failure for that request and continues serving other eligible documents

### Requirement: Preserve local content privacy
Mavis SHALL NOT transmit served document contents to external services as part of basic serving or rendering.

#### Scenario: Render a local document
- **WHEN** a user browses local documentation with default functionality
- **THEN** Mavis renders and serves it without sending its contents to an external service

### Requirement: Prevent active content in non-Markdown viewers
Mavis SHALL escape text and code displayed in built-in viewers and SHALL preserve script-disabled sandboxing for raw active-content responses.

#### Scenario: Text file contains markup
- **WHEN** a user opens a text or code file containing HTML or script-like source in the built-in viewer
- **THEN** Mavis displays the source as literal text without creating executable document elements

#### Scenario: Open raw active content
- **WHEN** a user requests raw HTML, SVG, or JavaScript content
- **THEN** Mavis returns a sandboxed response with scripts disabled
