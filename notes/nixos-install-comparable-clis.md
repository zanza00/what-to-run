# Nix and NixOS installation in comparable CLI tools

Research note, 2026-11-01, verified against primary sources: each project's own repository, the
nixpkgs repository, and the home-manager repository. I read every file cited below from the
repository's default branch (`master` or `main`) on that date, and I checked nixpkgs and
home-manager on their `master` branches. "nixpkgs master" below means the unstable branch; the
current stable release branch is `nixos-26.05`. Nothing comes from blog posts or third-party
write-ups. An absent flake or module is a finding after a check, not an assumption. Latest release
tags come from each project's GitHub releases page (`releases/latest`), checked the same day.

Purpose: PR #61 adds an in-repo flake to what-to-run (packages only, `x86_64-linux`,
`aarch64-linux`, `aarch64-darwin`, `buildGoModule` from source, a manually bumped `version`
attribute that the release workflow checks against the tag) and tells NixOS users to add the
flake as an input. This note collects what comparable CLIs do, so the maintainer can judge
whether that approach matches the ecosystem.

## Summary

- All eleven projects below are packaged in nixpkgs under `pkgs/by-name/`. On nixpkgs master the
  version equals the latest upstream release for nine of them; go-task is one release behind and
  skim is two behind. The stable channel trails further: `nixos-26.05` has fzf 0.72.0 (upstream
  0.74.5) and go-task 3.48.0 (upstream 3.54.0).
- Six of eleven ship an upstream flake: lazygit, television, gum, yazi, atuin, and skim. The other
  five have none. Every upstream flake exports packages and devShells except skim, which exports
  devShells only. Four export an overlay. None exports `homeManagerModules` or `nixosModules`.
- Every home-manager module in this set ships from the home-manager project, not from an upstream
  flake. Nine of the eleven projects have one; gum and go-task do not. The modules for the
  shell-widget tools (fzf, skim, zoxide, navi, atuin, starship) all wire shell integration, and
  fzf's module warns about the Ctrl-R conflict with atuin and mcfly.
- The projects' own docs almost always point at nixpkgs: `nix-env -iA nixpkgs.<name>` (fzf, gum,
  starship, zoxide, go-task) or `nix run nixpkgs#<name>` (television, lazygit). Only atuin
  recommends its own flake (`nix profile install "github:atuinsh/atuin"` plus a Cachix cache),
  and only yazi documents the flake as a system input ("keep up with the bleeding edge") plus a
  Cachix cache. lazygit frames its flake as "the latest development version" and keeps nixpkgs as
  the install channel. navi and skim say nothing about installing on Nix.
- No upstream flake derives its version from the git tag. The Rust projects read `Cargo.toml`
  (atuin plain, yazi with a date and rev suffix, television with a "-nightly" suffix), lazygit
  builds `version = "dev"` with the git revision in the binary, and gum bumps a manual attribute
  that rotted: its flake says 0.15.2 while the repo ships v2.0.2.

| project | upstream flake | in nixpkgs | home-manager module | what their docs tell NixOS users |
|---|---|---|---|---|
| fzf | no | yes, current | yes | `nix-env -iA nixpkgs.fzf` |
| lazygit | yes: packages, devShells, checks, overlays | yes, current | yes | nixpkgs first; flake is the development channel, plus an overlay example |
| television | yes: packages, apps, checks, devShells | yes, current | yes | `nix run nixpkgs#television` |
| gum | yes: packages, overlays, version stale | yes, current | no | `nix-env -iA nixpkgs.gum` |
| yazi | yes: packages, devShells, formatter, overlays | yes, current | yes | nixpkgs, configuration.nix, home-manager template, flake input, Cachix |
| atuin | yes: packages, devShells, formatter, overlays | yes, current | yes | `nix profile install "github:atuinsh/atuin"` + Cachix |
| starship | no | yes, current | yes | `nix-env -iA nixpkgs.starship`, home-manager module, configuration.nix |
| zoxide | no | yes, current | yes | `nix-env -iA nixpkgs.zoxide` |
| go-task | no | yes, one release behind | no | `nix-env -iA nixpkgs.go-task` |
| navi | no | yes, current | yes | nothing |
| skim | devShells only, no packages | yes, two releases behind | yes | nothing |

## fzf

1. Upstream flake: none. There is no `flake.nix` on `master`
   (<https://github.com/junegunn/fzf/tree/master>).
2. nixpkgs: yes, attribute `fzf`, version 0.74.5, which equals the latest release v0.74.5
   (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/fz/fzf/package.nix>). The stable
   channel has 0.72.0
   (<https://github.com/NixOS/nixpkgs/blob/nixos-26.05/pkgs/by-name/fz/fzf/package.nix>).
3. home-manager: yes, shipped by home-manager. The module installs the package, writes colors,
   `defaultCommand` and `defaultOptions`, and wires the shell integrations for bash, zsh, fish
   and nushell by sourcing fzf's own key-bindings scripts (Ctrl-T, Ctrl-R, Alt-C). When fzf and
   atuin or mcfly both bind Ctrl-R, it warns and names the options that decide who owns the key
   (<https://github.com/nix-community/home-manager/blob/master/modules/programs/fzf.nix>).
4. Docs: the README package-manager table says: "Nix | NixOS, etc. | `nix-env -iA nixpkgs.fzf`"
   (<https://github.com/junegunn/fzf/blob/master/README.md>).
5. Version handling: not applicable, no upstream flake.

## lazygit

1. Upstream flake: yes, on `master`, built with flake-parts
   (<https://github.com/jesseduffield/lazygit/blob/master/flake.nix>). Outputs: `packages.default`
   and `packages.lazygit`, `devShells.default`, `checks.build`, `overlays.default`. No
   `homeManagerModules`, no `nixosModules`.
2. nixpkgs: yes, attribute `lazygit`, version 0.66.2, the latest release
   (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/la/lazygit/package.nix>).
3. home-manager: yes, shipped by home-manager. The module writes the YAML config from
   `settings`, and the "integrations" define a shell function (default name `lg`) that cds into
   the directory lazygit was in when it exited
   (<https://github.com/nix-community/home-manager/blob/master/modules/programs/lazygit.nix>).
4. Docs: the README "NixOS" section leads with nixpkgs: "On NixOS, lazygit is packaged with nix
   and distributed via nixpkgs", with `nix-shell -p lazygit`, `nix run nixpkgs#lazygit`, and
   `environment.systemPackages` in `configuration.nix`. A second subsection, "Using the official
   lazygit flake", says: "This repository includes a nix flake that provides the latest
   development version and additional development tools", and shows `nix run
   github:jesseduffield/lazygit`, `nix develop`, and an overlay for flake-based systems
   (<https://github.com/jesseduffield/lazygit/blob/master/README.md>).
5. Version handling: `version = "dev"`, and `gitCommit = inputs.self.rev or inputs.self.dirtyRev
   or "dev"` goes into ldflags. The flake never claims a release version; it labels itself dev
   and carries the revision (<https://github.com/jesseduffield/lazygit/blob/master/flake.nix>).
   The flake is actively maintained: the last commit touching it is from 2026-09-21
   (<https://github.com/jesseduffield/lazygit/commits/master/flake.nix>).

## television

1. Upstream flake: yes, on `main`, built with flake-parts and crane
   (<https://github.com/alexpasmantier/television/blob/main/flake.nix>). Outputs:
   `packages.default`, `apps.default`, `checks`, `devShells.default`. No overlays, no
   `homeManagerModules`.
2. nixpkgs: yes, attribute `television`, version 0.15.9, the latest release
   (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/te/television/package.nix>).
3. home-manager: yes, shipped by home-manager. The module writes `settings` to
   `$XDG_CONFIG_HOME/television/config.toml`, writes cable channels, and wires shell
   integrations for bash, fish, nushell and zsh
   (<https://github.com/nix-community/home-manager/blob/master/modules/programs/television.nix>).
4. Docs: the README package-manager table says "Nix | `nix run nixpkgs#television`"
   (<https://github.com/alexpasmantier/television/blob/main/README.md>). The website installation
   page repeats it: "Television is available on `nixpkgs`" with the same command
   (<https://github.com/alexpasmantier/television/blob/main/docs/getting-started/01-installation.md>).
   Neither mentions the flake.
5. Version handling: `version = tvCargo.package.version + "-nightly"` — read from `Cargo.toml`
   with a suffix, not from a tag and not manual
   (<https://github.com/alexpasmantier/television/blob/main/flake.nix>).

## gum

1. Upstream flake: yes, on `main`, but thin: `packages.default` imports a plain `default.nix`,
   plus `overlays.default`. No devShells, no `homeManagerModules`
   (<https://github.com/charmbracelet/gum/blob/main/flake.nix>). The wrapped `default.nix` carries
   a manual `version = "0.15.2"`
   (<https://github.com/charmbracelet/gum/blob/main/default.nix>). That attribute rotted: the
   latest release is v2.0.2 (2026-09-24), `flake.nix` was last touched 2024-05-25 and
   `default.nix` 2025-02-15
   (<https://github.com/charmbracelet/gum/releases/latest>,
   <https://github.com/charmbracelet/gum/commits/main/flake.nix>,
   <https://github.com/charmbracelet/gum/commits/main/default.nix>).
2. nixpkgs: yes, attribute `gum`, version 2.0.2, the latest release — nixpkgs is more current than
   gum's own flake
   (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/gu/gum/package.nix>).
3. home-manager: no module
   (<https://github.com/nix-community/home-manager/tree/master/modules/programs>).
4. Docs: the README install block says `nix-env -iA nixpkgs.gum`. It never mentions the flake
   (<https://github.com/charmbracelet/gum/blob/main/README.md>).
5. Version handling: a manually bumped attribute, currently about eighteen months behind the
   releases.

## yazi

1. Upstream flake: yes, on `main`
   (<https://github.com/sxyazi/yazi/blob/main/flake.nix>). Outputs: `packages.yazi-unwrapped`,
   `packages.yazi`, `packages.default`, `devShells.default`, `formatter`, `overlays.default` and
   `overlays.yazi`. No `homeManagerModules`.
2. nixpkgs: yes. The build lives in `yazi-unwrapped`, version 26.9.1, the latest release
   (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/ya/yazi-unwrapped/package.nix>).
   The user-facing `yazi` is a wrapper that puts optional dependencies on PATH and can set
   `YAZI_CONFIG_HOME`
   (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/ya/yazi/package.nix>).
3. home-manager: yes, shipped by home-manager. The module configures the package, the TOML
   settings, plugins, flavors, `initLua`, a shell wrapper (for cd-on-quit) and shell integrations
   (<https://github.com/nix-community/home-manager/blob/master/modules/programs/yazi.nix>).
4. Docs: the installation page has the fullest Nix section of the set. It shows `nix-env -iA
   nixos.yazi` / `nixpkgs.yazi`, a `configuration.nix` example, and says: "You can also manage
   Yazi's configuration using home-manager". A "Nix flakes" section says: "The upstream
   repository provides a flake so that Nix users can easily keep up with the bleeding edge", with
   a complete `inputs.yazi.url = "github:sxyazi/yazi"` example for NixOS and home-manager, an
   overlay alternative, and a Cachix cache with substituters and a public key
   (<https://github.com/yazi-rs/yazi-rs.github.io/blob/main/docs/installation.md>).
5. Version handling: `version = Cargo.toml workspace.package.version + "pre" +
   lastModifiedDate + shortRev` — automatic from the manifest plus the revision
   (<https://github.com/sxyazi/yazi/blob/main/flake.nix>).

## atuin

1. Upstream flake: yes, on `main`, built with flake-parts, crane and fenix
   (<https://github.com/atuinsh/atuin/blob/main/flake.nix>). Outputs: `packages.atuin` and
   `packages.default`, `devShells.default`, `formatter`, `overlays.default`. No
   `homeManagerModules`.
2. nixpkgs: yes, attribute `atuin`, version 18.24.4, the latest release. Atuin's own `atuin.nix`
   points at this path in a comment
   (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/at/atuin/package.nix>,
   <https://github.com/atuinsh/atuin/blob/main/atuin.nix>).
3. home-manager: yes, shipped by home-manager. The module writes `settings` to the config file,
   runs the atuin daemon, and wires shell integrations for bash, fish, nushell and zsh
   (<https://github.com/nix-community/home-manager/blob/master/modules/programs/atuin.nix>).
4. Docs: the installation guide recommends the upstream flake, not nixpkgs: "This repository is
   a flake, and can be installed using `nix profile`: `nix profile install
   "github:atuinsh/atuin"`. Linux builds of `main` are cached on Cachix. Run `cachix use atuin`
   first to download them instead of building from source"
   (<https://github.com/atuinsh/atuin/blob/main/docs/docs/guide/installation.md>).
5. Version handling: `version = (lib.importTOML ./Cargo.toml).workspace.package.version` —
   automatic from the Rust manifest
   (<https://github.com/atuinsh/atuin/blob/main/atuin.nix>).

## starship

1. Upstream flake: none. There is no `flake.nix` on `master`
   (<https://github.com/starship/starship/tree/master>).
2. nixpkgs: yes, attribute `starship`, version 1.27.0, the latest release
   (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/st/starship/package.nix>).
3. home-manager: yes, shipped by home-manager. The module writes `settings` and `presets` to
   `starship.toml` and wires shell integrations for bash, fish, zsh, ion and nushell
   (<https://github.com/nix-community/home-manager/blob/master/modules/programs/starship.nix>).
4. Docs: the README package table says "NixOS | nixpkgs | `nix-env -iA nixpkgs.starship`"
   (<https://github.com/starship/starship/blob/master/README.md>). The website's installing page
   covers all three levels: "Imperatively" with `nix-env -iA nixos.starship`, "Declarative,
   single user, via home-manager" with `programs.starship`, and "Declarative, system-wide, with
   NixOS" with `environment.systemPackages`
   (<https://github.com/starship/starship/blob/master/docs/installing/README.md>).
5. Version handling: not applicable, no upstream flake.

## zoxide

1. Upstream flake: none. There is no `flake.nix` on `main`
   (<https://github.com/ajeetdsouza/zoxide/tree/main>).
2. nixpkgs: yes, attribute `zoxide`, version 0.10.0, the latest release (2026-07-04)
   (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/zo/zoxide/package.nix>).
3. home-manager: yes, shipped by home-manager. The module installs the binary, passes `options`
   to `zoxide init`, and writes the shell integrations for bash, fish, nushell and zsh
   (<https://github.com/nix-community/home-manager/blob/master/modules/programs/zoxide.nix>).
   This is the closest model for what-to-run's widget: the shell rc wiring lives in home-manager,
   not upstream.
4. Docs: the README install tables say `nix-env -iA nixpkgs.zoxide` for Linux and macOS, link
   the nixpkgs package file, and the README carries a "Built with Nix" badge
   (<https://github.com/ajeetdsouza/zoxide/blob/main/README.md>).
5. Version handling: not applicable, no upstream flake.

## go-task

1. Upstream flake: none. There is no `flake.nix` on `main`
   (<https://github.com/go-task/task/tree/main>).
2. nixpkgs: yes, attribute `go-task`, version 3.53.1; the latest release is v3.54.0, so master
   is one release behind, and `nixos-26.05` has 3.48.0
   (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/go/go-task/package.nix>,
   <https://github.com/NixOS/nixpkgs/blob/nixos-26.05/pkgs/by-name/go/go-task/package.nix>).
3. home-manager: no module
   (<https://github.com/nix-community/home-manager/tree/master/modules/programs>).
4. Docs: the installation page has a "Nix" section: `nix-env -iA nixpkgs.go-task`, with a
   "Source" link to the nixpkgs package file
   (<https://github.com/go-task/task/blob/main/website/src/latest/docs/installation.md>).
5. Version handling: not applicable, no upstream flake.

## Two closer comparables: navi and skim

what-to-run's nature is an fzf-style picker that runs project tasks and binds to a key in the
shell. fzf above is the closest well-known match. Two more projects in the same family are
worth the same five answers: navi (an interactive cheatsheet tool that picks and runs commands,
usable as a shell widget) and skim (an fzf-like finder with widget keybindings).

### navi

1. Upstream flake: none. There is no `flake.nix` on `master`
   (<https://github.com/denisidoro/navi/tree/master>).
2. nixpkgs: yes, attribute `navi`, version 2.24.0, the latest release
   (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/na/navi/package.nix>).
3. home-manager: yes, shipped by home-manager. The module writes `settings` to the navi config
   and wires bash, fish and zsh integrations that call `navi widget`
   (<https://github.com/nix-community/home-manager/blob/master/modules/programs/navi.nix>).
4. Docs: the README recommends `brew install navi` and says nothing about Nix, even though navi
   is in nixpkgs and home-manager
   (<https://github.com/denisidoro/navi/blob/master/README.md>).
5. Version handling: not applicable, no upstream flake.

### skim

1. Upstream flake: yes, on `master`, but development-only. Its description is "Nix flake for
   skim development", and it exports `devShells` (default, tests, utils, and more) plus a
   formatter. There is no `packages` output
   (<https://github.com/skim-rs/skim/blob/master/flake.nix>).
2. nixpkgs: yes, attribute `skim`, version 5.7.4; the latest release is v5.9.0, so master is two
   releases behind (<https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/sk/skim/package.nix>).
3. home-manager: yes, shipped by home-manager. The module configures `defaultCommand`,
   `defaultOptions`, the file, change-dir and history widget commands and options, and shell
   integrations for bash, fish and zsh
   (<https://github.com/nix-community/home-manager/blob/master/modules/programs/skim.nix>).
4. Docs: the README package-manager table lists no Nix; the only Nix mention is a
   troubleshooting note about line-feed issues. The dev flake is not advertised
   (<https://github.com/skim-rs/skim/blob/master/README.md>).
5. Version handling: no package in the flake, so none.

## Patterns

- nixpkgs is the baseline. All eleven projects are in nixpkgs, and the maintainers of the
  actively updated ones keep master current with releases. "Being in nixpkgs" is what makes the
  `nix-env -iA nixpkgs.<name>` docs line true, and that line is the most common recommendation in
  the set (fzf, gum, starship, zoxide, go-task; `nix run nixpkgs#` for television and lazygit).
- An upstream flake is common but always secondary. Six of eleven ship one, and five of those
  are also in nixpkgs with current versions. No project treats the flake as the only channel.
- Flake shape is uniform: packages plus devShells, sometimes an overlay, a formatter or checks.
  No project in this set exports `homeManagerModules` or `nixosModules` from its flake, and every
  home-manager module comes from the home-manager project itself.
- Docs about the flake are rare and hedged. lazygit calls its flake "the latest development
  version"; yazi says the flake is for users who want to "keep up with the bleeding edge"; only
  yazi shows the flake-as-input configuration that PR #61 documents, and only atuin makes the
  flake the recommended install.
- Version handling never uses the tag, because a flake cannot read one: flake metadata exposes
  the revision and date, not the tag. The projects split three ways: read the language manifest
  (atuin, yazi, television — automatic because release automation bumps Cargo.toml), hardcode
  "dev" plus the revision (lazygit), or bump manually (gum — and gum's attribute rotted for
  about eighteen months).
- home-manager modules exist for nine of eleven projects, including TUI apps (lazygit,
  television, yazi) and not only shell-widget tools. For the widget tools the module is where the
  shell wiring lands: fzf and skim expose widget keybindings as options, zoxide and navi write
  the init snippet into the rc files, and fzf's module warns when fzf and atuin or mcfly both
  bind Ctrl-R, naming the options that decide the owner.
- Binary caches are rare: yazi and atuin run Cachix and document it. The Go projects (lazygit,
  gum) do not.

## What this suggests for what-to-run

- The shape of PR #61 matches the ecosystem: packages plus devShells, built from source with
  buildGoModule, no `nixosModules` or `homeManagerModules`. lazygit's flake is the closest
  sibling, including the go.mod-driven Go choice and the source build. Nothing in the comparable
  set exports NixOS or home-manager modules, so the PR does not under-deliver by omitting them.
- The manually bumped `version` attribute is the one pattern with a bad track record: gum's
  rotted and nobody noticed. PR #61 adds the guard none of the comparables has — the release
  workflow fails when the flake version does not match the tag — which closes exactly the gap
  gum fell into. lazygit's "dev" plus revision is the low-maintenance alternative; reading the
  version from a manifest is not available to a Go module, since go.mod carries no version.
- The unusual part is the docs, not the flake: PR #61 tells NixOS users to install only through
  the flake input, and what-to-run is not in nixpkgs yet (no package under `pkgs/by-name/wh/`
  or `pkgs/by-name/wt/` on master, checked 2026-11-01;
  <https://github.com/NixOS/nixpkgs/tree/master/pkgs/by-name/wh>). In the comparable set, every
  project
  that documents Nix leads with nixpkgs, and the two that document the flake input (yazi,
  lazygit) were in nixpkgs first. A nixpkgs PR (which would live at
  `pkgs/by-name/wh/what-to-run/package.nix`) plus a docs line pointing at it would move
  what-to-run to the standard end-state: nixpkgs as the install channel, the flake as the
  bleeding-edge channel. Until that lands, the flake is the only channel, so the current docs
  are correct but singular in this set.
- A home-manager module is the natural next channel for a shell-widget CLI, but it would come
  from a home-manager PR (`programs.wtr` with shell-integration options, like fzf's and skim's
  widget options), not from this repository's flake. None of the comparables shipped one early;
  fzf-style widget tools all ended up with one.
- A Cachix cache would keep `nix profile install` users from building from source, but only
  yazi and atuin run one in this set, so it is an optimization, not a norm.

## Unverified

- I read files, not evaluations: I did not run `nix build` on any flake, so output lists come
  from reading `flake.nix`, not from `nix flake show`.
- I checked the stable channel (`nixos-26.05`) only for fzf and go-task, not for all eleven
  packages.
- Docs coverage: I checked each project's README and, where the README delegates, the in-repo
  docs source (television, atuin, go-task, starship, yazi). I did not sweep every page of the
  websites, the fzf wiki, or navi's wiki, so a stray Nix mention elsewhere is possible.
- For navi I could not enumerate the wiki (the wiki pages I fetched returned nothing), so "docs
  say nothing about Nix" rests on the README.
- Release dates come from the GitHub API for the latest release of each project; I did not check
  intermediate releases, so "one release behind" for go-task may be days old.
