---
title: Installation
linkTitle: Installation
description: Download or build a standalone Mavis binary for your platform.
weight: 30
icon: download
---

## Prebuilt binaries

Download the archive for your system from [Mavis releases](https://github.com/thorstenkramm/mavis/releases/latest).
Archives are named `mavis-VERSION-OS-ARCH`, where `VERSION` includes the leading `v`, such as `v1.0.0`.

| Operating system | CPU | Archive |
| --- | --- | --- |
| macOS (Intel) | `amd64` | `mavis-VERSION-darwin-amd64.tar.gz` |
| macOS (Apple Silicon) | `arm64` | `mavis-VERSION-darwin-arm64.tar.gz` |
| Linux (x86-64) | `amd64` | `mavis-VERSION-linux-amd64.tar.gz` |
| Linux (ARM64) | `arm64` | `mavis-VERSION-linux-arm64.tar.gz` |
| Windows (x86-64) | `amd64` | `mavis-VERSION-windows-amd64.zip` |

The binaries do **not** require Go, a language runtime, a package installation, or the repository. Each archive contains
the executable, README, quick start, installation and serving guides, and CLI, configuration, security, platform,
Markdown, and browser compatibility references. A separate `.sha256` file accompanies every archive.

### macOS and Linux

The following example uses Apple Silicon. Replace `v1.0.0` and `darwin-arm64` with the release tag and platform you
downloaded.

```shell
curl -fLO https://github.com/thorstenkramm/mavis/releases/download/v1.0.0/mavis-v1.0.0-darwin-arm64.tar.gz
curl -fLO https://github.com/thorstenkramm/mavis/releases/download/v1.0.0/mavis-v1.0.0-darwin-arm64.tar.gz.sha256
```

Verify the checksum on macOS:

```shell
shasum -a 256 -c mavis-v1.0.0-darwin-arm64.tar.gz.sha256
```

Or on Linux:

```shell
sha256sum -c mavis-v1.0.0-linux-amd64.tar.gz.sha256
```

Extract and run the binary:

```shell
tar -xzf mavis-v1.0.0-darwin-arm64.tar.gz
cd mavis-v1.0.0-darwin-arm64
./mavis serve /path/to/documentation
```

If macOS prevents an unidentified downloaded executable from opening, review the release checksum and your local
security policy before allowing it. Mavis does not ask you to bypass signatures or notarization requirements.

### Windows

In PowerShell, replace the example version with your downloaded release tag:

```powershell
$archive = '.\mavis-v1.0.0-windows-amd64.zip'
$expected = ((Get-Content "$archive.sha256" -Raw) -split '\s+')[0]
if ((Get-FileHash $archive -Algorithm SHA256).Hash -ne $expected) { throw 'Checksum mismatch' }
Expand-Archive -LiteralPath $archive -DestinationPath .
```

After verification and extraction, run:

```powershell
cd .\mavis-v1.0.0-windows-amd64
.\mavis.exe serve C:\path\to\documentation
```

Windows supports TCP listeners. Unix-socket mode returns an actionable unsupported-platform error; see
[Platform compatibility](../reference/platforms.md).

## Build from source

Building from source requires the Go version declared in `go.mod`, Git, and `make`:

```shell
git clone https://github.com/thorstenkramm/mavis.git
cd mavis
make build
```

The resulting binary is `./mavis` on macOS and Linux, or `mavis.exe` on Windows. Once built, it has the same standalone
runtime behavior as a release binary.

## Verify a source build

```shell
make lint
make test
make coverage
```

For a release candidate, install GoReleaser v2 and Python 3, then validate the configuration and build all five archives
without publishing:

```shell
goreleaser check
goreleaser release --snapshot --clean --skip=publish
python3 scripts/format-release-checksums.py dist
```

The checksum helper verifies every digest and adds the archive filename required by the `shasum -c` and `sha256sum -c`
commands above. `scripts/smoke-release.go` checks an extracted binary on its native operating system.

Tag-driven GitHub Actions builds without publishing, verifies the packaged documentation and native binary on all five
target platforms, and only then creates the GitHub release from those exact archives and generated release notes. Tags
with a prerelease suffix, such as `v1.1.0-rc.1`, are published as prereleases.

See [Quick start](quick-start.md) to serve your first directory.
