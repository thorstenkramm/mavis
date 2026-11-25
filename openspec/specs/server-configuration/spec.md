# server-configuration Specification

## Purpose

Defines deterministic TOML configuration and CLI overrides for content visibility, exclusions, and server listener defaults.

## Requirements

### Requirement: Load configuration from the content root
Mavis SHALL read optional TOML configuration from `.mavis.conf` in the served root, supporting top-level `exclude` and `include_hidden` plus `host`, `port`, and `socket` under `[server]`.

#### Scenario: Read TCP configuration
- **WHEN** `.mavis.conf` defines exclusions, hidden-file behavior, and TCP host/port values
- **THEN** Mavis applies those values to the serving session

#### Scenario: Configuration file is absent
- **WHEN** no `.mavis.conf` exists in the served root
- **THEN** Mavis starts with documented defaults rather than reporting an error

### Requirement: Provide safe configuration defaults
By default, Mavis SHALL use host `127.0.0.1`, port `3344`, no Unix socket, no exclusions, and `include_hidden = false`.

#### Scenario: Start without overrides
- **WHEN** neither configuration nor CLI flags provide settings
- **THEN** Mavis uses loopback TCP on port `3344` and hides hidden files and folders

### Requirement: Override server defaults through the CLI
CLI server settings SHALL override corresponding `.mavis.conf` values for the current run without modifying the file.

#### Scenario: Override a configured port
- **WHEN** the file configures port `4000` and the CLI supplies port `5000`
- **THEN** Mavis listens on port `5000` and leaves `.mavis.conf` unchanged

### Requirement: Override hidden-file visibility through the CLI
An explicitly supplied `--include-hidden` value SHALL override `include_hidden` from `.mavis.conf` for the current run without modifying the file. When the flag is absent, the file or default value SHALL apply.

#### Scenario: Enable hidden content for one run
- **WHEN** the file disables hidden content and the CLI supplies `--include-hidden`
- **THEN** Mavis includes eligible hidden content subject to exclusions and leaves `.mavis.conf` unchanged

#### Scenario: Disable configured hidden content
- **WHEN** the file enables hidden content and the CLI supplies `--include-hidden=false`
- **THEN** Mavis hides hidden content for that run and leaves `.mavis.conf` unchanged

#### Scenario: Preserve the file setting without the flag
- **WHEN** the file enables hidden content and the CLI omits `--include-hidden`
- **THEN** Mavis retains hidden-file inclusion for that run

### Requirement: Combine exclusion sources
Mavis SHALL apply the union of CLI and `.mavis.conf` exclusions so content matching either source is excluded.

#### Scenario: Rules from both sources match
- **WHEN** the file excludes `drafts/**` and the CLI excludes `*.tmp`
- **THEN** Mavis excludes content matching either pattern throughout the session

### Requirement: Reject ambiguous listener configuration
Mavis SHALL reject configuration that selects a Unix socket together with host or port settings instead of starting multiple listeners or silently choosing one.

#### Scenario: Socket and TCP settings conflict
- **WHEN** effective configuration contains a socket path and an explicitly configured host or port
- **THEN** Mavis reports the conflicting listener settings and exits nonzero

### Requirement: Validate configuration before serving
Mavis SHALL reject malformed TOML, unexpected keys, invalid value types, malformed or root-escaping exclusion patterns, invalid ports, empty socket paths, and conflicting settings with actionable diagnostics.

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
