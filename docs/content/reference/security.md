---
title: Security and privacy
linkTitle: Security
description: Content visibility, filesystem boundaries, network exposure, and privacy.
weight: 40
icon: lock
---

## Content visibility

Mavis applies one authorization policy to directory listings, Markdown documents, and direct file requests. Eligible
entries are regular files and directories below the selected content root. Unsupported special files are not served.

Hidden files and folders are excluded by default. A path is hidden when **any** component begins with `.`, on every
supported operating system. Set `include_hidden = true` or use `--include-hidden` to make hidden content eligible;
`--include-hidden=false` explicitly disables it for one run.

The root `.mavis.conf` is always private, including when hidden content is included. It never appears in listings and
cannot be retrieved directly or through a symbolic-link alias. Hidden-file inclusion does not override exclusion
rules. A `.mavis.conf` in another directory has no special configuration role and follows the normal hidden-file and
exclusion policy. Mavis also denies other names for the root configuration file by filesystem identity.
Use the stored spelling of each filename in links, even on case-insensitive filesystems.

## Exclusions

Exclusion patterns are slash-based and relative to the served root. Mavis checks the complete path and every parent
folder, so excluding `drafts` also blocks `drafts/review/notes.md`. Excluded content is omitted from listings and is
unavailable through direct requests, including requests for descendants of an excluded folder.

The same exclusions apply to both the requested logical path and a symbolic link's resolved target. An allowed name
therefore cannot expose an excluded target, and an excluded alias cannot expose an otherwise eligible target. See the
[configuration reference](configuration.md) for wildcard syntax and validation rules.

## Filesystem boundaries

Mavis decodes and validates request paths before accessing the filesystem. It rejects traversal segments, encoded
traversal, NUL bytes, backslash-based traversal, absolute paths, Windows volume paths, and non-canonical path forms.
It does not pass request paths to a general-purpose file server.

Every request resolves symbolic links again and verifies containment with path-relative filesystem checks. A link is
eligible only when its target remains inside the served root and both its logical and resolved paths pass the same
visibility and exclusion rules. Links outside the root are not listed or served. Directories, regular files, and their
current contents are read on demand rather than copied to another content store.

### Concurrent local filesystem changes

Filesystem entries can change between validation and opening—a time-of-check/time-of-use race. Mavis re-resolves paths
on every request, opens only resolved targets, and centralizes its checks, but the baseline does **not** claim
protection against a concurrent hostile local process mutating the served filesystem. Do not run Mavis with elevated
privileges or serve a tree that an untrusted local adversary can modify.

## Errors and response handling

Missing, hidden, excluded, invalid, and out-of-root requests receive a non-revealing not-found response. Unexpected
read or open failures receive a generic server error; detailed diagnostics remain in local server logs. A failure for
one file does not prevent unrelated eligible content from being served.

Non-Markdown files use extension-based MIME detection with content sniffing and a binary fallback. Responses include
`X-Content-Type-Options: nosniff`. Viewer-supported files have an HTML viewer at their normal URL and original bytes at
`?raw=1`; other formats retain raw responses. Text and code viewers escape source as literal text rather than creating
active markup. Mavis selects a viewer from a file's first 512 bytes and reads the complete file only to display valid
UTF-8 text, so images, PDFs, and downloads are streamed rather than held in memory.

Raw files have a sandboxed content security policy with scripts disabled, including HTML, SVG, and JavaScript opened
directly. Raw PDFs alone permit framing by the same Mavis origin so the built-in viewer can display them, using
`frame-ancestors 'self'` and `X-Frame-Options: SAMEORIGIN`. Other raw content retains `frame-ancestors 'none'` and
`X-Frame-Options: DENY`. Viewer and raw representations use the same content-authorization checks.

## Network exposure

The default listener is loopback-only. Non-loopback binding requires explicit configuration and displays a warning:
anyone with network access may read the served files. Mavis does not provide authentication. Use it only on networks
where that exposure is acceptable. Unix sockets provide local-only transport with owner-only permissions on supported
platforms; they do not add user authentication.

TCP requests must use the configured hostname or bound address and port. A loopback connection also accepts
`localhost`. For wildcard bindings, clients can use the IP address they connected to. If using a reverse proxy,
configure its upstream Host header accordingly. This check does not use request-controlled DNS lookups.

## Privacy

Mavis renders and serves documents locally without transmitting their contents to an external service. It contains no
analytics, telemetry, account system, remote rendering service, or remote content store. Application templates and
assets are included in the binary.

Documents can still contain author-supplied external links or media. Following a link or loading external media is a
browser network operation outside Mavis's local rendering pipeline and can disclose ordinary connection metadata to
that destination. Use local media and review document links when fully offline reading is required.
