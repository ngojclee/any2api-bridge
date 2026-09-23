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
