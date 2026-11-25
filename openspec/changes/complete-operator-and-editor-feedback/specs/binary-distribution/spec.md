# Spec Delta

## ADDED Requirements

### Requirement: Publish service deployment guides
The public and packaged documentation SHALL explain how to install, start, verify, troubleshoot and remove Mavis as a service on Linux, macOS and Windows, including a Linux-only HTTPS reverse-proxy deployment.

#### Scenario: Install a Linux or macOS service
- **WHEN** an operator follows the platform guide
- **THEN** the systemd service or launchd LaunchDaemon runs as a dedicated `mavis` user and serves the selected read-only content root

#### Scenario: Install a Windows service
- **WHEN** a Windows operator follows the Servy guide
- **THEN** they can register `mavis.exe` with the correct arguments, working directory and service identity, and verify the loopback TCP endpoint

#### Scenario: Terminate HTTPS at Caddy on Linux
- **WHEN** an operator follows the Caddy guide for `https://mavis.example.com`
- **THEN** a hardened systemd unit runs Mavis on a Unix socket, Caddy connects through its group permissions, and Mavis opens no backend TCP port

### Requirement: Show repository status badges
The repository README SHALL display labelled, linked badges for Mavis main-branch CI, latest release, Go version and license.

#### Scenario: Inspect or follow a badge
- **WHEN** a reader views the README on GitHub
- **THEN** each badge identifies its subject and links to the corresponding Mavis workflow, release, module definition or license
