---
title: Platform compatibility
linkTitle: Platforms
description: Supported operating systems, CPU architectures, listeners, and runtime requirements.
weight: 45
icon: desktop
---

## Operating systems and CPUs

Mavis provides standalone release binaries for these targets:

| Operating system | CPU architecture | Typical hardware | Unix sockets |
| --- | --- | --- | --- |
| macOS | `amd64` | Intel Macs | Supported |
| macOS | `arm64` | Apple Silicon Macs | Supported |
| Linux | `amd64` | x86-64 systems | Supported |
| Linux | `arm64` | ARM64 systems | Supported |
| Windows | `amd64` | x86-64 systems | Not supported; use TCP |

These are the baseline build targets; Mavis does not provide 32-bit, Windows ARM64, or other architecture archives.
Use a currently supported operating system with a compatible browser. Release smoke tests run on native macOS, Linux,
and Windows runners for every published target.

No target requires Go or another language runtime at execution time. Application templates, CSS, and JavaScript are
embedded in the executable. Go is needed only to build from source or run the release tooling.

## TCP and Unix-socket listeners

All platforms support TCP with the default loopback address `127.0.0.1:3344`. A Unix socket can be selected with
`--socket` on macOS and Linux. Mavis defaults to owner-only permissions (`0600`); `server.socket_group` opts into
group access (`0660`). Use a group resolvable by the installed binary: standalone builds may use the local group
database rather than network directory services. Mavis refuses to
remove an arbitrary existing path, and cleans up only the socket owned by the current process.

Windows returns this kind of actionable error instead of silently falling back to TCP:

```text
unix-socket listeners are not supported on Windows; use TCP host and port settings
```

Do not combine `--socket` with an explicitly configured host or port on any platform. See the
[configuration reference](configuration.md) for listener selection and conflicts.

Binding any non-loopback TCP address is an explicit choice and produces a warning: Mavis has no authentication, so
anyone with network access may read the served files. Keep the loopback default unless that exposure is acceptable.
See [Security and privacy](security.md).

## Browser and Markdown compatibility

The responsive reading interface targets current stable Chrome/Edge, Firefox, and Safari. The tested versions,
keyboard behavior, automated accessibility results, and Safari/WebKit distinction are recorded in
[Browser compatibility](browser-compatibility.md).

Mavis recognizes `.md` and `.markdown` case-insensitively and renders core GitLab Flavored Markdown behavior. It does
not claim specialized GitLab extension parity. See [Markdown compatibility](markdown.md) for the supported feature
matrix and intentional limitations.

For archive names, download instructions, and checksums, see [Installation](../getting-started/installation.md).

## Service deployment verification

| Guide | Verification |
| --- | --- |
| [Linux systemd](../guides/linux-systemd.md) | Debian 13 arm64, systemd 257: unit validation, service user `mavis`, HTTP response and stop verified. |
| [Linux Caddy](../guides/linux-caddy.md) | Same VM, Caddy 2.6.2: hardened unit and Caddyfile validated. HTTPS tested with a local CA, `0660` socket accessible to Caddy but not `nobody`, no Mavis TCP listener. |
| [macOS launchd](../guides/macos-launchd.md) | Plist syntax checked on macOS 27.0.1; native system LaunchDaemon lifecycle not yet verified. |
| [Windows Servy](../guides/windows-servy.md) | Instructions checked against Servy 11.0 documentation; native Windows service lifecycle not yet verified. |
