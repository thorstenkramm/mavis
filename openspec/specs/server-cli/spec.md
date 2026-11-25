# server-cli Specification

## Purpose

Provides the command-line entry point and server lifecycle through which users serve a documentation directory safely and predictably.

## Requirements

### Requirement: Serve a directory through the public CLI
The system SHALL provide `mavis serve <directory>` as the only serving command, using the supplied directory as the content root and `localhost:3344` as the default address.

#### Scenario: Start with defaults
- **WHEN** a user runs `mavis serve .` without listener overrides
- **THEN** Mavis serves the current directory at `http://localhost:3344`

#### Scenario: Reject the former command spelling
- **WHEN** a user runs `mavis serv .`
- **THEN** Mavis rejects the command and reports the valid `serve` spelling without starting a server

### Requirement: Report startup information without launching applications
Mavis SHALL print the serving address or socket path and absolute content root after startup and SHALL NOT open a browser or another application.

#### Scenario: TCP startup
- **WHEN** the TCP listener starts successfully
- **THEN** Mavis prints its URL and content root while leaving browser launch to the user

#### Scenario: Unix-socket startup
- **WHEN** a supported Unix-socket listener starts successfully
- **THEN** Mavis prints its socket path and content root without printing a misleading TCP URL

### Requirement: Support explicit listener selection
Mavis SHALL allow users to select a configurable TCP host and port or a Unix socket, subject to operating-system support.

#### Scenario: Custom TCP listener
- **WHEN** a user supplies a valid host and port
- **THEN** Mavis listens on that TCP address instead of the default

#### Scenario: Unix-socket listener
- **WHEN** a user supplies only a valid socket path on a supported system
- **THEN** Mavis serves the documentation through that socket rather than TCP

### Requirement: Warn about unauthenticated network exposure
Mavis SHALL require deliberate configuration for non-loopback binding and SHALL warn that anyone with network access may read the served files; authentication SHALL NOT be required.

#### Scenario: Bind beyond loopback
- **WHEN** a user explicitly configures a non-loopback host
- **THEN** Mavis displays a clear unauthenticated-access warning before or during startup

### Requirement: Fail startup with actionable diagnostics
Mavis SHALL report invalid roots, invalid listeners, and listener conflicts clearly and SHALL exit nonzero when startup cannot complete.

#### Scenario: Port already in use
- **WHEN** the configured TCP port cannot be bound
- **THEN** Mavis identifies the listener failure and exits nonzero without serving partial content

#### Scenario: Missing content root
- **WHEN** the supplied directory does not exist or is not a readable directory
- **THEN** Mavis identifies the invalid root and exits nonzero

### Requirement: Stop the server cleanly
Mavis SHALL stop through normal CLI interruption and release its listener during graceful shutdown.

#### Scenario: Interrupt a running server
- **WHEN** a user stops Mavis with `Ctrl+C`
- **THEN** Mavis shuts down and releases the TCP port or Unix socket for a subsequent run
