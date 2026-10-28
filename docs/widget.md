# Widget setup

The widget opens a fuzzy picker with the commands it finds in the current
directory.

## zsh and bash

Add the line for your shell to its rc file, then start a new shell. To use it
right away, `source` the file in the current shell.

```sh
# zsh: ~/.zshrc
eval "$(wtr init zsh)"
```

```sh
# bash 4.3 or later: ~/.bashrc
eval "$(wtr init bash)"
```

## Windows

### PowerShell

Add this to `$PROFILE`, then start a new PowerShell session. To load the
profile right away, run `. $PROFILE` in the current session.

```powershell
# PowerShell 5.1 or 7+
wtr init powershell | Out-String | Invoke-Expression
```

### cmd.exe

The widget needs PowerShell. `wtr` also runs from cmd.exe, but its quoting is
limited: use PowerShell when an argument must survive an exact round trip.

## Key binding

The widget is bound to **Ctrl-O** by default.

To change it, pass `--key` to `init`, for example
`wtr init zsh --key ctrl-g`. zsh and bash also accept a raw sequence in the
form their `bindkey` or `bind` takes (`^[x` for zsh, `\e[15~` for bash).
PowerShell takes key names only (`ctrl-o`, `alt-x`), not raw sequences.

## Fullscreen

With the widget open, press `ctrl-f` to expand the inline picker to fullscreen.
Press it again to go back.

## Running the picker

`wtr` runs the picker without the shell widget:

```sh
wtr          # fullscreen picker
```

To get the same inline picker without the widget, use `--inline`:

```sh
wtr --inline # inline picker, below the current prompt
```

If the current directory declares no tasks, the inline picker stays open on a
message listing the files it looked for. Close it with `enter` or `esc`. A
fullscreen run reports the error and exits.

With the widget, `enter` runs the selected command and `tab` only inserts it in
the prompt. Without the widget, `wtr --inline` runs the command itself, and
`tab` prints the command line instead.
