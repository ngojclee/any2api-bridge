# r26 — Central CPA Request-After-Auth `call_id` Normalization for OpenAI/Codex (`v0.5.27`)

## Problem

Non-OpenAI `/v1/responses` translators in CPA emit synthetic tool call IDs of
the form `cpa_tid_v1_<base64>` (73–107 chars) on both `call_id` and `id`. When a
client later switches the thread to an OpenAI/Codex model (`to_format: codex` /
`openai`), OpenAI rejects `call_id` > 64 chars (`HTTP 400`) and non-`fc_` item
`id`s.

## Fix

- `src/callid_sanitize.go`: zero-allocation byte precheck
  (`hasPotentialOversizedCallID`) followed by deterministic SHA-256 clamping
  (`call_<sha256[:48]>`, 53 chars — identical to Codex Desktop's local gateway
  and Rust sidecar) for `call_id` / `tool_call_id` > 64 chars and stripping of
  synthetic `cpa_tid_*` / `call_*` / oversized `id` fields on `input[]` items.
- `src/dispatch.go` (`handleInterceptAfter`): runs normalization only after CPA
  selects an OpenAI/Codex target (`codex`, `openai`, `openai-response`,
  `openai-responses`, `codex:*`, or `ChatGPT`), leaving `devin` / `claude` /
  other non-OpenAI providers completely untouched.
