# TOML configuration rules

Use these rules when changing Mavis configuration, configuration examples, or
the configuration loader.

## Source of truth

- Mavis parses TOML with `github.com/pelletier/go-toml/v2`, not Viper or
  BurntSushi/toml.
- `internal/config` defines the schema, validation, defaults, source presence,
  and CLI/file merging.
- `docs/content/reference/configuration.md` is the annotated configuration reference. README.md
  contains only the minimal quick start and links to detailed documentation.
- `.mavis.conf` uses TOML and lives in the served content root.
- Treat `.mavis.conf` as user configuration. Do not overwrite a user's file or
  copy local paths, credentials, or other private values into examples.
- The root `.mavis.conf` must never be listed or served, including when hidden
  files are explicitly included.

## Structure and style

- Use top-level `exclude` and `include_hidden` keys and the `[server]` table
  with `host`, `port`, and optional `socket` only as defined by the implemented
  schema.
- Add keys only alongside implementation, tests, OpenSpec requirement updates,
  and documentation. Do not document planned settings as available.
- Use `snake_case` for new keys and preserve established key names.
- Match the documentation examples: left-aligned keys and table headers, `=`
  assignment, and blank lines between unrelated settings.
- Use `# ` for comments and `# key = value` for commented-out settings.
- Use arrays only where the schema accepts them; `exclude` is an array of
  wildcard strings.

## Values

- Quote strings with double quotes unless a TOML literal string makes a
  supported path value clearer. Do not show Windows socket examples: Windows
  socket mode is explicitly unsupported in the baseline.
- Use unquoted `true` and `false` for `include_hidden`.
- Use an unquoted integer for `port`; quote `host` and `socket` strings.
- Keep wildcard patterns slash-based and root-relative. Do not use absolute or
  root-escaping patterns.
- Do not imply environment-variable interpolation or path expansion that the
  loader does not implement.

## Examples and defaults

- Describe each setting's purpose, whether it is optional, its default, and any
  CLI override.
- State that defaults are host `127.0.0.1`, port `3344`, no socket, no
  exclusions, and `include_hidden = false`.
- State that CLI flags override file values for server settings, while CLI and
  file exclusion lists are combined.
- Explain that defaults do not conflict with socket mode, but an explicitly
  configured host or port combined with `socket` is an error.
- Explain that hidden files remain excluded by default and exclusion rules
  still apply when `include_hidden = true`.
- Preserve safe defaults in examples. Do not enable non-loopback binding or
  hidden-file inclusion merely to demonstrate the keys.
- Include a separate Unix-socket example rather than combining `socket` with
  `host` or `port` in one example.
- Use illustrative paths and reserved example domains only; never use real
  secrets or private document contents.

## Validation and maintenance

- Validate examples with a TOML parser and against the implemented schema.
  Syntax validity alone does not prove keys, types, or combinations are valid.
- For schema, loader, or CLI configuration changes, run
  `go test ./internal/config ./internal/cli` and the full quality checks in
  `make lint` and `make test`.
- Cover meaningful defaults, validation, source precedence, exclusion merging,
  and listener conflicts in tests.
- Keep `docs/content/reference/configuration.md`, README.md, CLI help, and OpenSpec requirements
  consistent when configuration behavior changes.
- Unknown keys are configuration errors. Do not silently accept undocumented
  keys in examples or tests.
- Do not require a Node-based TOML formatter solely to edit configuration.
  Preserve established formatting unless a formatter is deliberately adopted.
- Keep credentials and private paths out of diagnostics and logs. Configuration
  errors may identify setting names but must not expose document contents.

## Naming

- Use "Mavis" in prose and `mavis` for commands, binaries, and file names.
