# Proposal

## Why

The browser interface looked unfinished: a bare top bar, a link list for directories, raw YAML for front matter and one fixed light palette. Readers asked for a professional appearance that still works for any Markdown repository.

## What Changes

- Put the logo, the search field and a theme switch in one top bar. **BREAKING** for the previous layout: search no longer spans the viewport above the header, and the top bar no longer repeats the page title.
- Show a directory trail above documents, viewers and listings.
- Show front matter as a schema-free tree that depends on structure only, collapsed by default, with the original YAML source one step away.
- Show file-type icons in the file browser and in directory listings, and list directories as a table with name and type.
- Add a dark theme that follows the operating system and can be switched and remembered.
- Widen the reading column up to a fixed maximum and make the download control a compact icon button.
- Give the public documentation site the same logo and a green palette.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `content-search`: the search field moves into the top bar.
- `documentation-browser`: directory trail, typed icons, tabular listings, themes; the page title is shown by the browser only.
- `markdown-rendering`: front matter expands to a structural tree and the original source.

## Impact

Changes touch the embedded template, stylesheet and scripts, the page models, the navigation JSON (`icon`), the release smoke test, the Hugo theme and the public guides. `gopkg.in/yaml.v3`, already present in `go.sum`, becomes a direct dependency for the front-matter tree. Three icons from the MIT-licensed Material Icon Theme are embedded with their notice. Content access rules, routes, downloads and search behavior are unchanged.
