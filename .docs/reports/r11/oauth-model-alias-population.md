# OAuth model alias population — devin + codebuddy

Round 11. Filled `oauth-model-alias` in `/home/Docker/CLIProxyAPI/config.yaml`
for providers `devin` and `codebuddy` per owner list; all entries written
without `fork` (Keep original off). Backup at `config.yaml.bak-<ts>`.

## Merge result

- devin: 2 existing kept (`devin/swe-2→swe-2`, `devin/kimi-k3`), +135 new.
  Existing `devin/kimi-k3` alias updated `kimi-k3` → `Kimi K3` (owner's list
  intent); dead bare `kimi-k3→Kimi K3` dropped (no such upstream name).
- codebuddy: 4 existing kept (deepseek-v4.1-flash, hy3, hy4-preview, kimi-k3),
  +16 new (`codebuddy/x` → bare `x`). Bare rows in the owner list were already
  aliases of existing entries — skipped.

## Live verification (post-restart, plugin v0.5.17 loaded)

| Client model | Result | Resolved upstream |
|---|---|---|
| Claude Opus 5.5 | 200 | devin/claude-opus-5-5 |
| SWE-2 | 200 | devin/swe-2 |
| swe-2 (old alias) | 200 | devin/swe-2 |
| SWE-1.6 | 200 | devin/swe-1-6 |
| GPT-5.5 High Thinking Fast | 200 | devin/gpt-5-5-high |
| Kimi K3 | 200 | devin/kimi-k3 |
| kimi-k3 | routes to codebuddy | codebuddy vendor 429 (provider-wide) |
| balanced-model / kimi-k2.6 | routes to codebuddy | vendor 429 (provider-wide) |
| Gemini 3 Flash | routes to devin | upstream insufficient_quota |

Codebuddy vendor returns 429 for all models incl. pre-existing
`deepseek-flash:free` — account-side rate limit, not an alias problem.
Devin `Gemini 3 Flash` upstreams exist but the account lacks quota.

Catalog: 285 models; no duplicate `devin/x` originals alongside aliases
(fork off = alias replaces original id).

## Follow-up (v0.5.18 + v0.5.19)

- v0.5.18: models table "Client alias" cell wrapped in `<code>` — matches the
  monospace upstream column (font/spacing mismatch the owner spotted).
- v0.5.19: `effectiveDirectModelAliases` — under `single_id` the
  `account.models` alias cache now stores the effective catalog id (the wire
  name clients call) instead of a materialized `prefix/x` that no longer
  exists in the catalog. Antigravity cache shows bare slugs; chatgpt shows
  `gpt-5.5-high` (stripped under raw_names). Verified via live scan/upsert
  through the management API in the owner's browser session.
- One transient: the first chatgpt scan-upsert request landed with
  single_id unset (prefix re-added, alias cache materialized prefixed);
  re-running the same request persisted `single_id: true` and restored the
  bare single-id provider rows. Flag state verified in config and via the
  accounts endpoint after the re-run.

## Follow-up 2 (v0.5.20) — usage dashboard charts

- Confirmed "Model usage share" / "Traffic by client" are functional: they
  render passive usage records the bridge collects from upstream headers.
- Token usage trend: new SVG stacked-bar chart (input/output/cache-read)
  with dashed cache-hit-rate line on a 0-100% axis; no JS chart lib.
- Model usage share: SVG donut + per-label stable hash colors (dot + bar).
- formatUsageNumber: `1,569,159` separators; `>=1M` compacts to `10.5M`,
  `>=1B` to `1.2B`; full value kept in `title` tooltips on metric cards.
- Live-verified in owner browser: 8 donut segments, 22 bars, 15 rate dots,
  metrics show `239` / `10.5M` / `1.6M / 41,235`.
