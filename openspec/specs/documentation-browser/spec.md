# documentation-browser Specification

## Purpose

Provides an accessible, responsive reading and navigation experience for browsing directories and rendered documentation in a browser.

## Requirements

### Requirement: Present a two-pane desktop reading experience
On desktop viewports, Mavis SHALL display file navigation on the left and rendered Markdown or a supported non-Markdown viewer with any applicable current-document table of contents on the right.

#### Scenario: Read a document on desktop
- **WHEN** a user opens a Markdown document at a common desktop viewport
- **THEN** the file browser remains available on the left and the content and table of contents appear on the right

#### Scenario: View a non-Markdown file on desktop
- **WHEN** a user opens a supported text, code, image, or PDF file at a common desktop viewport
- **THEN** the file browser remains available on the left and the selected file appears in the content area

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

### Requirement: Derive Markdown page titles from the first H1
When a rendered Markdown document contains an H1, Mavis SHALL use the plain text of its first H1 for the browser and top-bar page title. Documents without an H1 SHALL retain their existing filename or directory-derived title.

#### Scenario: Document has an H1
- **WHEN** a user opens a Markdown document whose first H1 differs from its filename
- **THEN** the browser and top bar display the first H1 text while the H1 remains in the rendered document

#### Scenario: Document has multiple H1 headings
- **WHEN** a user opens a Markdown document containing more than one H1
- **THEN** Mavis uses only the first H1 as the page title

#### Scenario: Document has no H1
- **WHEN** a user opens a Markdown document without an H1
- **THEN** Mavis displays the existing filename or directory-derived page title

### Requirement: Display text and code within the documentation layout
Mavis SHALL display eligible UTF-8 text and code files as escaped, monospaced source in the content area while keeping file navigation available.

#### Scenario: Open a source file
- **WHEN** a user selects an eligible text or code file
- **THEN** Mavis displays its source in the viewer area with file navigation visible and provides access to the raw file

### Requirement: Display images and PDFs within the documentation layout
Mavis SHALL display eligible images and PDFs in the content area rather than replacing the documentation interface, and SHALL provide access to the raw file.

#### Scenario: Open an image
- **WHEN** a user selects an eligible image
- **THEN** the image appears in the viewer area and file navigation remains visible

#### Scenario: Open a PDF
- **WHEN** a user selects an eligible PDF
- **THEN** the browser's PDF display appears in the viewer area and file navigation remains visible

### Requirement: Provide an expandable iconic folder tree
Mavis SHALL display distinct file and folder icons and SHALL let users independently expand or collapse any eligible folder, with icon and arrow state indicating whether it is expanded.

#### Scenario: Expand a folder without leaving the current document
- **WHEN** a user activates a collapsed folder's expansion control
- **THEN** its eligible children appear in the file browser, its indicators show the expanded state, and the current content remains open

#### Scenario: Collapse an expanded folder
- **WHEN** a user activates an expanded folder's expansion control
- **THEN** its descendants are hidden and its indicators show the collapsed state

#### Scenario: Navigate with folder controls
- **WHEN** a user operates folder expansion controls with a keyboard
- **THEN** each control is reachable, has an accessible name and state, and can be activated without a pointer

#### Scenario: Open a document after changing folder states
- **WHEN** a user opens another document or refreshes the page
- **THEN** Mavis retains the user's folder expansion states and automatically reveals the current document's ancestry

### Requirement: Resize desktop file navigation
Mavis SHALL let desktop users resize the file browser within usable minimum and maximum bounds while keeping the content area usable, and SHALL retain the selected width across navigation and refresh.

#### Scenario: Drag the navigation separator
- **WHEN** a desktop user drags the file-browser separator to a different allowed width
- **THEN** the file browser and content area adjust without overlapping

#### Scenario: Resize with a keyboard
- **WHEN** a desktop user focuses the separator and presses supported arrow keys
- **THEN** the file browser width changes by a predictable increment and its accessible value is updated

#### Scenario: Navigate after resizing
- **WHEN** a user opens another document or refreshes the page
- **THEN** Mavis restores the previously selected file-browser width

### Requirement: Keep file-browser names on one line
Mavis SHALL display each file and folder name on a single unwrapped line and SHALL contain overflow within horizontal file-browser scrolling rather than horizontal page scrolling.

#### Scenario: Display a long filename
- **WHEN** a filename does not fit within the selected file-browser width
- **THEN** the complete name remains on one line and the user can scroll the file browser horizontally without scrolling the whole page sideways
