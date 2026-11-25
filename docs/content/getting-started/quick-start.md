---
title: Quick start
linkTitle: Quick start
description: Serve a directory of Markdown documents and open it in your browser.
weight: 20
icon: play-circle
---

## 1. Install Mavis

Download and extract a standalone binary for your platform using [Installation](installation.md), or build the current
source from the repository root:

```shell
make build
```

Neither a downloaded nor a source-built binary requires Go at runtime.

## 2. Serve a documentation directory

Run the executable from its extracted or build directory:

```shell
cd ~/my-project-docs
/path/to/mavis serve .
```

On Windows PowerShell, use the extracted executable path, for example:

```powershell
.\mavis.exe serve .
```

Mavis prints the content root and serving address. By default the address is:

```text
http://localhost:3344
```

Mavis does not open a browser automatically. Open the printed URL yourself.

## 3. Browse the documentation

- Select folders and documents in the left-hand file browser.
- Read rendered Markdown in the right-hand content pane.
- Use the current document's table of contents to jump between headings.
- On narrow screens, use **Files** to open navigation and **Close** or **Escape** to dismiss it.
- Edit a source file and manually refresh the browser to see the latest content.

A directory's `README.md` becomes its default page when present. Other directories show their eligible folder and file
listing.

## 4. Stop the server

Press `Ctrl+C`. Mavis shuts down and releases the TCP port or Unix socket so you can start it again.

## Where next

- [Serving documentation](../guides/serving-documentation.md): directories, links, assets, and refresh behavior
- [Configuration](../reference/configuration.md): CLI flags, TOML, exclusions, and listeners
- [Platform compatibility](../reference/platforms.md): operating systems, CPUs, and socket support
- [Security](../reference/security.md): visibility and filesystem boundaries
