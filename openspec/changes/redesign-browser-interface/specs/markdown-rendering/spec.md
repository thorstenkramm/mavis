# Spec Delta

## MODIFIED Requirements

### Requirement: Present collapsible YAML front matter
When front matter is present, Mavis SHALL display a thin collapsed panel above the document that indicates its presence. Expanded, the panel SHALL present every entry by structure alone, without depending on any key name: a plain value beside its key, and a list or map as a collapsed branch that expands to the same presentation at any depth. The panel SHALL also offer the original YAML source in monospaced text, and SHALL show the source alone when the front matter cannot be parsed.

#### Scenario: Open a document with front matter
- **WHEN** a user opens a Markdown document with leading YAML front matter
- **THEN** a collapsed front-matter panel appears above the document without adding front-matter content to the TOC

#### Scenario: Expand front matter
- **WHEN** a user activates the front-matter panel with a pointer or keyboard
- **THEN** the panel lists each top-level entry with its escaped value, and lists and maps appear as collapsed branches labelled with their size

#### Scenario: Expand a nested value
- **WHEN** a user activates a branch at any depth
- **THEN** its entries appear with the same presentation as the top level

#### Scenario: Inspect the source
- **WHEN** a user activates the source control in the expanded panel
- **THEN** the escaped original YAML appears in a monospaced, readable format

#### Scenario: Front matter is not valid YAML
- **WHEN** a user expands the panel of a document whose front matter cannot be parsed
- **THEN** the panel shows the escaped original source only

#### Scenario: Front matter is very deep or large
- **WHEN** front matter nests or repeats beyond Mavis's limits, or reuses anchors
- **THEN** Mavis marks the omitted part and names aliases instead of expanding them

#### Scenario: Document has no front matter
- **WHEN** a user opens a Markdown document without leading YAML front matter
- **THEN** Mavis does not display a front-matter panel
