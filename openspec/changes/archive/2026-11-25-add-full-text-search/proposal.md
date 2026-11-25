# Proposal

## Why

Users cannot currently locate documentation by its contents, which makes larger documentation trees difficult to navigate. Mavis needs local full-text search that finds German compounds and other partial-word matches without introducing a stale index, significant memory overhead, or a large dependency tree.

## What Changes

- Add an accessible, full-width search bar above the existing header, file browser, and content, with server-rendered results outside the file browser.
- **BREAKING**: Reserve `/_mavis/search` for search, so a content file previously available at that exact URL is shadowed like other internal Mavis routes.
- Search current eligible Markdown, text, and code files using case-insensitive substring matching, including matches inside words and compounds.
- Present Google-like results led by the matching filename, with its root-relative path and an escaped excerpt containing words before and after the match; link each result to its normal viewer page.
- Apply existing hidden-file, exclusion, root-confinement, symlink, and private-configuration rules to search traversal and results.
- Handle empty queries, no matches, unreadable files, oversized work, and cancelled requests without exposing filesystem details or preventing unrelated content from being served.
- Keep search local and current by scanning on demand; do not add Bleve, a persistent index, a file watcher, or a refresh signal.

## Capabilities

### New Capabilities

- `content-search`: Local full-text search across eligible served content, including query behavior, result presentation, access control, freshness, and resource limits.

### Modified Capabilities

None. Existing content visibility and browser navigation behavior remains unchanged.

## Impact

- Add search traversal and matching under `internal/content`, reusing resolver visibility and path-safety policy.
- Extend `internal/web` routing, page models, templates, and styles with the search form and results page.
- Add unit, integration, accessibility, security, boundary, and release-smoke coverage; document query behavior and limitations.
- Use the Go standard library only; no new runtime service, external search service, index storage, or third-party dependency is required.
