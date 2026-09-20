# R8 - Any2Api Bridge Model Namespace + Upstream Scan

Role: Dev CPA Plugin - Vision
Session ID: 01a0822e-74ad-7e82-80b3-c2f60d76a76e
Planner: 01a0b405-1930-7c61-9fff-67f7855c8772
Repo: D:/Python/projects/CPA Plugin/agy-identity-bridge
Work directly on `main`. No worktree, no branch.

## Context (measured)

- `model_namespace` works but only publishes the 8 models declared in the
  mirrored `openai-compatibility` provider block.
- The full Antigravity model set lives in a different provider/auth path, so
  the bridge does not see it.
- The owner wants an optional second namespace level so a model can read
  `any2api/<namespace>/<model>`, while keeping the current single-prefix
  behavior working.

## Deliverables

1. Default model namespace.
   - When `model_namespace` is empty, publish under a default `any2api`
     prefix instead of bare IDs (mirrors the codebuddy `codebuddy/` pattern).
   - Keep `model_namespace` as the override.
2. Optional nested namespace.
   - Support an account/provider namespace segment so the public id can be
     `any2api/<namespace>/<model>`; keep `any2api/<model>` valid when the
     extra segment is absent.
   - Executor must strip all bridge-owned prefix segments before calling
     upstream.
3. Upstream model scan.
   - The direct-account scan must populate the catalog from the upstream
     `/v1/models` response and preserve aliases/capabilities, so the console
     can show the full model set instead of only the mirrored config list.
4. Tests for prefix stripping, nested namespace, and scan merge.

## Stop lines

- Do not touch the four inherited dirty test files or `output/`.
- No live CPA mutation or restart, no secrets committed.
