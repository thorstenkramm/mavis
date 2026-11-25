---
title: Mavis
description: Serve a directory of Markdown and supporting files as a browsable documentation site.
headline: Your Markdown, ready to read.
lead: >-
  Mavis turns a documentation directory into a browsable local site with rendered core GitLab Flavored Markdown,
  folder navigation, document tables of contents, and no separate site-building step.
note: Mavis renders and serves your documents locally by default, without a hosted service or runtime installation.
terminal: |-
  $ cd ~/my-project-docs
  $ mavis serve .
  Serving: http://localhost:3344
  Root: /Users/me/my-project-docs

  Open the printed URL in your browser.
featuresTitle: Documentation without the build pipeline
features:
  - icon: file-text
    title: Core GLFM rendering
    text: Read tables, task lists, fenced code, links, and other everyday Markdown as clean HTML.
    link: /reference/markdown
  - icon: folder
    title: Your directory, directly
    text: Browse the folders and documents you already have. No import, database, or generated content store.
    link: /guides/serving-documentation
  - icon: list
    title: Navigation that stays useful
    text: Move through folders, jump to document headings, and use ordinary browser history and heading URLs.
    link: /guides/serving-documentation
  - icon: settings
    title: Small, explicit configuration
    text: Use CLI flags or root-level TOML for exclusions, hidden files, TCP listeners, and Unix sockets.
    link: /reference/configuration
  - icon: lock
    title: Content boundaries
    text: Keep excluded, hidden, configuration, and out-of-root content unavailable through listings and direct URLs.
    link: /reference/security
  - icon: desktop
    title: Desktop and mobile reading
    text: Use a two-pane desktop layout or dismissible file navigation on smaller screens, with keyboard access throughout.
    link: /reference/browser-compatibility
exampleTitle: Configuration that stays with the docs
exampleText: >-
  A root-level `.mavis.conf` holds reusable exclusions and server defaults. CLI flags override listener settings,
  while exclusion patterns from both sources combine.
example: |-
  exclude = ["drafts/**", "*.tmp"]
  include_hidden = false

  [server]
  host = "127.0.0.1"
  port = 3344
resourcesTitle: Start exploring
resources:
  - link: /getting-started/quick-start
  - link: /reference/configuration
  - link: /reference/security
installTitle: Install the standalone binary
install:
  - name: Download a release
    code: |-
      # Download and extract the archive for your OS and CPU.
      # No Go or language runtime is required.
      mavis serve /path/to/documentation
    link: /getting-started/installation
  - name: Build from source
    code: |-
      git clone https://github.com/thorstenkramm/mavis.git
      cd mavis
      make build
      ./mavis serve ./testdata/docs
    link: /getting-started/installation
---
