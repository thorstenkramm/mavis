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

Mavis discovers `pandoc` in its startup `PATH`. To disable DOCX and skip discovery:

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

### `edit_url_template`

Optional top-level HTTPS URL for the repository editor. Place it before any TOML table. Include the branch and repository
subdirectory in the URL and one `{path}` segment for the file relative to the served root. For example:

```toml
edit_url_template = "https://github.com/thorstenkramm/mavis/edit/main/docs/content/{path}"
# GitLab alternative: "https://gitlab.example.org/team/manual/-/edit/main/docs/{path}"
```

Omit the key to hide edit links. A bare repository URL, credentials, or a placeholder in the host or query is invalid.

### `[server]`

With `socket` (or `--socket`), set `socket_group = "mavis-proxy"` to share the socket with a named local group.
The socket user stays the same; group sharing sets its mode to `0660`. Omit the key for owner-only `0600` access.
The group must exist, the service must be allowed to assign it, and the proxy needs search permission on the parent directory.
`socket_group` cannot be used with TCP.
See the [Linux Caddy deployment](../guides/linux-caddy.md) for a complete socket configuration and hardened unit.

`host` and `port` select a TCP listener; `socket` selects a Unix socket. Explicit CLI values override file values for
the current run. Host `127.0.0.1` and port `3344` are defaults and therefore do not conflict with socket mode unless
one of them is also supplied explicitly.

Binding to a non-loopback host requires explicit configuration. Mavis displays a warning that anyone with network
access may read the served files and does not enable authentication. Use non-loopback binding only when that exposure
is acceptable.

### `[downloads]`

`export_docx` defaults to `true`. Mavis looks up `pandoc` once in its process `PATH` and uses the resolved path for the session.
If no executable is found, Mavis starts without DOCX. Setting `export_docx = false` skips the lookup.

To select a particular installation, change the service's `PATH` or place a symlink named `pandoc` in a trusted directory
on that `PATH`. Restart Mavis to repeat discovery. If upgrading a configuration that sets `downloads.pandoc`, remove
that key: Mavis rejects it as unknown.

## Validation

Unknown keys, malformed TOML, invalid value types, empty hosts, invalid ports, empty socket paths,
invalid exclusion patterns, and listener conflicts are startup
errors. Mavis names the affected setting and exits without opening a listener. Environment-variable expansion, path
interpolation, and configuration discovery outside the served root are not performed.
