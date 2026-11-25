# mavis

**Ma**rdown **Vi**ew **S**erver (Mavis) serves a directory of Markdown and supporting files as rendered HTML with
folder navigation and a table of contents for the current document.

## Quick start

Download a standalone archive from [Mavis releases](https://github.com/thorstenkramm/mavis/releases/latest), or build
from the repository root:

```shell
make build
```

See [Installation](docs/content/getting-started/installation.md) for archive names, extraction, and checksums.

Serve the included example documentation:

```shell
./mavis serve ./testdata/docs
```

Open <http://localhost:3344> yourself. Mavis prints the address and absolute content root after startup and never
opens a browser automatically. Press `Ctrl+C` to stop it.

To serve your own documentation, run `mavis serve <directory>` in place of the example command. Prebuilt release
binaries do not require Go at runtime.

## Configuration

By default, Mavis serves non-hidden files from the selected root on loopback TCP port `3344`. Use repeatable
`--exclude` flags or a root-level `.mavis.conf` TOML file to exclude files and folders. Hidden files can be included
explicitly, while exclusion rules and the private `.mavis.conf` boundary still apply.

```toml
exclude = ["drafts/**", "*.tmp"]
include_hidden = false

[server]
host = "127.0.0.1"
port = 3344
```

CLI server settings override file values; exclusion rules from both sources are combined. See the complete
[configuration reference](docs/content/reference/configuration.md), [CLI reference](docs/content/reference/cli.md),
and [security guidance](docs/content/reference/security.md).

Mavis supports core [GitLab Flavored Markdown](docs/content/reference/markdown.md) and provides a responsive two-pane
reading interface. See [platform compatibility](docs/content/reference/platforms.md) for supported operating systems,
CPUs, and Unix sockets. Full GitLab extension parity, automatic browser reload, and full-text search of served content
are outside the baseline.
