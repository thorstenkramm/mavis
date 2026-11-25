---
title: CLI reference
linkTitle: CLI
description: The mavis serve command, flags, output, and shutdown behavior.
weight: 10
icon: terminal
---

## `mavis serve <directory>`

Serve one documentation directory. The directory argument is required and may be `.` for the current directory.
The former `serv` spelling is not an alias.

| Flag | Purpose | Default |
| --- | --- | --- |
| `--host` | TCP hostname or address | `127.0.0.1` |
| `--port` | TCP port | `3344` |
| `--socket` | Unix-socket path instead of TCP | none |
| `--exclude` | Root-relative wildcard exclusion; repeat or combine with file rules | none |
| `--include-hidden` | Include dotfiles and dot directories unless excluded | `false` |

Explicit CLI scalar values override corresponding `.mavis.conf` settings for one run. Use
`--include-hidden=false` to disable hidden content configured by the file. Exclusions from the CLI and file are
combined. Configuring a socket together with an explicitly supplied host or port is an error.

## Startup output and errors

Mavis prints the absolute content root and URL or socket path after the listener starts. It does not launch a
browser. Invalid roots, malformed configuration, listener conflicts, and unavailable ports produce actionable
errors and a nonzero exit status.

Binding to a non-loopback address displays a warning that anyone with network access may read the served files.
Mavis does not require authentication.

## Shutdown

Press `Ctrl+C` for graceful shutdown. Mavis releases its TCP listener or removes the Unix socket it owns.
