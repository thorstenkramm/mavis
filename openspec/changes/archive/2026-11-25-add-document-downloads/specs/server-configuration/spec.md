# Spec Delta

## MODIFIED Requirements

### Requirement: Load configuration from the content root
Mavis SHALL read optional TOML configuration from `.mavis.conf` in the served root, supporting top-level `exclude` and `include_hidden`, `host`, `port`, and `socket` under `[server]`, and `export_docx` and `pandoc` under `[downloads]`.

#### Scenario: Read TCP configuration
- **WHEN** `.mavis.conf` defines exclusions, hidden-file behavior, and TCP host/port values
- **THEN** Mavis applies those values to the serving session

#### Scenario: Read DOCX converter configuration
- **WHEN** `.mavis.conf` defines a non-empty `pandoc` executable string under `[downloads]`
- **THEN** Mavis uses that executable instead of PATH discovery when DOCX is enabled

#### Scenario: Disable DOCX
- **WHEN** `.mavis.conf` sets `export_docx = false` under `[downloads]`
- **THEN** Mavis disables DOCX regardless of pandoc availability and skips executable resolution

#### Scenario: Configuration file is absent
- **WHEN** no `.mavis.conf` exists in the served root
- **THEN** Mavis starts with documented defaults rather than reporting an error

### Requirement: Provide safe configuration defaults
By default, Mavis SHALL use host `127.0.0.1`, port `3344`, no Unix socket, no exclusions, `include_hidden = false`, `export_docx = true`, and no explicit pandoc path.

#### Scenario: Start without overrides
- **WHEN** neither configuration nor CLI flags provide settings
- **THEN** Mavis uses loopback TCP on port `3344`, hides hidden files and folders, and attempts one pandoc PATH lookup at startup

#### Scenario: Default startup finds pandoc
- **WHEN** default DOCX settings apply and pandoc is present in PATH at startup
- **THEN** Mavis enables DOCX with the discovered executable

#### Scenario: Default startup does not find pandoc
- **WHEN** default DOCX settings apply and pandoc is absent from PATH at startup
- **THEN** Mavis starts normally with DOCX unavailable

### Requirement: Validate configuration before serving
Mavis SHALL reject malformed TOML, unexpected keys, invalid value types, malformed or root-escaping exclusion patterns, invalid ports, empty socket paths, empty configured converter paths, and conflicting settings with actionable diagnostics.

#### Scenario: Malformed TOML
- **WHEN** `.mavis.conf` cannot be parsed
- **THEN** Mavis identifies the configuration error and does not start a listener

#### Scenario: Invalid value
- **WHEN** a setting has an unsupported type or value
- **THEN** Mavis names the invalid setting and exits nonzero

#### Scenario: Unknown key
- **WHEN** `.mavis.conf` contains an unrecognized key
- **THEN** Mavis identifies the configuration error and does not start a listener

#### Scenario: Invalid exclusion pattern
- **WHEN** an exclusion contains a malformed or root-escaping wildcard pattern
- **THEN** Mavis names the invalid exclusion and does not start a listener

#### Scenario: Empty converter path
- **WHEN** `.mavis.conf` explicitly sets `downloads.pandoc` to an empty or whitespace-only string
- **THEN** Mavis names the invalid setting and does not start a listener
