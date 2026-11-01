# Widget setup

Press `ctrl-o` ([or your own key](#key-binding)) to open the inline picker below
your prompt, `enter` to run the selected task, `tab` to put the command in the
prompt without running it, or `ctrl-f` to toggle fullscreen.

## zsh and bash

Add the line for your shell to its rc file, then start a new shell (or `source`
the file in the current shell):

```sh
# zsh: ~/.zshrc
eval "$(wtr init zsh)"

# bash 4.3+: ~/.bashrc
eval "$(wtr init bash)"
```

## PowerShell

Add this line to `$PROFILE`, then start a new session (or run `. $PROFILE`):

```powershell
# PowerShell 5.1 or 7+
wtr init powershell | Out-String | Invoke-Expression
```

On Windows, the widget requires PowerShell.

## Key binding

To change the shortcut from `ctrl-o`, pass `--key` to `init`:

```sh
eval "$(wtr init zsh --key ctrl-g)"
```

In zsh and bash, `--key` also accepts raw key sequences in the format that
`bindkey` and `bind` expect (`^[x` in zsh, `\e[15~` in bash). PowerShell
accepts key names only (`ctrl-o`, `alt-x`).

## Running without the widget

You can also run `wtr` directly as a command:

```sh
wtr          # fullscreen picker
wtr --inline # inline picker below the prompt
```
