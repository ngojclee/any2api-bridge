# r23 — CPA v8 nested provider layout + per-model remove

## Problem

Every write path in the console failed with
`{"error":"matching CPA provider was not found"}` and the models table had no
way to delete stale rows for either Antigravity or ChatGPT.

Root cause: the running CPA image (v8.0.3, `eceasy` build) uses the new
config layout documented in its shipped `config.example.yaml` — credential
provider groups live under the `api-keys:` root
(`api-keys.openai-compatibility`) and each provider row carries `keys:`
instead of `api-key-entries:`. The plugin only looked for a top-level
`openai-compatibility:` list, so it discovered zero providers and every
match/patch path bailed out before reaching name/base-URL matching.

## Changes

- `providerspec.go`: new `openAICompatSection()` returns the owning container
  plus a `nested` flag. The v8 `api-keys.openai-compatibility` section wins
  over a legacy top-level list (matching CPA's "new field wins" rule); an
  empty v8 list still claims the section when no legacy list exists.
  `openAICompatEntries()` and `compatAPIKeys()` now read both layouts
  (`keys` accepted alongside `api-key-entries`/`api_key_entries`).
- `management.go`: `replaceOpenAICompatEntry()` writes the row back into the
  container it was found in (never creates a shadow top-level section) and
  runs `normalizeProviderKeyField()` — rows in a v8 section are migrated to
  `keys:`, rows in a legacy section to `api-key-entries:`. The provider
  editor's credential writes drop `keys` when emitting `api-key-entries` so
  a stale v8 list can't shadow the fresh one.
- `direct_provider.go`: `mergeDirectProviderAPIKeys()` reads all three key
  spellings and clears `keys` when it writes `api-key-entries`
  (write-back re-emits the right name). `findDirectProviderIndex()` now
  trims trailing slashes on both sides of the base-URL comparison.
  `redactDirectProviderPayload()` also strips `keys`/`api_key_entries` so
  v8 credentials can never leak into a preview payload.
- `patchDirectProviderConfig()` gained a `pruneIDs` parameter; matching
  provider model rows (by name, prefix-stripped bare name, or alias) are
  dropped via the new `pruneDirectProviderModelRows()`.
- `direct_service.go` + `management.go`: new authenticated management route
  `POST /direct/accounts/models/remove` (`handleDirectAccountModelRemove`).
  Removes the model from the account store, then patches the provider row
  with the removed id as a prune target. Store removal succeeds even when
  the CPA provider can't be matched; the response reports
  `provider_written`/`provider_error` instead of failing.
- `direct_ui.go`: models table gained an Actions column with a per-row
  `Remove` button (danger styling, `window.confirm` gate, disabled while in
  flight, notice + reload on completion) — same delegated click handler as
  the other persistent writes. Works for both provider views because the
  rows render per account.

## Tests

New `direct_layout_test.go` covers: nested discovery, nested-over-legacy
precedence, legacy fallback, write-back staying nested with `keys:` emitted,
legacy layout keeping `api-key-entries`, provider index matching by
case-insensitive name and trailing-slash base URL, row pruning (bare and
prefixed ids), and the remove handler end-to-end (store + provider, 404 for
untracked ids). `direct_ui_test.go` asserts the remove route and control
exist in the console page. `go vet` clean, `go test ./...` green.

## Deploy

`.so` built by CI on tag `v0.5.25`, installed to the CPA plugins dir,
`store.version` pinned, container restarted — verified `plugin loaded
version=0.5.25` and a live models/remove call.
