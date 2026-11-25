# content-search Specification

## Purpose

Provides local, current full-text search across eligible documentation while preserving Mavis's access controls, responsive layout, and content privacy.

## Requirements

### Requirement: Provide accessible documentation search
Mavis SHALL provide a labelled, full-width search bar above the existing header, file browser, and content. Neither the search input nor its results SHALL appear inside the file browser. Search and result navigation SHALL work by keyboard and without JavaScript.

#### Scenario: Submit a search
- **WHEN** a user enters a query and submits the top search bar
- **THEN** Mavis displays results in the main content area, retains the query in the search bar, and keeps file navigation separate

#### Scenario: Inspect search placement
- **WHEN** a user inspects the interface on desktop or mobile
- **THEN** the search bar spans the top of the interface and neither its input nor its results are contained in the file browser

#### Scenario: Search on a narrow screen
- **WHEN** a user searches at a mobile-width viewport
- **THEN** the top search bar and main-content results remain usable without horizontal page scrolling

### Requirement: Match literal text within current files
Mavis SHALL search eligible UTF-8 file contents using Unicode-aware, case-insensitive literal substring matching. The complete trimmed query SHALL be matched as one substring, including inside words and compounds; filenames SHALL NOT be searched.

#### Scenario: Match a compound word
- **WHEN** a user searches for `sicherheit` and an eligible document contains `Informationssicherheit`
- **THEN** that document appears in the results

#### Scenario: Match a multi-word phrase
- **WHEN** a user enters a query containing spaces
- **THEN** Mavis matches the complete phrase rather than treating the words as independent terms

#### Scenario: Query has different letter case
- **WHEN** a query differs from document text only by Unicode letter case
- **THEN** Mavis treats the text as a match

#### Scenario: Query is empty
- **WHEN** a user submits an empty or whitespace-only query
- **THEN** Mavis presents an empty-search prompt without scanning file contents

### Requirement: Search only eligible content
Search SHALL apply the same visibility, exclusion, root-confinement, symlink, exact-path, and private-configuration rules as ordinary content requests and SHALL NOT transmit queries or content to an external service.

#### Scenario: Matching content is excluded
- **WHEN** a query matches a hidden, excluded, or private-configuration file that is not eligible for serving
- **THEN** neither its path nor its content appears in the results

#### Scenario: Matching content is outside the root
- **WHEN** a query matches a file reachable only through a symbolic link outside the served root
- **THEN** that file is not searched or disclosed

### Requirement: Present safe deterministic results
For each matching file, Mavis SHALL show its filename as the prominent link, its root-relative path as secondary text, and an escaped Google-like excerpt with words before and after the match. Results SHALL be path-ordered, with each file appearing once.

#### Scenario: Identify a result by filename and path
- **WHEN** results include matching files, including identical filenames in different directories
- **THEN** each filename is the prominent normal-content link and its full root-relative path distinguishes its location

#### Scenario: Show context around a match
- **WHEN** a match has source words on both sides
- **THEN** its excerpt contains words before and after the highlighted match and uses ellipses where surrounding source is omitted

#### Scenario: Open a result
- **WHEN** a user activates a filename result link
- **THEN** Mavis opens the file's normal Markdown or non-Markdown viewer URL

#### Scenario: Reserved routes shadow matching files
- **WHEN** a file's logical URL is handled by an internal route
- **THEN** that path is omitted before applying the result limit

#### Scenario: An alias excludes an otherwise accessible match
- **WHEN** a matching file is excluded under one directory alias but accessible through another path
- **THEN** Mavis returns a link through an eligible path without duplicating the file

#### Scenario: Matching text contains markup
- **WHEN** an excerpt contains HTML or script-like source
- **THEN** Mavis displays it as literal text and highlights matches without creating executable elements

#### Scenario: Search has no matches
- **WHEN** no eligible file contains the complete query
- **THEN** Mavis displays a clear no-results message

### Requirement: Reflect content changes without an index
Mavis SHALL read current eligible content for each non-empty search and SHALL NOT require an index rebuild, refresh signal, or server restart after content changes.

#### Scenario: Search after editing
- **WHEN** a user edits a document and repeats a search without restarting Mavis
- **THEN** the results reflect the edited content

#### Scenario: Search after removing a document
- **WHEN** a previously matching document is removed and the search is repeated
- **THEN** the removed document no longer appears

### Requirement: Bound and isolate search work
Mavis SHALL reject queries longer than 200 Unicode characters, return at most 100 results per search with a truncation notice, and skip files larger than 10 MiB. Binary files SHALL be skipped silently; unreadable, removed, invalid UTF-8, and oversized text candidates SHALL produce a generic incomplete-results notice.

#### Scenario: Query is too long
- **WHEN** a user submits a query longer than 200 Unicode characters
- **THEN** Mavis returns a clear validation error without scanning file contents

#### Scenario: More than 100 files match
- **WHEN** more than 100 eligible files contain the query
- **THEN** Mavis returns the first 100 path-ordered results and states that additional results were omitted

#### Scenario: Binary files are present
- **WHEN** the served tree contains images, PDFs, or other binary files
- **THEN** Mavis excludes them from search without marking otherwise complete results as incomplete

#### Scenario: One text candidate cannot be searched
- **WHEN** a text candidate is unreadable, removed during traversal, invalid UTF-8, or larger than 10 MiB
- **THEN** Mavis omits that file, returns other matches, and discloses only that some files could not be searched

#### Scenario: Search exceeds its time budget
- **WHEN** a search cannot finish within 10 seconds
- **THEN** Mavis stops the scan and returns a generic timeout response without exposing filesystem details
