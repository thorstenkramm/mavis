---
title: Privacy
description: How Mavis and this documentation site handle local content and browser data.
---

**Effective date:** November 24, 2026

Mavis is a local-first command-line program. It renders and serves files from a directory you select and does not
transmit their contents to an external service as part of basic operation. It contains no analytics, telemetry,
accounts, or remote content store.

## Network access

Mavis uses a loopback listener by default. If you explicitly bind it to another interface, anyone with network
access may read the served files. Mavis warns about this behavior and does not provide authentication. External
links and assets referenced by your documents remain subject to the destinations and browser behavior you choose.

## This website

This documentation site is static and uses vendored UIkit assets rather than third-party CDN scripts. It includes no
analytics. Your light/dark color-theme choice is stored only in your browser's local storage under `mv-theme`.

When deployed to GitHub Pages, GitHub may process technical data such as IP addresses to deliver the site; see
[GitHub's privacy statement](https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement).

## Changes

Changes to this page are published with a new effective date. The history is available in the
[project repository](https://github.com/thorstenkramm/mavis/commits/main/docs/content/privacy.md).
