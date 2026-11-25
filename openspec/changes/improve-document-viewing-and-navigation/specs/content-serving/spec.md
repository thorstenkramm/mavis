# Spec Delta

## MODIFIED Requirements

### Requirement: Serve non-Markdown files with appropriate content types
Mavis SHALL serve original non-Markdown bytes with their appropriate `Content-Type` through an explicit raw representation. Normal navigation to eligible UTF-8 text or code, images, and PDFs SHALL return an HTML viewer; other files SHALL retain raw responses.

#### Scenario: Request an image
- **WHEN** a user requests the raw representation of an eligible PNG image
- **THEN** Mavis returns the image with an image content type rather than an HTML viewer

#### Scenario: Request another file type
- **WHEN** a user requests the raw representation of an eligible PDF, text file, or downloadable asset
- **THEN** Mavis returns the original bytes with an appropriate content type and leaves inline-versus-download behavior to the browser

#### Scenario: Open a viewer-supported file
- **WHEN** a user navigates normally to an eligible UTF-8 text or code file, image, or PDF
- **THEN** Mavis returns the corresponding HTML viewer and provides a link to its raw representation

#### Scenario: Request an unsupported viewer format
- **WHEN** a user navigates to an eligible file for which Mavis has no built-in viewer
- **THEN** Mavis returns the raw file response with an appropriate content type

### Requirement: Enforce exclusions at every access path
Mavis SHALL prevent wildcard-excluded files and folders from appearing in listings or being retrieved through viewer pages, raw representations, direct requests, or crafted requests.

#### Scenario: Request excluded content directly
- **WHEN** a user requests either the viewer or raw URL corresponding to an excluded file or a file below an excluded folder
- **THEN** Mavis does not return the content through either representation

## ADDED Requirements

### Requirement: Prevent active content in non-Markdown viewers
Mavis SHALL escape text and code displayed in built-in viewers and SHALL preserve script-disabled sandboxing for raw active-content responses.

#### Scenario: Text file contains markup
- **WHEN** a user opens a text or code file containing HTML or script-like source in the built-in viewer
- **THEN** Mavis displays the source as literal text without creating executable document elements

#### Scenario: Open raw active content
- **WHEN** a user requests raw HTML, SVG, or JavaScript content
- **THEN** Mavis returns a sandboxed response with scripts disabled
