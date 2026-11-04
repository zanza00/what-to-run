# Warning identity (piece 1)

The warning becomes a typed value with its own identity — a kind and a
dismissal key — instead of a bare error whose text doubles as the identity.
Grilled with the maintainer on 2026-11-04 (eight questions, one re-asked);
the full spec is published to the local tracker
(`.scratch/warning-identity/spec.md`); this page is the durable record.
Candidate: "Warning identity" in
[arch-review-2026-11](arch-review-2026-11.md), piece 1 of the
[attack plan](attack-plan.md). Terms: `GLOSSARY.md` (warning, warning kind,
dismissal key).

## What was broken (all at 2026-11-04, pre-change)

- `warningKey` (warning_modal.go) normalizes the message text and doubles as
  both display and dismissed-key: any reword in any release silently
  un-dismisses every dismissal of that warning.
- The source badge names a file relative to the start directory
  (discover.go:218-231: `justfile` from the root, `../justfile` from a
  subdir), so the same broken file changes identity with the subdirectory.
- `hasThemeWarning` (tui.go:748-757) substring-matches "theme" in the
  message; any warning mentioning the word triggers the theme-save
  re-discovery.
- A bad `disable` entry produces two warnings today: the rc loader's
  validation and core's enabled-sources safety net (config.go:263,
  discover.go:280), with different texts, so dismissing one leaves the
  other.
- The dismissal of one warning reads the cache file three times
  (load + read inside save, plus the reload in the command).

## The decisions (2026-11-04)

- Typed `Warning` in core — kind + dismissal key + message — travelling the
  whole seam as a warning slice: the rc loader, discovery, and the picker's
  two synthetic warnings (unknown theme flag, unusable shell flag). It
  implements `error` with the message. Config already imports core, so the
  type lives there.
- The display remains the message, byte-identical to today's
  whitespace-normalized text: box body, count, startup open, no-tasks
  error. The identity is never displayed. No differential sweep owed: no
  frame or display byte changes.
- Kinds: a closed set of five subjects — `theme`, `shell`, `rc`, `source`,
  `registry`. The kind names the subject, not the origin: theme warnings
  from the flag and from the rc file share the kind, which is what the
  theme-reload guard asks.
- Key grammar `kind:subject[:detail]`, one line, no spaces, offender values
  sanitized. The message wording never enters the key.
- Granularity: `source` keys on source ID + leaf name with the parent-walk
  prefix stripped (one broken file, one identity, from any subdirectory; no
  read/parse distinction), `rc` on file origin + problem + offending value
  (a new problem still warns), `theme` on origin + name, `shell` and
  `registry` on the raw stored value. Known accepted collision: two broken
  files of one source sharing a leaf name share the entry.
- The picker dedupes by key at list assembly; the first occurrence's message
  wins the display. This is what collapses the `disable` twin.
- `hasThemeWarning` becomes a kind check; the substring match dies.
- Migration: lookup duale — an entry matches by key or by legacy text (the
  normalized message an old version stored). No startup reordering, no
  eager rewrite; saves stay append-only with the new key; the prune on a
  re-discovery compares both forms and removes healed warnings' entries
  under today's rule. Honesty: entries already drifted before this change
  match nothing today and may pop once more; everything that matches today
  keeps matching. The file may hold both forms mixed indefinitely.
- Scope riders settled with the piece: the triple read dies (save reads
  once, writes once, returns the set; the dismiss path does not reload);
  CHANGELOG under `[Unreleased]` (dismissals survive releases and
  subdirectory moves; one warning for a bad `disable` entry); docs untouched
  (the cache file's documented path and behaviour stay true); one draft PR
  based on `main` (first piece, nothing stacked); no tickets, the piece is
  small.

## Ratified after the code review (2026-11-04)

The two-axis code review of PR #116 surfaced two edge cases the grilling had
not covered. The maintainer ratified both on 2026-11-04; no code change
followed from them.

- 2026-11-04: a wrong-typed `disable` entry carries no offending value, so
  its key ends at the problem (`rc:<origin>:disable-entry`,
  `NewRCWarning` in config.go). Two such entries in one file collapse into
  one warning, and a wrong-typed entry added later to the same file does not
  warn again after a dismissal. The alternative — keying on a rendering of
  the non-string value — would name something the user cannot recognize in
  the dismissed-warnings file.
- 2026-11-04: when the parent search cannot resolve its starting directory,
  the warning keys on that directory in place of the leaf name
  (discover.go, `NewSourceWarning`), as its comment says. This is the one
  source warning whose identity varies with the starting subdirectory.
  Ratified because every alternative is worse: a source-only key would let
  two different unresolvable directories share one dismissal, and keying on
  the error text would put the message into the key.

## Verification plan

`task all` before every commit; the glyph-count tests of the box and the
no-tasks screen pass unmodified (the display's byte-identity). New tests at
the two producer seams and the dismissal helpers: key stability across a
reword and across a subdirectory, the legacy-text lookup, the prune of both
forms, the `disable` twin collapsing to one warning, the theme guard on
kind, and the single-read save. Implementation follows the implement skill
(TDD, own two-axis code review before committing).

If the implementation changes any decision above, add a dated claim here and
mark the old one superseded — do not rewrite it.
