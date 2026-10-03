# r22 — thinking yaml keys + include_variants scan

## Problem 1 — invalid thinking keys in config.yaml

`thinkingSpec` had only JSON tags. When a direct account was serialized into
`config.yaml` (plugin `direct_accounts` store and the `openai-compatibility`
provider `models:` rows), yaml.v3 lowercased the Go field names and emitted
`zeroallowed:` / `dynamicallowed:`. CPA's strict decoder expects the kebab
schema keys `zero-allowed` / `dynamic-allowed`, so every management save of
config.yaml failed with:

    yaml: unmarshal errors:
      field dynamicallowed not found in type registry.ThinkingSupport
      field zeroallowed not found in type registry.ThinkingSupport

That strict-parse failure also blocked unrelated provider edits through the
CPA AI Providers UI — the "cannot delete old models" symptom the operator
reported.

## Fix

- `thinkingSpec` now carries explicit yaml tags (`min`, `max`,
  `zero-allowed`, `dynamic-allowed`, `levels`) matching the CPA schema and
  the hand-built kebab map the provider editor already writes.
- Both readers (`compatModels`, `directAccountModelFromMap`) additionally
  accept the legacy `zeroallowed`/`dynamicallowed` spellings so stores written
  by older builds keep parsing until the next write normalizes them.

## Problem 2 — scan could not inventory flat effort variants

agy2api's public `/v1/models` advertises one entry per model family; the flat
`*-low/medium/high/...` slugs stay hidden (callable, not discoverable). The
bridge therefore wrote only family ids into the provider model table.

## Fix

- New per-account flag `include_variants` (config `include_variants`, console
  "Effort variants" checkbox in the Models tab). It rides the same
  checkbox → `directFlagOverrides` → stored account path as raw_names /
  manual_alias / single_id, and the read view exposes it as
  `include_variants`.
- When set, scans call `{base}/models?include_variants=1`. The identity
  signature keeps covering the bare `/models` path — upstream verifiers sign
  the path only — so signed probes are unaffected.
- "Fetch models" (preview) honors the checkbox too; upsert persists it.

## Files

- `src/providerspec.go` — yaml tags, legacy read keys
- `src/direct_accounts.go` — `IncludeVariants` field/view/parse, legacy read
- `src/direct_provider.go` — flag persistence + overrides
- `src/direct_service.go` — scan query plumbing
- `src/management.go` — `probeProviderModelSpecs(spec, query)`, `probeModelsURL`
- `src/direct_ui.go` — "Effort variants" chip + request params
- `src/thinking_yaml_test.go` — new regression tests
- `src/direct_provider_test.go` — flag round-trip coverage
- `Makefile`, `registry.json`, `src/version.go` — 0.5.24

## Verification

- `go vet ./src`, `go test ./src` — pass (new tests assert kebab keys are
  emitted, legacy keys still parse, and the probe URL gains the query).

## Follow-up

- Deploy: build `any2api-bridge-v0.5.24.so`, install to the CPA plugins dir,
  pin `store.version`/`release-tag` to `v0.5.24`, restart cli-proxy-api, then
  normalize the two stale `zeroallowed`/`dynamicallowed` sections already on
  disk (next store write does it; or edit in place).
