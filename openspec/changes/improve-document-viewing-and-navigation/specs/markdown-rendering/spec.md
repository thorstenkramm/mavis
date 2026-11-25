# Spec Delta

## ADDED Requirements

### Requirement: Separate leading YAML front matter from Markdown
Mavis SHALL recognize a YAML front-matter block only at the start of a Markdown document and SHALL exclude its delimiters and contents from the rendered Markdown body and table of contents.

#### Scenario: Document begins with front matter
- **WHEN** a Markdown document begins with a delimited YAML front-matter block followed by headings and content
- **THEN** Mavis renders only the following Markdown and builds the TOC only from its headings

#### Scenario: Delimiters appear later in a document
- **WHEN** delimiter-like lines appear outside a leading front-matter block
- **THEN** Mavis handles them as ordinary Markdown rather than front matter

### Requirement: Present collapsible YAML front matter
When front matter is present, Mavis SHALL display a thin collapsed panel above the document that indicates its presence and expands to show the original YAML source in monospaced text.

#### Scenario: Open a document with front matter
- **WHEN** a user opens a Markdown document with leading YAML front matter
- **THEN** a collapsed front-matter panel appears above the document without adding front-matter content to the TOC

#### Scenario: Expand front matter
- **WHEN** a user activates the front-matter panel with a pointer or keyboard
- **THEN** the panel expands to reveal the escaped YAML source in a monospaced, readable format

#### Scenario: Document has no front matter
- **WHEN** a user opens a Markdown document without leading YAML front matter
- **THEN** Mavis does not display a front-matter panel

### Requirement: Use raw representations for embedded local assets
Mavis SHALL resolve embedded local image sources against the source document or served root as applicable and SHALL use their raw representations so viewer pages are never substituted for image content.

#### Scenario: Embed a relative image
- **WHEN** a Markdown document embeds an eligible image from a relative path
- **THEN** the rendered image loads the raw image bytes from the resolved location

#### Scenario: Embed a root-relative image
- **WHEN** a Markdown document embeds an eligible image using a root-relative path
- **THEN** the rendered image loads the raw image bytes from the served root

#### Scenario: Link rather than embed a viewable file
- **WHEN** a Markdown document contains an ordinary link to an eligible viewable non-Markdown file
- **THEN** the link opens that file's documentation viewer rather than its raw representation
