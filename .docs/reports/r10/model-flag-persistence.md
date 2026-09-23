# r10 — Model-flag persistence + compact checkbox labels (v0.5.13, v0.5.14)

## Bugs fixed

1. **Projection drop (v0.5.13)**: `directAccountsForConfig` projected
   accounts back to `direct_accounts` without `raw_names` / `manual_alias` /
   `single_id`, so flags never reached config.yaml even when applied.
2. **Body vs query (v0.5.14, the actual user-visible bug)**: the console's
   `call()` posts checkbox params inside the JSON **body**, but
   `handleDirectProviderScanUpsert` read them only via
   `firstRequestQueryValue` (query string). The overrides never arrived, so
   every Scan & write stored `false` and checkboxes reverted to unchecked.
   `directFlagOverrides` now reads the JSON body first and falls back to
   query values (and accepts both `"1"`/`"0"` strings and JSON booleans).

## Changes

- `src/direct_provider.go` — `directAccountsForConfig` now writes the three
  flags when true, so they persist in `plugins.configs.any2api-bridge.
  direct_accounts` (bind-mounted config.yaml → survives restarts).
  `directAccountFromMap` already parsed them back; only the writer was missing.
- `src/direct_ui.go` — checkbox labels shortened to `Raw names`,
  `Manual aliases`, `Single ID`; full explanations moved into `title`
  tooltips on each chip.
- `src/direct_provider_test.go` — added
  `TestDirectAccountModelFlagsPersistThroughConfigProjection` (projection →
  parse round-trip).

## Release

- v0.5.13: commit `5962e86` — projection writes flags + compact labels.
- v0.5.14: commit `06d0501` — `directFlagOverrides` reads the JSON body.
- Deployed `any2api-bridge-v0.5.14.so` → `/home/Docker/CLIProxyAPI/plugins/
  linux/amd64/` (0755), pinned `version: 0.5.14` + `release-tag: v0.5.14`.
- `plugin loaded ... version=0.5.14`; CPA removed the v0.5.13 binary.

## Behavior

Tick → Scan & write models → flags stored on the account → console and all
later scans/upserts reuse them until changed. Untick → same path persists
`false`.
