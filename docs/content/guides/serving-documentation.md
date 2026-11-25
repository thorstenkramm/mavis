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

## Markdown documents

Mavis recognizes `.md` and `.markdown` extensions case-insensitively. Relative links to other Markdown documents
open their rendered Mavis pages and preserve heading fragments:

```markdown
See [the setup guide](../guide/setup.md#prerequisites).
```

External absolute URLs remain unchanged. Root-relative paths such as `/images/architecture.png` resolve against the
served root. Other relative assets resolve against the document that references them.

## Non-Markdown files

Eligible images, PDFs, text files, and other assets are returned with an appropriate content type. The browser
chooses whether to display a supported format inline or offer it as a download.

## Update content

Edit the source files normally, then manually refresh the browser. Mavis reads current content per request; it does
not automatically reload the page or require a server restart.

## Control visibility

Hidden files are excluded by default. Wildcard exclusions apply to navigation and direct requests, and the root
`.mavis.conf` is never served. See [Configuration](../reference/configuration.md) and
[Security](../reference/security.md).
