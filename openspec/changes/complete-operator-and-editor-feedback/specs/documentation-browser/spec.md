# Spec Delta

## ADDED Requirements

### Requirement: Link editable pages to their repository source
When an edit URL template is configured, Mavis SHALL place a keyboard-accessible “Edit page” link with a decorative Git icon beneath the rendered Markdown or UTF-8 text/code viewer. It SHALL substitute the eligible file's root-relative logical path with each segment URL-escaped. Rendering the control SHALL require neither JavaScript nor a request to the repository host.

#### Scenario: Edit a file with special characters
- **WHEN** the viewer displays `guides/Über #1%.md` with a configured template
- **THEN** the link preserves the `guides/` hierarchy and encodes the filename as data without changing the destination host, query or fragment

#### Scenario: Edit a directory README
- **WHEN** the user reads a directory index backed by `guide/README.md`
- **THEN** the edit link identifies `guide/README.md`, not the directory URL

#### Scenario: Edit a text or code file
- **WHEN** a user navigates by keyboard with JavaScript disabled on a text/code viewer
- **THEN** the user can focus the visibly labelled edit link below the source and follow it normally

#### Scenario: No applicable editor
- **WHEN** the template is absent or the response is an image or PDF viewer, directory listing, search result page, raw file, download or error
- **THEN** Mavis adds no edit control to that response
