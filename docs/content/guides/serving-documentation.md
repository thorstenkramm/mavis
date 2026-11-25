---
title: Serving documentation
linkTitle: Serving documentation
description: Work with directories, Markdown links, assets, and content updates.
weight: 10
icon: folder
---

## Serve the current directory

```shell
mavis serve .
```

The positional directory becomes the only content root. Mavis serves files directly from that location rather than
copying them into a separate store.

## Directory pages

When a directory contains an eligible `README.md`, opening that directory displays the rendered README. Otherwise,
Mavis displays its eligible folders and files. Navigation loads the current directory and ancestry instead of
scanning the entire tree for every page.

## File browser

Icons identify folders and file types such as Markdown, YAML, and Python; other files share a neutral icon. Use the
arrow button beside a folder to load and reveal its children without leaving the current document; activate it again to hide the descendants. Each folder retains its own state. The buttons
are keyboard accessible with **Tab**, **Enter**, and **Space**, while folder names remain ordinary links to directory
pages.

Mavis stores the folders you expand locally in the browser and restores them after navigation or refresh. The current
document's ancestry is always revealed, but folders opened only for that reason are not saved. If browser storage is
unavailable, expansion continues to work for the open page. Without JavaScript, root entries, current ancestry, and ordinary folder links remain available; opening a folder
link renders that directory and its ancestry on the server.

## Directory trail and color theme

Above each document, viewer, and directory listing, a trail links every parent folder back to the content root.
Directory listings show each entry's icon, name, and type: **Folder**, **Markdown**, or the file's extension.

Mavis follows the operating system's light or dark preference. The button at the right of the top bar switches the
theme and stores the choice locally in the browser under `mavis.theme.v1`. Without JavaScript, Mavis uses the light
theme.

## Markdown documents

Mavis recognizes `.md` and `.markdown` extensions case-insensitively. Relative links to other Markdown documents
open their rendered Mavis pages and preserve heading fragments:

```markdown
See [the setup guide](../guide/setup.md#prerequisites).
```

External absolute URLs remain unchanged. Root-relative paths such as `/images/architecture.png` resolve against the
served root. Other relative assets resolve against the document that references them.

## Non-Markdown files

Eligible UTF-8 text and code files open in an escaped, monospaced viewer. Images and PDFs open in the content area as
well, so the file browser remains visible. Each viewer provides **View raw file** for the original bytes. Other file
formats retain their raw responses and appropriate content types.

Append `?raw=1` to a content URL when linking directly to original bytes rather than its viewer. Embedded Markdown
images use this representation automatically. Viewer and raw URLs apply the same visibility and exclusion rules.

## Update content

Edit the source files normally, then manually refresh the browser. Mavis reads current content per request; it does
not automatically reload the page or require a server restart.

## Control visibility

Hidden files are excluded by default. Wildcard exclusions apply to navigation and direct requests, and the root
`.mavis.conf` is never served. See [Configuration](../reference/configuration.md) and
[Security](../reference/security.md).

## Edit a page in its repository

To offer an **Edit page** link below Markdown and text/code pages, set `edit_url_template` at the top of `.mavis.conf`,
for example `edit_url_template = "https://git.example.org/team/manual/-/edit/main/docs/{path}"`. Restart Mavis.
A directory index links to its `README.md`. Following the link opens the repository editor; editing permissions are managed there.
