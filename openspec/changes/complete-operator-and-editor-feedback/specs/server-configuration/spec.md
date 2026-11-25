# Spec Delta

## ADDED Requirements

### Requirement: Configure repository editing by template
Mavis SHALL accept an optional top-level `edit_url_template` string specifying an absolute HTTPS URL with a fixed host and exactly one `{path}` placeholder as a complete URL path segment. Credentials, unknown placeholders and blank or malformed templates SHALL be rejected before serving. Absence disables edit links.

#### Scenario: Configure a repository subdirectory
- **WHEN** `.mavis.conf` sets `edit_url_template = "https://github.com/thorstenkramm/mavis/edit/main/docs/content/{path}"`
- **THEN** Mavis accepts the template without contacting a repository or discovering a branch

#### Scenario: Reject an unsafe or incomplete template
- **WHEN** the value is a bare repository URL, uses HTTP, or puts the placeholder in the host instead of the path
- **THEN** startup fails with a diagnostic identifying `edit_url_template`

### Requirement: Opt into group access to the effective Unix socket
Mavis SHALL accept `[server].socket_group` as an optional nonblank local group name. If set, an effective Unix socket path is required after CLI overrides. The setting SHALL NOT turn on socket mode implicitly.

#### Scenario: Select the socket on the command line
- **WHEN** the file supplies a group name and `--socket` supplies the socket path with no conflicting TCP settings
- **THEN** Mavis uses that group for the selected socket on a supported system

#### Scenario: Reject a group on a TCP listener
- **WHEN** a group is configured and the effective listener would use TCP
- **THEN** Mavis reports the incompatible setting and does not listen
