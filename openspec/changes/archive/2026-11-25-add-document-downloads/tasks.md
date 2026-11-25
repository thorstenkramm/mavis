# Tasks

## 1. Configuration and original downloads

- [x] 1.1 Add `Downloads.ExportDOCX` (default true) and optional `Downloads.Pandoc`, with startup PATH discovery, explicit-path precedence, false override, and strict empty-value validation; verify config/server tests for PATH found/not found, explicit absolute/PATH names, invalid paths, disabled detection with a configured path, wrong types, unknown keys, and restart-only PATH changes, and document both settings in `docs/content/reference/configuration.md` with a passing `make docs`.
- [x] 1.2 Implement `?download=markdown` and `?raw=1&download=1` original-byte attachments with safe UTF-8 filenames and existing raw behavior preserved; verify content/web tests for BOM/front matter, edits, content types, inline-versus-attachment disposition, Unicode/header-special names, unknown formats, and hidden/excluded/private/outside-root boundaries.

## 2. Built-in PDF export

- [x] 2.1 Pin the evaluated `goldmark-pdf` pseudo-version and add embedded Roboto/Roboto Mono fonts, licenses, and the preloaded offline font cache; verify real PDF generation with a fail-on-request HTTP transport, inspect module/binary growth and licenses, and cross-compile all release targets with `CGO_ENABLED=0`.
- [x] 2.2 Implement front-matter-free GFM PDF conversion through the `gopdf` backend, including consistent glyph measurement/substitution and panic recovery; verify recording-backend tests for German text, headings, emphasis, lists, tables, links, code, BOM/front matter, CJK/emoji replacements, and literal HTML, plus real-output PDF signature and error-path tests.
- [x] 2.3 Implement resolver-backed relative image loading and visible placeholders for unsupported formats; verify tests for PNG/JPEG/GIF, SVG, alt text, remote URLs, traversal, hidden/excluded/private files, internal/outside symlinks, disappearing files, and size limits without unauthorized bytes or conversion-wide failure.
- [x] 2.4 Add the bounded export service and `?download=pdf` responses with safe filenames, complete-buffer validation, two workers, 10/50 MiB limits, and 30-second invalidation; verify fake-worker tests for concurrency, late results, deadline, cancellation, panic, output overflow, generic 400/413/500/503 responses, and document PDF capabilities/limitations in `docs/content/guides/downloading-documents.md` with a passing `make docs`.

## 3. Optional DOCX export

- [x] 3.1 Build self-contained sanitized article HTML with resolver-authorized data-URI images and no interface/front-matter content; verify tests for current H1/body, tables/lists/links/code, PNG/SVG, markup sanitization, and removal of remote, excluded, private, traversal, and outside-root resources.
- [x] 3.2 Implement direct `pandoc --sandbox` execution through the executable resolved once at startup, with private temporary files, minimal environment, bounded logs/output, timeout termination, and `?download=docx`; verify fake-executable tests for argument safety, no per-request PATH lookup, success, malformed DOCX, nonzero exit, timeout, cleanup, and disabled/not-found responses, run a conditional real-pandoc integration test, and document automatic detection, opt-out, and service PATH/permissions with a passing `make docs`.

## 4. Download interface and integrated verification

- [x] 4.1 Add top-right native download menus to Markdown and non-Markdown viewer toolbars with only applicable formats, accessible icons/links, and responsive 44-pixel targets; verify template/accessibility tests for keyboard/no-JavaScript behavior, DOCX visibility when PATH-found, not-found, and explicitly disabled, dropdown containment, focus, contrast, and overflow, and add menu instructions to the download guide with a passing `make docs`.
- [x] 4.2 Extend `scripts/smoke-release.go` for original attachments, offline PDF, startup PATH-based DOCX availability and opt-out, safe filenames, and access boundaries; verify the native release smoke passes, then run `make test`, `make coverage`, `make lint`, duplication/vulnerability checks, and strict validation of this change and all main specs with at least 85% coverage.
- [x] 4.3 Verify menus and downloads in current Chrome/Edge, Firefox, and WebKit at desktop/mobile sizes, open representative PDFs and DOCX files in compatible viewers to check German text/tables/code/images/placeholders, and run accessibility scans; verify no new violations or horizontal overflow and record versions, output limitations, and results in `docs/content/reference/browser-compatibility.md` before rerunning `make docs`.
