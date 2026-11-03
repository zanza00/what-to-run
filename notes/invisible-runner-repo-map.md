# Invisible runner — repo map

Mapped 2026-11-03 on branch `explore/invisible-runner`, read-only survey. Goal:
`wtr build` at the prompt becomes the task's real command line before acceptance.

## 1. The three shell widgets (internal/shell)

Each widget is a script that `wtr init <shell>` prints; `prog` is the invoked
name, `bin` the launched binary, `key` the combo (default `ctrl-o`, zshinit.go:14).

- zsh — `ZshInit` (zshinit.go:33). Install: `eval "$(wtr init zsh)"`, wrapped by
  `task bind` (Taskfile.yml:50). Sets `WTR_BIN`/`WTR_FLAGS` (zshinit.go:55-56).
- bash — `BashInit` (bashinit.go:38). Install: `eval "$(wtr init bash)"`, wrapped
  by `task bind-bash` (Taskfile.yml:57); needs bash 4.3+ (bashinit.go:27, 149).
- powershell — `PowerShellInit` (powershellinit.go:35). Install:
  `wtr init powershell | Out-String | Invoke-Expression`; no `task` target.

Protocol (fzf-style, identical in all three): the widget captures stdout and
launches `"$WTR_BIN" --inline --shell posix|powershell` (zshinit.go:80,
bashinit.go:88, powershellinit.go:74); the picker writes two lines — mode
(`run`/`insert`) then the command line — via `reportChoice` (tui.go:203-211,
called at tui.go:184-186). The widget then runs or inserts:

- zsh: `run` → `LBUFFER+=<line>; zle accept-line` (zshinit.go:89-91); `insert` →
  `LBUFFER+=` (zshinit.go:94). Reads `LBUFFER`+`RBUFFER` (the whole line);
  accepts via `zle accept-line`. No accept hook.
- bash: splice into `READLINE_LINE` at `READLINE_POINT` (bashinit.go:99-100).
  Accepting from shell code is impossible (bashinit.go:18-23): a macro tail
  (`WTR_TAIL`) is rebound to `accept-line` (bashinit.go:105, 110-118, 146-155).
- powershell: `Insert` + `AcceptLine` (powershellinit.go:87-88); buffer readable
  via `GetBufferState` (unused today); key bound via `Set-PSReadLineKeyHandler`
  (powershellinit.go:101-102).

## 2. cmd/wtr surface (main.go, 227 lines)

One subcommand: `init zsh|bash|powershell [--key combo]` (main.go:32, runInit
main.go:113). Flags: `--inline` (40), `--debug-area` (46), `--theme` (48),
`--shell` (51), `--list-themes` (56), `--version` (59), `--preview` (62);
`flag.Parse()` at main.go:66 drops positional args — `wtr build` today opens
the picker with an empty filter and ignores "build". No initial query/filter
flag exists: `tui.Options` (tui.go:43) has no query field and the filter starts
empty (tui.go:337). No non-interactive resolve path; the closest are
`--preview` (tui.go:214) and tab-in-a-terminal printing the line (tui.go:188-190).

## 3. internal/core: model and matching

`Task` (task.go:12-27): `Source`, `Name`, `Desc`, `Args []string`, `Dir`.
`Command()` builds the exec.Cmd (task.go:29). `CommandLineFor(shell)` renders
the argv as a shell line, quoted per dialect; PowerShell gets a `&` prefix when
the command word needs quotes (task.go:39-47).

Matching lives in the TUI, not core: `rankTasks` (tui.go:865) sorts by
`taskMatchScore` (tui.go:905) — case-insensitive subsequence over name, command
line and description; contiguous beats subsequence; exact name adds
`exactNameBonus` 500 and prefix 200 (tui.go:911, 938-939). There is no
"exactly one match" decision anywhere; every consumer gets a ranked list. A
resolver would filter `core.Discover` output (discover.go:323) by exact name
and require len==1 — names can still collide across sources (rc tasks shadow
same-named source tasks, discover.go:328-332, but two sources may share one).
Edge wiring to reuse: `Env`/`Env.discover()`/`productionEnv` (env.go:15, 49, 71).

## 4. rc file [[tasks]]

Fields: `name`, `command`, `description`, `cwd` (config.go:315-353,
docs/config.md:66-70). No `env` anywhere. `rcTasks` (discover.go:297) runs the
command through `sh -c`/`cmd /c` (shellCommandFor, shell.go:79-85) and resolves
`cwd` into `Task.Dir` (discover.go:313-315). So name→command-line is complete
for source tasks (quoting round-trips for posix/powershell; cmd is best-effort,
shell.go:93-96), but: rc tasks lose the raw command (`Args` is `sh -c <cmd>`),
and `Dir` cannot ride a bare command line — the rewrite needs a `cd` prefix.

## 5. Docs that already promise close behavior

README.md:15-16: "shell widget first: selected commands run through your shell
and stay in your terminal history" — the invisible runner extends this to
unmodified typed lines. docs/widget.md:3-6 documents ctrl-o/enter/tab/ctrl-f and
the `ABC=1` prefix kept before the command; :39-51 "running without the widget".
docs/task-sources.md and docs/config.md claim nothing about resolution.

## 6. Naming

`prog = progName(os.Args[0])` (main.go:23, progNameFor main.go:175) — basename,
`.exe` stripped on Windows — feeds usage, messages and the picker title.
`widgetBin` (main.go:202) computes what the init scripts embed as `WTR_BIN`: the
bare invoked name (PATH-resolved at keypress) or an absolute path. A shell hook
can therefore know the invocation name: compare the line's first word against
`$WTR_BIN` (basename when absolute).

## Gaps

1. No subcommand or flag resolves a query to a command without the TUI
   (main.go:66, 86).
2. No uniqueness decision ("exactly one match"); the scorer only ranks.
3. No accept-line/Enter hook in any widget; only the bound key is intercepted.
4. rc tasks lose the raw command in `Task`; `Dir` cannot ride a command line;
   PowerShell quoting needs the `&` prefix.
5. No quiet resolve contract: hooks want line-on-success, silence + exit 1 on
   no match; `reportChoice` always prints a mode line.

## Design options

A (recommended) — `resolve` subcommand + tui entry point. Dispatch next to
`init` (main.go:32): `wtr resolve [--shell posix] <name>`. New exported
`tui.Resolve(w, opts, query)` beside `Run`/`Preview` (tui.go:94/214):
resolveEnv → `Env.discover` → new pure helper `resolveName(tasks, query)` next
to `rankTasks` (exact name, require unique) → print `CommandLineFor`, exit 0;
no/ambiguous match → print nothing, exit 1. A task with `Dir` emits
`cd <quoted dir> && <line>` (or Dir tasks are skipped in v1). zsh hook in
`ZshInit`: `zle -N accept-line` wrapper checks first word == `$WTR_BIN`
basename, calls the resolver, rewrites `BUFFER`, else falls through to the
builtin. bash: bind Enter to the macro+tail chain of bashinit.go:146-155
(riskier: fights user Enter bindings). PowerShell: `-Chord Enter` handler with
`GetBufferState`/`AcceptLine`. Keeps cmd/wtr thin, tui owns the entry points,
Env stays the only edge wiring.

B — decision rule in core. `core.Resolve(tasks, name) (Task, bool)` beside
`Discover`; the tui entry point only wires env. Better layering (rule testable
without tui), at the cost of a second exported seam.

C — reuse the widget protocol: the resolver prints `reportChoice`'s `run\n<line>`
so hooks parse what widgets already parse. Less new surface, but drags insert
semantics where only run is needed, and two lines buy nothing over exit codes.
