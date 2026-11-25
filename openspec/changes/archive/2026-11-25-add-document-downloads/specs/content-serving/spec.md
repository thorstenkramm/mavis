# Spec Delta

## MODIFIED Requirements

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
