# Widget setup

The widget opens the task picker with a keypress and places the selected
command in the prompt, so it runs through the shell and becomes part of the
terminal history.

## zsh and bash

Add the line for your shell to its rc file, then start a new shell (or evaluate
it in the current one). Use `wtr` instead of `what-to-run` if that is the
command name you installed.

```sh
# zsh: ~/.zshrc
eval "$(what-to-run init zsh)"
```

```sh
# bash 4.3 or later: ~/.bashrc
eval "$(what-to-run init bash)"
```

## PowerShell

Add this to `$PROFILE`:

```powershell
# PowerShell 5.1 or 7+
what-to-run init powershell | Out-String | Invoke-Expression
```

`Out-String` preserves the newlines and the `#` comments of the generated
script; piping `init` straight into `Invoke-Expression` would collapse it to one
line and comment out everything after the first `#`.

## Key binding

The widget is bound to **Ctrl-S**. Pass `--key` to `init` to choose another key,
for example `--key ctrl-g`. zsh and bash also accept a raw sequence as their
`bindkey`/`bind` takes it (`^[x` for zsh, `\e[15~` for bash); PowerShell takes
key names only (`ctrl-s`, `alt-x`), not raw sequences.

Ctrl-S and Ctrl-Q are the terminal's flow-control keys. The generated zsh and
bash integrations run `stty -ixon` when one of them is bound, so the key
reaches the widget; `stty ixon` restores the default.

## Shell dialect

The picker shows one command line per task and the inline widget places it in
the prompt, so every argument is quoted for the shell that reads it back. The
zsh and bash widgets pass `--shell posix`, the PowerShell widget passes
`--shell powershell`, and a direct run uses `auto`: PowerShell on Windows,
POSIX elsewhere. `--shell cmd` targets cmd.exe.

cmd.exe expands `%VAR%` and `!VAR!` even inside double quotes, and a newline
cannot be represented in one command line, so arguments containing them cannot
be reproduced exactly. When an exact round-trip matters, PowerShell is the
better choice.

## Running the picker

`wtr` (or `what-to-run`) runs the picker without the shell widget:

```sh
wtr          # full-screen picker
wtr --inline # inline picker, below the current prompt
```

Full-screen mode is the default. When you run a task there, the picker returns
after the task finishes, and the output remains in the terminal scrollback.
Inline mode is intended for shell widgets and requires an interactive terminal;
without one it warns and falls back to full-screen.

If the current directory declares no tasks, the inline picker stays open on a
message listing the files it looked for (close it with `enter` or `esc`), while
a full-screen run reports the error and exits.

With the shell widget, `enter` runs the selected command and `tab` only inserts
it in the prompt. When `wtr --inline` runs in a terminal without the widget, it
executes the command itself, or prints the command line after `tab`.
