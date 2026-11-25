---
title: Configuration
linkTitle: Configuration
description: TOML schema, defaults, wildcard exclusions, and CLI precedence.
weight: 20
icon: settings
---

Mavis reads optional TOML configuration from `.mavis.conf` in the served root. The file itself is never served.
Command-line flags configure one run without changing the file.

## Defaults

| Setting | Default |
| --- | --- |
| `exclude` | `[]` |
| `include_hidden` | `false` |
| `[server].host` | `127.0.0.1` |
| `[server].port` | `3344` |
| `[server].socket` | none |
| `[downloads].export_docx` | `true` |
| `[downloads].pandoc` | none; discover `pandoc` in `PATH` |

With no configuration or flags, `mavis serve .` listens on loopback TCP and prints `http://localhost:3344`.

## TCP example

```toml
exclude = ["drafts/**", "*.tmp"]
include_hidden = false

[server]
host = "127.0.0.1"
port = 3344
```

The same listener can be selected explicitly for one run:

```shell
mavis serve . --host 127.0.0.1 --port 3344
```

## Unix-socket example

```toml
exclude = ["drafts/**", "*.tmp"]
include_hidden = false

[server]
socket = "/tmp/mavis.sock"
```

Or select a socket through the CLI:

```shell
mavis serve . --socket /tmp/mavis.sock
```

A socket is an alternative to TCP, not an additional listener. Defaults do not conflict with socket mode, but an
explicitly configured host or port combined with `socket` is an error. This includes conflicts between file and CLI
values. Unix sockets are supported on macOS and Linux; Windows reports an actionable unsupported-listener error.

## DOCX export example

```toml
[downloads]
export_docx = true
pandoc = "/opt/pandoc/bin/pandoc"
```

Omit `pandoc` to discover an executable named `pandoc` in `PATH`. Set `export_docx = false` to disable DOCX
and skip executable discovery, even when `pandoc` is configured:

```toml
[downloads]
export_docx = false
```

## CLI flags

All serving flags belong to `mavis serve <directory>`:

| Flag | Value and behavior |
| --- | --- |
| `--host <host>` | TCP hostname or address; overrides `[server].host` |
| `--port <port>` | TCP port from `1` through `65535`; overrides `[server].port` |
| `--socket <path>` | Unix-socket path instead of TCP; overrides `[server].socket` |
| `--exclude <pattern>` | Adds an exclusion; repeat the flag for multiple patterns |
| `--include-hidden` | Sets `include_hidden = true` for this run |
| `--include-hidden=false` | Explicitly disables hidden content even when the file enables it |

Explicit scalar flags override corresponding file values. If `--include-hidden` is absent, the file or default value
applies. Exclusions are different: file and CLI patterns are combined rather than replaced.

For example:

```shell
mavis serve . --port 5000 --exclude "drafts/**" --exclude "*.tmp" --include-hidden=false
```

## Keys

### `exclude`

An array of slash-based wildcard patterns matched against paths relative to the served root. Patterns do not begin
with `/`; for example, use `drafts/**`, not `/drafts/**`. Path separators are normalized before matching on every
operating system.

| Syntax | Meaning | Example |
| --- | --- | --- |
| `*` | Zero or more characters within one path component | `*.tmp` excludes root-level temporary files |
| `?` | One character within a path component | `draft?/notes.md` matches `draft1/notes.md` |
| `[abc]` | One character from a class | `draft[12]/notes.md` matches `draft2/notes.md` |
| `{one,two}` | One of the listed alternatives | `{drafts,tmp}/**` matches either directory tree |
| `**` | Any number of path components, recursively | `**/*.tmp` excludes temporary files at any depth |

Mavis checks the complete path and every parent directory. Consequently, excluding `drafts` also excludes
`drafts/review/notes.md`, and excluding `docs/*` excludes descendants of matched directories such as `docs/guide`.
The pattern `*.tmp` affects only the root because a single `*` does not cross `/`; use `**/*.tmp` for all depths.

CLI and file exclusion lists are combined. Empty, absolute, malformed, and root-escaping patterns—including any
pattern containing a `..` path component—are startup errors.

### `include_hidden`

A boolean controlling files and folders with a path component beginning with `.`. The default is `false`. Exclusion
rules still apply when this is `true`, and `.mavis.conf` always remains private. Use `--include-hidden` or
`--include-hidden=false` to override the file value explicitly for one run.

### `[server]`

`host` and `port` select a TCP listener; `socket` selects a Unix socket. Explicit CLI values override file values for
the current run. Host `127.0.0.1` and port `3344` are defaults and therefore do not conflict with socket mode unless
one of them is also supplied explicitly.

Binding to a non-loopback host requires explicit configuration. Mavis displays a warning that anyone with network
access may read the served files and does not enable authentication. Use non-loopback binding only when that exposure
is acceptable.

### `[downloads]`

`export_docx` controls optional DOCX conversion and defaults to `true`. When it is enabled and `pandoc` is omitted,
Mavis performs one `PATH` lookup for `pandoc` during startup. If the executable is absent, Mavis still starts and DOCX
is unavailable. Setting `export_docx = false` always disables DOCX and skips all executable lookup, including a
configured `pandoc` value.

Set `pandoc` to a non-empty executable name or path to override automatic discovery of the default name. A name is
resolved through `PATH`; an absolute or relative path is checked directly. Mavis uses the resolved executable for the
whole serving session. An explicitly configured executable that cannot be resolved is a startup error; Mavis does not
fall back to another `pandoc` in `PATH`. Empty and whitespace-only values are configuration errors even when
`export_docx = false`.

Installing, removing, or replacing pandoc and changing `PATH` or executable permissions take effect only after Mavis
restarts. Services often have a different `PATH` from an interactive shell: configure the service manager's `PATH` or
use an absolute path, and ensure the Mavis service account can access and execute pandoc and any files it requires.

## Validation

Unknown keys, malformed TOML, invalid value types, empty hosts, invalid ports, empty socket paths, empty explicit
pandoc values, unresolvable explicit pandoc executables, invalid exclusion patterns, and listener conflicts are startup
errors. Mavis names the affected setting and exits without opening a listener. Environment-variable expansion, path
interpolation, and configuration discovery outside the served root are not performed.
