# Widget setup

The widget opens a fuzzy picker that lists the commands it finds in the current
directory.

## zsh and bash

Add the line for your shell to its rc file, then start a new shell. To use the
widget right away, `source` the file in the current shell.

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

On Windows, the widget works only in PowerShell. `wtr` also runs from cmd.exe,
but cmd.exe handles quotes in a limited way, so some arguments can change. Use
PowerShell when the exact command matters.

## Key binding

Ctrl-O opens the widget by default.

To change the key, add `--key` to the `init` command, for example
`wtr init zsh --key ctrl-g`. In zsh and bash, you can also use a raw key
sequence, in the format that `bindkey` and `bind` expect: `^[x` for zsh,
`\e[15~` for bash. PowerShell accepts key names only (`ctrl-o`, `alt-x`), not
raw sequences.

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

If there are no tasks in the current directory, the inline picker stays open
and shows the files it looked for. Close it with `enter` or `esc`. A fullscreen
run shows the error and exits.

With the widget, `enter` runs the selected command and `tab` only puts it in
the prompt. Without the widget, `wtr --inline` runs the command itself, and
`tab` prints the command line instead.
