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

## CPA-side verdict: unproven — no live lane to test

The handoff's open item ("does CPA's translated `Payload` keep these
fields") needs a live CPA→agy2api request. On the target host
(10.21.1.101:8317) the `openai-compatibility` provider for agy2api
(`http://10.21.4.101:8123/v1`) is `disabled: true`, and no
`ln.Antigravity`/`any2api`/`agy`/`antigravity` model is routable — every
spelling returns `model_not_found`. Bare `gemini-3.8-flash` belongs to the
`devin` provider (429 quota), not the bridge.

→ `OriginalRequest` fallback stays unimplemented on purpose. It is the
documented escape hatch only if a future live probe proves CPA's
`Payload` drops a contract field; wiring it blind would bypass any
CPA-side normalization for no proven benefit.

## Acceptance status

| Check | Status |
|---|---|
| Plugin preserves request tool fields | ✅ unit-verified (new tests) |
| Plugin preserves `delta.tool_calls` bytes | ✅ unit-verified |
| `[DONE]` skipped / `event: error` passed | ✅ unit-verified |
| CPA `Payload` completeness | ⏳ needs live lane (agy2api provider disabled) |
| Non-stream `message.tool_calls` round-trip | ⏳ same |
| Stream pass-through end-to-end | ⏳ same |

## Install state

Deployed plugin on the host: `any2api-bridge-v0.5.25.so` (matches repo
VERSION). This round adds tests + a non-functional refactor, so no `.so`
rebuild/install is required. When a functional change does land: build
Linux amd64 c-shared → install to
`/home/Docker/CLIProxyAPI/plugins/linux/amd64/` (0755) → pin
`plugins.configs.any2api-bridge.store.version` in `config.yaml` →
restart `cli-proxy-api`. Both paths are bind mounts, so the update
survives restarts.
