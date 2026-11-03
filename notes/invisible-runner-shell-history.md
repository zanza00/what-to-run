# Invisible runner: resolution at Enter and the history entry

Investigated 2026-11-03 on `explore/invisible-runner`, read-only: tmux 3.8 panes
driving zsh 5.9.2 and bash 5.3.20, isolated HISTFILEs, a stub resolver resolving
only `build`; prototypes in `/tmp/wtr-exp/`. `[v]` = verified that day on this
machine, `[c]` = cited. Goal: `wtr build` + Enter asks a synchronous
`wtr resolve <word>` (exit 0 + one command line iff exactly one task matches,
no TTY use), runs the resolved line, records that line in history, and leaves
any other shape untouched. The resolver does not exist yet (today `flag.Parse`
drops positionals — see `notes/invisible-runner-repo-map.md`).

## zsh

### 1. Wrap accept-line `[v]` — recommended

    _wtr_accept_line() {
      emulate -L zsh
      local -a words=(${(z)BUFFER})
      local resolved word=${(Q)words[2]}
      if (( ${#words} == 2 )) && [[ ${words[1]} == wtr ]] && [[ $word != -* ]] \
     && [[ $word != *[\$\'\"\ ]* && $word != *\\* && $BUFFER != *$'\n'* ]] \
     && resolved=$(wtr resolve "$word"); then BUFFER=$resolved; fi
      if (( ${+widgets[wtr-orig-accept-line]} )); then zle wtr-orig-accept-line
      else zle .accept-line; fi
    }
    typeset -g wtr_prev=${widgets[accept-line]}
    if [[ $wtr_prev == user:* ]]; then zle -N wtr-orig-accept-line ${wtr_prev#user:}; fi
    zle -N accept-line _wtr_accept_line

- The rewritten buffer is the line accepted, executed, and recorded: `wtr build`
  never appears, in `fc -l` and in HISTFILE after exit `[v]`. zsh commits the
  accepted line at accept, before execution: `$history[$HISTCMD]` and
  `fc -ln $HISTCMD` show the running line; `fc -l -1` lists the event *before*
  the current one, and `-0` is invalid `[v]`.
- `$widgets[accept-line]` is the plain string `builtin` in zsh 5.9.2, not
  `builtin:accept-line`; user widgets are `user:<fn>` `[v]`. Guarding with
  `!= builtin:*` misfires and registers a function named `builtin` `[v]`.
- Composability: capture the previous user widget and chain as above; verified in
  both load orders with a capture-and-chain stand-in plugin — history is correct
  either way `[v]`. zsh-autosuggestions wraps accept-line with this exact pattern
  (dispatches on `builtin` / `user:*` / `completion:*`, invokes via
  `zle <name> -- "$@"`) `[v: plugin source, local antidote cache]`; atuin's zsh
  script does not wrap accept-line, it only binds Ctrl-R and arrows `[v: atuin
  init zsh]`.

### 2. `wtr()` function + `print -s` `[v]` — fallback only

    wtr() {
      local resolved
      if (( $# == 1 )) && [[ $1 != -* ]] && resolved=$(wtr resolve "$1"); then
        print -s -- "$resolved"; eval " $resolved"; return $?
      fi
      command wtr "$@"
    }

- Both lines land in history — the typed `wtr build` (committed at accept), then
  the resolved line; both persist to HISTFILE `[v]`. No supported way to replace
  the just-accepted entry from a running command: `$history` is read-only
  (`read-only variable: history`), `fc` cannot delete `[v]`. Use this shape only
  where zle is missing: it cannot meet the spec alone.

### zsh pitfalls

- Flags and any other 2+-word or multiline shape: never rewrite; bare `wtr`
  stays untouched and opens the picker as today `[v]`. Cursor position is
  irrelevant: rewrite only immediately before accepting, never touch BUFFER on
  the no-rewrite path `[v]`.
- `${(z)}` + `${(Q)}` mirror the argv the real binary sees, so `wtr "build"`
  and `wtr bu\ild` rewrite `[v]`; the guard also rejects newlines, trailing
  backslashes and any `$` (`${(Q)}` does not expand parameters), so
  continuations and parameter words pass through.
- Testing gotcha: `zsh -f` saves no history at exit (NO_RCS — man zsh) `[v]`,
  and a fresh HOME summons the first-run wizard, eating the first keys `[v]`.

## bash

### 1. Enter macro + bind -x + tail `[v]` — recommended

The idiom `internal/shell/bashinit.go` ships for the picker key, applied to
Enter: rebind `\C-m` (and `\C-j`) in emacs-standard/vi-insert/vi-command to a
macro `RUN+TAIL`; the bind -x widget rewrites `READLINE_LINE` and the tail is
bound to `accept-line`, so readline itself submits the rewritten line (wiring as
in bashinit.go, bash 4.3+):

    _wtr_enter_widget() {
      _wtr_tail accept-line   # first: Enter must keep working no matter what
      local word=${READLINE_LINE#wtr } resolved
      if [[ $READLINE_LINE == wtr\ * && -n $word && $word != -* \
     && $word != *[[:space:]\'\"\\$]* && $READLINE_LINE == "${READLINE_LINE%%$'\n'*}" ]] \
     && resolved=$(wtr resolve "$word"); then
        READLINE_LINE=$resolved; READLINE_POINT=${#READLINE_LINE}
      fi
    }

- History records the resolved line and not `wtr build` `[v]`; plain lines,
  empty Enter, and unclosed-quote continuation behave normally `[v]`.
- `bind -x` alone cannot accept: it rewrites the line and leaves it at the
  prompt `[v]`. The naive `bind '"\C-m": "\C-xw\C-m"'` recurses (the macro's
  `\C-m` re-expands into itself; a counter-guard stopped it after 5 rounds), so
  the tail must be a different, invisible sequence `[v]`. Every Enter pays the
  dispatch, and a slow resolver stalls Enter: keep the fast path fork-free.

### 2. `wtr()` function + `history -s` + `eval` `[v]` — no-rebind alternative

    wtr() {
      local resolved
      if (( $# == 1 )) && [[ $1 != -* ]] && resolved=$(wtr resolve "$1"); then
        local lastno; read -r lastno _ <<< "$(history 1)"
        (( lastno == HISTCMD )) && history -s -- "$resolved"   # skip if history off (HISTCMD=0)
        eval " $resolved"; return
      fi
      command wtr "$@"
    }

- Bash records the accepted line before execution; HISTCMD is its index `[v]`.
  Key finding: `history -s` **replaces the running line's entry** — it does not
  append (a standalone `history -s X` only looks like an append because it
  replaces the running `history -s X` line) `[v]`. So `history -s -- "$resolved"`
  alone swaps `wtr build` for the resolved line; no `history -d` needed `[v]`.
- Never pair `history -d $HISTCMD` with a following `history -s`: once the
  running entry is deleted, `-s` replaces the *last* entry and clobbers the
  previous command `[v]`. `-d` alone does delete the running line `[v]`.
- With `HISTCONTROL=ignorespace` a leading-space line is never recorded; `-s`
  then appends benignly, nothing is clobbered `[v]`. The `lastno == HISTCMD`
  guard skips the rewrite when the running line is not the last entry (history
  disabled, HISTCMD=0 `[v]`).
- Persistence verified with the default overwrite-at-exit and with
  `shopt -s histappend` `[v]`. A `history -a; history -n` PROMPT_COMMAND recipe
  flushes/merges our entry harmlessly; a `history -c; history -r` reload would
  drop in-memory-only entries before they reach the file `[c]`.

### Prior art (bash)

atuin embeds bash-preexec (DEBUG-trap + PROMPT_COMMAND) and its Ctrl-R widget
sets `READLINE_LINE`/`READLINE_POINT` plus a macro-chain key bound to
accept-line, the same route as our tail `[v: atuin init bash]`. fzf pushes its
selection into `READLINE_LINE` and only deletes entries on request
(`builtin history -d` + `history -w` under histappend) `[v: fzf 0.74.5]`. direnv
hooks PROMPT_COMMAND only `[v]`; mcfly is not installed here `[c]`. None of them
rewrites the accepted line's history entry; they only push text into the buffer.

## Shared pitfalls

- Quoting: the resolver returns the line in the shell's dialect; the widget path
  stores the exact text verbatim, but the bash function path `eval`s the line —
  reject any word with quotes, backslashes, spaces or `$` (both sketches do),
  since parsing it faithfully would risk executing substitutions; when in doubt,
  do not rewrite.
- Fallback on ambiguity: exit nonzero and leave the line untouched; the picker
  opens as it does today (bare `wtr`), or the binary runs. Keep the rewrite
  opt-in and the shape checks conservative: flags, extra words, and multiline
  buffers all pass through, and on any resolver error accept the original line.
- The resolver runs synchronously on Enter, captured on stdout, and must not
  touch the TTY; a timeout that falls back to leaving the line untouched is
  safer than a stuck Enter key.
