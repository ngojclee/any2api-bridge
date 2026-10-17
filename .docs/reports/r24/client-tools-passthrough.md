# r24 — Client-executed tools passthrough verification

Source: `agy2api/.docs/handoffs/any2api-cpa-client-tools.md`

## Question

Do OpenAI client-executed tool fields survive `harness → CPA
(any2api-bridge) → agy2api` in both directions?

- Request: `tools`, `tool_choice` (string + named object variants),
  `parallel_tool_calls`, `agy_tool_execution`, assistant `tool_calls`,
  `role:"tool"` + `tool_call_id`.
- Response/stream: `message.tool_calls`, `delta.tool_calls` (with `index`),
  `finish_reason:"tool_calls"`, `event: error` frames.

## Plugin-side verdict: already transparent, no functional change needed

`buildUpstreamRequest` sends `req.Payload` base64-verbatim; the only body
touches are `rewritePayloadModelWithEffort` / `stripPayloadModelPrefix`,
both `map[string]any` round-trips that keep unknown fields. Stream path
emits each SSE event's raw `data:` payload through `host.stream.emit`;
`[DONE]` alone is skipped (CPA writes its own tail).

## Changes this round

- `src/executor.go`: extracted `streamEmitPayload` (the emit-envelope
  marshal) from `emitUpstreamEvents` so tests can decode what the bridge
  puts on the wire. Behavior identical.
- `src/executor_tools_test.go` (new): regression tests from the handoff —
  - `TestExecutorPreservesClientToolContractFields`: full
    `normalizeExecutorModel` + `buildUpstreamRequest` path with `tools`,
    named `tool_choice`, `parallel_tool_calls: false`,
    `agy_tool_execution`, assistant `tool_calls`, `role:"tool"` message →
    decoded upstream body deep-equals the client body (model prefix strip
    aside).
  - `TestExecutorPreservesToolChoiceVariants`: `auto`/`required`/`none`
    string forms and `parallel_tool_calls: true` survive verbatim.
  - `TestSSEStreamParserPreservesToolCallDelta`: `delta.tool_calls` frame
    split mid-arguments across reads → emitted byte-identical;
    emit-envelope round-trip decodes to the same bytes.
  - `TestSSEStreamParserPassesErrorEventsAndSkipsDone`: `event: error`
    data passes through; `data: [DONE]` is still dropped.

`go vet ./src` clean, `go test ./src` all green.

## CPA-side verdict: verified live — `Payload` keeps the full contract

After the `openai-compatibility` Antigravity provider was re-enabled
(`disabled: false`), a live probe through CPA `10.21.1.101:8317` with
`claude-sonnet-5-5` (unique to the agy2api lane — bare
`gemini-3.8-flash` belongs to the `devin` provider) confirmed:

- Non-stream `tools` + `tool_choice:auto` + `parallel_tool_calls:false`
  → response `finish_reason="tool_calls"` with populated
  `message.tool_calls` (`id`, `type`, `function.name`,
  `function.arguments`). CPA's inbound parse and outbound write lose
  nothing.
- Round-trip: assistant `tool_calls` + `role:"tool"` + `tool_call_id`
  history accepted upstream (no 422 — agy2api's pending-call validation
  passed), and the model chained a second tool call correctly.
- Stream `stream:true`: exactly one `delta.tool_calls` chunk carrying
  `index`, then `finish_reason="tool_calls"`, then `[DONE]` —
  byte-faithful pass-through.
- `agy_tool_execution` travels as an ordinary JSON field through the
  same path (covered by the unit tests; nothing in CPA's known
  chat-completions schema would single it out).

→ `OriginalRequest` fallback is definitively **not needed**. It stays
documented in the handoff as the escape hatch only.

## Acceptance status

| Check | Status |
|---|---|
| Plugin preserves request tool fields | ✅ unit-verified (new tests) |
| Plugin preserves `delta.tool_calls` bytes | ✅ unit-verified |
| `[DONE]` skipped / `event: error` passed | ✅ unit-verified |
| CPA `Payload` completeness | ✅ live-verified (non-stream + round-trip) |
| Non-stream `message.tool_calls` round-trip | ✅ live-verified |
| Stream pass-through end-to-end | ✅ live-verified (`delta.tool_calls` + `[DONE]`) |

## Install state

Deployed plugin on the host: `any2api-bridge-v0.5.25.so` (matches repo
VERSION). This round adds tests + a non-functional refactor, so no `.so`
rebuild/install is required. When a functional change does land: build
Linux amd64 c-shared → install to
`/home/Docker/CLIProxyAPI/plugins/linux/amd64/` (0755) → pin
`plugins.configs.any2api-bridge.store.version` in `config.yaml` →
restart `cli-proxy-api`. Both paths are bind mounts, so the update
survives restarts.
