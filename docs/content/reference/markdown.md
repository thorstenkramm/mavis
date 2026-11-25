---
title: Markdown compatibility
linkTitle: Markdown
description: Supported core GitLab Flavored Markdown and intentional limitations.
weight: 30
icon: file-text
---

Mavis recognizes `.md` and `.markdown` files case-insensitively, including names such as `GUIDE.MD`. It reads the
current source on every request and renders it with goldmark's core GitHub/GitLab-style Markdown features.

## Supported core behavior

| Feature | Mavis behavior |
| --- | --- |
| CommonMark | Text, headings, lists, links, emphasis, block quotes, and fenced code blocks |
| Tables | Header and body cells render as HTML tables |
| Task lists | Checked and unchecked items render as disabled checkboxes |
| Strikethrough | `~~removed~~` renders as deleted text |
| Automatic links | Plain HTTP and HTTPS URLs become links |
| Heading navigation | Headings receive IDs, with numeric suffixes for duplicates, and a hierarchical current-document TOC |
| Page titles | The first H1 supplies the browser and top-bar title; documents without an H1 use their filename or directory name |
| YAML front matter | A leading delimited block appears in a collapsed, expandable monospaced panel and is excluded from the body and TOC |
| Document links | Relative `.md` and `.markdown` links resolve against the source document; query strings and fragments are preserved |
| Assets | Relative images and files resolve against the source document; root-relative paths resolve against the served root; embedded images use raw-content URLs |
| External URLs | Absolute external URLs and fragment-only links retain their destinations |

Local destinations must pass the same visibility, exclusion, and root-containment rules as direct requests. Mavis
removes destinations that are excluded or escape the served root rather than exposing a link to them. URL rewriting is
performed on the Markdown syntax tree, so examples inside code spans and fenced code blocks are not rewritten.

Front matter starts with `---` on the document's first line and ends with `---` or `...`. A UTF-8 byte order mark before
the opening delimiter is accepted. Mavis displays the original YAML source without interpreting the metadata; an
unterminated opening delimiter remains ordinary Markdown. Activate
the **Front matter** panel with a pointer or keyboard to inspect the source.

## HTML safety

Raw HTML from Markdown is omitted by the renderer. The generated HTML then passes through a sanitization allowlist.
Safe formatting, heading IDs, tables, lists, images, links, language classes on code blocks, and disabled task
checkboxes are preserved. Scripts, event handlers, unsafe URL schemes, forms, and active embeds are not executable in
the Mavis interface. Application pages also use a restrictive Content Security Policy as defense in depth.

If a document cannot be rendered, Mavis presents a generic safe error fragment without exposing internal diagnostics.
The failure is isolated to that request; other eligible documents remain available.

## Intentional limitations

Mavis supports the core behavior above and does **not** claim full GitLab Flavored Markdown parity. The following
specialized behavior is outside the baseline:

- Mermaid and other diagram rendering;
- mathematical notation and KaTeX;
- GitLab alert blocks, footnotes, definition lists, and emoji shortcodes;
- GitLab-specific issue, merge-request, user, and other project references;
- include directives, wiki-specific links, and repository-aware transformations;
- syntax highlighting (the language class is retained, but Mavis does not add a highlighter).

Unsupported constructs may appear as ordinary text, standard Markdown output, or omitted content. Their output can
differ from GitLab, but they must not compromise the current page or make other documents unavailable.

See [Serving documentation](../guides/serving-documentation.md) for link and asset examples and
[Security and privacy](security.md) for content-access boundaries.
