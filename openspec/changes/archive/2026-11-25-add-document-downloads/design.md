# Design

## Context

See `proposal.md` for motivation and the three files under `specs/` for the behavior contract.

Mavis currently serves original non-Markdown bytes through `?raw=1`, renders Markdown with goldmark/GFM, and authorizes every logical path through `content.Resolver`. Configuration rejects unknown keys. There is no conversion worker, attachment response, or export resource adapter.

An evaluation of `goldmark-pdf` at commit `76079c6194277ecd318924223db61477cc507c71` with `gopdf` produced good basic German documents without cgo. Tables, task lists, code highlighting, links, and PNG images worked. SVG returned `image: unknown format`, emoji/CJK disappeared with the default font, and the default font cache performed network requests. The latest tagged release, v0.4.2, predates the required backend fixes.

## Goals / Non-Goals

**Goals:**

- Keep PDF generation in-process, offline, deterministic, and available on every supported platform.
- Reuse Mavis's Markdown parsing and resolver policy rather than giving converters unrestricted filesystem access.
- Return complete attachments only after successful bounded conversion.
- Preserve a no-JavaScript download interface and existing viewer/raw behavior.
- Make the optional DOCX process boundary explicit and testable.

**Non-Goals:**

- Browser-identical PDF layout, SVG rasterization, color emoji, CJK font packs, or full Unicode coverage.
- PDF text extraction, PDF/A validation, editable forms, or conversions between arbitrary file formats.
- A built-in DOCX writer or converter command-line arguments in configuration.
- WeasyPrint, Python, headless Chrome/Edge, Ghostscript, LaTeX, or persistent conversion caches/services.

## Decisions

### Use query-selected representations at existing content URLs

Keep each document's canonical URL unchanged and select downloads with query parameters:

- `?download=markdown` for original Markdown source;
- `?download=pdf` for generated PDF;
- `?download=docx` for generated DOCX;
- `?raw=1&download=1` for original non-Markdown attachments.

The web handler intercepts Markdown download values before normal rendering. The content file handler adds attachment disposition only when `download=1`; plain `?raw=1` retains current inline behavior and security headers. Unknown or inapplicable formats return 400, disabled or undiscovered DOCX returns 404, oversized sources return 413, deadline expiry returns 503, and other conversion failures return 500.

Use `mime.FormatMediaType` to construct `Content-Disposition` with the basename only, an ASCII fallback, and UTF-8 extended filename encoding. Generated formats replace the Markdown extension with `.pdf` or `.docx`. Set explicit content types, `X-Content-Type-Options: nosniff`, `Cache-Control: no-store`, and length only after the complete output is available.

### Add a native top-right download menu

Add a document toolbar for Markdown and extend the existing non-Markdown viewer toolbar. Align a `<details>`/`<summary>` download menu to the right, with an SVG download icon, accessible name, and ordinary links for formats. Directory listings do not receive a menu.

Native disclosure behavior supplies keyboard and no-JavaScript support. CSS positions the dropdown within the viewport, provides visible focus and WCAG AA contrast, and keeps mobile targets at least 44 by 44 pixels. DOCX links are omitted entirely when DOCX is disabled or no converter was discovered. Existing **View raw file** links remain independent of attachment downloads.

### Isolate export orchestration from HTTP

Add an internal export service with small PDF and DOCX interfaces injected into the web handler. It owns the shared two-slot semaphore, 30-second contexts, 10 MiB source reads, 50 MiB limited writers, panic-to-error conversion, and generic error categories. HTTP code resolves content and writes responses but does not invoke goldmark-pdf or pandoc directly.

Read source once per conversion and strip front matter through a shared render helper before either generated format. Original Markdown downloads bypass this helper and retain every byte. Conversion writes into a bounded buffer or private temporary file; no bytes reach the response until generation and format validation succeed. Validate a PDF signature and DOCX ZIP/container structure before download.

For PDF, run conversion on a bounded worker goroutine and select between its result and the request deadline. A timed-out result is discarded. The occupied worker slot remains held until an uncancellable library call actually returns, preventing overdue work from exceeding the two-worker bound. DOCX uses `exec.CommandContext`, which terminates its process on deadline.

### Pin goldmark-pdf and use only its gopdf backend

Depend on the evaluated pseudo-version for commit `76079c619427`, not v0.4.2 and not a floating branch. Construct goldmark with GFM and `goldmark-pdf` as its renderer, explicitly supplying `gopdf.New` through `pdf.WithPDF`. Do not import or configure the `fpdf` backend. Recover library panics at the export boundary and record detailed diagnostics only in server logs.

Disable HTML-style escaping for PDF text so literal angle brackets appear correctly; Markdown HTML never becomes active browser content in this path. Retain the library's simplified layout rather than attempting to duplicate Mavis's HTML/CSS. The first H1 and filename supply document metadata where available.

**Alternatives considered:** WeasyPrint and Chrome provide higher fidelity but violate the standalone-runtime goal. Direct `gopdf` is leaner but requires owning the entire Markdown layout engine. `tdewolff/canvas` has a much larger compiled dependency surface. Ghostscript does not consume Markdown or HTML directly.

### Embed fonts and prevent all runtime font requests

Ship Roboto regular, italic, bold, and bold-italic plus Roboto Mono, together with their license notices, as embedded assets. Prepopulate the `go-swiss/fonts.Cache` with the exact Google-font URL keys used by the pinned dependency and these embedded bytes. This satisfies both `gopdf.New`'s unconditional Roboto preload and the renderer's later style-font loads without network access. Cache writes are unnecessary.

Add an offline regression test with an HTTP transport that fails on every request. Cross-compile with `CGO_ENABLED=0` on the existing release matrix and audit binary size and module growth during implementation.

Wrap the PDF text interface so measuring and drawing apply identical glyph normalization. Inspect embedded font coverage and replace each unsupported rune with U+FFFD when covered, otherwise `?`. This prevents silent CJK/emoji loss and keeps text widths consistent with rendered output.

### Authorize PDF images through a resolver-backed filesystem

Implement the `http.FileSystem` expected by `goldmark-pdf` over logical paths relative to the current Markdown document. Reject schemes, absolute paths, backslashes, traversal, remote URLs, excluded content, and outside-root symlinks through resolver checks. Permit only current regular files with size caps.

PNG, JPEG, and GIF pass through to `gopdf`. For SVG and other eligible but unsupported image formats, return a fixed embedded PNG placeholder reading **Preview unavailable** rather than passing undecodable bytes. The renderer's image alternative text remains available below the placeholder. Unauthorized resources return an error that becomes a warning and omitted image, not a failed document or leaked file.

### Resolve DOCX availability once at startup

Add `Downloads.ExportDOCX` and `Downloads.Pandoc` to effective configuration, with `ExportDOCX` defaulting to true and no explicit executable path. A nullable file-configuration boolean distinguishes an explicit false override from the default.

Apply this startup sequence before constructing the web handler:

1. When `export_docx = false`, skip executable resolution and construct no DOCX converter, even if `downloads.pandoc` is present.
2. When a non-empty `downloads.pandoc` is configured and DOCX is enabled, resolve that exact name or path. Failure is an actionable startup error rather than a silent fallback to PATH.
3. Otherwise, call `exec.LookPath("pandoc")` once. A hit enables DOCX with the resolved path; a miss leaves DOCX disabled and does not prevent startup.

Cache the resolved executable for the serving session. Do not repeat PATH lookup per request, so installing/removing pandoc or changing PATH takes effect only after restart. Empty explicit paths and wrong TOML types remain strict configuration errors.

### Convert DOCX from self-contained sanitized HTML

Reuse Mavis's normal renderer to obtain sanitized article HTML and its first-H1 title. Do not include navigation, search, front-matter panels, TOC, or other interface elements. Resolve eligible local images through the same resource policy and inline them as data URIs, including SVG where pandoc supports it. Remove remote and unauthorized sources before conversion.

Write the self-contained HTML to a private temporary directory and invoke the configured executable directly, without a shell or user-supplied arguments:

```text
pandoc --sandbox -f html -t docx -o <private-output> <private-input>
```

Use a minimal environment, bounded stdout/stderr logs, a 30-second command context, and output stat/read caps. Remove the temporary directory on every path. Invoke the executable resolved during startup rather than performing another lookup. The optional `[downloads].pandoc = "/path/to/pandoc"` overrides PATH discovery; `[downloads].export_docx = false` takes precedence over both discovery and an explicit path.

## Risks / Trade-offs

- **The required goldmark-pdf fixes are untagged and upstream calls gopdf buggy** → Pin the evaluated commit, run a broad fixture suite, recover panics, and make dependency upgrade a deliberate reviewed operation.
- **Simplified PDF output may disappoint users expecting browser fidelity** → Label PDF limitations in documentation and never substitute remote conversion silently.
- **A PDF library call may not honor cancellation during CPU layout** → Bound workers and input/output, invalidate late results, and retain the worker slot until return; only DOCX can be killed at the process boundary.
- **Embedded fonts increase binary size and cover only selected scripts** → Ship a small explicit font set with licenses and visible replacement glyphs rather than runtime downloads.
- **Image placeholders change document meaning** → Use a visible placeholder and preserve alt text; do not pretend the original SVG was rendered.
- **Resolver checks and subsequent file opens have filesystem race windows** → Use resolved targets and regular-file/size checks as elsewhere in Mavis, fail closed on errors, and add symlink/traversal regression tests.
- **Pandoc remains a large optional installation** → It is never required for PDF or original downloads and is absent from service requirements unless DOCX is enabled.

## Migration Plan

1. Add configuration and original attachment behavior with tests; existing URLs remain compatible.
2. Add the pinned PDF dependency, embedded fonts/licenses, offline font cache, export service, and PDF fixtures.
3. Add optional pandoc DOCX conversion and authorized HTML resource inlining.
4. Add the native download menu, responsive styling, user/configuration documentation, and release smoke assertions.
5. Run race tests, 85% minimum coverage, lint/duplication/vulnerability checks, cross-platform builds, PDF/DOCX compatibility checks, and desktop/mobile browser verification.
6. Release normally. No data migration or persistent state is required; existing configurations work unchanged and rollback only removes the new menu/formats and dependency.
