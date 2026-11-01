# Nix install verification

Started 2026-11-01 by the maintainer's request: can a trial NixOS distro install
and use the wtr CLI, before the feature PR merges?

## Merge first, or test first?

Test first works. On 2026-11-01 the answer was: a flake ref accepts a branch, so
anyone can install the unmerged work with:

```sh
nix profile install github:zanza00/what-to-run/feat/channel-nix
```

The commands documented in `docs/install.md` at `feat/channel-nix@fa47b0e`
(2026-11-01) use the default branch, `github:zanza00/what-to-run`, and only work
after the merge: `main` at `399a138` had no `flake.nix`. The docs also tell users
to pin releases with `github:zanza00/what-to-run/<tag>`.

## What CI already covers

On 2026-11-01, at PR #61 head `fa47b0e`, the `checks.yml` job `nix` runs
`nix flake check` and `nix build .#default` on `ubuntu-latest` and `macos-latest`,
both green. CI builds from a checkout, so it does not cover fetching the flake by
a GitHub ref, `nix profile install`, or running the installed binary.

## How we test here

On 2026-11-01 the host had no local nix; it has rootless podman 5.8.9, docker
29.9.2 and qemu. The maintainer chose a minimal NixOS container over a full VM:
the official `nixos/nix` image under podman, with docker and `--option sandbox
false` as fallbacks. A full qemu VM to validate the NixOS snippet from
`docs/install.md` (`environment.systemPackages`) stays an optional follow-up.

## Flake facts

At `feat/channel-nix@fa47b0e` (2026-11-01):

- `packages.default` = `what-to-run`, a `buildGoModule` forced onto `go_1_27`
  when nixpkgs carries it; `go.mod` requires Go >= 1.27.2 and the flake throws
  when nixpkgs is older.
- The `checkPhase` runs `go test ./...`, so a Nix build also runs the Go test
  suite.
- `version = "0.9.0"` is injected via ldflags, so `wtr --version` must print it.
- Systems: `x86_64-linux`, `aarch64-linux`, `aarch64-darwin`. Intel Macs are out
  of scope: `docs/install.md` sends them to Homebrew.
- `flake.lock` pins nixpkgs `nixos-unstable` rev
  `31ecbaca1bdefa45dffc409f0250bd8039ea0010`.

## Result

On 2026-11-01 the container test PASSED end to end (background agent
`nix-verify`; full report in the session scratchpad as `nix-verify-report.md`):

- `nixos/nix:latest` under rootless podman 5.8.9, Nix 2.35.4. That image ships
  `sandbox = false`, so the test did not exercise a sandboxed build; the CI
  `nix` job at the same commit did, green on both runners.
- The branch ref resolved to `fa47b0ee8c552eba2854919dee9b35157495e3aa`, the
  exact PR #61 head.
- `nix profile install` built `what-to-run-0.9.0` from source in about 52
  seconds, toolchain substituted from cache.nixos.org, and the check phase ran
  `go test ./...`: all four packages ok.
- `/root/.nix-profile/bin/wtr --version` printed `wtr 0.9.0`; `--help`,
  `init zsh` and the one-shot `nix run -- --version` all worked. Without a tty,
  `wtr` exits 1 with a clean `bubbletea: could not open TTY` error, as expected.
- Total wall time about 2.5 minutes. Only `x86_64-linux` was exercised in the
  container; `aarch64-darwin` only through CI.

Verdict: PR #61 needs no merge to be testable, and its user install path works
at `fa47b0e`. Merging stays the maintainer's manual call, and the default-ref
commands from `docs/install.md` go live only after it.

On 2026-11-01 at 17:02 UTC, after `origin/main` was merged into the branch
(merge commit `9c98c4b`, see [Packaging channel
PRs](packaging-channel-prs.md)), the whole CI ran green on the PR: test on
both runners, cross, the `nix` job on both runners, the changelog entry job,
and the `CI required` gate. So the flake builds sandboxed in CI on the exact
tree that the container test verified from the user install path.
