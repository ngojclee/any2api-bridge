# Direct Account and Channel Plan

## Why This Changes

The current plugin mirrors a matched CPA provider into a plugin-owned executor.
That preserves dynamic identity headers, but it duplicates provider state,
creates a virtual model publisher, and forces collision-management behavior.

The new design uses CPA's original `openai-compatibility` channels as the
runtime source of truth. Each account is a normal CPA channel. Any2Api Bridge
manages those channels and injects the per-request identity headers that cannot
be static.

## Required Invariants

1. `agy2api` and `gpt2api` remain separate provider kinds and use their
   existing header families.
2. Identity timestamps and HMAC signatures remain dynamic. They must never be
   persisted as static provider headers.
3. API keys, static header values, and identity secrets are write-only in all
   management responses and HTML pages.
4. A scan must not overwrite an operator's alias, enabled state, image flag,
   thinking setting, or routing priority.
5. Direct mode must not create a plugin-owned provider, synthetic auth record,
   mirrored model catalog, or model namespace.
6. Existing mirror-mode routing stays unchanged until the operator explicitly
   migrates an account.
7. No R6 action changes a live CPA configuration, plugin artifact, container,
   or browser account.

## Account Model

An account is metadata owned by Any2Api Bridge and credentials owned by its
matching CPA `openai-compatibility` channel.

```text
account_id                 Stable plugin identifier
provider_kind              agy2api | gpt2api
label                      Operator-facing account name
channel_name               Matching CPA openai-compatibility channel
prefix                     CPA model prefix for deterministic interception
base_url                   Upstream OpenAI-compatible base URL
enabled                    Account routing switch
priority                   CPA channel priority
identity_signing_enabled   Dynamic header injection switch
models[]                   Upstream id, alias, enabled, capabilities
```

The management write payload may contain an API key and static headers. Those
values are written directly into the CPA channel configuration, but never
returned by Any2Api Bridge. Read models expose only `api_key_configured` and
per-header configured state.

One account maps to one CPA channel. Multiple accounts for the same provider
kind are therefore supported naturally: each can have a distinct upstream key,
headers, prefix, priority, and catalog.

## Direct Request Path

```text
client
  -> CPA model routing selects one direct openai-compatibility channel
  -> Any2Api Bridge after-auth interceptor resolves the account from prefix
  -> plugin adds signed AGY or Any2API headers to the request options
  -> CPA openai-compatibility executor sends the normal upstream request
  -> agy2api or gpt2api gateway
```

Static provider headers belong to the CPA channel. Dynamic identity headers
are generated in the after-auth interceptor. The plugin's legacy executor is
not part of direct mode.

Local CPA v7.3.7 source shows the generic request-interceptor path carries
post-auth header changes into executor options, and the OpenAI-compatible
executor applies those options to its upstream HTTP request. R6 must preserve
this as focused, reproducible source-level test evidence before a live
migration is considered complete.

## Provider Kinds

### Antigravity / AGY2API

- Inject the existing `X-AGY-*` identity headers.
- Use the current canonical path-sensitive signature message.
- Support multiple AGY accounts with separate CPA channel credentials.

### ChatGPT / GPT2API

- Inject the existing `X-Any2API-*` identity headers.
- Preserve `conversation_id` only when the caller supplied one.
- Scan the gpt2api gateway model endpoint and save selected models to the
  account's direct CPA channel.

## Model Scanning and Aliases

The scanner calls the account's upstream model endpoint using the account
credential and static headers. It accepts an OpenAI `{ "data": [...] }`
catalog and a bounded plain-list fallback.

On refresh:

1. Normalize, de-duplicate, sort, and bound discovered upstream ids.
2. Retain existing account rows and their aliases/capabilities when ids match.
3. Mark missing previously-known models as unavailable instead of deleting them
   automatically.
4. Present additions for selection before publishing them to CPA.
5. Write the selected model rows and aliases directly to the matching CPA
   channel.

Aliases are explicit per account. They are not generated from a virtual
provider prefix, and duplicate aliases across active CPA channels must be
rejected with a clear conflict response.

## Management UI Plan

The embedded plugin UI becomes an operational console rather than a mirrored
provider editor.

```text
Sidebar
  Antigravity / AGY2API
  ChatGPT / GPT2API
  Usage
  Logs
  Settings
```

Each provider group contains:

- Accounts: account table, add/edit, enable, priority, connection test, scan.
- Models: upstream id, client alias, enabled state, image capability, thinking
  capability, and selection actions.
- Headers: names and configured-state only; write/update controls do not echo
  values.
- Routing: CPA prefix, channel name, priority, direct-mode health, and
  migration state.

Usage and Logs stay global, but provider/account filters apply consistently to
all charts, tables, and exported data. Existing usage analytics are retained
and refactored to use the selected filter as the single source of truth.

## Migration and Rollback

1. Back up CPA `config.yaml`, plugin configuration, and current plugin files.
2. Import a mirrored provider as a disabled direct account draft.
3. Validate the derived CPA channel payload without writing it.
4. Create one non-conflicting direct test prefix and run the approved smoke
   request.
5. Verify the gateway accepted a valid signed identity and the expected
   conversation/account routing occurred.
6. Publish the direct account under its production prefix, then disable the
   legacy mirrored execution path.
7. Keep the legacy config and documented rollback switch until direct traffic
   has sustained successful smoke evidence.

Rollback is a routing configuration action: re-enable the original legacy
provider before disabling direct mode. Never delete a credential or account
automatically during migration.

## R8 Implementation Notes

R8 moves the direct path to the original `openai-compatibility` provider row:

- `/direct/accounts/upsert` merges the account credential, static headers, and
  selected models into the matching original provider.
- `/direct/accounts/scan/upsert` fetches the upstream `/v1/models` catalog,
  merges it while preserving aliases/capabilities, and writes the resulting
  model rows to the original provider.
- Multiple accounts may share one provider; CPA selected-auth metadata can
  resolve the account identity when available.
- Direct mode does not use `model_namespace` or a plugin-owned virtual provider.
- The legacy mirror remains available behind `direct_mode_enabled` for rollback.
