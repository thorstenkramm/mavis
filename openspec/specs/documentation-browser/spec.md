# documentation-browser Specification

## Purpose

Provides an accessible, responsive reading and navigation experience for browsing directories and rendered documentation in a browser.

## Requirements

### Requirement: Present a two-pane desktop reading experience
On desktop viewports, Mavis SHALL display file navigation on the left and rendered content with its current-document table of contents on the right.

#### Scenario: Read a document on desktop
- **WHEN** a user opens a Markdown document at a common desktop viewport
- **THEN** the file browser remains available on the left and the content and table of contents appear on the right

### Requirement: Navigate the file hierarchy
Mavis SHALL let users navigate folders, open eligible documents, and identify the current document in the file browser.

#### Scenario: Open a nested document
- **WHEN** a user navigates into a folder and selects a document
- **THEN** Mavis displays that document and identifies it as current in the file browser

#### Scenario: Return to a folder
- **WHEN** a user navigates back toward a parent folder
- **THEN** Mavis presents the corresponding eligible directory hierarchy without requiring a CLI restart

### Requirement: Support browser and heading navigation
Mavis SHALL support browser back/forward navigation between documents and shareable URLs for document headings.

#### Scenario: Use browser history
- **WHEN** a user opens one document after another and selects browser Back
- **THEN** Mavis returns to the prior document and its navigation state

#### Scenario: Open a heading URL
- **WHEN** a user opens a rendered document URL containing a valid heading fragment
- **THEN** the browser presents the document at the corresponding section

### Requirement: Provide responsive narrow-screen navigation
Mavis SHALL keep reading and file navigation usable on mobile and narrow screens, including a way to reach the file browser when space is limited.

#### Scenario: Read on a narrow screen
- **WHEN** a user opens Mavis at a mobile-width viewport
- **THEN** the user can read the document and open and dismiss file navigation without horizontal page scrolling

### Requirement: Meet baseline accessibility expectations
Mavis SHALL provide keyboard-accessible controls, visible focus, semantic navigation and document structure, and WCAG AA color contrast for text and controls.

#### Scenario: Navigate without a pointer
- **WHEN** a user operates file navigation and document controls with a keyboard
- **THEN** all controls are reachable and usable with a visible focus indicator

#### Scenario: Inspect page structure
- **WHEN** an accessibility tool examines the interface
- **THEN** navigation, headings, links, and document content expose semantic roles and structure

### Requirement: Present useful empty and error states
Mavis SHALL provide clear, actionable states for empty directories, missing or excluded content, unreadable files, and invalid requests without exposing unnecessary system details.

#### Scenario: Directory has no visible documents
- **WHEN** a user opens a served directory with no eligible documents
- **THEN** Mavis explains that no visible documents are available rather than showing a blank page

#### Scenario: Requested content is unavailable
- **WHEN** a user requests missing, excluded, or unreadable content
- **THEN** Mavis presents an appropriate error state while keeping site navigation available where possible
