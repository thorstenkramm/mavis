# Spec Delta

## Purpose

Turns local Markdown documents into safe, navigable HTML with core GitLab Flavored Markdown behavior and working document relationships.

## ADDED Requirements

### Requirement: Recognize Markdown documents case-insensitively
Mavis SHALL treat filenames ending in `.md` or `.markdown` as Markdown documents regardless of extension letter case.

#### Scenario: Common extensions
- **WHEN** a directory contains `guide.md`, `guide.markdown`, and `GUIDE.MD`
- **THEN** Mavis renders each as a Markdown document

### Requirement: Render core GitLab Flavored Markdown
Mavis SHALL render core GLFM, including tables, task lists, strikethrough, fenced code blocks, headings, lists, links, and other standard Markdown content.

#### Scenario: Render core extensions
- **WHEN** a document contains a table, task list, strikethrough, and fenced code
- **THEN** the generated HTML presents those constructs with their core GLFM meaning

#### Scenario: Encounter an unsupported extension
- **WHEN** a document uses a specialized GitLab extension outside the supported core
- **THEN** Mavis handles it without compromising the page or other documents and documents the compatibility limitation

### Requirement: Generate stable heading navigation
Mavis SHALL generate heading anchors and table-of-contents entries for the current rendered document, linking each entry to its corresponding section.

#### Scenario: Navigate a long document
- **WHEN** a rendered document contains multiple headings
- **THEN** its table-of-contents data reflects that document's heading hierarchy and links to the matching anchors

### Requirement: Rewrite relative document links
Mavis SHALL rewrite relative links to served Markdown documents into rendered Mavis routes while preserving heading fragments.

#### Scenario: Follow a cross-folder document link
- **WHEN** a document links to `../guide/setup.md#prerequisites`
- **THEN** the rendered link opens the rendered setup document at the prerequisites section, subject to access rules

### Requirement: Preserve external and root-relative URL behavior
Mavis SHALL leave external absolute URLs unchanged and resolve root-relative paths against the Mavis served root.

#### Scenario: External link
- **WHEN** a document contains an external HTTPS link
- **THEN** the rendered link retains the external destination

#### Scenario: Root-relative asset
- **WHEN** a document references `/images/architecture.png`
- **THEN** Mavis resolves the path against the served root rather than the current document's folder

### Requirement: Preserve relative asset links
Mavis SHALL resolve relative image and other asset references against the source document's location and SHALL apply normal content access rules.

#### Scenario: Image beside the document
- **WHEN** a document references `images/diagram.png`
- **THEN** the rendered page loads the eligible asset relative to that document

#### Scenario: Excluded asset
- **WHEN** a referenced asset is excluded or outside the served root
- **THEN** Mavis does not expose the asset through the rendered link

### Requirement: Prevent active-content injection
Mavis SHALL sanitize rendered document HTML so Markdown-supplied scripts, event handlers, and unsafe embeds cannot execute in the Mavis interface.

#### Scenario: Document contains a script
- **WHEN** a Markdown document contains a script or event-handler payload
- **THEN** Mavis does not return it in an executable form within the documentation interface

#### Scenario: Document contains safe formatting
- **WHEN** a Markdown document uses supported non-executable formatting
- **THEN** sanitization preserves that formatting and its intended links

### Requirement: Isolate Markdown rendering failures
Mavis SHALL prevent invalid or unusual Markdown in one document from making the rest of the documentation site unavailable.

#### Scenario: Problematic document
- **WHEN** one document cannot be rendered normally
- **THEN** Mavis presents an appropriate error or safe fallback for that document while other eligible documents remain browsable
