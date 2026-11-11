# r25 — CPA v8 Direct Provider Matching, Bare-Model Auth Resolution & Release Sync (`v0.5.26`)

## Context

1. `main` (`e98fb58`) carried the `r22`–`r24` test suites (`direct_layout_test.go`,
   `thinking_yaml_test.go`, `executor_tools_test.go`) on top of `cb93eb4` while
   the corresponding `v0.5.24` (`6019b5c`) and `v0.5.25` (`f806438`) source
   commits and `streamEmitPayload` helper remained off `main`, leaving
   `registry.json` at `0.5.23` and `go test ./...` failing to compile on `main`.
2. On production CPA `v8.0.63` (`config-version: 8`), two Direct Mode edge cases
   remained even in `v0.5.25`:
   - `api-keys.openai-compatibility` entries use reverse-proxy domains
     (`gpt.api.*`, `agy.api.*`) while `direct_accounts` probe LAN endpoints, so
     `findDirectProviderIndex` returned `-1` (`original provider was not found`).
   - `Antigravity` runs with `raw_names: true` + `single_id: true` (bare catalog
     IDs, no `antigravity/` prefix) and CPA v8 populates
     `metadata.selected_auth_id` as `openai-compatibility:<channel>:<hash>`,
     which `resolveDirectAccountForPayload` did not match.

## Changes

- Restored `v0.5.24` + `v0.5.25` implementation onto `main` (`openAICompatSection`
  for `api-keys.openai-compatibility` + `keys`, kebab-case `thinkingSpec` YAML
  tags, `include_variants` scan flag, and `POST /direct/accounts/models/remove`)
  plus `streamEmitPayload` in `src/executor.go`.
- `src/direct_provider.go` (`findDirectProviderIndex`): keep exact
  `(channel_name, base_url)` matching as pass 1, then fall back to a unique
  case-insensitive `channel_name` match without overwriting the provider's
  configured `base-url`. Ambiguous duplicate names still return `-1`.
- `src/direct_service.go` (`scanDirectAccountModels`): fall back to the matched
  CPA provider's `compatAPIKeys` when `account.APIKey` is empty so browser-console
  upstream scans work when credentials live only in `keys`.
- `src/direct_channels.go` (`resolveDirectAccountForPayload`,
  `directAccountMatchesCandidate`, `compatAuthChannelName`,
  `resolveDirectAccountByUniqueModel`):
  - Match CPA v8 `openai-compatibility:<channel>:<hash>` `selected_auth_id`
    against the direct account channel/prefix/id.
  - Never hijack shared bare models when `selected_auth_id` explicitly names a
    different OAuth provider (e.g. `devin:...`).
  - Fall back to a unique enabled tracked model when `selected_auth_id` is empty.
- Bumped `src/version.go`, `Makefile`, and `registry.json` to `0.5.26` so GitHub
  Releases, CPA Plugin Store, and the mounted `.so` stay aligned.
