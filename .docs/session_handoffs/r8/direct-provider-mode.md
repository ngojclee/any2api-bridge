# R8 (corrected) - Direct Provider Mode (drop mirror)

Role: Dev CPA Plugin - Vision
Session ID: 01a0822e-74ad-7e82-80b3-c2f60d76a76e
Planner: 01a0b405-1930-7c61-9fff-67f7855c8772
Repo: D:/Python/projects/CPA Plugin/agy-identity-bridge
Work directly on `main`. No worktree, no branch.

This supersedes the earlier R8 namespace brief.

## Direction change (owner decision)

Stop mirroring a provider into a plugin-owned executor. Follow the
opencode-go / cpa-account-config-manager pattern: the plugin edits the
ORIGINAL `openai-compatibility` provider directly.

- Operator picks an existing provider family (Antigravity / ChatGPT).
- Adding an account adds an `api-key-entries` entry to that provider.
- The plugin writes the provider's `models` list (fetched from upstream
  `/v1/models`) and can set alias/capability rows.
- No plugin-owned virtual provider, no model_namespace, no mirror spec.
- Keep the request interceptor: it still injects `X-AGY-*` / `X-Any2API-*`
  per request so agy2api / gpt2api and the Any2Api Connector get identity.
  CPA v7.3.7 applies after-auth interceptor headers to the native
  openai-compatibility executor, so the plugin executor is no longer needed.

## Deliverables

1. Direct provider write path: update the original provider row's
   `api-key-entries`, `models`, and headers via the CPA management API
   (or config write + reload), never a mirrored copy.
2. Model scan: fetch upstream `/v1/models` and merge into the provider's
   `models` list, preserving aliases/capabilities.
3. Multi-account: several accounts = several `api-key-entries` on the same
   provider; the interceptor must still resolve the correct account/identity
   (use CPA's selected-auth metadata when available).
4. Keep the R7 direct-account console; repoint it at the original provider.
5. Legacy mirror mode stays behind `direct_mode_enabled` for rollback.
6. Tests for provider upsert, key-entry merge, model scan merge, and prefix
   stripping removal.

## Verify

- `go test ./...` and `go vet ./...` pass.
- No live CPA mutation in this round.

## Stop lines

- Do not touch the four inherited dirty test files or `output/`.
- Do not remove the legacy mirror code yet.
