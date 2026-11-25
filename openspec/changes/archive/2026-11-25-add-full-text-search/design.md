# Design

## Context

See `proposal.md` for motivation and `specs/content-search/spec.md` for the behavior contract.

Mavis currently resolves every content request through `content.Resolver`, which owns visibility, exclusions, exact spelling, symlink confinement, and `.mavis.conf` privacy. The web layer has reserved internal routes for assets and lazy navigation, a shared server-rendered page template, and a `form-action 'none'` CSP. There is no content index or filesystem watcher.

Measurements on the target ISMS tree found 400 text files totaling 6.1 MiB. A warm substring scan took about 27 ms per query. A lean in-memory Bleve index took about 0.5 seconds to build and retained about 40 MiB. Bleve whole-token matching found `sicherheit` in 55 files, versus 150 for substring matching; prefix matching still missed compounds where the query occurs inside a token.

## Goals / Non-Goals

**Goals:**

- Reuse the resolver as the single access-policy boundary for traversal and result URLs.
- Provide predictable substring and phrase search with bounded memory, output, concurrency, and latency.
- Keep search usable without JavaScript and preserve existing desktop and mobile navigation behavior.
- Return useful snippets without introducing active content or filesystem diagnostics.

**Non-Goals:**

- Stemming, typo tolerance, relevance scores, Boolean operators, or independent term matching.
- Accent folding, multi-character case folding, or language-specific compound splitting.
- Full-text extraction from PDF, DOCX, images, or other binary formats.
- Persistent or in-memory indexes, watchers, refresh signals, background jobs, and external search services.
- Filename-only matches, pagination, highlighted matches in the opened document, or rendering Markdown before matching.

## Decisions

### Scan current files through a content-layer search API

Add `internal/content/search.go` with a context-aware resolver search method and typed results containing the display filename, logical path, snippet segments, truncation state, and incomplete state. The web layer supplies limits and translates failures to HTTP responses; it does not walk the filesystem itself.

Traverse from the root through `Resolver.List` so every logical entry receives the same policy checks as navigation and direct access. Use each resource's resolved `TargetPath` to track visited directories, preventing internal symlink cycles and duplicate recursive scans. Files are read from their already-resolved paths, one at a time. Result links use `content.URLFor`, preserving escaping for spaces, percent signs, and Unicode names.

**Alternative considered:** Bleve in memory. It improves query latency but adds roughly 8 MiB to release binaries and about 50 modules, requires refresh machinery, uses more memory, and its token queries miss the German compound matches required here. Prefix queries recover tokens beginning with a query but not tokens containing it later. N-gram indexing would approach substring behavior at still greater index cost.

### Treat the trimmed query as one literal substring

Validate that the decoded query is UTF-8, trim surrounding Unicode whitespace, and count Unicode characters rather than bytes. Empty input renders the prompt without calling the content search. Non-empty input is matched against raw file source, including Markdown punctuation and front matter; internal spaces remain significant, so a multi-word query behaves as a phrase.

Use Go's Unicode lowercase mapping on query and source. Matching and snippet offsets operate on rune slices rather than byte offsets from a lowered string, because UTF-8 byte lengths can change under case mapping. This provides simple one-rune case mapping without locale-specific or multi-character folds: for example, umlauts remain significant and `ß` is not treated as `ss`.

**Alternative considered:** splitting on whitespace and requiring every term. That is common search-engine behavior, but it makes phrase expectations ambiguous and needs a separate phrase syntax. A single literal substring is deterministic and matched the requested compound behavior in the measurements.

### Detect text candidates and read each with a hard cap

Inspect at most the leading 512 bytes using the same content-type detection approach as the viewer, supplemented by recognized text extensions. Obvious images, PDFs, and other binary formats are outside search scope and skipped silently. Read a text candidate through `io.LimitReader` at 10 MiB plus one byte, then reject oversized or invalid UTF-8 content. Only one capped source and its rune/lowered representations are live per worker.

This deliberately searches extensionless UTF-8 documentation as well as known Markdown, text, and code extensions. It does not render Markdown merely to search it, avoiding repeated rendering cost and differences between raw-source and visible-text queries.

### Keep result memory and snippet output bounded

Record at most the 101 lexicographically smallest matches, then return the first 100 and use the extra entry only to set the truncation notice. Sort by logical slash-separated path, not filesystem-specific absolute paths. Store one snippet per file rather than every occurrence.

Build a Google-like excerpt around the first match with up to eight whitespace-delimited words before it and twelve after it. Preserve the exact source text and highlight every query occurrence inside that excerpt. Add ellipses where source is omitted, and cap the excerpt at 240 runes with rune-safe truncation so an unusually long token cannot create an unbounded result.

Represent the excerpt as text/match segments; the template escapes each segment and wraps only match segments in `<mark>`. Newlines remain source whitespace and collapse naturally in HTML. No source string becomes `template.HTML`. The filename is derived with `path.Base` for the prominent link, while the complete logical path appears as secondary text to distinguish equal filenames.

### Add a reserved server-rendered route and full-width top search bar

Register `GET /_mavis/search` and `HEAD /_mavis/search` beside the existing internal routes. Other methods return 405 with `Allow: GET, HEAD`. The reserved path takes precedence over same-named user content. Search responses use `Cache-Control: no-store` and existing security headers.

Place a dedicated, viewport-wide search region immediately after the skip link and before the existing top bar and two-pane layout. Its native GET form has a persistent visible label, `type="search"`, and submit button; neither the form nor any search output is a descendant of the file-browser navigation. The search field grows within a readable maximum width and wraps cleanly on narrow screens.

Render results only in the ordinary main article, using the filename as the large linked heading, the root-relative path as secondary text, and the contextual excerpt below. Root file navigation remains in its separate sidebar. Preserve desktop navigation and mobile focus trapping by treating the new search region as background content while the drawer is open. Add responsive styles for the top search region, filename links, paths, excerpts, notices, and `<mark>` with WCAG AA contrast and contained overflow. Google-like refers to this presentation only; results retain deterministic path ordering rather than relevance ranking.

Change the HTML-page CSP to `form-action 'self'` so the native form can submit only to Mavis. Raw file responses retain `form-action 'none'`; no inline script or new external connection is introduced.

**Alternative considered:** a JavaScript-only search box and JSON endpoint. That would avoid the form CSP change but unnecessarily make a basic server-side capability depend on JavaScript.

### Apply one deadline and bounded concurrency to the whole search

Use a 10-second context deadline covering semaphore wait, traversal, reads, and matching. Allow two concurrent searches per handler; additional searches wait only until their own deadline or an earlier slot. Check the context between directory levels and files and after each capped read, since ordinary filesystem reads are not themselves cancellable.

Return 400 for invalid or oversized queries, 503 for deadline expiry, and 500 for a root traversal failure, all with generic messages. Client cancellation stops work without attempting a detailed response. Per-file and descendant-directory failures set the incomplete flag, are logged without being copied into the response, and do not discard matches from healthy content. Timeout returns the timeout response rather than partial results.

## Risks / Trade-offs

- **Large trees make every query perform filesystem I/O** → The 10 MiB file cap, two-worker limit, 10-second deadline, and request cancellation bound impact; current content remains the priority over indexed speed.
- **Cold-cache searches can exceed the target latency** → Return a generic timeout instead of leaving a request running indefinitely; do not claim a fixed sub-second guarantee.
- **Simple lowercasing is not full linguistic search** → Document exact substring, phrase, case, and diacritic behavior; do not imply stemming or transliteration.
- **Raw-source snippets can contain Markdown or YAML syntax** → This is intentional and keeps matching consistent with file contents; escape all output.
- **A per-file read may finish just after cancellation** → The cap bounds the delay and memory; check context immediately afterward before further work.
- **Symlink aliases are searched once per resolved directory** → This prevents cycles and duplicate work; result paths remain valid logical aliases within the served root.
- **Allowing same-origin forms slightly widens the HTML CSP** → Restrict it to `'self'`, preserve the stricter raw-content policy, and add a regression assertion.

## Migration Plan

1. Add the content search implementation and tests without changing existing content URLs.
2. Add the reserved route, full-width top search bar, filename-led result template, styles, and same-origin form CSP.
3. Update user documentation with query semantics, binary exclusions, limits, and the reserved path.
4. Run race tests, coverage at the existing 85% minimum, lint/security checks, release smoke tests, and desktop/mobile browser checks.
5. Deploy with the normal binary release. No configuration, data migration, index directory, or startup work is required; rollback is a binary rollback, with the reserved search route disappearing.
