---
name: docs-writing
description: Write or edit README.md and docs/*.md, or document a new task source.
---

# Write the what-to-run docs

## Ownership

Every fact has a **single source of truth**: one file states it in full, and every other page links to that file by title (`see [Configuration](docs/config.md)`).

| File | Owns |
| --- | --- |
| `README.md` | pitch, install shortcuts, widget one-liner, source bullet list, config summary, inspiration, license |
| `docs/install.md` | install channels, setup scripts, release file naming |
| `docs/widget.md` | zsh, bash and PowerShell widget setup, key binding, fullscreen toggle, direct runs |
| `docs/config.md` | rc schema, per-project overrides, precedence, best-effort parsing, warnings, themes and theme flags |
| `docs/task-sources.md` | source registry table and per-source caveats |
| `docs/development.md` | prerequisites, dev tasks, local build, README demo |

When adding a task source, add one row to `docs/task-sources.md` and one bullet to `README.md` (`## Task sources`) in `defaultSources` registry order. Add a `##` section in `docs/task-sources.md` only when the table cannot hold the caveats.

## Glossary

Use these exact domain terms across all docs.

| Term | Meaning |
| --- | --- |
| task | a runnable command that a source declares |
| source | one family of project files that the picker reads |
| discovery | the process that finds a source file and parses its tasks |
| picker | the TUI list of tasks |
| row | one line of the picker, for one task |
| badge | the source file shown at the right of a row |
| widget | the shell function that opens the picker from a key |
| inline | the picker drawn below the prompt |
| fullscreen | the picker that takes the whole terminal |
| rc file | the TOML files the picker reads: the global file and the project file |
| global file | `$XDG_CONFIG_HOME/what-to-run/config.toml`, or `%AppData%\what-to-run\config.toml` on Windows |
| project file | `.what-to-run.toml` in a project |
| shell dialect | the quoting rules for `sh`, `cmd` and `powershell` |
| theme | a named look from the registry |

## Structure and voice

Order sections by what the reader does. Read `docs/widget.md` as the structural and stylistic model.

Write in **CEFR B2** American English:

- **One idea per sentence**, most under 25 words, with varied sentence length. Keep a cause with its effect in one sentence ("`cmd.exe` handles quotes in a limited way, so some arguments can change.").
- **Active voice** with the component as subject or "you" as the reader: "The picker reads the rc file." / "Add this line to your rc file."
- **Condition first**: "If there are no tasks in the current directory, the inline picker stays open."
- **Action in the verb**: "Ctrl-O opens the widget by default." / "To change the key, add `--key` to the `init` command."
- **Plain verbs and full forms**: write "shows", "puts", "add", "stay the same", and full forms ("do not", "it is") with periods instead of semicolons.
- **Command name**: write `wtr` in user docs (`./bin/wtr` belongs only in `docs/development.md`).

## Completion criteria

Every doc change is done when all four checks hold:

1. **Single owner**: no fact is stated in two files; cross-references link to `[Title](path#anchor)` and every target heading exists.
2. **Ground truth**: every flag, subcommand and key in the touched sections matches `wtr --help` and `cmd/wtr`.
3. **Formatting**: `task check-md` exits 0 with no diff.
4. **Source parity**: `go test ./cmd/wtr` passes.
