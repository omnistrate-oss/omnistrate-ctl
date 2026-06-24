# Deployment-Cell Debug TUI Redesign

Date: 2026-06-23

## Problem

The `deployment-cell debug` TUI renders the left amenity list and the right
detail panel with an unbounded `lipgloss.JoinHorizontal`. When a selected
amenity's rendered values / manifest / logs are large:

- the right panel grows taller than the terminal and the whole layout reflows
  ("the screen jumps"),
- the left list is pushed off-screen,
- there is no way to scroll the content.

The `instance debug` command already solves this with a fixed
header/body/footer, a manual scroll offset clamped against content, and
line-wrapping (`expandLinesToVisual`). We will follow that pattern.

## Goal

Redesign the deployment-cell debug TUI as a two-tier (instance-style)
interface: a scrollable amenity **list screen** that drills into a scrollable,
tabbed **detail screen** per amenity. Bound all panels to the terminal size and
make content scroll instead of reflow.

## Architecture

One bubbletea program, two screens (mirrors `dagModel.inDetail`):

- `deploymentCellDebugModel` (existing, refactored) is the **list screen** and
  gains `inDetail bool` + `detail *amenityDetailModel`. `Update`/`View`
  delegate to the detail model when `inDetail` is true. `tea.WindowSizeMsg` is
  forwarded to the detail model.

### List screen

- Fixed header: `Deployment Cell <id>` + template line.
- Bordered, height-bounded amenity list that scrolls (cursor + scroll offset,
  clamped, normalized so the cursor stays visible) when amenities overflow.
- Footer: `↑↓: select   enter: open   q: quit`.
- `enter` builds an `amenityDetailModel` for the selected amenity and sets
  `inDetail = true`.

### Detail screen (`amenityDetailModel`, new file `cmd/deploymentcell/debug_detail.go`)

- Header (2 lines): `Amenity · <name> · <type> · <status>` and a
  source/workflow line. Workflow/run IDs are omitted when absent
  (reuse `workflowLine`).
- Tab bar built from the amenity's `availableViews()`:
  - Helm → Rendered Values / Template Values / Helm Logs
  - KubernetesManifest → Rendered Manifest / Template Manifest / Cluster Status
  - default → Status
- Bordered, height-bounded scrollable body with a `[pos/total %]` indicator in
  the bottom border.
- Footer: `tab/shift+tab: switch view   ↑↓/pgup/pgdn/home/end: scroll   y: copy   esc: back   q: quit`.

### Content pipeline per tab (correctness-critical)

Raw text → (values tabs only) `toPrettyJSON` → split to source lines →
`expandLinesToVisual(lines, maxWidth)` → slice `[scroll : scroll+bodyH]` →
highlight **each visual line** (`syntax.HighlightLine(line, "values.json")` for
value tabs; plain for manifests/logs).

Wrapping happens on raw text so ANSI escape codes are never split across visual
lines.

## Shared helpers (`internal/tui`)

- `wrap.go` — move `visualLine`, `softWrapLine`, `expandLinesToVisual` out of
  `cmd/instance/debug_plan_dag_render.go`; repoint instance call sites. Pure,
  dependency-free. Add `wrap_test.go`.
- `clipboard.go` — move the dependency-free `CopyToClipboard(text)` here.
  Instance's `copyToClipboardCmd`/`clipboardResultMsg` wrapper stays in instance
  and calls `tui.CopyToClipboard`. deployment-cell gets its own small `tea.Cmd`
  wrapper.
- `internal/syntax`: drop the now-unused `HighlightJSON` (detail screen
  highlights per visual line via `HighlightLine`).

## Edge cases

- Empty amenity list → existing message + quit.
- Placeholder / unparseable payloads (`toPrettyJSON` returns `false`) → render
  plain, no crash.
- Narrow / short terminals → clamp widths/heights to minimums (≥20 wide, ≥1 tall).
- Resize in detail → propagate `WindowSizeMsg`, re-clamp scroll against
  recomputed visual-line count.
- Tab switch resets scroll to 0; scroll re-clamped every render.
- Long amenity names → existing `truncate`.

## Testing

- `internal/tui/wrap_test.go`: wrap boundaries, continuation line numbers
  (`sourceNum`), empty input, width ≤ 0.
- `cmd/deploymentcell`: keep existing `toPrettyJSON` / `workflowLine` tests; add
  tests for a pure `rawContentForTab(status, view)` helper (returns the
  unhighlighted string per selected view) and a `maxScroll(total, bodyH)` clamp
  helper.
- Detail `Update`/`View` stay thin and delegate to these tested helpers.
