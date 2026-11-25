---
title: Searching documentation
linkTitle: Searching documentation
description: Find current text with local literal substring and phrase search.
weight: 30
icon: search
---

## Search from the top bar

Enter text in the full-width **Search documentation** field above the file browser and content, then press **Enter** or
activate **Search**. Mavis displays results in the main content area and retains the query in the search bar. Search,
result links, and the form work with a keyboard and without JavaScript.

Each result leads with the matching filename as its normal document or viewer link. The complete root-relative path
appears below it to distinguish equal filenames in different folders. A contextual excerpt highlights matching text and
shows surrounding source words. Results use deterministic path order rather than relevance ranking.

The reserved route is `/_mavis/search?q=...`. Files whose URLs are shadowed by the search, navigation, or asset routes
do not count toward the result limit. If several paths reach the same file, search returns one accessible link.

## Literal substrings and phrases

Mavis searches the complete trimmed query as one literal, case-insensitive substring. It finds matches inside words and
compounds, so `sicherheit` matches `Informationssicherheit`. Spaces inside the query remain significant: `security policy`
is one phrase, not two independent terms.

Matching uses Unicode simple lowercase mapping. It does not perform stemming, typo tolerance, accent folding,
language-specific compound splitting, or multi-character case folding. Umlauts remain significant, and `ß` is not treated
as `ss`. Filenames alone are not searched.

Search reads raw Markdown, text, and code source, including Markdown punctuation and YAML front matter. Consequently, an
excerpt can contain source syntax that is not visible in the rendered page. Matching source is escaped before display.

## Current and bounded results

Mavis scans current eligible files on demand. It has no persistent index, file watcher, refresh signal, or external search
service. Repeating a search after an edit or deletion reflects the current filesystem without a restart.

Images, PDFs, and other binary files are skipped silently. Extensionless UTF-8 text is eligible. Oversized, unreadable,
removed, or invalid UTF-8 text candidates are omitted and produce a generic notice that some files could not be searched;
other matches remain available.

| Limit | Value |
| --- | ---: |
| Query length | 200 Unicode characters |
| Results | 100 path-ordered files, with a truncation notice |
| File size | 10 MiB per text candidate |
| Search deadline | 10 seconds |
| Concurrent searches | 2 |

An empty or whitespace-only query displays a prompt without scanning file contents. A query over 200 characters is
rejected before scanning. A search that exceeds its deadline returns a generic timeout response rather than partial
results.

## Visibility and privacy

Search uses the same hidden-file, exclusion, exact-spelling, root-confinement, symlink, and `.mavis.conf` privacy rules as
ordinary content requests. Queries and document contents stay in the local Mavis process and are not sent to an external
search service. See [Security and privacy](../reference/security.md).
