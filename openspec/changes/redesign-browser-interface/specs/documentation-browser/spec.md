# Spec Delta

## MODIFIED Requirements

### Requirement: Derive Markdown page titles from the first H1
When a rendered Markdown document contains an H1, Mavis SHALL use the plain text of its first H1 for the browser page title. Documents without an H1 SHALL retain their existing filename or directory-derived title.

#### Scenario: Document has an H1
- **WHEN** a user opens a Markdown document whose first H1 differs from its filename
- **THEN** the browser title displays the first H1 text while the H1 remains in the rendered document

#### Scenario: Document has multiple H1 headings
- **WHEN** a user opens a Markdown document containing more than one H1
- **THEN** Mavis uses only the first H1 as the page title

#### Scenario: Document has no H1
- **WHEN** a user opens a Markdown document without an H1
- **THEN** Mavis displays the existing filename or directory-derived page title

## ADDED Requirements

### Requirement: Show a directory trail
Mavis SHALL show, above documents, file viewers and directory listings below the content root, a trail that links each parent folder and names the current page.

#### Scenario: Open a nested document
- **WHEN** a user opens `area/topic/page.md`
- **THEN** the trail links `area` and `area/topic` and names `page.md` as the current location

#### Scenario: Open the content root
- **WHEN** a user opens the content root
- **THEN** Mavis shows no trail

### Requirement: Identify entries by type
Mavis SHALL show an icon for folders and for recognized file types in the file browser and in directory listings, and a neutral icon for every other file. Directory listings SHALL state each entry's name and type.

#### Scenario: List a mixed directory
- **WHEN** a user opens a directory containing a folder, a Markdown file, a YAML file and an unrecognized file
- **THEN** each entry shows its icon, name and type, and the unrecognized file shows the neutral icon and its extension

### Requirement: Offer light and dark themes
Mavis SHALL provide light and dark themes that both meet the color-contrast requirement. It SHALL follow the operating system's preference, let the user switch themes from the top bar, and retain that choice across navigation and refresh.

#### Scenario: Open Mavis with a dark system preference
- **WHEN** a user with a dark system preference and no saved choice opens a page
- **THEN** Mavis displays the dark theme

#### Scenario: Switch the theme
- **WHEN** a user activates the theme control and opens another page
- **THEN** the chosen theme is still applied

#### Scenario: JavaScript is unavailable
- **WHEN** a user opens a page without JavaScript
- **THEN** Mavis displays the light theme and every other control keeps working
