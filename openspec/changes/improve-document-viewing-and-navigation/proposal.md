# Proposal

## Why

Mavis currently loses its navigation layout when users open non-Markdown files, treats front matter as document content, and provides only a fixed-width, partially expanded file browser. These limitations make mixed-format documentation harder to read and navigate than Markdown-only content.

## What Changes

- Use the plain text of a Markdown document's first H1 as the browser and top-bar page title; retain the existing filename or directory title when no H1 exists.
- Recognize a leading delimited YAML front-matter block separately from Markdown. Exclude it from the rendered body and TOC, and present its source in a collapsed, expandable, monospaced panel above the document.
- Add built-in viewers for eligible UTF-8 text and code files, images, and PDFs so these formats open in the content area while file navigation remains available. Text and code remain escaped and monospaced; syntax highlighting is not added.
- Provide explicit access to each viewed file's original bytes and preserve appropriate raw content types, content boundaries, and active-content protections. Embedded Markdown images must continue to load as images rather than viewer pages.
- **BREAKING:** Normal navigation to a viewer-supported non-Markdown file will return its HTML viewer instead of the raw file response; the raw representation will use an explicit raw-content URL.
- Add file-type and folder icons, independent folder expand/collapse controls with state indicators, and nested browsing beyond the active ancestry while continuing to identify the current document.
- Allow desktop users to resize the file browser within usable bounds. Keep every filename on one line and place overflow in a horizontally scrollable navigation pane rather than wrapping names or scrolling the whole page.
- Preserve keyboard accessibility, responsive mobile navigation, ordinary browser history, and the existing manual-refresh behavior.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `content-serving`: Distinguish viewer and raw representations of eligible non-Markdown files while preserving content types, authorization, privacy, and failure isolation.
- `documentation-browser`: Add mixed-format viewers, H1-derived page titles, expandable iconic folder navigation, desktop resizing, and single-line filename overflow behavior.
- `markdown-rendering`: Separate leading YAML front matter from Markdown and heading extraction, expose its source for display, and keep embedded assets compatible with raw-content URLs.

## Impact

- Affected code: content navigation models, Markdown render results and link rewriting, web handlers and page models, HTML templates, CSS, JavaScript, and associated unit and integration tests.
- Affected behavior: page titles, front-matter presentation, non-Markdown response representations, raw-content links, folder-tree state, and desktop layout controls.
- Security: raw HTML and other active content must remain non-executable; text must be escaped; viewer embedding must use narrowly scoped CSP changes; all viewer and raw routes must share existing content authorization.
- Documentation: update user guides and Markdown, security, and browser-compatibility references for viewers, front matter, and navigation controls.
