# R6 Direct Account Foundation Report

Date: 2026-09-18
Lane: CPA/Vision
Repository: `D:/Python/projects/CPA Plugin/agy-identity-bridge`
Branch: `codex/r6-direct-account-foundation`
Base: `origin/main` at `ef325de`

## Outcome

R6 implements the source-level direct-account foundation behind an opt-in
`direct_mode_enabled` flag. Legacy mirror mode remains the default and its
existing behavior is preserved.

## Measured

- Added `src/direct_accounts.go`:
  - bounded account model (`maxDirectAccounts`, `maxDirectModels`);
  - normalized `account_id`, `provider_kind`, `channel_name`, `prefix`,
    `base_url`, `enabled`, `priority`, `identity_signing_enabled`, write-only
    API key/header fields, and bounded model rows;
  - validation for unsafe account ids/prefixes, duplicate account ids,
    duplicate active prefixes, cross-kind channel collisions, and duplicate
    active model aliases;
  - redacted read views exposing `api_key_configured` and per-header
    `configured` state only.
- Added `src/direct_channels.go`:
  - direct account resolution from CPA channel name, prefix, or requested model
    prefix;
  - dynamic AGY and Any2API header-family selection;
  - CPA OpenAI-compatible channel payload adapter;
  - model catalog parser accepting OpenAI `data[].id` and a bounded plain-list
    fallback, with de-duplication and preservation of existing alias/enabled/
    capability metadata.
- Updated `src/config.go`:
  - added `direct_mode_enabled` and `direct_accounts`;
  - direct mode defaults off;
  - direct account entries are normalized and bounded at load time.
- Updated `src/dispatch.go`:
  - direct mode never creates the plugin executor auth record;
  - direct mode does not advertise executor/model-provider capabilities;
  - direct mode uses the after-auth interceptor to emit signed `X-AGY-*` or
    `X-Any2API-*` headers based on the matched account kind.
- Updated `src/models.go`:
  - direct mode returns an empty model-registration response and does not
    publish a mirrored catalog.
- Updated `src/management.go` and `src/providers.go`:
  - added a small redacted JSON boundary at
    `GET /v0/management/plugins/any2api-bridge/direct/accounts`;
  - diagnostics expose direct mode state and account count.

## Tests Run

- `gofmt` on changed Go files: passed.
- `go test ./...`: passed.
- `go vet ./...`: passed.
- `go vet ./src`: passed.
- `go test ./src -v`: passed.

Required R6 tests now covered:

- account validation rejects invalid ids, unsafe prefixes, duplicate active
  prefixes, duplicate account ids, cross-kind channel collisions, and duplicate
  active model aliases;
- direct channel payload preserves selected models and aliases while read
  responses never expose API key or header values;
- AGY and GPT accounts receive their correct dynamic header families without
  cross-contamination;
- direct mode does not advertise executor/model-provider capabilities and
  returns an empty legacy model-registration response;
- model parser accepts OpenAI `data[].id`, de-duplicates, bounds output, and
  retains existing alias/enabled metadata;
- legacy configuration without `direct_mode_enabled` retains the existing
  mirror behavior and tests remain green.

## Derived

- R6 is a source-level foundation only. It does not write CPA channels or
  accounts.
- The management boundary is intentionally JSON-only; the full sidebar UI is
  deferred to R7.
- Dynamic identity headers are still generated per request and are not stored
  in static account headers.

## Unverified / Remaining Risk

- No live CPA config was mutated, no plugin was installed, and CPA was not
  restarted.
- The direct path was not exercised against a live CPA `openai-compatibility`
  channel in this round.
- The source-level interceptor-to-executor header propagation claim from the
  architecture plan remains to be validated against the deployed CPA host in a
  later round.
- R7 still needs to render the direct account management contract and add
  operator workflows for scan, selection, and publish.

## Boundaries Held

- No live CPA or Docker mutation.
- No plugin install or restart.
- No credential, raw header, prompt body, or reasoning trace in this report.
