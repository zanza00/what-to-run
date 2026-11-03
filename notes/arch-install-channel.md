# Arch install channel

How Arch Linux users install `wtr`, and why the AUR is not part of it.
History, not a snapshot: every claim is dated and cites the commit it was
true at.

- **2026-11-03**: the maintainer decided the AUR is out, because AUR
  registration has been closed for the channel's whole life: the
  `AUR_SSH_PRIVATE_KEY` secret was never set, and the release workflow
  always skipped the step with a warning. Until accounts reopen, the AUR
  is not up for discussion. The removal is PR #110 (branch
  `chore/remove-aur` at `c44cc76`, a draft when this was written): commit
  `de5ac10` deletes `scripts/publish-aur.py` with its 9 tests, the `aur`
  publisher from `.goreleaser.yaml` and the AUR block from `release.yml`;
  commit `dbe52f5` drops every AUR row and check from the release skill;
  commit `c44cc76` tightens the stack prose.
- **2026-11-03**: the Arch channels are the pacman repository on gh-pages
  (PR #108, branch `feat/pacman-repo` at `29b1a4d`: one database per
  architecture, one entry per package name, every package kept on disk)
  and the `.pkg.tar.zst` in every release (PR #107, branch
  `feat/arch-package` at `ec7ac95`) for `pacman -U` installs and
  downgrades. The stack reads #107, then #108, then #110; after the
  parents merge, #110 retargets onto `main`.
- **2026-11-03**: bringing the AUR back, if registration ever reopens,
  means reverting PR #110, setting the secret again and re-adding the
  channel rows to the release skill. No placeholder in the tree points
  there: this page is the map.
- **2026-11-03**: the released changelog history, the 0.1.2 "Added an AUR
  publishing step" entry and the 0.9.0 line "Homebrew and AUR publish the
  single binary", keeps its AUR mentions: the changelog is a log.
- **2026-11-03**: known gap, outside the removal spec: the method table
  in `docs/install.md` has no Arch row, although the README install
  bullet names pacman. A candidate ticket for the next docs pass.
  *(Superseded 2026-11-03 by the claim below.)*
- **2026-11-03**: the known gap closed. The method table in
  `docs/install.md` lists pacman for Arch Linux and links to the Arch
  Linux section, like the apt, dnf and Scoop rows. The fix is commit
  `73b9533` on branch `ticket/04-close-stack-findings`, stacked on
  `chore/remove-aur` (PR #110). The same commit also quiets the
  shellcheck SC2034 finding in the verify-npm job of `release.yml`.
