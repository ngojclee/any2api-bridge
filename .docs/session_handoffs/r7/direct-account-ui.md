# R7 Handoff - Direct Account Management UI

Project: Any2Api Bridge
Round: R7
Role: Dev CPA Plugin - Vision
Session ID: 01a0822e-74ad-7e82-80b3-c2f60d76a76e
Planner session: 01a0b405-1930-7c61-9fff-67f7855c8772
Repo: D:/Python/projects/CPA Plugin/agy-identity-bridge
Branch: codex/r7-direct-account-ui
Suggested worktree: D:/Python/projects/CPA Plugin/agy-identity-bridge-r7-ui
Base main: 3d98c93

## Problem

R6 landed the direct-account foundation behind `direct_mode_enabled`: the
account metadata model, CPA channel payload adapter, model catalog parser,
and redacted JSON boundary at
`GET /v0/management/plugins/any2api-bridge/direct/accounts`. What is missing
is the operator-facing management UI over that contract: the owner wants a
sidebar with provider groups (Antigravity / AGY2API and ChatGPT / GPT2API),
each with Accounts, Models, Headers, and Routing views.

## Outcome

The embedded plugin UI becomes an operational console for direct accounts:
operators create accounts per provider kind, scan upstream models, keep
aliases, and see routing state - without touching the legacy mirror config.

## Owned Scope

- `src/direct_accounts.go`, `src/direct_channels.go` - only what the UI
  needs to render;
- `src/management.go` - management JSON contract and embedded HTML;
- `src/usage.go` - reuse the usage dashboard data shape;
- focused tests for the new UI/service boundary;
- `.docs/reports/r7/direct-account-ui.md`.

Do not edit `src/executor.go`, `src/providerspec.go`, or the legacy mirror
flow. Do not edit `ui/` of `gpt2api` - this is the CPA plugin's own embedded
UI.

## Deliverables

1. **Sidebar provider groups.**
   - `Antigravity / AGY2API` and `ChatGPT / GPT2API` as the two top-level
     groups.
   - Each group contains: `Accounts`, `Models`, `Headers`, `Routing`.
   - Global `Usage`, `Logs`, `Settings` below.

2. **Accounts view per provider.**
   - Table of direct accounts: label, channel name, prefix, base_url
     (redacted), api_key_configured flag, enabled, priority, model count.
   - Actions: add account, edit label/prefix/channel/priority, enable/disable,
     test connection, fetch models.
   - One account maps to one CPA `openai-compatibility` channel; multiple
     accounts per provider kind are supported naturally.

3. **Models view per account.**
   - Upstream model id, client alias, enabled flag, image capability,
     thinking capability.
   - Scan/fetch calls the account's upstream `/v1/models` with the account
     credential and static headers; merge preserves existing
     alias/enabled/capability metadata.
   - Selected models publish to the matching CPA channel payload.

4. **Headers view per account.**
   - Static header names and configured state only; write/update controls
     never echo values.

5. **Routing view per account.**
   - CPA prefix, channel name, priority, direct-mode health, migration state.

6. **Usage + Logs + Settings.**
   - Reuse the existing usage analytics (usage share, traffic by client) and
     bounded event log; apply provider/account filters consistently.
   - Secrets, API keys, static header values, and identity secrets are
     write-only everywhere.

7. **JSON contract.**
   - Extend the existing `/v0/management/plugins/any2api-bridge/direct/`
     boundary with the account list/detail/scan/publish calls the UI needs.
   - Keep responses redacted; never return API keys or header values.

## Required Tests

- account list/detail JSON is redacted (no api_key, no header values);
- scan parses OpenAI `data[].id` + bounded plain-list fallback, deduplicates,
  preserves existing alias/enabled/capability metadata;
- publish writes the selected model rows + aliases to the channel payload;
- provider-kind header family selection stays correct (AGY vs Any2API);
- legacy mirror config is untouched when direct mode is off.

## Stop Lines

Do not:

- mutate the live CPA configuration, plugin installation, Docker container,
  browser account, deployment, or credentials;
- restart CPA;
- remove or rewrite `src/executor.go` or legacy mirror code;
- change `CONTROL_PLANE_*`, `HUB_MCP_*`, or any deployment env;
- commit secrets, raw headers, browser state, or live response bodies.

## Reporting

Commit/push `codex/r7-direct-account-ui`, write
`.docs/reports/r7/direct-account-ui.md`, and leave the final answer in your
own thread. Do not send inbound `REPORT_BACK`.
