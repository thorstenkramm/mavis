# Spec Delta

## MODIFIED Requirements

### Requirement: Provide accessible documentation search
Mavis SHALL provide a labelled search field in the top bar, above the file browser and content. Neither the search input nor its results SHALL appear inside the file browser. Search and result navigation SHALL work by keyboard and without JavaScript.

#### Scenario: Submit a search
- **WHEN** a user enters a query and submits the top-bar search field
- **THEN** Mavis displays results in the main content area, retains the query in the search field, and keeps file navigation separate

#### Scenario: Inspect search placement
- **WHEN** a user inspects the interface on desktop or mobile
- **THEN** the search field is part of the top bar and neither its input nor its results are contained in the file browser

#### Scenario: Search on a narrow screen
- **WHEN** a user searches at a mobile-width viewport
- **THEN** the search field and main-content results remain usable without horizontal page scrolling

#### Scenario: Jump to search from the keyboard
- **WHEN** a user presses `Alt+/` outside a text field with JavaScript enabled and search is available
- **THEN** focus moves to the search field
