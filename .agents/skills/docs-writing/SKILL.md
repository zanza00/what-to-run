---
name: docs-writing
description: Write and update what-to-run's documentation — README.md and docs/*.md. Covers where each fact belongs, the project glossary, B2 English and the house voice, and the checks to run before finishing. Use when writing or editing the docs, adding a source (docs/task-sources.md and the README bullet list), documenting a config key, theme, flag or shell integration, or when the user asks to update the README or "i docs".
---

# Write the what-to-run docs

Each detail has one owner. The README shows the way in and gives the shortest
form of each step. `docs/` holds the reference. `AGENTS.md` is for agents and
is not user documentation.

Write in B2 English (CEFR): clear for an upper-intermediate reader and natural
for a native speaker. `docs/widget.md` is the reference page for this voice.

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

The README summarizes. It does not repeat a setup step in full. Put the details
in `docs/` and link to them.

A new source needs one row in `docs/task-sources.md`, in registry order, and one
bullet in the README "Task sources" section. `go test ./cmd/wtr` fails when
either is missing. The test checks the table order, not the bullet order: keep
the bullets in the same order anyway. Add a `##` section in
`docs/task-sources.md` only when the source has caveats that the table cannot
hold, as the `package.json` and Taskfile sections do.

## Glossary

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
what the reader does, not by the code structure. `docs/widget.md` is the model:

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

Group shells under their operating system. Put an explanation next to the step
it explains, in one short paragraph. Do not write a separate essay.

## Voice and level

- One idea in each sentence. Keep most sentences under 25 words, but do not
  split a cause from its effect: "cmd.exe handles quotes in a limited way, so
  some arguments can change" stays one sentence.
- Vary the length. A page of 10-word sentences reads like a machine.
- Prefer the common word: "shows" not "reports", "puts" not "inserts", "add" not
  "pass", "there are no tasks" not "the directory declares no tasks".
- Put the action in the verb: "Ctrl-O opens the widget", not "the widget is
  bound to Ctrl-O".
- Avoid C1 idioms and nominalizations: "stay exactly the same", not "survive an
  exact round trip". Write "handles quotes in a limited way", not "its quoting
  is limited".
- Keep the technical terms: picker, rc file, prompt, fullscreen, raw key
  sequence. B2 is about general English, not the domain vocabulary.

## Rules

- Use the imperative for steps and "you" for options. Do not use "we", "our" or
  "us". Make the component the subject: "the picker shows", "wtr quotes".
- Use the active voice: "the picker reads the rc file", not "the rc file is
  read".
- Put the condition first: "If there are no tasks, ...".
- Do not use contractions or semicolons.
- Use American spelling: behavior, color, center.
- Use one term for one thing. Introduce "fuzzy picker" once, then write
  "picker".
- Do not write "simply", "just", "obviously" or "easy".
- Put code, flags, keys and file names in backticks. Put commands in fenced
  blocks.
- Write `wtr`, not `./bin/wtr`, except in `docs/development.md`.
- Name the destination in a link: "see [Configuration](docs/config.md)" from the
  README, not "click here".

## Before you finish

1. Run `task check-md`.
2. Run `go test ./cmd/wtr` when the source list changed. It checks the
   `docs/task-sources.md` table and the README bullets.
3. Check every command against `wtr --help` and the code. Do not document a flag
   that does not exist.
4. Follow every new link and check the anchor against the heading.
5. Search for the paragraph you wrote. If it already exists, link to it instead
   of repeating it.
6. Read the page aloud. Split a sentence that runs out of breath. Join two short
   sentences with "so" or "because" when they belong together.
7. Run `task all` when the same change touches Go code.
8. Put the docs and the code in the same commit.

## Examples

| Before | After |
| --- | --- |
| The widget is bound to **Ctrl-O** by default. | Ctrl-O opens the widget by default. |
| If the current directory declares no tasks... | If there are no tasks in the current directory... |
| To change it, pass `--key` to `init`. | To change the key, add `--key` to the `init` command. |
| Its quoting is limited: use PowerShell when an argument must survive an exact round trip. | It handles quotes in a limited way, so some arguments can change. Use PowerShell when the exact command matters. |
| Simply add the following line to your shell's rc file, and then make sure to restart your shell so that the widget becomes available. | Add this line to your rc file. Start a new shell to load the widget. |
