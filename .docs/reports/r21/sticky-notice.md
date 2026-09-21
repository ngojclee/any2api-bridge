# R21 - Sticky console notice

Made the direct-console `.notice` element sticky at the top of the content
column (like the management editor `.notice-bar`) and added the missing
`ok`/`err` color variants so success/error toasts are visually distinct and
stay visible while scrolling.

## Changes
- `src/direct_ui.go`: `#notice` now `position:sticky; top:0; z-index:50`
  with a drop shadow, plus `.notice.ok` and `.notice.err` styles matching
  the existing warn palette.

## Validation
- `go test ./...` -> pass.
- Bumped `Makefile`/`registry.json` to 0.5.5 for release.
