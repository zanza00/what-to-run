# NixOS install landscape

Research note: `notes/nixos-install-comparable-clis.md` (2026-11-01) compares
eleven CLI tools similar to what-to-run - fzf, lazygit, television, gum, yazi,
atuin, starship, zoxide, go-task, navi and skim - on how they install on Nix
and NixOS. The note cites each claim against the project's repository, nixpkgs
and home-manager; this page only records the findings that bear on
what-to-run's own choices.

Findings on 2026-11-01 (from the note):

- All eleven are packaged in nixpkgs under `pkgs/by-name/`, and nixpkgs master
  carried the latest release for nine of them. Their docs almost always point
  at nixpkgs first.
- Six of eleven ship an upstream flake; every one exports packages and
  devShells, four an overlay, and none exports `homeManagerModules` or
  `nixosModules`. PR #61's packages-only shape matches the ecosystem.
- Home-manager modules exist for nine of eleven, and every one ships from the
  home-manager project, never from an upstream flake. For the shell-widget
  tools the module is where shell integration lands.
- No comparable flake derives its version from the git tag. gum's manually
  bumped attribute rotted (0.15.2 in-repo against v2.0.2 released), which is
  the exact failure PR #61's release-workflow check guards against.
- what-to-run was not in nixpkgs on 2026-11-01 (checked `pkgs/by-name/wh` and
  `wt`), so leading the docs with the flake was correct but singular in this
  set: the two comparables that document the flake as an input, yazi and
  lazygit, were in nixpkgs first and keep nixpkgs as the main channel.

Implications recorded for the maintainer: keep PR #61 as is; a nixpkgs
submission under `pkgs/by-name/` completes the standard pattern and would let
the docs lead with nixpkgs; a home-manager `programs.wtr` module is the
natural later channel for the shell widget. No comparable ships a NixOS
module, so none is needed.
