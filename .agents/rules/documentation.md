# Documentation site rules

Use these rules for Mavis user documentation, Hugo templates, and documentation CI.

## Structure

- The Hugo project root is `docs/`; public content lives only in `docs/content/`.
- Maintainer documents such as the PRD live in `docs/internal/` and are not
  published.
- Keep the `getting-started`, `guides`, and `reference` sections and their
  menu order unless the information architecture is deliberately changed.
- The self-contained theme is `docs/themes/mavis`, adapted from Mailfire's
  Hugo theme. Do not copy Mailfire artwork, email content, verification files,
  or generated output.
- UIkit is vendored under `docs/themes/mavis/assets/vendor/uikit/`; do not load
  site assets from a CDN. Keep its version, checksum, and license documentation
  current when updating it.

## Authoring

- Update user documentation alongside the behavior it describes. Do not present
  planned behavior as released; mark baseline documentation appropriately
  until implementation lands.
- Use relative Markdown links between content files so Hugo can validate them.
- Keep quick-start content concise and put complete settings in the reference
  section. Internal PRDs and OpenSpec artifacts are not public user guides.
- Preserve accessible navigation, search, dark mode, copy buttons, and
  responsive layouts when changing the theme.
- Do not commit `docs/public`, `docs/resources`, `.hugo_build.lock`, or
  `docs/assets/generated`.

## Verification

- Use `make docs` for the production build. It runs Hugo with minification,
  garbage collection, clean destination, and warnings treated as errors.
- Use `make docs-serve` for local preview.
- The documentation workflow builds pull requests and deploys only from main.
  It uses the pinned Hugo version and full Git history for update dates.
- Run `./run-all-tests.sh` for the full local quality suite when the Go module
  is available. It includes the documentation build when Hugo is installed.
