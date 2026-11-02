# Box module (FP4)

One deep module draws the modal boxes; the callers keep content and
placement. Settled with the maintainer on 2026-11-01; implemented as PR1 +
PR2. Terms: see `GLOSSARY.md` at the repo root (as of `494da47`).

## The decisions (2026-11-01)

- Placement stays OUT of the box: `render` returns lines, the caller wraps
  them in a `placedBox`. Three rules (help-anchored, centered, own-x) are
  three callers' decisions.
- `innerW` is an INPUT (a content fact: the warning wraps text, the theme
  modal sizes a name column). The box owns the clamp `innerW ≤ w-4`, the
  guard, borders, padding.
- `nil` from `render` means "the space is too small"; it propagates to
  `boxes()`, which skips the box.
- The notice stays OUT: one adapter is a hypothetical seam; the requirement
  is two adapters for a real one.
- BYTE-IDENTICAL is a hard constraint: no visible change, existing tests
  pass unmodified, tests are only added.
- Everything package-private in `internal/tui`; `box.go` is a new file in
  the same package.
- lipgloss v2@v2.0.6 cannot embed a title in a border (verified in
  `borders.go` of the pinned module: `renderHorizontalEdge` cycles runes
  only), so the box stays hand-drawn.

## PR1 — warning modal (done)

As of `8674bdb` (branch `refactor/tui-box-module`, PR #84, draft,
`skip-changelog`):

- `internal/tui/box.go`: `boxChrome = 3`, `type box` (title, body, footer,
  border `color.Color`, innerW), `func (b box) render(h, w int) []string`.
- `internal/tui/warning_modal.go`: `warningModalChrome` and
  `warningModalFooter` are gone; `warningModalLines` keeps its signature and
  delegates the drawing. Content (title, hints, wrap, dismissal) unchanged.
- `internal/tui/box_test.go`: seam tests (guard, clamp, title cut, body
  pad/truncate/spacer, footer pad/drop, line count, border color).
- No existing test was edited. `task all` green on 2026-11-02. An
  end-to-end check built the binary from `main` and from the branch and
  byte-compared two `--preview` frames (80×24, warning modal open, same
  invoked name): identical, 2572 bytes.
- The blank spacer row is `body = append(body, "")`: an empty padded body
  row is byte-identical to the old explicit spacer.

## Code review of PR1 (2026-11-02)

Two-axis review (Standards + Spec, fresh subagents, diff `main...HEAD`).
Spec: faithful, no missing requirement, no wrong implementation; the clamp
and the footer-drop were verified as no-ops for PR1's caller. Standards: no
hard violation; two minor findings and four judgement calls. Resolved by
design: the footer-drop branch (box.go:59) is dead for PR1 but is PR2's
requirement — the theme modal drops hints wider than innerW, byte for byte;
and the theme modal's hand-drawn box is transitional duplication until PR2
merges. Actionable in PR1: telegraphic field comments in box.go:24-28
(AGENTS.md Comments rule wants full sentences) and the bare `w < 10` literal
next to its named twin `boxChrome+1`. Deferred to PR2: the stale
"overlays" word in the `boxes()` comment (tui.go:1295) contradicts the
placed-box glossary term introduced by this diff.

## PR2 — theme modal (done)

As of 2026-11-02 (branch `refactor/tui-box-theme-modal`, PR #85, draft,
`skip-changelog`, stacked on PR #84): commits `a4c8cc9` (the placed-box
wording in the `boxes()` comment, tui.go, comment only) and `3a0a419`
(the conversion: −41/+28 in theme_modal.go; theme_test.go carries only the
two `themeModalChrome → boxChrome` renames). The implementer's sweep:
627,200 combinations, zero differences. The chief of staff verified
independently with a second harness over the boundary region (h 1–10 ×
w 8–26 × 14 themes × 7 offsets × 3 terminal-color variants = 55,860
combinations, raw bytes): zero differences; `task all` green.

**2026-11-02 correction.** The claim "the box's drop-if-wider reproduces
`themeModalFooter` byte for byte" was FALSE. lipgloss v2 emits a zero-width
escape pair for `Style.Render("")`: the old `themeModalFooter` rendered the
dropped hints as `dim.Render("")` (escapes present), while the box replaced
a dropped footer with bare `""`. The PR2 sweep found 62,524 differing
combinations (h 4–32 × w 10–20, where innerW < 17), all invisible to the
test suite and to ANSI-stripped assertions. Corrected flow, decided by the
chief of staff: the CALLER pre-drops the plain hints
(`if lipgloss.Width(themeModalHints) > innerW { hints = "" }`) and passes
`m.styles.dim.Render(hints)`; the box's drop-if-wider stays as a defensive
guard, the same status as the clamp — byte-invisible for both callers. PR1
was never affected: the warning modal's hints always fit after
`warningModalHintText`, and the empty case is still dim-rendered by the
caller.

Dies: `themeModalChrome` (→ `boxChrome`), `themeModalFooter` (pad and drop
move into the box), the border assembly in `themeModalLines`. Stays:
`themeModalLayout`, `themeModalRow` (row painting is FP3 territory), the
windowing (FP6 territory), the title with the ↑/↓ counter, persistence.
`themeModalLines` keeps its signature; the footer hint drop reproduces
byte-for-byte via the box's drop-if-wider. After PR1 merges, rebase PR2
onto `main` (repo rule for stacks).

The full design detail lives in the session spec; this page is the durable
record. If PR2 changes any decision above, add a dated claim here and mark
the old one superseded — do not rewrite it.
