#!/usr/bin/env zsh
#
# Local dev harness: bind Ctrl-S to the local build of what-to-run.
# Source it, do not execute it:
#
#   make build
#   source scripts/ctrl-s.zsh
#
# Then pressing Ctrl-S on a prompt line opens the picker anchored on the line
# right below the prompt, filling the space down to the bottom of the screen
# (same idea as television's --inline). q, esc or ctrl+c close it.
#
# enter writes the chosen command at the prompt and runs it (it lands in the
# shell history like a typed command); tab only places it in the prompt, ready
# to be edited. The picker reports the choice on stdout (captured below) and
# renders on the terminal itself.
#
# Override the binary path with:
#   WTR_BIN=/path/to/binary source scripts/ctrl-s.zsh
# Extra flags for the widget, e.g. to see the managed area border:
#   WTR_FLAGS=--debug-area source scripts/ctrl-s.zsh
#
# NOTE: Ctrl-S is the XOFF flow-control character, so the harness disables
# IXON for the session (otherwise the terminal freezes instead of delivering
# the key to zle). Run `stty ixon` to restore the default behaviour.

typeset -g WTR_BIN="${WTR_BIN:-${${(%):-%x}:A:h:h}/bin/what-to-run}"
typeset -g WTR_FLAGS="${WTR_FLAGS:-}"

if [[ ! -x $WTR_BIN ]]; then
  print -u2 "ctrl-s.zsh: $WTR_BIN not found or not executable — run 'make build' first"
  return 1
fi

stty -ixon 2>/dev/null

_what_to_run_widget() {
  zle -I
  # zle runs commands with stdin not connected to the terminal; use /dev/tty
  # explicitly (same as television's shell integration). stdout is captured:
  # the picker renders on the terminal and prints its result here — a mode
  # line ("run" or "insert") followed by the command line.
  local out
  out=$("$WTR_BIN" --inline ${=WTR_FLAGS} < /dev/tty)

  local -a lines
  lines=("${(@f)out}")
  if (( ${#lines} >= 2 )); then
    case ${lines[1]} in
      run)
        BUFFER=${lines[2]}
        zle accept-line
        return
        ;;
      insert)
        LBUFFER+=${lines[2]}
        ;;
    esac
  fi
  zle reset-prompt
}

zle -N what-to-run-widget _what_to_run_widget
bindkey '^S' what-to-run-widget

print "ctrl-s.zsh: Ctrl-S now opens ${WTR_BIN} --inline (IXON off)"
