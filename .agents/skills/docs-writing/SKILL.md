---
name: docs-writing
description: Write what-to-run's user docs in the house voice, with one owner per fact, B2 English, the glossary and the finish checks. Use when writing or editing README.md or docs/*.md, adding a task source, or documenting a config key, theme, flag or shell widget.
---

# Write the what-to-run docs

`README.md` shows the way in and gives the shortest form of each step. `docs/`
holds the reference. `AGENTS.md` is for agents, not for users.

The house voice is B2 English (CEFR): clear for an upper-intermediate reader and
natural for a native speaker. `docs/widget.md` is the reference page.

Every fact has one owner: one file states it in full, and every other page links
to that file instead of repeating it.

## Where each fact belongs

| File | Owns | Links to |
| --- | --- | --- |
| `README.md` | pitch, install shortcuts, widget one-liner, source bullet list, config summary, inspiration, license | `docs/install.md`, `docs/widget.md`, `docs/task-sources.md`, `docs/config.md` |
| `docs/install.md` | every install channel, the setup scripts, upgrades from a two-name release | `docs/widget.md` from "Next steps" |
| `docs/widget.md` | zsh and bash setup, the Windows shells (PowerShell and cmd.exe), key binding, fullscreen toggle, direct runs | — |
| `docs/config.md` | the rc schema, per-project overrides, precedence, best-effort parsing, warnings, themes and theme flags | `docs/task-sources.md` for the source IDs |
| `docs/task-sources.md` | the registry table, the source ID rules, per-source notes | — |
| `docs/development.md` | prerequisites, tasks, README demo, local build, release secrets and pipeline | `.github/workflows/release.yml` for the pipeline source |
| `AGENTS.md` | rules for agents | — |

The README links to the owner of each detail and never repeats a setup step.

A new source needs one row in `docs/task-sources.md`, in registry order, and one
bullet in the README "Task sources" section: `go test ./cmd/wtr` fails when
either is missing. The test checks the table order, so keep the bullets in the
same order too. Add a `##` section in `docs/task-sources.md` only when the table
cannot hold the caveats, as the `package.json` and Taskfile sections do.

## Glossary

These are the project terms. Keep them in the prose: B2 covers general English,
not this vocabulary. Introduce "fuzzy picker" once, then write "picker".

| Term | Means | Do not write |
| --- | --- | --- |
| task | a runnable command that a source declares | script, target, entry |
| source | one family of project files that the picker reads | provider, parser, integration |
| discovery | the process that finds a source file and parses its tasks | scan, lookup |
| picker | the full TUI list of tasks | menu, selector, dialog |
| row | one line of the picker, for one task | item, entry |
| badge | the source file shown at the right of a row | tag, label |
| widget | the shell function that opens the picker from a key | plugin, hook, extension |
| inline | the picker drawn below the prompt | embedded, popup |
| fullscreen | the picker that takes the whole terminal | full screen |
| rc file | the TOML files the picker reads: the global file and the project file | config file, settings file |
| global file | `$XDG_CONFIG_HOME/what-to-run/config.toml`, or `%AppData%\what-to-run\config.toml` on Windows | user config |
| project file | `.what-to-run.toml` in a project | local config |
| shell dialect | the quoting rules for `sh`, `cmd` and `powershell` | shell type |
| theme | a named look from the registry | skin, colorscheme |

## Page shape

Start with one sentence that says what the thing is. Then order the sections by
what the reader does, not by the code. `docs/widget.md` is the model:

```text
# Widget setup
one sentence: what the widget is
## zsh and bash
## Windows
### PowerShell
### cmd.exe
## Key binding
## Fullscreen
## Running the picker
```

Group shells under their operating system, and put an explanation next to the
step it explains, in one short paragraph.

## The house voice

- One idea per sentence, most under 25 words, and vary the length: a page of
  10-word sentences reads like a machine. Keep a cause with its effect:
  "cmd.exe handles quotes in a limited way, so some arguments can change" stays
  one sentence.
- Make the component the subject, and address the reader as "you": "the picker
  shows", "add the line". Never "we", "our" or "us".
- Use the active voice: "the picker reads the rc file", not "the rc file is
  read".
- Put the condition first: "If there are no tasks, ...".
- Prefer the common word: "shows" not "reports", "puts" not "inserts", "add" not
  "pass", "there are no tasks" not "the directory declares no tasks".
- Put the action in the verb: "Ctrl-O opens the widget", not "the widget is
  bound to Ctrl-O".
- Stay with plain English: "stay exactly the same", not "survive an exact round
  trip". Write "handles quotes in a limited way", not "its quoting is limited".
- Write full forms and one sentence per period: "do not", "it is", and a period
  where a semicolon tempts you.
- Use American spelling: behavior, color, center.
- Leave out "simply", "just", "obviously" and "easy": the fact carries itself.

Calibrate a draft against these pairs:

| Before | After |
| --- | --- |
| The widget is bound to **Ctrl-O** by default. | Ctrl-O opens the widget by default. |
| If the current directory declares no tasks... | If there are no tasks in the current directory... |
| To change it, pass `--key` to `init`. | To change the key, add `--key` to the `init` command. |
| Its quoting is limited: use PowerShell when an argument must survive an exact round trip. | It handles quotes in a limited way, so some arguments can change. Use PowerShell when the exact command matters. |
| Simply add the following line to your shell's rc file, and then make sure to restart your shell so that the widget becomes available. | Add this line to your rc file. Start a new shell to load the widget. |

## Formatting

- Put code, flags, keys and file names in backticks, and commands in fenced
  blocks.
- Write `wtr`, not `./bin/wtr`, except in `docs/development.md`.
- Name the destination in the link: `see [Configuration](docs/config.md)`.

## Before you finish

1. Run `task check-md`. It must report no diff.
2. Run `go test ./cmd/wtr` when the source list changed. It must pass.
3. Check every command you touched against `wtr --help` and the code. If you
   cannot find a command, delete it from the docs.
4. Open every new link. The anchor must match a heading.
5. Search the docs for the fact you added. If it already has an owner, link to
   it instead of writing it again.
6. Run `task all` when the change touches Go code. It must pass.
7. Commit the docs with the code they describe.
