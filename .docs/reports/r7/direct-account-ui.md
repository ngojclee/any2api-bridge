# R7 Direct Account Management UI Report

Date: 2026-09-19
Lane: CPA/Vision
Repository: `D:/Python/projects/CPA Plugin/agy-identity-bridge`
Worktree: `D:/Python/projects/CPA Plugin/agy-identity-bridge-r7-ui`
Branch: `codex/r7-direct-account-ui`
Base: `origin/main` at `f185bf6`

## Outcome

R7 adds the first operator-facing direct-account console over the R6
foundation. It is deliberately preview-safe: the page renders account state and
channel payloads, but does not write live CPA configuration.

## Measured

- Added `src/direct_service.go`:
  - redacted account detail endpoint;
  - account model scan endpoint using the existing upstream model probe;
  - raw catalog merge endpoint for deterministic parser testing;
  - publish preview endpoint returning a redacted CPA channel payload plus
    configured secret/header state only.
- Added `src/direct_ui.go`:
  - embedded direct-account console at `GET .../direct`;
  - sidebar provider groups for `Antigravity / AGY2API` and
    `ChatGPT / GPT2API`;
  - per-provider `Accounts`, `Models`, `Headers`, and `Routing` views;
  - global `Usage`, `Logs`, and `Settings` views;
  - account table, model metadata, static header configured state, routing
    health, draft account form, scan actions, and publish preview;
  - write-only API key/header controls that never echo entered values.
- Updated `src/management.go`:
  - registered `/direct`, `/direct/accounts/detail`,
    `/direct/accounts/scan`, `/direct/accounts/scan/raw`, and
    `/direct/accounts/publish`;
  - routed resource and management requests to the console or JSON boundary.

## Tests Run

- `gofmt` on changed Go files: passed.
- `go test ./...`: passed.
- `go vet ./...`: passed.

Focused coverage added:

- account detail JSON never returns API key, header values, or query secrets;
- raw scan accepts OpenAI `data[].id`, de-duplicates, bounds output, marks
  missing existing rows unavailable, and preserves alias/enabled/capability
  metadata;
- publish preview writes selected models and aliases to the channel payload
  while the management response stays redacted;
- direct console contains provider groups, provider views, global views, and
  no secret values;
- direct mode off leaves legacy mirror capabilities/model registration
  untouched.

## Derived

- R7 is an operational console boundary, not a live migration writer.
- `Preview publish` returns the selected CPA channel shape with credentials
  represented as configured state only. The underlying `directChannelPayload`
  helper still supports the true write shape for a later approved mutation
  round.
- Account creation is a draft form only; persisting a new account belongs to a
  future controlled save/migration workflow.

## Unverified / Remaining Risk

- No live CPA config was mutated, no plugin was installed, and CPA was not
  restarted.
- The scan endpoint was tested at the service/parser boundary; no live upstream
  credential was used.
- Browser-level visual validation was not run because the plugin UI is embedded
  Go HTML and this round was source/test scoped.
- The console does not yet save edited accounts, headers, aliases, or publish
  channel payloads to CPA config; that remains a later explicit write path.

## Boundaries Held

- No live CPA, Docker, deployment, credential, or restart change.
- `src/executor.go` and `src/providerspec.go` were not edited.
- No gpt2api `ui/` changes.
- No secrets, raw headers, browser state, or live response bodies included in
  this report.
