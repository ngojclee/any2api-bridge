# any2api-bridge Agent Rules

## Shared workflow (owner standard)

- Work directly on `main` in this checkout. No worktrees, no feature branches
  unless the planner explicitly asks.
- Preserve inherited dirty files (`src/endpoint_test.go`,
  `src/forwarding_test.go`, `src/identity_profiles_test.go`,
  `src/sse_test.go`) and `output/`. Never revert them.
- Stage only your own paths; never `git add -A`.
- After changes: build, test, commit, push to `origin main`.
- Write a short file-first report under `.docs/reports/<round>/`.
- Never print, commit, or log secrets, identity secrets, or raw headers.

## Direction

- Direct-provider mode is the target: edit the ORIGINAL
  `openai-compatibility` provider, add accounts as `api-key-entries`
  (with per-key `weight` and provider `priority`), fetch models from upstream
  `/v1/models`. The legacy mirror stays behind `direct_mode_enabled` for
  rollback only.
- Keep the request interceptor so `X-AGY-*` / `X-Any2API-*` still reach
  agy2api / gpt2api and the Any2Api Connector.

## CPA plugin release checklist

1. Bump the version in code, `Makefile`, and `registry.json`.
2. Build the Linux amd64 shared object with the local/self-hosted runtime.
3. Install to `/home/Docker/CLIProxyAPI/plugins/linux/amd64/` (mode `0755`),
   one `.so` per plugin id.
4. Pin `plugins.configs.any2api-bridge.store.version` + `release-tag` in
   `/home/Docker/CLIProxyAPI/config.yaml`.
5. Restart `cli-proxy-api`, verify `plugin loaded plugin_id=any2api-bridge
   version=<ver>` and `configured`.
6. Restart keeps updates because `plugins/` and `config.yaml` are bind mounts.

## Live load vs restart (CPA v7.3.7)

- CPA v7.3.7 has plugin hot reload (`pluginhost: plugin hot reloaded ...`).
- Hot reload only runs when `ApplyConfig` fires: process startup, an accepted
  config reload, or a config save through the CPA management API. Dropping a new
  `.so` and editing `config.yaml` directly on disk does not reliably trigger it.
- To load a new `.so` without a restart, apply the plugin config through the
  management API; otherwise use one container restart.
- Keep `store.version` matched to the file on disk, or CPA skips the plugin.

## Verification

```bash
go build ./src
go vet ./...
go test ./...
```
