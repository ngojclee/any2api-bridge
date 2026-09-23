# Direct provider: stale upstream-name pruning (v0.5.7)

Round: r6 (continued)
Date: 2026-09-23

## Why

gpt2api renamed its upstream model namespace `chatgpt-web/*` -> `chatgpt/*`
(model release 0.5.35). A re-run of "Scan & write models" marks the old
`chatgpt-web/*` account models `Unavailable`, but v0.5.6 still merged them into
the provider `models:` list and never removed existing provider rows, so dead
upstream names would stay registered in the CPA catalog.

## Changes

- `mergeDirectProviderModels` now builds an `unavailable` set from account
  models flagged `Unavailable` and:
  - drops existing provider rows whose `name`/`id` matches a gone upstream id
    (only rows this account tracked; hand-added rows are untouched),
  - skips unavailable account models when writing new rows.
- `handleDirectProviderScanUpsert` refuses an empty 200 from `/v1/models`
  (`502`, "provider config left unchanged") so a transient empty catalog cannot
  mark everything unavailable and prune the whole provider.
- `prefixDirectModelAliases` materializes `model.Alias = upstream_id` when the
  upstream id already carries the account prefix (gpt2api now serves
  `chatgpt/x`), so the console CLIENT ALIAS column shows the effective
  client-facing name instead of blank.

## Resulting provider row after "Scan & write models"

ChatGPT account (prefix `chatgpt`):

    name:  chatgpt/gpt-5.5-high    # upstream wire name
    alias: gpt-5.5-high            # bare; CPA registers x + chatgpt/x

Antigravity account (prefix `antigravity`):

    name:  gemini-3.8-flash
    alias: gemini-3.8-flash        # registers gemini-3.8-flash + antigravity/gemini-3.8-flash

## Tests

- `TestDirectProviderUpsertPrunesUnavailableUpstreamNames`
- `TestPrefixDirectModelAliasesMaterializesNamespacedUpstreamID`
- `go vet ./...` + `go test ./...` clean.
