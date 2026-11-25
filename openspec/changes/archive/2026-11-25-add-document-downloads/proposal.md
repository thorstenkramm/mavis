# Proposal

## Why

Users can view documents in Mavis but cannot save them in a portable format. Mavis needs convenient original-file downloads and built-in PDF export without imposing Python, a browser, or another runtime on every installation.

## What Changes

- Add a keyboard-accessible download icon and dropdown in the top-right of the document or file viewer.
- Offer Markdown documents as original Markdown, built-in PDF, and optionally DOCX; offer other eligible files as raw downloads with attachment filenames.
- Generate simplified PDFs inside Mavis using `goldmark-pdf` with its pure-Go `gopdf` backend, embedded fonts, current Markdown content, and resolver-authorized local images.
- Replace unsupported images and glyphs visibly rather than failing the entire PDF; do not claim browser-identical layout, SVG rendering, emoji, or CJK coverage.
- Enable DOCX at startup when pandoc is found in PATH or configured explicitly, using sandboxed conversion of self-contained authorized HTML; allow `export_docx = false` to override detection and disable DOCX.
- Apply existing content visibility and root-confinement rules to every export and use safe content types, attachment headers, timeouts, output limits, and bounded concurrency.
- Extend configuration with optional `[downloads]` settings for DOCX enablement and the pandoc executable; PDF, Markdown, and raw downloads require no external tool.

## Capabilities

### New Capabilities

- `document-downloads`: Accessible download selection and secure generation of original, PDF, and DOCX representations of eligible content.

### Modified Capabilities

- `content-serving`: Add an explicit attachment representation while preserving ordinary raw responses and all existing access controls.
- `server-configuration`: Accept and validate optional DOCX enablement and pandoc path settings, with PATH detection enabled by default.

## Impact

- Add PDF/DOCX export code, authorized resource adapters, embedded font assets and licenses, and download response handling.
- Extend `internal/web`, `internal/content`, `internal/render`, `internal/config`, and server dependency injection.
- Pin `github.com/stephenafamo/goldmark-pdf` to the evaluated `gopdf`-capable commit and add its transitive Go dependencies; retain pure-Go cross-compilation.
- Add optional pandoc process execution for DOCX, but no Python, WeasyPrint, headless browser, Ghostscript, or persistent conversion service.
- Add unit, integration, security, accessibility, export-compatibility, cross-platform, and release-smoke tests plus user/configuration documentation.
