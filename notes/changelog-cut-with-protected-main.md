# Cutting the changelog when the default branch only accepts pull requests

Research note, 2026-10-31, verified against primary sources (repository files, pull requests, commits,
tags and releases read through the GitHub API today, plus the official documentation of release-please,
changesets, semantic-release, GoReleaser, git-cliff and GitHub). The question: the `main-guard` ruleset
requires every change to `main` to go through a merged PR, so the commit the `release` skill pushes to
`main` before tagging — `## [Unreleased]` renamed into `## [X.Y.Z] - YYYY-MM-DD` — can no longer land,
while `.github/workflows/release.yml` still requires the tagged commit to be an ancestor of `origin/main`.

## Summary

- **The release PR**: a small PR cuts the version section, merges through the normal process, and the
  release manager tags the commit the merge put on the default branch. Verified in traefik/traefik
  (428 "Prepare release" PRs) and hashicorp/terraform ("Prepare before X release" PRs; the tag sits on
  the squash commit). go-task/task and golangci/golangci-lint cut the same way but by pushing directly,
  which is exactly the access the ruleset removes.
- **Bot release PRs**: release-please keeps a "chore(main): release X.Y.Z" PR open, updates
  `CHANGELOG.md` inside it, and tags the merged commit when the maintainer merges; it requires
  Conventional Commits. changesets keeps a "Version Packages" PR fed by `.changeset/*.md` files added
  in feature PRs and publishes via CI on merge.
- **Bots with an exception**: semantic-release pushes the release commit straight to the protected
  branch with a token or GitHub App that the ruleset's bypass list allows; GitHub rulesets explicitly
  support GitHub Apps on bypass lists. `main-guard` has no bypassers, so this needs a policy change.
- **Tag first**: the release ships before the changelog section exists on main; the changelog PR lands
  afterwards. Verified in hashicorp/vault (the 2.2.1 section reached `main` about 21 hours after the
  tag), prometheus/prometheus (release branch merged back to `main` 4 hours after the tag) and
  grafana/loki ("Reflect recent releases in CHANGELOG.md on main branch" PRs).
- **No changelog file**: the GitHub Release is the changelog — github/cli and ollama ship no
  `CHANGELOG.md` at all; charmbracelet projects and GoReleaser itself generate the release notes from
  commits (GoReleaser `changelog:` config, git-cliff, GitHub's notes generator).
- Fit for this repo: **the release PR**. It keeps the Keep a Changelog file, the hand-written entries,
  the PR-added-entry rule from PR #80 and the hand-pushed annotated tag; only the delivery of the cut
  changes, from a direct push to a merged PR.

## 1. The release PR: cut the section in a PR, merge it, tag the merged commit

The release manager opens a small PR that renames the unreleased changelog section into the version
section (and bumps version files when they exist), merges it through the ordinary PR process, and then
tags the commit that the merge produced on the default branch. The tag is on the protected branch by
construction.

- **traefik/traefik** runs every release like this: the changelog history shows 428 PRs titled
  "Prepare release vX" ([search](https://github.com/traefik/traefik/pulls?q=%22Prepare+release%22+in%3Atitle)).
  For v3.7.17, [PR #14179](https://github.com/traefik/traefik/pull/14179) "Prepare release v3.7.17"
  was opened at 2026-10-29T09:52:15Z and merged at 10:08:06Z; the merged
  [commit 04e8ccf8](https://github.com/traefik/traefik/commit/04e8ccf84ada1dcb450cec06554fe27812903134)
  touches only `CHANGELOG.md`, and the tag v3.7.17 points at that commit. Their sections are
  `## [vX.Y.Z](tree-link) (YYYY-MM-DD)` with per-entry PR links
  ([CHANGELOG.md](https://github.com/traefik/traefik/blob/38d53dd06c53c7e9bd08e57b217e72686d823610/CHANGELOG.md)).
- **hashicorp/terraform** cuts with "Prepare before X release" PRs: the squash commit
  [5e83043c](https://github.com/hashicorp/terraform/commit/5e83043c788062d9bc735e6b86809ac6c0b2a409)
  "Prepare before 1.17.0-alpha20260812 release
  ([#39006](https://github.com/hashicorp/terraform/pull/39006))" (2026-08-12T10:47:50Z) changes
  `CHANGELOG.md` and `version/VERSION`, and the tag v1.17.0-alpha20260812 sits on it. A follow-up
  "Cleanup after X release" PR reopens the `(Unreleased)` section, and "Prepare main branch for
  post-v1.17 development ([#39165](https://github.com/hashicorp/terraform/pull/39165))" moves to the
  next minor line. The changelog keeps a `## 1.18.0 (Unreleased)` section at the top
  ([CHANGELOG.md](https://github.com/hashicorp/terraform/blob/main/CHANGELOG.md)). The 1.13.0 cut took
  ten minutes end to end ([#37470](https://github.com/hashicorp/terraform/pull/37470), opened and
  merged on 2025-08-20).
- **go-task/task** is the same flow minus the PR: maintainers push `chore: changelog for #NNNN`
  commits to `main` after each merge, and the cut is the "v3.54.0" commit
  [b5aef1c2](https://github.com/go-task/task/commit/b5aef1c287a9be8038347a17646bdf4bc7185bce)
  (2026-10-01T20:21:39Z), which renames `## Unreleased` into `## v3.54.0 - 2026-10-01` and bumps
  `internal/version/version.txt`; the annotated tag v3.54.0 follows two seconds later and points at it.
  No PR for it exists — the flow works only because maintainers can push to `main`.
  ([CHANGELOG.md](https://github.com/go-task/task/blob/main/CHANGELOG.md))
- **golangci/golangci-lint** cut through PRs in the past
  ([#5765](https://github.com/golangci/golangci-lint/pull/5765),
  [#1457](https://github.com/golangci/golangci-lint/pull/1457)) but today the release manager pushes
  the "chore: prepare release" commit
  [25c5369a](https://github.com/golangci/golangci-lint/commit/25c5369a93aacb8cccea0d46e1569ca4a194564d)
  (2026-10-16, `CHANGELOG.md` plus version constants) straight to `main` and tags it eight minutes
  later. Their post-release chores, however, are automated as PRs from a fork because the bot has no
  push access: [post-release.yml](https://github.com/golangci/golangci-lint/blob/main/.github/workflows/post-release.yml)
  uses `peter-evans/create-pull-request` with `push-to-fork: golangci-releaser/golangci-lint`.

## 2. Bot-managed release PRs: release-please and changesets

Both tools replace the "cut by hand" step with a permanently refreshed release PR. They differ in
what they require as input and what they do when the PR merges.

- **release-please** ([googleapis/release-please](https://github.com/googleapis/release-please)) needs
  Conventional Commits: "It does so by parsing your git history, looking for Conventional Commit
  messages, and creating release PRs." The mechanics are documented in the README: "When you're ready
  to tag a release, simply merge the release PR. Both squash-merge and merge commits work with
  Release PRs. When the Release PR is merged, release-please takes the following steps: 1. Updates
  your changelog file (for example `CHANGELOG.md`), along with other language specific files (for
  example `package.json`). 2. Tags the commit with the version number 3. Creates a GitHub Release
  based on the tag." The project releases itself this way — 203 "chore(main): release" PRs, for
  example [#2888](https://github.com/googleapis/release-please/pull/2888) "chore(main): release
  17.11.2", merged two minutes after the bot opened it. The maintainer keeps control of the merge; the
  bot owns the tag.
- **changesets** ([changesets/changesets](https://github.com/changesets/changesets)) requires
  `.changeset/*.md` files, markdown entries with YAML front matter for the bump type, added in feature
  PRs: "The best time to capture this information is when submitting a PR"
  ([detailed-explanation](https://github.com/changesets/changesets/blob/main/docs/detailed-explanation.md)).
  In CI, [changesets/action](https://github.com/changesets/action) "Version packages and create or
  update a pull request with the changes" — the "Version Packages" PR that bumps versions and writes
  the changelogs — and "Publish packages to npm" once it merges. changesets publishes through CI on
  merge; nothing tags automatically until that publish runs. The project releases itself this way —
  137 "Version Packages" PRs, for example
  [#2291](https://github.com/changesets/changesets/pull/2291), open for a week and then merged.

## 3. Bots that push: give the bot an exception

The third pattern keeps a release commit on the protected branch but pushes it with elevated rights.

- **semantic-release** computes the version from commit messages in CI and needs write access to the
  repository: "semantic-release requires authentication to push to your repository and publish
  releases" and "requires push access to the project Git repository in order to create Git tags",
  satisfied by `GH_TOKEN`/`GITHUB_TOKEN` — "In GitHub Actions, the workflow-provided `GITHUB_TOKEN`
  is automatically available; prefer this or a GitHub App installation token when available; use a
  personal access token only when required"
  ([CI configuration](https://semantic-release.org/usage/ci-configuration/)). The release commit
  itself comes from the [@semantic-release/git](https://github.com/semantic-release/git) plugin:
  "prepare: Create a release commit, including configurable file assets" — "for each release a
  release commit will be pushed to the remote Git repository". The same README warns: "Please
  consider our recommendation against making commits during your release ... to avoid unnecessary
  headaches." semantic-release's own releases show the alternative: the tag v25.0.9 sits on the
  squash-merged fix PR commit
  [8d905a56](https://github.com/semantic-release/semantic-release/releases/tag/v25.0.9), with no
  release commit at all.
- **The GitHub mechanism** is the ruleset bypass list: "When you create a ruleset, you can allow
  certain users to bypass the rules in the ruleset. This can be users with a certain role, such as
  repository administrator, or it can be specific teams or GitHub Apps"
  ([About rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets)).
  So a release bot can be granted a sanctioned exception by adding its App (or a token's owner) to
  the bypass list of the protecting ruleset. `main-guard` has no bypassers today, so this pattern
  needs a deliberate policy change, not just tooling.

## 4. Tag first: the changelog is documentation that can lag the tag

Here the release goes out first, and the changelog section reaches `main` only afterwards — the
changelog is treated as documentation, not as an input to the release.

- **hashicorp/vault** collects entries as files inside feature PRs: "Please include a file within your
  PR named `changelog/#.txt`, where `#` is your pull request ID" and "You do not need to include the
  link at the end of the summary that appears in CHANGELOG.md, those are generated automatically by
  the changelog-building process"
  ([CONTRIBUTING.md](https://github.com/hashicorp/vault/blob/main/CONTRIBUTING.md)). The release
  itself is tagged by an automation PR — the tag v2.2.1 sits on
  [0e7c41b9](https://github.com/hashicorp/vault/commit/0e7c41b936) "This is an automated pull request
  to build all artifacts for a release (#32173)" (2026-10-22T20:27:49Z) — while the `## 2.2.1`
  section ("### October 22, 2026") first lands on `main` with "Update CHANGELOG for 2.2.1, 1.21.13,
  1.19.24" ([0c511745](https://github.com/hashicorp/vault/commit/0c511745), 2026-10-23T17:15:57Z),
  about 21 hours after the tag. Earlier sections went the same way, for example "Update CHANGELOG 2.2.0
  release ([#19893](https://github.com/hashicorp/vault/pull/19893))" (2026-10-15).
  ([CHANGELOG.md](https://github.com/hashicorp/vault/blob/main/CHANGELOG.md))
- **prometheus/prometheus** cuts the section on a release branch and merges it back: the tag v3.15.1
  sits on the release-branch merge
  [bae3eced](https://github.com/prometheus/prometheus/commit/bae3eced0a) of
  [#20127](https://github.com/prometheus/prometheus/pull/20127) "prepare-3.15" (tag 2026-10-21T13:24:37Z),
  and the `## 3.15.1 / 2026-10-21` section reaches `main` with "Merge branch 'release-3.15'" at
  17:33:22Z, four hours after the tag. The `main` changelog keeps a `## main / unreleased` section on
  top ([CHANGELOG.md](https://github.com/prometheus/prometheus/blob/main/CHANGELOG.md)).
- **grafana/loki** reconciles `main` after releases with PRs such as
  ["chore: Reflect recent releases in CHANGELOG.md on main branch (#21621)"](https://github.com/grafana/loki/pull/21621)
  (2026-04-20) and ["docs: Update CHANGELOG for missing releases (#20263)"](https://github.com/grafana/loki/pull/20263).
- The reverse ordering also exists: etcd-io/etcd commits "CHANGELOG(3.7): Set 3.7.3 release date"
  (2026-10-22T21:51:47Z) before the tag v3.7.3 (2026-10-22T23:03:14Z) — the dated section is on `main`
  first, like pattern 1.

## 5. No changelog file: the GitHub Release is the changelog

- **github/cli** keeps no `CHANGELOG.md` at all (not in the
  [repository root](https://github.com/cli/cli), and a code search for a changelog path returns
  nothing), and its releases are GitHub's automatically generated notes — "## What's Changed" with PR
  links and a "Full Changelog" compare link
  ([v2.103.3](https://github.com/cli/cli/releases/tag/v2.103.3)). GitHub documents the generator:
  "Automatically generated release notes provide an automated alternative to manually writing release
  notes" ([docs](https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes)).
- **ollama/ollama** likewise ships no `CHANGELOG.md`
  ([root](https://github.com/ollama/ollama)); its release notes are hand-written highlights plus the
  compare link ([v0.40.13](https://github.com/ollama/ollama/releases/tag/v0.40.13)). The "Update
  Changelog for vX" PR flow this research was asked to check could not be verified: no such PR titles
  exist in the repository today.
- **charmbracelet** projects (bubbletea, lipgloss) have no `CHANGELOG.md`; their release notes are
  generated from commits by GoReleaser through the shared config
  [charmbracelet/meta/goreleaser-lib.yaml](https://github.com/charmbracelet/meta/blob/main/goreleaser-lib.yaml)
  — `changelog: sort: asc, use: github`, a `{{ .SHA }}: {{ .Message }} (@author)` format, commit
  filters and groups (New!/Fixed/Docs/Deps) — visible in the
  [bubbletea v2.1.1](https://github.com/charmbracelet/bubbletea/releases/tag/v2.1.1) and
  [lipgloss v2.0.6](https://github.com/charmbracelet/lipgloss/releases/tag/v2.0.6) notes.
- **GoReleaser itself** does not keep a `CHANGELOG.md` — the only file of that name in the repository
  is its own documentation page — and its releases carry a "## Changelog" generated by the
  `changelog:` block of its [.goreleaser.yaml](https://github.com/goreleaser/goreleaser/blob/main/.goreleaser.yaml)
  (`sort: asc`, `use: github`, per-commit format with author logins, and a long `filters.exclude`
  list), for example [v2.19.0](https://github.com/goreleaser/goreleaser/releases/tag/v2.19.0). The
  docs describe exactly this feature: "You can customize how the changelog is generated using the
  `changelog` section in the config file", with `use: github|gitlab|gitea|github-native` (the last
  "uses the GitHub release notes generation API"), format templates and commit filters
  ([goreleaser.com/customization/changelog](https://goreleaser.com/customization/changelog/)). The
  same page notes that `disable: true` also ignores "any changelog files passed via
  `--release-notes`" — so release notes can come from commits or from a file.
- **git-cliff** is the generator itself: "git-cliff can generate changelog files from the Git history
  by utilizing conventional commits as well as regex-powered custom parsers"
  ([README](https://github.com/orhun/git-cliff)). The project still keeps a `CHANGELOG.md`, cut in a
  maintainer-pushed "chore(release): prepare for v2.14.2" commit
  ([5d996de8](https://github.com/orhun/git-cliff/commit/5d996de8220d26c0cb8ff329d31791570cab262f),
  2026-09-18, six seconds before the tag), while its release workflow generates the GitHub Release
  notes at publish time with the
  [git-cliff action](https://github.com/orhun/git-cliff/blob/main/.github/workflows/cd.yml)
  (`--latest --no-exec`) and feeds them to the upload step as the release body.

## Unverified

- What generates golangci-lint's `CHANGELOG.md` sections: the file is machine-formatted ("### v2.15.0
  / *Released on 2026-10-16*" with numbered lists and a `<!-- START --->` marker), but the repo has
  no cliff.toml and no git-cliff references, so the tool is unidentified.
- Whether the surveyed repos protect their default branch with PR-only rulesets: the public rulesets
  API was rate-limited, so the flows above are stated from their PR, commit and tag evidence only.
- The historical changelog flows of github/cli and ollama/ollama: neither repo has a changelog file
  today, and no commits on their default branches ever touch one, so the older flows cannot be
  verified from the current repository state.
- git-cliff.com blocked automated reading; the git-cliff README in the source repository is cited
  instead.

## Fit for this repo

what-to-run's constraints: a Keep a Changelog `CHANGELOG.md` whose entries are written by hand and
landed in feature PRs (enforced by `scripts/check-changelog.sh` from PR #80); the version lives only
in the annotated tag, pushed by hand after the maintainer confirms; `.github/workflows/release.yml`
publishes on tag push and requires the tag commit to be an ancestor of `origin/main`; merges are
manual and agents open draft PRs.

- **Pattern 1 (release PR) fits directly** and is the recommendation. The cut becomes a one-file PR:
  rename `## [Unreleased]` into `## [X.Y.Z] - YYYY-MM-DD`, add a fresh empty `## [Unreleased]`, open it
  as a draft (docs-only, so the PR #80 job passes trivially — the check fires only when `internal/`,
  `cmd/`, `npm/` or `scripts/` change). The maintainer marks it ready and merges it when asked in
  that turn, exactly like every other PR. Any merge method works, because the tag does not target the
  PR head but whatever commit the merge put on `main`. Then: fetch, verify `origin/main`, run
  `task all`, tag `origin/main`'s head with `git tag -a vX.Y.Z -m vX.Y.Z`, push the tag — the
  workflow's `git merge-base --is-ancestor HEAD origin/main` passes by construction. The `release`
  skill changes in one place: step 2's "Commit it now; step 3 pushes it to `main`" becomes "open the
  draft cut PR; after the maintainer merges it, tag the merged commit".
- One follow-on from pattern 1: a squashed cut PR leaves one commit on `main` inside the next tag's
  commit range, and today's `.goreleaser.yaml` `changelog:` block excludes only `^Merge `, so
  "docs: cut the changelog for vX.Y.Z (#N)" would appear in the next release's generated notes.
  Either exclude that subject in the GoReleaser filters or accept one housekeeping line.
- **Pattern 4 (tag first) is the fallback** if even one PR per release is unwanted: tag the release
  with the section still under `[Unreleased]`, then merge the cut PR afterwards, like Vault. The
  trade-off is explicit: the `CHANGELOG.md` inside the tag does not carry the version header, so the
  skill's rule "the tag must contain the renamed section" is dropped instead of the direct push.
- **Pattern 2 does not fit**: release-please requires Conventional Commits and writes the changelog
  from commit subjects, replacing the hand-written entries that the `docs-writing` skill and PR #80
  exist to keep; changesets requires `.changeset/*.md` files, a second bookkeeping format next to
  Keep a Changelog. Both take the tag (or the publish) away from the maintainer's confirmed,
  hand-pushed annotated tag.
- **Pattern 3 is excluded by policy**, not by tooling: it needs a bypass entry in `main-guard` (a
  GitHub App on the ruleset's bypass list, or a token whose owner is exempt), and the ruleset was
  chosen with no bypassers.
- **Pattern 5 contradicts the repo's decisions**: `CHANGELOG.md` is the user-facing changelog that PR
  #80's CI job feeds; deleting it would also orphan the Keep a Changelog format and the skill's
  release-summary step. GoReleaser's commit-derived release notes already exist as a second,
  machine-generated record, so the repo already has pattern 5's convenience without giving up the
  curated file.
- If the cut PR should ever be opened by a bot instead of an agent, golangci-lint's
  `post-release.yml` is the template: `peter-evans/create-pull-request` with `push-to-fork` lets a
  token holder that cannot push `main` still open the PR.
