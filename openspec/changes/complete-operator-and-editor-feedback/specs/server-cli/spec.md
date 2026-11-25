# Spec Delta

## ADDED Requirements

### Requirement: Share a Unix listener with a named group
On Linux and macOS, Mavis SHALL preserve the socket's user owner and set its group owner to the configured group and its mode to `0660` before serving. With no group configured the socket SHALL retain mode `0600`. Mavis SHALL fail startup if the requested group access cannot be established.

#### Scenario: Connect from a reverse proxy on Linux
- **WHEN** the proxy user belongs to the configured group and can traverse the socket directory
- **THEN** it can connect to the Unix listener while a different unprivileged user outside the group cannot

#### Scenario: Start without group sharing
- **WHEN** an operator starts a Unix listener without configuring a group
- **THEN** Mavis keeps the owner-only socket mode and does not require group lookup or ownership changes

#### Scenario: Ownership or permissions cannot be established
- **WHEN** the group is unknown or setting ownership or permissions fails
- **THEN** Mavis reports the failing operation, serves no requests, and cleans up its own new socket without removing a replacement path

#### Scenario: Request a group-shared socket on Windows
- **WHEN** a Windows operator selects a Unix socket with a group
- **THEN** startup reports that socket mode is unsupported and does not fall back to TCP
