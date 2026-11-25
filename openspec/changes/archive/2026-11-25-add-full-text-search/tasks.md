# Tasks

## 1. Content search engine

- [x] 1.1 Implement context-aware recursive search through `content.Resolver`, including visited-target symlink-cycle protection and bounded lexicographic result collection; verify with `internal/content` tests for hidden/excluded/private content, outside-root links, internal aliases, cycles, special path spelling, deterministic order, and the 100-result truncation notice.
- [x] 1.2 Implement trimmed literal-substring matching and rune-safe, word-based contextual text/match segments; verify with table-driven tests for German compounds, Unicode case and byte-length changes, multi-word phrases, raw Markdown/front matter, filename-only non-matches, up to eight words before and twelve after the first match, ellipses, the 240-rune excerpt cap, and repeated highlighted occurrences.
- [x] 1.3 Implement leading-sample text detection, silent binary exclusion, 10 MiB reads, current-file scanning, and per-file/directory failure isolation; verify with tests for images/PDFs, extensionless text, invalid UTF-8, oversized and disappearing files, edits/deletions between searches, incomplete notices, context cancellation, and deadline expiry.

## 2. Search HTTP interface and documentation

- [x] 2.1 Add the reserved `/_mavis/search` GET/HEAD route, query decoding and validation, 10-second deadline, two-search concurrency bound, and generic 400/500/503 responses; verify handler tests for methods, empty input, invalid UTF-8, 200-character boundaries, timeout/cancellation, concurrency wait, no-store headers, `form-action 'self'`, unchanged raw-file CSP, and shadowing same-named content.
- [x] 2.2 Add a full-width native GET search bar above the existing header and layout, plus main-content results with prominent filename links, secondary root-relative paths, and escaped `<mark>` excerpts; verify template and handler tests that search input/output are outside the file browser, equal filenames are distinguished, context surrounds matches, special-character URLs and XSS-like source are escaped, and retained queries, states, keyboard semantics, overflow containment, contrast, and no-JavaScript behavior work.
- [x] 2.3 Document the full-width top search bar, filename-led contextual results, literal phrase/substring semantics, simple case and diacritic behavior, raw-source/front-matter matching, binary exclusions, limits, timeout behavior, and the reserved route in `docs/content/guides/searching-documentation.md` and the security guide as applicable; verify the documented examples against tests and run `make docs` without warnings.

## 3. Integrated verification

- [x] 3.1 Extend `scripts/smoke-release.go` with compound search, top-bar placement outside the file browser, filename/path results, safe contextual excerpts, current edits, and hidden/excluded/private boundaries; verify the release smoke test passes against a freshly built native binary.
- [x] 3.2 Run `make test`, `make coverage`, `make lint`, duplication and vulnerability checks, and strict validation of `add-full-text-search` plus all main specs; verify all checks pass and total coverage remains at least 85%.
- [x] 3.3 Verify the full-width top search bar and filename-led contextual results in current Chrome/Edge, Firefox, and WebKit at desktop and mobile widths, including keyboard-only use, JavaScript disabled, separation from the file browser, drawer focus trapping with the search region inert, target sizes, no horizontal overflow, and automated accessibility scans; verify zero new violations and record versions and results in `docs/content/reference/browser-compatibility.md` before rerunning `make docs`.
