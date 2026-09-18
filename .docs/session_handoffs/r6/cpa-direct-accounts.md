# R6 Handoff: Direct Account Foundation

Project: Any2Api Bridge

Date: 2026-09-18

Planner: `/root`

Worker role: Dev CPA Plugin - Vision

Repository: `D:/Python/projects/CPA Plugin/agy-identity-bridge`

## Outcome

Implement the source-level direct-account foundation described in:

`.docs/architecture/direct-account-channel-plan.md`

This is a foundation round. It must make direct CPA channel management
testable and opt-in while preserving current production behavior by default.

## Owned Scope

- `src/direct_accounts.go` and focused tests;
- `src/direct_channels.go` and focused tests;
- minimal changes in `src/config.go`, `src/providers.go`, `src/dispatch.go`,
  and `src/management.go` needed for the direct-account contract;
- R6 report at `.docs/reports/r6/cpa-direct-accounts.md`;
- documentation updates directly related to the implemented contract.

## Deliverables

1. Add an opt-in direct routing mode. Legacy mirror mode remains the default.
2. Add a validated, bounded account metadata model for `agy2api` and
   `gpt2api`, with a stable `account_id`, CPA channel name, prefix, label,
   enabled flag, priority, and signing-enabled flag.
3. Implement a CPA OpenAI-compatible channel payload adapter:
   - one account maps to one channel;
   - static API key and static headers are accepted only on write;
   - read models redact all secret values;
   - model rows support upstream id, alias, enabled state, and capabilities.
4. Add direct-mode account resolution from CPA channel/prefix so the existing
   after-auth interceptor chooses `X-AGY-*` or `X-Any2API-*` dynamically.
5. In direct mode, do not register a plugin executor, synthetic auth record,
   virtual provider, mirrored model catalog, or model namespace.
6. Add a model catalog parser suitable for `/v1/models` responses. It must
   normalize, de-duplicate, bound, and preserve existing aliases when merging.
7. Expose a small management JSON contract or internal service boundary that
   R7 can render. Do not build the full sidebar UI in this round.

## Required Tests

- account validation rejects invalid ids, unsafe prefixes, duplicate active
  prefixes, and cross-kind channel collisions;
- direct channel payload preserves selected models and aliases but never emits
  a secret in read responses;
- AGY and GPT accounts receive their correct dynamic header families;
- direct mode does not advertise executor/model-provider capabilities;
- model parser accepts OpenAI `data[].id`, de-duplicates, bounds output, and
  retains existing alias/enabled metadata;
- legacy configuration still behaves as before when direct mode is absent or
  disabled.

## Stop Lines

Do not:

- mutate the live CPA configuration, plugin installation, Docker container,
  browser account, deployment, or credentials;
- restart CPA;
- remove or rewrite `src/executor.go` or legacy mirror code;
- change the existing usage dashboard beyond data types needed by the new
  contract;
- commit inherited dirty test files or `output/`;
- send credentials, raw headers, browser state, or reasoning traces in a
  report.

## Branch and Report

Use a dedicated worktree and branch:

`codex/r6-direct-account-foundation`

Commit only owned files, push the branch, and write a file-first report:

`.docs/reports/r6/cpa-direct-accounts.md`

Then send only this compact pointer to planner thread
`01a0b405-1930-7c61-9fff-67f7855c8772`:

```text
REPORT_BACK
Round: R6
Role: CPA/Vision
Session ID: 01a0822e-74ad-7e82-80b3-c2f60d76a76e
Status: DONE | PARTIAL | BLOCKED | NEEDS_PLANNER
Branch: <branch>
Commit: <commit>
Report: .docs/reports/r6/cpa-direct-accounts.md
Tests: <short result>
Blockers: <none or short line>
Next: planner review
```
