# Design

## Context

See `proposal.md` for motivation and the three delta specs for behavior contracts. The current web handler sends non-Markdown requests directly to the raw file handler, while its page template owns navigation only for Markdown and directory pages. Navigation currently contains the active ancestry only. The renderer returns HTML and TOC data but no document metadata, and front matter is parsed as ordinary Markdown.

The implementation must retain the existing resolver as the single content-authorization boundary, keep raw active content non-executable, avoid client-side routing, and preserve the mobile drawer and ordinary document URLs.

## Goals / Non-Goals

**Goals:**

- Reuse the existing server-rendered page and resolver for Markdown, directory, viewer, and raw responses.
- Keep embedded assets, explicit raw links, and PDF embedding on a consistent raw URL contract.
- Scale folder expansion without recursively reading the entire served tree for every page.
- Use native HTML and small embedded JavaScript controls with keyboard and mobile behavior equivalent to pointer behavior.
- Avoid new runtime services and external browser assets.

**Non-Goals:**

- YAML semantic validation or using front-matter fields to override document titles.
- Syntax highlighting, WYSIWYG editing, or execution of viewed code.
- Arbitrary HTML preview or relaxing raw HTML/SVG/JavaScript sandboxing.
- A client-side routing framework, automatic browser reload, or a fully eager recursive directory tree.
- Persisting browser preferences on the server or sharing them between browsers.

## Decisions

### 1. Use `?raw=1` on the existing content URL for original bytes

Normal navigation to a viewer-supported file returns the HTML page; the same escaped path with the `raw=1` query parameter returns the original file response. Other query parameters and fragments are preserved when constructing raw asset URLs. Unsupported viewer formats continue directly to the raw handler.

Centralize raw URL construction so web viewer links and Markdown image rewriting cannot diverge. Ordinary links to non-Markdown files remain viewer links; only embedded local images and explicit raw actions select the raw representation.

**Alternatives considered:** Separate raw path prefixes would require additional route and link-mapping rules and would make rollback less compatible. Static file URLs outside the resolver would weaken the central authorization boundary.

### 2. Classify viewers from resolved content and keep their models separate

The web handler will classify a resolved regular file as Markdown, text/code, image, PDF, or raw-only. MIME detection will combine the file extension with a content sniff bounded to the first 512 bytes, so classification never loads a whole file. Text/code viewing additionally requires valid UTF-8 and is the only case that reads the complete source. Images will use an explicit browser-safe set, including SVG only as an image source; PDFs require a PDF content type.

A viewer page model will contain the viewer kind, display name, escaped text where applicable, and raw URL. The template will render text in `<pre><code>`, images in `<img>` with useful alternative text, and PDFs in a titled `<iframe>`. No TOC is shown for non-Markdown viewers.

**Alternatives considered:** Extension-only routing misclassifies extensionless or misleading files. Browser sniffing of an HTML wrapper is not a substitute for server-side viewer selection.

### 3. Extract front matter and the first H1 during Markdown processing

Before Goldmark parsing, recognize only an opening `---` line at the start of the source, after an optional UTF-8 byte order mark, and a closing `---` or `...` line, accepting LF and CRLF input. Return the enclosed YAML source separately and parse only the remaining Markdown. An unterminated opening delimiter remains ordinary Markdown.

Extend the render result with plain-text document title and front-matter source fields. While walking the parsed AST for the TOC, capture only the first level-one heading as the title. The web handler retains its current title when that field is empty.

Render front matter outside sanitized Markdown HTML as a native `<details>` panel with a concise `<summary>` and a template-escaped `<pre><code class="language-yaml">` body. No YAML parser dependency is needed because Mavis displays rather than interprets the metadata.

**Alternatives considered:** A YAML parser would add validation and dependencies without changing the requested display behavior. A Goldmark extension could parse metadata, but explicit source separation more directly guarantees that front matter cannot enter heading IDs or the TOC.

### 4. Add a same-origin lazy navigation endpoint

Keep server-rendered navigation for the root and active ancestry. A folder expansion button will request `GET /_mavis/navigation?path=<root-relative-path>` when its children are not already loaded. The endpoint will use the same resolver and return only eligible entry names, relative paths, content URLs, and directory flags as JSON with `Cache-Control: no-store`; it will never return absolute filesystem paths.

JavaScript will build entries with DOM APIs and `textContent`, set `aria-expanded`, and hide or reveal the associated group. Folder names remain ordinary links, so directory navigation and browser history do not depend on the expansion API. Initial page rendering and active-ancestry navigation remain usable without the API.

**Alternatives considered:** Rendering every descendant eagerly can make large trees slow and lets one unreadable distant directory affect unrelated pages. Client-side routing would add unnecessary history and focus complexity.

### 5. Store folder and width preferences locally and provide accessible controls

Store expanded relative folder paths and desktop width in browser `localStorage`, with in-memory fallback if storage is unavailable. Current-document ancestry is always expanded, even if a stored state says otherwise, and folders revealed only as ancestry are not added to the stored state. Storage contains no file contents or absolute root path.

Use distinct embedded inline SVG icons for folders and common file categories, marked decorative so accessible names come from visible text and controls. Folder buttons will expose expanded state, and arrows will visually match that state.

Desktop layout will use a CSS width variable and a focusable vertical `role="separator"` control. Pointer capture will handle dragging; Left/Right arrows will resize by a fixed increment. Width will be clamped to a usable range and exposed through `aria-valuemin`, `aria-valuemax`, and `aria-valuenow`. In narrow desktop windows the effective maximum drops to leave the reading area at least 480 pixels, without overwriting the stored preference. The separator is hidden on mobile, where the existing drawer sizing remains authoritative.

Navigation rows will use `white-space: nowrap`; nested lists will size to their content and the sidebar will own horizontal scrolling. The reading area and body will retain their overflow guards.

**Alternatives considered:** Server sessions would add state and lifecycle complexity for cosmetic preferences. Pointer-only resizing would conflict with the accessibility baseline.

### 6. Make narrowly scoped CSP adjustments

For application pages, change `connect-src` from `'none'` to `'self'` for lazy navigation and add `frame-src 'self'` for PDF viewers. Keep `object-src 'none'`, script and style restrictions, and existing external-image behavior.

Keep `frame-ancestors 'none'` and script-disabled sandboxing on raw HTML, SVG, JavaScript, and other active content. Permit `frame-ancestors 'self'`, with a matching `X-Frame-Options: SAMEORIGIN`, only on raw PDF responses so the application can embed the browser's PDF display. Raw responses retain `nosniff` and their existing content types.

**Alternatives considered:** Permitting arbitrary frames or disabling the raw sandbox would expose the application to active document content. Using `<object>` would require relaxing `object-src` and offers no advantage over the iframe.

### 7. Verify behavior at unit, HTTP, browser, and release levels

Add focused renderer tests for front matter, titles, and image URLs; resolver/navigation tests for lazy entry eligibility; and web integration tests for every viewer, raw representation, CSP, and error path. Extend release smoke fixtures to assert both a text viewer and its raw response, plus H1 title and front-matter behavior.

Exercise folder expansion, persistence, keyboard resizing, pointer resizing, long-name overflow, front-matter disclosure, text safety, images, and PDFs in the supported Chrome/Edge, Firefox, and WebKit browser matrix. Retain the existing mobile drawer and accessibility checks, then run `run-all-tests.sh` and workflow validation.

## Risks / Trade-offs

- **[Lazy navigation adds a same-origin API]** → Keep it read-only, non-revealing, resolver-backed, and covered by direct and crafted-request tests.
- **[A raw PDF must be frameable by the application]** → Relax `frame-ancestors` only for PDF and only to `'self'`; retain script-disabled sandboxing and test the response policy.
- **[Very large text files can increase per-request memory use]** → Read each source once, avoid syntax-highlighting copies, and keep raw access available; introduce a specified size limit separately rather than silently truncating content.
- **[Stored expansion paths may refer to deleted or excluded folders]** → Ignore unavailable entries on demand and always rebuild current ancestry from the current resolver results.
- **[Browser PDF behavior varies]** → Use the native iframe viewer, retain a raw-file action, and verify all supported engines instead of implementing a PDF renderer.
- **[JavaScript-disabled clients cannot lazily expand arbitrary folders]** → Preserve ordinary folder links and server-rendered active-ancestry navigation as the fallback.

## Migration Plan

1. Introduce render metadata and raw URL helpers, then update Markdown link rewriting and tests.
2. Add viewer and navigation endpoint routes while retaining the raw handler and resolver as the response boundary.
3. Update templates, styles, and browser controls, followed by user and security documentation.
4. Update the standalone release smoke checks and run the complete local and CI verification suites.

No configuration or persistent server data requires migration. Explicit `?raw=1` links remain raw responses if rolled back to the previous version because the former non-Markdown route already ignored query parameters. Rolling back the application removes viewers and browser preferences but leaves served files unchanged.
