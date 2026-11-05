# Agent process

How agents work in this repo. Decisions by the maintainer, dated. The repo
rules that live in `AGENTS.md` (draft PRs, stacked PRs, manual merges,
`task all`, Simplified Technical English comments) are not repeated here.

- **2026-11-02**: work is delegated to fresh-context subagents; the main
  session coordinates, writes specs and verifies (chief-of-staff mode).
  The user asked for this explicitly after the box-module grilling.
- **2026-11-02**: implementers must use the `implement` skill: TDD at
  pre-agreed seams, regular typechecks, the full suite once at the end, and
  the implementer itself calls `code-review` on its own work before
  committing. The first box-module implementer did NOT run that final
  code-review (it was not in its brief); the maintainer noticed. Every
  implementer brief from now on says: load `implement` and follow it.
- **2026-11-02**: the agent wiki lives in `notes/` beside the research
  notes; `notes/index.md` is its index. One topic per file, dated claims,
  commit citations, supersede instead of rewrite.
- **2026-11-02**: domain terms crystallize into `GLOSSARY.md` at the repo
  root (created with box/modal/placed box in `494da47`) via the
  `domain-modeling` skill, as decisions land — not in bulk.
- **2026-11-01**: before any Go task, load `golang-how-to` (repo rule,
  AGENTS.md); before UI code, also `bubbletea`.
- **2026-11-03**: subagent briefs must inline the implement discipline.
  Children in this runtime cannot load `implement` through the Skill tool,
  so briefs say: read `.agents/skills/implement/SKILL.md` and follow it.
  Children also have no subagent spawner, so the `code-review` skill's two
  axes run in-context inside a child; only the main session can spawn the
  axes in parallel. Learned in the AUR-removal implement-spec run (PR
  #110), where every implementer hit both limits.
- **2026-11-05**: a spec names a new type, file and field with the one
  word its comments' prose will use, and the implementer checks the word
  against the prose before the first commit (the "persisting rule"). The
  maintainer ratified the rule after the piece-5 review: the piece-3 and
  piece-4 Standards reviews both caught a spec-named operation drifting
  from its comment prose (the `reprise`/`settle` case), and the rule,
  written into the FP5 spec and brief, took the naming findings to zero.

## Recurring gotchas

- **2026-11-02**: lipgloss v2 `Style.Render("")` is NOT empty output: it
  emits a zero-width escape pair (e.g. `\x1b[38;5;241m\x1b[m`). A byte
  comparison sees it; an ANSI-stripped assertion does not. The PR2 sweep
  caught 62,524 differences of exactly this kind that the whole test
  suite was blind to.
- **2026-11-02** (retro of the box-implementer session): after dropping
  authored files (specs, docs, GLOSSARY entries) into the tree, run
  `task check-md` before dispatching an implementer. `task all` runs
  dprint *fmt*, which silently fixes; CI runs dprint *check*, which fails —
  an implementer handed a non-clean file either reddens CI or silently
  edits your file (the 2026-11-02 GLOSSARY.md `_Avoid_` case).
- **2026-11-02**: dprint config enforces `emphasisKind: "asterisks"`: write
  `*Avoid*`, not `_Avoid_`, in Markdown (the first GLOSSARY.md draft failed
  `task all` on this).
- **2026-11-02**: the invoked binary name appears in the picker title
  (AGENTS.md rule): when byte-comparing two builds, invoke both under the
  same filename or the title line differs.
- **2026-11-02**: `git diff main --stat` shows unstaged working-tree state
  too; use the three-dot `git diff main...HEAD` to review only commits.
- **2026-11-03**: create a ticket's worktree before spawning its
  implementer. The AUR-removal implementer arrived at a missing worktree and
  had to build it itself; its brief assumed the worktree existed. The
  implement-spec dispatch order is: integration branch, ticket worktree,
  then the brief.
- **2026-11-05**: a spec's descriptive prose can coin a domain word the
  same way its names do. The FP1 spec said "the register" and the
  collision with the sources registry and the color register surfaced
  only in the worktree, at Standards-review time. Check a spec's
  umbrella vocabulary against GLOSSARY.md and the existing terms before
  dispatching an implementer, the way the persisting rule checks names
  before the first commit.
