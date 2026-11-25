# Tasks

## 1. Markdown metadata and asset URLs

- [x] 1.1 Implement leading YAML front-matter separation for LF and CRLF sources, closing `---` and `...` delimiters, and unterminated-delimiter fallback; add renderer tests proving front matter stays out of the body, heading IDs, and TOC, then run `go test ./internal/render`.
- [x] 1.2 Extend render results with the first H1's plain-text title and apply it to browser and top-bar titles with the existing fallback; test formatted headings, multiple H1s, no-H1 documents, and directory READMEs with `go test ./internal/render ./internal/web`.
- [x] 1.3 Add centralized raw-content URL construction and use it for eligible relative and root-relative embedded images while preserving ordinary viewer links, queries, fragments, and authorization; verify with `go test ./internal/render ./internal/content`.
- [x] 1.4 Render escaped front matter in a collapsed native disclosure panel with monospaced YAML and no panel when absent; add web tests and update `docs/content/reference/markdown.md`, then run `go test ./internal/web` and `make docs`.

## 2. Mixed-format viewers and raw responses

- [x] 2.1 Implement extension-plus-content viewer classification, UTF-8 validation, and `?raw=1` dispatch for text/code, images, PDFs, and raw-only files; test normal and raw representations, unsupported formats, and misleading extensions with `go test ./internal/web ./internal/content`.
- [x] 2.2 Add the escaped monospaced text/code viewer inside the shared page layout with navigation and a raw-file action; test markup-like source, current navigation, read failures, and refresh behavior with `go test ./internal/web`.
- [x] 2.3 Add image and PDF viewer templates with accessible names, raw-file actions, and same-origin frame support; test both desktop viewer layouts and the narrowly changed application CSP with `go test ./internal/web`.
- [x] 2.4 Restrict `frame-ancestors 'self'` to raw PDFs while retaining sandboxing and `frame-ancestors 'none'` for raw active content; add viewer/raw exclusion tests and update the serving and security guides, then run `go test ./internal/web ./internal/content` and `make docs`.

## 3. Expandable iconic file browser

- [x] 3.1 Add the read-only lazy navigation JSON endpoint using decoded root-relative paths and resolver-backed listings; test eligible entries, exclusions, hidden files, traversal, unsupported methods, no-store responses, and absence of absolute paths with `go test ./internal/content ./internal/web`.
- [x] 3.2 Render server-side folder expansion buttons, decorative file/folder icons, state arrows, active ancestry, and current-document state; verify accessible names, `aria-expanded`, ordinary folder links, and no-JavaScript fallback with `go test ./internal/web`.
- [x] 3.3 Implement lazy child loading with `textContent`, independent expand/collapse behavior, localStorage restoration with in-memory fallback, and forced current-ancestry expansion; verify pointer and keyboard operation, persistence across navigation/refresh, and stale-path handling in browser tests.
- [x] 3.4 Document folder icons, expansion controls, keyboard behavior, persistence, and the non-JavaScript fallback in the serving and browser guides; run `make docs` and check the rendered guidance.

## 4. Resizable and overflow-safe navigation

- [x] 4.1 Implement the CSS sidebar width variable, single-line file rows, content-sized nested lists, and sidebar-only horizontal scrolling; extend responsive tests and verify long names at desktop and mobile widths without body overflow using `go test ./internal/web` and browser viewport checks.
- [x] 4.2 Add the desktop ARIA separator with bounded pointer dragging, keyboard increments, accessible values, and localStorage restoration; verify drag, arrow keys, limits, navigation/refresh persistence, and in-memory fallback in browser tests.
- [x] 4.3 Preserve the existing mobile drawer, focus trap, Escape behavior, contrast, and reduced-motion behavior while hiding desktop resizing on narrow screens; update `docs/content/reference/browser-compatibility.md`, then run accessibility tests and `make docs`.

## 5. Release and system verification

- [x] 5.1 Extend `scripts/smoke-release.go` with H1 title, front matter, text viewer, raw response, and content-boundary assertions; rebuild with `make build` and verify with `go run -race scripts/smoke-release.go -binary ./mavis`.
- [x] 5.2 Run the supported Chrome/Edge, Firefox, and WebKit browser checks for Markdown, front matter, all viewers, folder controls, resizing, persistence, long names, mobile navigation, and keyboard accessibility; record versions and results in `docs/content/reference/browser-compatibility.md` and verify with `make docs`.
- [x] 5.3 Run `./run-all-tests.sh`, GitHub workflow lint, and `openspec validate improve-document-viewing-and-navigation --strict`; confirm at least 85% coverage and resolve every failure without suppressing security checks.
