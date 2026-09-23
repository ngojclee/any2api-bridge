# r10 — Model-flag persistence + compact checkbox labels (v0.5.13)

## Bug fixed

`directAccountsForConfig` projected accounts back to `direct_accounts`
without `raw_names` / `manual_alias` / `single_id`. Every scan-upsert wrote
the config via `storeDirectSettings` → `applyPluginConfiguration` reloaded it
→ flags silently reverted to `false` in memory and on disk. Checkbox state
was lost after each Scan & write and after every CPA restart.

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

- Commit `5962e86`, tag `v0.5.13`, CI build green.
- Deployed `any2api-bridge-v0.5.13.so` → `/home/Docker/CLIProxyAPI/plugins/
  linux/amd64/` (0755), pinned `version: 0.5.13` + `release-tag: v0.5.13`.
- `plugin loaded ... version=0.5.13`; CPA removed the v0.5.12 binary.

## Behavior

Tick → Scan & write models → flags stored on the account → console and all
later scans/upserts reuse them until changed. Untick → same path persists
`false`.
