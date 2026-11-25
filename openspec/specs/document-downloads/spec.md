# document-downloads Specification

## Purpose

Provides accessible original-file, PDF, and DOCX downloads while preserving content access controls, local privacy, and Mavis's standalone deployment model.

## Requirements

### Requirement: Provide an accessible download menu
Mavis SHALL place a download icon and dropdown at the top-right of document and file-viewer pages. The menu SHALL work with a keyboard and without JavaScript and SHALL offer only formats available for the current file.

#### Scenario: Open a Markdown download menu
- **WHEN** a user activates the download control on a Markdown document
- **THEN** Mavis offers original Markdown and PDF, plus DOCX when enabled at startup

#### Scenario: Open another file's download menu
- **WHEN** a user activates the download control on an eligible non-Markdown file
- **THEN** Mavis offers a raw download without Markdown-only formats

#### Scenario: Use the menu on mobile
- **WHEN** a user opens the menu at a mobile-width viewport
- **THEN** its controls remain visible, keyboard accessible, and usable without horizontal page scrolling

### Requirement: Download current original content
Mavis SHALL return the current original bytes of eligible Markdown and other files as attachments with safe filenames and appropriate content types.

#### Scenario: Download Markdown source
- **WHEN** a user downloads a Markdown document as Markdown
- **THEN** Mavis returns its current unmodified source, including any BOM and front matter, as an attachment

#### Scenario: Download after an edit
- **WHEN** a file changes on disk and the user repeats its original-file download
- **THEN** Mavis returns the edited bytes without requiring a restart

#### Scenario: Filename needs encoding
- **WHEN** an eligible filename contains Unicode or header-special characters
- **THEN** Mavis returns a valid attachment header without response splitting or losing the UTF-8 filename

### Requirement: Generate built-in PDF downloads
Mavis SHALL generate a simplified PDF from current Markdown body content without an external executable, runtime service, font download, or network access. Front matter SHALL be excluded and common text, headings, emphasis, lists, tables, links, code, and supported local raster images SHALL be represented.

#### Scenario: Download a German document
- **WHEN** a user downloads Markdown containing German text, headings, a table, and a code block as PDF
- **THEN** Mavis returns a valid PDF attachment containing that content with embedded fonts

#### Scenario: Document has front matter
- **WHEN** a Markdown document begins with YAML front matter
- **THEN** the PDF contains the document body but not the front-matter block

#### Scenario: Mavis has no network access
- **WHEN** a user requests a PDF on an offline host with no installed converter
- **THEN** Mavis still generates the PDF without downloading fonts or invoking another program

### Requirement: Degrade unsupported PDF content visibly
PDF export SHALL substitute a visible placeholder for unsupported image formats and a visible replacement glyph for unavailable font characters rather than silently omitting them or failing the whole export. Mavis SHALL NOT claim SVG, emoji, CJK, or browser-identical rendering.

#### Scenario: Document contains SVG
- **WHEN** an eligible Markdown document references a local SVG image
- **THEN** Mavis returns the remaining PDF content with a visible image placeholder instead of failing conversion

#### Scenario: Document contains unavailable glyphs
- **WHEN** Markdown contains emoji, CJK, or another character unavailable in the embedded fonts
- **THEN** the PDF contains a visible replacement glyph and preserves the remaining text

#### Scenario: Document contains a supported raster image
- **WHEN** Markdown references an eligible local PNG, JPEG, or GIF image
- **THEN** Mavis embeds that image subject to available layout support

### Requirement: Offer optional DOCX conversion
Mavis SHALL enable DOCX at startup when pandoc is found in PATH or configured explicitly, unless `export_docx = false`. It SHALL generate DOCX from self-contained authorized HTML in a sandboxed local process without fetching remote or unauthorized resources.

#### Scenario: Pandoc is found in PATH
- **WHEN** DOCX is not disabled and startup finds pandoc in PATH
- **THEN** Mavis enables DOCX using that executable

#### Scenario: Pandoc is not found
- **WHEN** DOCX is not disabled, no executable is explicitly configured, and startup cannot find pandoc in PATH
- **THEN** Mavis starts without DOCX and leaves other downloads available

#### Scenario: DOCX is explicitly disabled
- **WHEN** configuration sets `export_docx = false`, regardless of pandoc availability or an explicit path
- **THEN** Mavis skips converter resolution, omits DOCX from the menu, and reports direct DOCX requests as unavailable

#### Scenario: Explicit executable is invalid
- **WHEN** DOCX is enabled and an explicitly configured pandoc executable cannot be resolved
- **THEN** Mavis reports an actionable startup configuration error

#### Scenario: Download DOCX
- **WHEN** DOCX is enabled and a user downloads Markdown as DOCX
- **THEN** Mavis returns a valid DOCX attachment representing the current body, headings, tables, lists, links, code, and permitted local images

#### Scenario: Document references remote or outside content
- **WHEN** Markdown references a remote image or a file outside the eligible served content
- **THEN** Mavis does not fetch or embed that resource in the DOCX

### Requirement: Enforce content boundaries for every download
Every download and embedded resource SHALL use the same eligibility, exclusion, root-confinement, symlink, and private-configuration rules as ordinary content requests. Export SHALL NOT transmit document content to an external service.

#### Scenario: Request excluded content
- **WHEN** a user requests any download format for hidden, excluded, missing, or private-configuration content
- **THEN** Mavis does not return its bytes, path, generated representation, or filesystem details

#### Scenario: Markdown references an excluded image
- **WHEN** an otherwise eligible document references an excluded or outside-root image
- **THEN** Mavis does not embed that image or disclose its bytes in PDF or DOCX output

### Requirement: Bound export work and failures
Mavis SHALL reject Markdown sources larger than 10 MiB, cap generated files at 50 MiB, allow at most two concurrent conversions, and apply a 30-second deadline. Failures SHALL return a generic error without a partial attachment, process details, or filesystem paths.

#### Scenario: Source exceeds the limit
- **WHEN** a user requests PDF or DOCX for Markdown larger than 10 MiB
- **THEN** Mavis rejects conversion before producing an attachment and returns a clear size error

#### Scenario: Conversion exceeds the deadline
- **WHEN** PDF or DOCX conversion cannot finish within 30 seconds
- **THEN** Mavis stops or invalidates the conversion and returns a generic timeout response without partial output

#### Scenario: Converter fails
- **WHEN** conversion fails, exceeds the output cap, or the external converter exits unsuccessfully
- **THEN** Mavis returns a generic conversion error and continues serving unrelated content

#### Scenario: Download format is unknown
- **WHEN** a user requests an unsupported download format or a format unavailable for the file type
- **THEN** Mavis returns a clear client error without generating a file
