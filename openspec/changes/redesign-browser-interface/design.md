# Design

## Context

The content security policy allows only same-origin styles, scripts and images, so the interface uses classes, embedded SVG files and no inline styles or web fonts. Mavis serves arbitrary Markdown: front matter has no schema and may nest to any depth. See `proposal.md` for motivation.

## Goals / Non-Goals

Restyle and regroup existing controls without changing routes or access checks. No client-side routing, no font downloads, no interpretation of particular front-matter keys, and no reading of file contents for directory listings.

## Decisions

1. **Structure decides, never key names.** `parseFrontMatter` converts the YAML node tree into scalars, lists and maps. A scalar is shown beside its key as written; a list or map is a collapsed branch labelled with its size; the same rule applies at every depth. One entry per row avoids any width threshold. Aliases are named rather than expanded, and depth and node count are bounded, so anchors cannot inflate the tree. Unparseable front matter falls back to the source alone. Nested native `details` elements keep the panel usable without JavaScript.

2. **Icons by extension, one mapping on the server.** `iconFor` and `kindFor` map a name to an embedded icon and a type label. The navigation JSON carries the icon so the client does not repeat the mapping. Unknown types share a neutral icon.

3. **Themes as token sets.** `:root` and `:root[data-theme="dark"]` define the same custom properties. A small blocking script applies the saved or system preference before first paint; the toggle stores the choice under `mavis.theme.v1`. The contrast test checks both palettes.

4. **Search in the top bar.** The form, its label, its route and its no-JavaScript behavior are unchanged; only its position and size change. It moves below the logo at mobile widths. `/` focuses the field when JavaScript is available.

5. **Target sizes.** Controls are 36 CSS pixels high on desktop and 44 at mobile widths, where touch input is expected.

6. **Reading width.** The text column grows to 55rem, about 110 characters at the base size, and no further.
