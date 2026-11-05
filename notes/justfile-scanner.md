# Justfile scanner and split (piece 9)

The justfile parser's five copies of the same quote-aware byte walk die
into one scanner, and the fused flow splits into its three homes: the
pure document scan, the Files-port traversal and the selection policy
each own their contracts in their signatures and headers, the dead
error half of the interface dies, and the 78-line prose comment
distributes to the owners. Grilled with the maintainer on 2026-11-05
(eight questions, every recommendation accepted — and the spec's
uniform claims were verified per site before dispatch, applying the
agent-process gotcha of the same day); the synthesis lives in the
local tracker (`.scratch/justfile-scanner/spec.md`); this page is the
durable record. Candidate: "justfile: five quote scanners and a
fused flow" in [arch-review-2026-11](arch-review-2026-11.md), piece 9
of the [attack plan](attack-plan.md). Shipped on 2026-11-05 as PR #134
(merge commit `f0f9b2c`, the single implementer commit `9cb1e4d`).

## What was broken (all at 2026-11-05, at `cfc203a`)

- Five copies of the same walk, verified per site: every scanner
  repeated the core (walk the bytes, skip the escaped byte, toggle the
  quote on `'` and `"`) and differed only in what it reports —
  `justContinues` (:582, stops at `#`, answers whether a quote stayed
  open), `justHeaderColon` (:637, stops at `#`, reports a `:` outside
  quotes excluding `:=`), `justAttributeEnd` (:714, reports the
  position of `]`), `stripJustComment` (:738, returns the prefix
  before `#`), `splitJustAttributes` (:763, splits on the commas).
  The family was deliberate — `justHeaderColon`'s prose said it read
  quotes "like stripJustComment reads them" — but nothing shared the
  walk.
- `parseJustfileFor` fused three concerns that existed as functions
  under one roof: the pure scan (`justDocumentOf`, :174), the
  Files-port traversal (`justCollect`, :308) and the selection policy
  (`selectJustTasks`, :449). Its `error` return was dead (:145,
  pinned by TestParseJustfileNeverErrors), and the real contract was a
  78-line prose comment (:66-143).

## The decisions (2026-11-05)

- **Scanner and split in one piece**, one implementer, one commit: the
  scanner lives in the scanning half, and the split creates the homes
  the laws distribute to.
- **One walker, differences as data.** `justScan(text, stops string)
  (at int, open bool)` in `internal/core/just_scan.go` — the escape
  skip and the quote toggle live once; the callers hand their stops
  (`"#"` continues, `":#"` the header colon, `"]"` the attribute end,
  `"#"` the comment, `","` the split) and resume the scan on slices
  taken at stop positions, where no quote can be open. Each caller's
  comment keeps its own law: the `:=` exclusion stays
  `justHeaderColon`'s, the unterminated-quote answer stays
  `justContinues`'.
- **The split into the three homes** with honest signatures;
  `parseJustfileFor` becomes the thin composer, and the dead `error`
  dies with the signature (`TestParseJustfileNeverErrors` renamed
  `TestParseJustfileNeverFails` with its comment rewritten — the old
  name would lie about a signature with no error; four inputs and the
  nil-tasks pin unchanged).
- **The prose comment distributes to the owners**: each law moves to
  the header of the function that implements it, nothing stated
  twice, and the composer's header shrinks to the map.
- **The 81-case table passes case-for-case unchanged** (the
  attack-plan guardrail); the scaffolding adapts only where the
  vanished signature forbids the old shape.
- **Two legs of corpus differential** — the five old scanner bodies
  inlined against the walker, and the old `parseJustfileFor` against
  the split composer — plus the binary check and `task all`.
- **The `~/` reach-past stays out** ("do not widen Files for it"):
  `justImportPath` is untouched, piece 12's territory.

## The deletion taxonomy (2026-11-05, PR #134)

The 78 fused prose lines sorted honestly in review: 3 stayed with the
composer (the outcome, the imported-as-if-root phrase that
`justCollect`'s law supersedes, the four-line map), 11 were `//`
separators; ~32 laws moved into the three home headers; ~30 were
duplicate-law restatements whose satellite already stated the
identical law (the platform semantics to `justPlatformEnabled`, the
attribute list to `justAttributes`, the path resolution to
`justImportPath`, the nil-Files stop to `justCollect`'s body, the
continuation rules to `justLogicalLine`, the keyword grammar to
`justKeywordRe`/`justModuleRe`, the `:=` anchoring to
`justAssignmentRe`, the keyword-colon gate to `isJustSetting`); the
rest were dead cross-reference pointers ("read like justHeaderColon
reads them") — pointers, not laws, and `justScan`'s header is now the
single statement. No narration-only line existed.

## The proofs (2026-11-05, PR #134)

- **Scanner differential**: the five old bodies inlined verbatim from
  `git show cfc203a:internal/core/source_justfile.go`, driven beside
  `justScan` over 554 inputs (6 fixture files with the nested import
  trees, 25 synthetic documents, 40 edge strings: `#` inside quotes,
  escaped stops, backslash continuations, `:=`, `]` in doc strings,
  commas in quotes, triple-quoted strings, `~/` imports, module lines,
  unterminated quotes), each whole, per line and right-trimmed, × 5
  caller pairs, comparing exact booleans, indexes, prefixes and
  slices: **2,770 comparisons, 0 differences**. Mutation-checked: a
  broken `stripJustComment` and a broken `:=` resume both fail the
  harness, so it does not compare a function with itself.
- **Task-set differential**: the old `parseJustfileFor` inlined
  against the split composer over 123 drives (4 fixture roots through
  `osFiles`, 25 documents, 11 `mapFiles` trees: nested imports,
  diamond, cycle, module chains, a `~/` home import), each under
  linux and darwin — full `[]Task` with `DeepEqual`: **0
  differences**, the old error nil in all.
- **Integration**: both builds invoked as `wtr-preview` in the repo
  root under a fresh `XDG_CACHE_HOME` — `--preview` (2,417 B) and
  `--preview --theme nonexistent` (2,578 B) identical against
  `cfc203a`.
- **Permanent tests**: `just_scan_test.go`, 32 rows (the escape and
  quote edges per stop kind, red confirmed before the
  implementation). The 81-case table name-identical on both sides (77
  `name:` fields + 4 never-fail inputs); only 8 call sites shed the
  two-value form the signature forbids. `task all` green; the
  registry invariants and docs tests unmodified; `justImportPath`
  byte-identical.
- **The review verdicts.** The Spec axis: spec-satisfying — two
  precision nits closed in the amendment (the import law now lives
  once, at `justCollect`; the deletion taxonomy stated honestly
  above). The Standards axis: one hard term violation closed (the
  mixed "reading"/"parser" actor — the parser keeps the register's
  old word, "reading" stays the quote scan's) with four prose
  judgement calls (the inherited `i++` what-comment now says why —
  "the escaped byte cannot close the quote"; "functions own the
  contract" without the coined "homes"; `justCollect` "gathers the
  definitions"; `selectJustTasks`'s 46-word opening broken into
  single-idea sentences).

## What landed (2026-11-05, PR #134)

- `9cb1e4d` owns the just quote scan once and splits the flow: 4
  files, +358/-220 — `just_scan.go` (35 lines),
  `just_scan_test.go` (236), `source_justfile.go` (the five bodies
  and the fused prose die), `source_justfile_test.go` (47). `task
  all` green at that commit.
- `f0f9b2c` merged it into `main`. The piece-8 bookkeeping had landed
  first, with PR #133 (merge commit `556e4a5`).

If the code changes any decision above, add a dated claim here and mark the
old one superseded — do not rewrite it.
