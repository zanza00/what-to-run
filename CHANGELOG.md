# Changelog

All notable changes to this project are documented in this file.

The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Added npm as an install channel: `npm install -g what-to-run` installs a
  small launcher plus one optional dependency per platform carrying the Go
  binary, and runs no install script. It is the shortest install path on
  Windows, and `npx what-to-run@latest` runs the picker without installing it
  (56988de).

### Changed

- **Breaking:** the `disable` key in the rc file now takes source IDs (`make`,
  `cargo-make`) instead of file names. An entry that is not a source ID is
  dropped with a warning naming the ID to use, so a source disabled by a file
  name runs again until the entry is fixed (f9d17b6).
- Derived the file list of the no-tasks screen from the source registry and
  wrapped it to the picker width, so a narrow terminal still shows every file
  name (eb81162).
- Reused the filter results across a frame, which keeps the picker fast while
  a filter runs over a large task list (e952869).

## [0.9.0] - 2026-10-25

### Added

- Added a Scoop install channel for Windows: `scoop install
  zanza00/what-to-run` installs the `wtr` command on amd64 and arm64
  (2a7d823).

### Changed

- Changed the widget's default key from `ctrl-s` to `ctrl-o`, leaving the
  shell's incremental forward search and flow control alone. `--key ctrl-s`
  still binds the old key, and the widget still handles flow control for
  `ctrl-s` and `ctrl-q` (24fd0c7).

## [0.8.0] - 2026-10-24

### Added

- Added `.rpm` packages for Fedora and RHEL to the release assets, for amd64
  and arm64, signed with the same key as the apt repository (4337e19).
- Added an RPM repository for `dnf` and `yum`, published next to the apt
  repository on the project's GitHub Pages site (e5dba14).
- Added the one-line setup scripts `setup.deb.sh` and `setup.rpm.sh`, which
  configure the signed apt and dnf repositories; the RPM script also works on
  hosts with `yum` but no `dnf` (f7c2b2e).

### Changed

- Shipped the single `wtr` binary instead of the `wtr` and `what-to-run`
  pair: every archive and `.deb` is about half the size, and Homebrew and AUR
  publish the single binary (2343624).

### Removed

- Removed the `what-to-run` command; rc files that call it must use `wtr`
  (2343624).

## [0.7.0] - 2026-10-23

### Added

- Added Taskfile as a task source: the picker lists the tasks of
  `Taskfile.yml` and the other go-task default names sorted by name, and each
  row runs `task <name>`. Descriptions come from the `desc`, the first line of
  the `summary` or the command line, flattened to one line, and internal
  tasks are hidden (2a39224).
- Followed local Taskfile includes recursively: every included task is
  namespaced with its include key (`docs:build`) unless the include sets
  `flatten: true`, and remote or templated paths are skipped without losing
  the other tasks (71e8df0).
- Added justfile as a task source: the picker lists the recipes of `justfile`,
  `Justfile` and `.justfile` sorted by name, and each row runs `just <name>`.
  Descriptions come from the doc attribute or the comment above the recipe,
  flattened to one line; private recipes and recipes declared for other
  platforms are hidden, and attributes, comments and multi-line headers parse
  the way `just --list` shows them (49f6d7a).
- Followed local justfile imports recursively, keeping the root file's recipe
  names and commands, and skipping a missing, unreadable or cyclic file
  (11959a1).
- Followed local justfile modules, listing their recipes as `name::recipe`,
  chained for nested modules (b1d0b1f).
- Searched parent directories for the justfile and Taskfile sources, like
  `just` and `task` themselves do: the nearest file wins, and a parent file's
  badge shows its path relative to the current directory (c6d5b41).
- Added the `--shell` flag to choose the dialect command lines are quoted
  for: `posix`, `powershell` or `cmd`; `auto`, the default, uses PowerShell on
  Windows and POSIX elsewhere (7b5d69d).
- Added the `init powershell` subcommand, printing a PSReadLine widget for
  Windows PowerShell 5.1 and PowerShell 7: enter runs the chosen task, tab
  inserts its command line in the buffer, the prompt redraws after the widget
  runs, and `WTR_BIN` and `WTR_FLAGS` are honored (022c3f4).
- Shipped Windows builds, amd64 and arm64, as zip archives; on Windows the
  usage, the messages and the picker title show the program name without the
  `.exe` suffix (f841796).

### Changed

- Quoted the command line the picker shows and the widget inserts for the
  shell that reads it back: the zsh and bash widgets pass `--shell posix`, the
  PowerShell widget passes `--shell powershell`, and in PowerShell a quoted
  command word is prefixed with the call operator `&`, so a path with spaces
  runs (7b5d69d).
- Ran rc file tasks through the platform shell, `cmd /c` on Windows and
  `sh -c` elsewhere; on Windows the raw cmd.exe command line keeps quotes
  inside the command intact (7b5d69d, 871b508).

### Fixed

- Fell back to fullscreen when the terminal cannot report the cursor row, a
  Windows conhost console, instead of leaving the inline picker blank
  (7b5d69d).

## [0.6.0] - 2026-10-22

### Added

- Added the cargo aliases source: the picker reads `[alias]` in
  `.cargo/config` and `.cargo/config.toml` and lists each alias as a
  `cargo <name>` task, with the alias expansion, chains resolved the way
  cargo does, as the description (441e78e).

## [0.5.0] - 2026-10-19

### Added

- Added the `ctrl+f` key binding to the inline picker: it expands the picker to
  fullscreen, pressing it again returns the picker to the inline area below
  the prompt, and the help line shows the toggle hint (be08e69).

### Fixed

- Fixed the apt repository publishing: the release pipeline pushed the apt
  channel to gh-pages without a token, so the push failed and the channel did
  not update (31f5080).

## [0.4.0] - 2026-10-19

### Added

- Added the `devbox.json` source, running its `shell.scripts` through
  `devbox run <name>` (d6d1ed7).
- Added the `composer.json` source, running custom `scripts` through
  `composer run-script <name>` and leaving out Composer's lifecycle and plugin
  events (c0f3f21).
- Added the `deno.json` and `deno.jsonc` source, running its `tasks` through
  `deno task <name>` (eed2049).
- Added the `mise.toml` and `.mise.toml` source, running its `[tasks]` through
  `mise run <name>` (009c4f6).

## [0.3.1] - 2026-10-19

### Added

- Added an apt repository for Debian and Ubuntu, so the CLI installs and
  updates through `apt` (b376ada).

### Fixed

- Kept the inline picker open on a "no tasks found here" screen that lists the
  files it looked for, closed with enter or esc; before, the error the run
  printed was erased by the shell prompt redraw. Fullscreen runs still exit
  with the error (6664f70).

## [0.3.0] - 2026-10-19

### Changed

- Changed the cargo-make source to read tasks from `Makefile.toml` instead of
  `Cargo.toml`, and skipped the private and disabled tasks the picker cannot
  run directly (2d089c4).
- Renamed the cargo-make value of the `disable` key in the rc file from
  `Cargo.toml` to `Makefile.toml` (2d089c4).

## [0.2.0] - 2026-10-19

### Added

- Added the bash widget: `wtr init bash` prints the integration to evaluate
  in an interactive shell, bound to `ctrl-s` by default and to the same
  `--key` values as `init zsh`. The chosen task runs like a typed line, with
  prompt, history and exit status. Bash 4.3 or later is required (bd3f7e2).

## [0.1.2] - 2026-10-19

### Added

- Added Linux `.deb` packages holding both binaries to the releases, with
  `apt install` instructions in the README (fcf8237).
- Added an AUR publishing step that pushes the `what-to-run-bin` PKGBUILD,
  skipped with a message while no AUR key is set (fcf8237).
- Documented installing both binaries with mise through the `github:` backend
  (0b6d81a).

## [0.1.1] - 2026-10-19

### Fixed

- Silenced the zsh widget: `eval "$(wtr init zsh)"` no longer prints a
  greeting in every new shell (659fa4a).

## [0.1.0] - 2026-10-19

### Added

- Added the picker: it discovers tasks from `package.json` scripts,
  `Makefile` targets and cargo-make `[tasks]` in `Cargo.toml`, filters them
  while typing, and runs the selection with enter. A source that fails to
  parse is skipped and reported instead of hiding the other tasks (d205d52).
- Detected the package.json runner from the `packageManager` field or a lock
  file, with npm as the fallback (d205d52).
- Added inline mode with `--inline`, drawing the picker below the prompt while
  fullscreen stays the default, and `--debug-area` drawing a border around the
  managed area (f51bfac).
- Ran the chosen task in place in fullscreen; in inline mode the picker quits
  first and the shell runs the task at the prompt, with its exit code
  propagated (d205d52).
- Built the binary as both `wtr` and `what-to-run`; usage, messages and the
  picker title follow the invoked name (8921b3d, dbaa81f).
- Added the `init zsh` subcommand, printing the widget script to evaluate in
  `~/.zshrc`, with `--key` to pick the binding, `ctrl-s` by default, and
  `WTR_BIN` and `WTR_FLAGS` to point at another binary (d4b274b).
- Made the widget fzf-style: enter executes the chosen task through zle, with
  prompt, history and exit status, and tab only places the command on the
  prompt (7c1a633).
- Added the rc files `.what-to-run.toml` and
  `$XDG_CONFIG_HOME/what-to-run/config.toml`: `[[tasks]]` adds custom tasks
  run with `sh -c`, with an optional `cwd`; `package_manager` forces the
  package.json runner, and `disable` skips a source (6db8184).
- Parsed the rc files best-effort — unknown keys, wrong types and invalid
  entries drop only the offending item — and added per-project overrides
  under `[projects."<path>"]` in the user rc file; `XDG_CONFIG_HOME` and
  `XDG_CACHE_HOME` are honored on every platform (f96a503, ee0ec0b).
- Opened warnings in the `ctrl+w` modal, where `d` dismisses one for future
  runs (f96a503).
- Extended the filter to descriptions and command lines: matches rank
  fzf-style with contiguous matches first, and the matched characters are
  highlighted in each row (5a8becd).
- Showed each row as the command line, the description and the badge, hiding a
  description that duplicates the command of an rc task; the description
  stays readable on the selected row of every theme (f582e49, d9910d6,
  883b5d8).
- Added 14 named themes — classic, nord, dracula, tokyo, gruvbox, catppuccin,
  one-dark, monokai, solarized, everforest, kanagawa, rose-pine, system and
  mono — with `--theme` to pin one for a run, `--list-themes` to list them
  and `--preview` to show one (640c374, 74a3ac6, 43a1fa9).
- Opened the theme picker with `ctrl+t`: every theme previews live, enter
  keeps the choice and esc restores the previous one; keeping a theme saves it
  to the user rc file, confirmed by a transient toast (06adbb1, 4266bc5).
- Added `--version`, reporting the release tag or the build's git describe
  string (10f460f).
- Pointed at `wtr` from `what-to-run --help` and from the widget startup
  message when it resolves on PATH (7da6eae, f18a765).
- Added the Homebrew install channel: tag pushes publish the Linux and Darwin
  archives and update the formula in the `zanza00/homebrew-tap` tap, creating
  the formula on the first release (bbc550b, adfc5cd).

[Unreleased]: https://github.com/zanza00/what-to-run/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/zanza00/what-to-run/releases/tag/v0.9.0
[0.8.0]: https://github.com/zanza00/what-to-run/releases/tag/v0.8.0
[0.7.0]: https://github.com/zanza00/what-to-run/releases/tag/v0.7.0
[0.6.0]: https://github.com/zanza00/what-to-run/releases/tag/v0.6.0
[0.5.0]: https://github.com/zanza00/what-to-run/releases/tag/v0.5.0
[0.4.0]: https://github.com/zanza00/what-to-run/releases/tag/v0.4.0
[0.3.1]: https://github.com/zanza00/what-to-run/releases/tag/v0.3.1
[0.3.0]: https://github.com/zanza00/what-to-run/releases/tag/v0.3.0
[0.2.0]: https://github.com/zanza00/what-to-run/releases/tag/v0.2.0
[0.1.2]: https://github.com/zanza00/what-to-run/releases/tag/v0.1.2
[0.1.1]: https://github.com/zanza00/what-to-run/releases/tag/v0.1.1
[0.1.0]: https://github.com/zanza00/what-to-run/releases/tag/v0.1.0
