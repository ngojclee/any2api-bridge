# Namespaced wire names in provider model rows (v0.5.9)

## Change

`mergeDirectAccountModelRow` now writes `name` as the namespaced wire name via
`directProviderWireName`: `antigravity/x`, `chatgpt/x`. When the upstream id
already carries the account prefix it is kept verbatim, so gpt2api's native
`chatgpt/*` ids are unchanged.

Rationale: every provider's left column shows a uniform `<backend>/<model>`
form regardless of what the upstream `/v1/models` advertises. agy2api keeps a
bare catalog and accepts the prefixed wire name on input (strips it in
`resolve_model` and at the HTTP boundary); gpt2api serves `chatgpt/*` natively.

`mergeDirectProviderModels` additionally:

- indexes existing rows by their prefix-stripped name/id as a fallback key, so
  a rescan of a bare upstream id still finds a row stored as `antigravity/x`
  (exact keys still win over stripped keys);
- checks the unavailable set against the stripped row name too, so a dropped
  upstream model prunes its namespaced provider row.

Alias behaviour is unchanged: rows store bare aliases; prefixed aliases heal
back to bare; custom bare aliases are preserved.

## Tests

- Renamed `TestDirectProviderUpsertDoesNotApplyModelNamespacePrefixStripping`
  to `TestDirectProviderUpsertWritesNamespacedWireNames` and updated
  expectations (`agy/gemini-3.8-flash`).
- Updated lookups in `TestDirectProviderUpsertUpdatesOriginalProviderOnly`,
  `TestDirectProviderAliasStripsAccountPrefix`, and
  `TestDirectPublishPreviewWritesSelectedModelsToChannelPayload` for the
  namespaced row names.
- New `TestDirectProviderRescanFindsRowsByNamespacedWireName`: rescan merges a
  namespaced row by bare upstream id without duplicating, and prunes a dropped
  model's namespaced row.

`go vet ./src` + `go test ./...` green.

## Deploy

- `any2api-bridge-v0.5.9.so` installed; `store.version`/`release-tag` pinned
  to `0.5.9`/`v0.5.9`; CPA restarted; log confirms
  `plugin loaded plugin_id=any2api-bridge version=0.5.9` and the old v0.5.8
  `.so` was removed by CPA.
- Live check: `antigravity/gemini-3.8-flash` sent upstream to agy2api returns
  200; agy2api `/v1/models` still advertises bare ids.

## Operator step

Re-run "Scan & write models" on the `antigravity` account: provider rows gain
`name: antigravity/<id>` while aliases stay bare. The `chatgpt` account is
already consistent (upstream ids are natively `chatgpt/*`).

## Follow-up: raw_names + manual_alias opt-outs (v0.5.10)

Per-account flags on `directAccount`, exposed as two checkboxes in the
Models tab and sent with "Scan & write models". Both persist onto the
stored account through the existing mergeDirectAccountByID +
storeDirectSettings path, so the next scan reuses the choice.

- `raw_names`: row `name` stays at the upstream id — for operators who
  want the left column untouched.
- `manual_alias`: the plugin never writes row `alias`; existing operator
  aliases are preserved verbatim and new rows get none — the right
  column is fully hand-managed.

`handleDirectProviderScanUpsert` reads `raw_names`/`manual_alias` from
the request (explicit "1"/"true"/"0"/"false"); absent params keep the
stored flags.

Test: `TestDirectProviderRescanFindsRowsByNamespacedWireName` (v0.5.9),
`TestDirectProviderRawNamesAndManualAliasOptOut` (v0.5.10). v0.5.10
deployed: `.so` installed, version pinned, `plugin loaded
version=0.5.10` confirmed.
