---
name: docs-writing
description: Write and update what-to-run's documentation — README.md and docs/*.md. Covers where each fact belongs, the project glossary, simplified technical English for prose, and the checks to run before finishing. Use when writing or editing the docs, adding a source (docs/task-sources.md and the README bullet list), documenting a config key, theme, flag or shell integration, or when the user asks to update the README or "i docs".
---

# Write the what-to-run docs

Each detail has one owner. The README shows the way in and gives the shortest
form of each step. `docs/` holds the reference. `AGENTS.md` is for agents and
is not user documentation.

The style follows the principles of Simplified Technical English
([ASD-STE100](https://www.asd-ste100.org/), Issue 9). The controlled
dictionary is out of scope: the glossary below is the project's own.

## Where each fact belongs

| File | Owns | Links to |
| --- | --- | --- |
| `README.md` | pitch, install shortcuts, widget one-liner, source bullet list, config summary, inspiration, license | `docs/install.md`, `docs/widget.md`, `docs/task-sources.md`, `docs/config.md` |
| `docs/install.md` | every install channel, the setup scripts, upgrades from a two-name release | `docs/widget.md` from "Next steps" |
| `docs/widget.md` | zsh, bash and PowerShell setup, key binding, shell dialect, inline mode | `docs/development.md` for the local build |
| `docs/config.md` | the rc schema, per-project overrides, precedence, best-effort parsing, warnings, themes and theme flags | `docs/task-sources.md` for the source IDs |
| `docs/task-sources.md` | the registry table, the source ID rules, per-source notes | — |
| `docs/development.md` | prerequisites, tasks, README demo, local build, release secrets and pipeline | `.github/workflows/release.yml` for the pipeline source |
| `AGENTS.md` | rules for agents | — |

The README summarizes. It does not repeat a setup step in full. Put the
details in `docs/` and link to them.

A new source needs one row in `docs/task-sources.md`, in registry order, and
one bullet in the README "Task sources" section. `go test ./cmd/wtr` fails when
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

## Instructions and reference

The docs have two voices. Use the one that matches the section.

- **Procedure** (`docs/install.md`, `docs/widget.md` and the how-to sections
  of `docs/development.md`): write in the imperative, one action in each
  sentence, 20 words at most. Put a condition first: "If the shell is bash,
  add this line".
- **Reference** (`docs/config.md`, `docs/task-sources.md`): write in the
  present tense, 25 words at most, one topic in each paragraph, six sentences
  at most.

In both voices:

- Use the active voice: "the picker reads the rc file", not "the rc file is
  read".
- Keep the articles and the full sentence: "Add the line", not "Add line".
- Do not use contractions. Do not use semicolons: write two sentences.
- Use American spelling: behavior, color, center.
- Use one term from the glossary for one thing.
- Do not write "simply", "just", "obviously" or "easy". They add no fact.
- Address the reader as "you", or use the imperative. Do not write "we".
- Put code, flags, keys and file names in backticks. Put commands in fenced
  blocks.
- Write `wtr`, not `./bin/wtr`, except in `docs/development.md`.
- Name the destination in a link: "see [Configuration](docs/config.md)" from
  the README, not "click here".

## Before you finish

1. Run `task check-md`.
2. Run `go test ./cmd/wtr` when the source list changed. It checks the
   `docs/task-sources.md` table and the README bullets.
3. Check every command against `wtr --help` and the code. Do not document a
   flag that does not exist.
4. Follow every new link and check the anchor against the heading.
5. Search for the paragraph you wrote. If it already exists, link to it instead
   of repeating it.
6. Run `task all` when the same change touches Go code.
7. Put the docs and the code in the same commit.

## Examples

| Before | After |
| --- | --- |
| Configuration is loaded from a variety of locations which are described in the section below. | The picker reads two rc files. The next section gives the order. |
| Simply add the following line to your shell's rc file, and then make sure to restart your shell so that the widget becomes available. | Add this line to your rc file. Start a new shell to load the widget. |
| The tasks that are discovered are shown in the picker, which is a list that can be filtered; the selected task runs when you press enter. | The picker lists the discovered tasks. Type to filter them. Press enter to run the selected task. |
| By default the theme is classic, but this can be changed by the user via the theme picker which is opened with ctrl+t. | The default theme is classic. Press ctrl+t to choose another theme. |
