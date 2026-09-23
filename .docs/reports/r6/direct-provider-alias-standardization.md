# r6 — Direct-provider alias standardization

## Why
The CPA provider `prefix` field already registers `prefix/<alias>` for every
model row, so writing the *prefixed* account alias into the provider row
produced `prefix/prefix/x` clones and removed the bare model id — breaking
callers that still use bare names. Separately, "Scan & write models" did not
persist the account's model aliases, leaving the Models tab's CLIENT ALIAS
column empty after a scan.

## Changes
- `mergeDirectAccountModelRow` / `directAccountModelRow` now take the account
  prefix and store the **bare** alias in the provider row
  (`trimDirectAliasPrefix`). CPA registration then yields `x` +
  `antigravity/x` (or `chatgpt/x`) — bare keeps working, prefixed carries the
  signed identity lane.
- Existing provider aliases that carry the account prefix are healed back to
  bare on the next sync; custom bare aliases set by hand are preserved.
- `handleDirectProviderScanUpsert` persists the refreshed account models
  into `direct_accounts` before patching the provider, so the console shows
  the client alias that was pushed.

## Tests
- `TestDirectProviderAliasStripsAccountPrefix`: heal-prefixed, preserve-custom,
  strip-on-new-row.
- Full `go test ./...` green; `go vet` clean. c-shared artifact builds on the
  self-hosted runtime (no gcc on this workstation).

## Deploy
- Version bumped to 0.5.6 (Makefile + registry.json + ldflags).
- Install `.so` to `/home/Docker/CLIProxyAPI/plugins/linux/amd64/`, pin
  `store.version` + `release-tag` in CPA `config.yaml`, restart
  `cli-proxy-api`, verify `plugin loaded ... version=0.5.6`.
- Then re-run "Scan & write models" per account to heal provider aliases.
