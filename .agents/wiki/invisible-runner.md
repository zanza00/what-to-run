# Invisible runner

The decision record for the invisible runner: a typed `wtr build` that matches
exactly one task becomes the task's real command line before the shell accepts
it, so the shell executes and records `npm run build`, and `wtr build` never
lands in the history. The research lives in notes/invisible-runner-repo-map.md
and notes/invisible-runner-shell-history.md (both 2026-11-03); this page holds
the verdict, the design direction and the open decisions. On 2026-11-03 no code
for this exists, on branch `explore/invisible-runner`.

## Feasibility (2026-11-03, the chief of staff)

Feasible, and the mechanics are verified, not guessed: the shell-history note
marks every claim `[v]` (tmux-driven zsh 5.9.2 and bash 5.3.20 on this machine)
or `[c]` cited.

- zsh: wrap `accept-line` with a `zle -N` widget that rewrites `BUFFER` for the
  exact shape `wtr <word>` and chains to the previous widget. The rewritten
  buffer is the line accepted, executed and recorded, in `fc -l` and in
  HISTFILE. The capture-and-chain pattern is what zsh-autosuggestions uses,
  so both load orders compose. One gotcha: `$widgets[accept-line]` is the
  plain string `builtin` in zsh 5.9.2, not `builtin:accept-line`.
- bash: the Enter macro + `bind -x` + tail idiom that `internal/shell` already
  ships for the picker key, applied to `\C-m`: the widget rewrites
  `READLINE_LINE`, the invisible tail is bound to `accept-line`, and readline
  submits the rewritten line. The naive direct rebind recurses; atuin's Ctrl-R
  takes the same macro-tail route. A no-rebind alternative exists: a `wtr()`
  function where one `history -s` replaces the running line's entry — never
  pair it with `history -d`, which makes `-s` clobber the previous command.
- PowerShell: `GetBufferState`/`AcceptLine` and a `-Chord Enter` handler are
  the building blocks (repo map, section 1); the history behavior is
  unverified on 2026-11-03.

The `wtr()` function + `print -s` fallback fails the spec in zsh: both lines
land in history, and no supported way exists to replace the just-accepted
entry from a running command.

## The repo gaps (2026-11-03, the repo map)

- No subcommand resolves a query without the TUI: `wtr build` today opens the
  picker with an empty filter, because `flag.Parse` drops positionals
  (main.go:66).
- No uniqueness decision anywhere; the scorer only ranks (tui.go:865).
- No accept hook in any widget; only the bound key is intercepted.
- rc tasks lose the raw command (`sh -c` wrapping, discover.go:297) and
  `Task.Dir` cannot ride a bare command line.
- README.md:15-16 already promises that selected commands run through the
  user's shell and stay in history: the feature extends that promise.

## The design direction (2026-11-03, the chief of staff)

Option A of the repo map: a thin `wtr resolve <name>` subcommand, dispatched
next to `init`, backed by an exported `tui.Resolve` beside `Run`/`Preview`:
`Env.discover`, then a pure exact-name/require-unique helper beside
`rankTasks`. It prints `CommandLineFor` and exits 0 on the unique match, and
prints nothing and exits 1 otherwise — that is the hook contract "leave the
line untouched". Options B (the rule in core) and C (reuse the widget
protocol) are documented as trade-offs in the note. The hooks ship inside the
existing init scripts (`ZshInit`, `BashInit`, `PowerShellInit`) behind an
opt-in.

## Risks

The hook owns Enter: every command the user accepts passes through it, and a
bug or a slow resolver can stall or break Enter globally. Mitigations from
the shell note: bind the tail to `accept-line` first, keep the fast path
fork-free (the shape check is string-only; only `wtr <word>` lines fork the
resolver), fall back to untouched acceptance on any resolver error or
timeout, and gate the rewrite conservatively — flags, quotes, backslashes,
newlines, `$` and extra words all pass through.

## Open decisions (for the maintainer)

- Scope: zsh and bash first, or all three shells in v1.
- rc tasks with `cwd`: emit `cd <dir> && <line>`, or skip Dir tasks in v1.
- Fallback UX: whether `wtr <query>` should open the picker pre-filtered with
  the query, since the resolver makes the positional arg meaningful.
- The opt-in surface: an `init` flag, an env var, or a separate bind recipe.

## Next step

A spec (to-spec), then tickets (to-tickets), then implement-spec; the
procedures live on disk at `.agents/skills/to-spec/SKILL.md`,
`.agents/skills/to-tickets/SKILL.md` and `.agents/skills/implement-spec/SKILL.md`
(checked 2026-11-03). Tickets go under `.scratch/`, the convention from the
Python task sources stack, because the repo has no configured issue tracker.
