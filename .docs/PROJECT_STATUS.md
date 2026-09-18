# Any2Api Bridge Project Status

Updated: 2026-09-18

## Live Baseline

- Repository: `ngojclee/any2api-bridge`
- Main release: `v0.3.0` (`b9814be`)
- CPA plugin id: `any2api-bridge`
- Live mode: legacy mirrored-provider executor remains enabled for the existing
  Antigravity route.
- Live configuration, provider credentials, and CPA restarts are out of scope
  for R6.

## Target Architecture

Any2Api Bridge will manage direct CPA `openai-compatibility` channels for
provider accounts. Each account owns its endpoint, credential, static headers,
model catalog, aliases, and routing metadata. The plugin will continue to
inject per-request signed identity headers for `agy2api` and `gpt2api`.

The target does not use a virtual provider, mirrored model catalog, collision
guard, or plugin-owned executor for normal traffic. Legacy mirror mode remains
available only as a rollback path until direct mode is proven live.

## R6: Direct Account Foundation

Status: DONE, merged at `48ad75c`

Owner: Dev CPA Plugin - Vision

Goal:

1. Add the direct-account domain model and direct-channel adapter behind an
   opt-in routing mode.
2. Preserve the current legacy mirror behavior by default.
3. Prove direct mode uses interceptor-generated headers without creating a
   plugin executor, synthetic auth record, or virtual model catalog.
4. Add unit-level source evidence and a report. Do not mutate live CPA.

Tracked handoff:
`.docs/session_handoffs/r6/cpa-direct-accounts.md`

## R6 Review Result

Accepted and merged into `main` at `48ad75c`. Verified on `main` after merge:
`go test ./...` ok and `go vet ./...` clean. Added
`src/direct_accounts.go`, `src/direct_channels.go`, focused tests, a
`direct_mode_enabled` opt-in flag, and a redacted JSON boundary at
`GET /v0/management/plugins/any2api-bridge/direct/accounts`.

Legacy mirror behavior remains the default, so this merge does not change live
routing. Direct mode still needs a later live round against a real CPA channel
before any migration.

## Planned Follow-up

R7 will build the management UI over the proven direct-account contract:

- sidebar provider groups: Antigravity / AGY2API and ChatGPT / GPT2API;
- Accounts, Models, Headers, and Routing views per provider group;
- global Usage, Logs, and Settings views;
- account model scanning with aliases retained;
- write-only treatment for API keys, static header values, and signing
  secrets.
