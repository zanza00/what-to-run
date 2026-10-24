# Additional installation methods: analysis and rollout plan

Status: proposal, not implemented. Target: the next minor release (`0.9.0`)
or later.

This note evaluates additional installation methods for `what-to-run`, with a
focus on Windows. It records what already exists, which package managers are
worth adding, what each channel requires from this repository and from external
services, and in which order the work should land.

## Summary

The recommended sequence is:

1. **WinGet** — the official Windows package manager. It has the largest
   reach, needs no administrator rights, supports ARM64 natively, and its
   manifests can be updated from the existing GoReleaser release.
2. **Scoop** — the cheapest channel to add. It is user-level, supports ARM64
   natively, and a personal bucket needs no review.
3. **Chocolatey** — useful for enterprise and automation users, but the most
   expensive option: a separate Windows job, manual moderation, and an x64
   package that runs under emulation on ARM64.

Code signing is a separate, optional follow-up. The Microsoft Store/MSIX,
Snap, Flatpak, MSYS2, and MacPorts are deliberately out of scope for now.

## Current distribution

| Channel | Status |
| --- | --- |
| Homebrew | Published through `scripts/publish-homebrew.py`; required secret `HOMEBREW_TAP_TOKEN` |
| mise | Installs from the GitHub releases with the `github:` backend |
| apt repository | Published on `gh-pages`; required secret `APT_GPG_PRIVATE_KEY` |
| RPM repository | Published on `gh-pages`; same GPG key as apt |
| `.deb` and `.rpm` assets | Attached to each GitHub release |
| AUR | `what-to-run-bin` is implemented but skipped while `AUR_SSH_PRIVATE_KEY` is unset; AUR registration is closed |
| Tarballs and Windows zip | Attached to each GitHub release |
| `go install` | Compiles and installs `wtr` locally |

Windows currently has no package-manager channel: users either extract the zip
by hand or run `go install`. That is the gap this proposal closes.

### Release pipeline constraints

The release pipeline is tag-driven and runs GoReleaser on `ubuntu-latest`:

- `.github/workflows/release.yml` validates the tag, calls the reusable
  `checks.yml`, builds the assets, creates the GitHub release, and runs the
  custom publishers.
- The release carries exactly eleven assets: `checksums.txt`, four tarballs,
  two Windows zips, two `.deb` files, and two `.rpm` files. The workflow and
  `.agents/skills/release/scripts/verify-release.sh` both assert that exact
  set.
- The Windows zip contains `wtr.exe`. Every new channel must install the `wtr`
  command; the package identifiers stay as they are.
- `checks.yml` runs `goreleaser check` in its `cross` job, so a broken
  `.goreleaser.yaml` fails CI before a tag is pushed.
- The Homebrew, AUR, and package-repository publishers are Python scripts with
  tests under `scripts/`. Secrets that are optional (AUR and apt/RPM) are
  skipped with a warning rather than failing the release.
- GoReleaser's `winget` and `scoops` pipes are `ContinueOnError`: a failure in
  either channel does not fail the release. Verification must check them
  explicitly, otherwise a silent failure is only noticed by users.

New package-manager channels should not add assets to the GitHub release. They
publish manifests elsewhere (a fork, a bucket, or a Chocolatey package), so the
eleven-asset invariant stays unchanged.

## Windows package manager landscape

| Tool | Scope | Admin | ARM64 | Publication model | Effort |
| --- | --- | --- | --- | --- | --- |
| **WinGet** | Microsoft's package manager, built into Windows 10/11 with App Installer | No | Native | Pull request against `microsoft/winget-pkgs`; the first package is reviewed manually, later versions are PRs too | Medium |
| **Scoop** | User-level package manager, popular with developers | No | Native | Manifest in a Git bucket; a personal bucket has no review | Low |
| **Chocolatey** | Machine-level package manager, common in enterprise environments | Yes | x64 under emulation | Account, API key, and moderation on `community.chocolatey.org` | High |
| **Microsoft Store/MSIX** | Store distribution with Microsoft-managed signing | No | Native | Partner Center and MSIX certification | Very high |
| **MSYS2/pacman** | MSYS2 package repository | No | Partial | PKGBUILD in the MSYS2 repositories | Medium, niche |
| **Ninite / Patch My PC** | Closed catalogues; no public self-service submission | — | — | — | Not applicable |
| **UniGetUI** | GUI front-end for WinGet, Scoop, and Chocolatey | — | — | No separate packaging | None |

WinGet, Scoop, and Chocolatey are complementary: WinGet covers ordinary
Windows users, Scoop covers developers who want per-user installs, and
Chocolatey covers machines managed by administrators and automation. They can
coexist; the `PATH` order decides which `wtr` wins, which is a known and
acceptable limitation.

## WinGet

### Why first

- It is the default package manager on Windows 11 and on Windows 10 machines
  with an up-to-date App Installer.
- Portable installs need no administrator rights.
- The manifest supports both `amd64` and `arm64` natively.
- The zip with a single portable file maps directly to `wtr.exe`, which is
  already shipped in the Windows archives.
- GoReleaser OSS can generate the manifests and open the update pull request
  from the existing release job.

### One-time prerequisites

- **Package identifier**: use `zanza00.WhatToRun`. The first segment must match
  the publisher and should stay stable; changing it later means publishing a
  new package. The identifier is currently free in `winget-pkgs`.
- **Microsoft CLA**: the GitHub account that opens the pull request must sign
  the Microsoft CLA.
- **Fork**: create `zanza00/winget-pkgs` as a fork of
  `microsoft/winget-pkgs`. GoReleaser can sync the fork but does not create it.
- **Token**: a classic personal access token with the `public_repo` scope,
  stored as the `WINGET_TOKEN` Actions secret. `GITHUB_TOKEN` cannot open a
  pull request against another repository.
- **First package**: open the first manifest as a normal "New package" pull
  request, manually or with `komac new`/`wingetcreate`, and wait for it to be
  merged. Review times vary; the first package commonly takes one to three
  weeks. Later versions can be automated.

### Manifest generation

GoReleaser's `winget` pipe selects the Windows zip archives, emits
`InstallerType: zip` with `NestedInstallerType: portable`, and derives
`NestedInstallerFiles` from the binaries inside each archive. Because the zip
contains `wtr.exe`, the generated manifest aliases `wtr`; no separate
packaging step is needed.

### Configuration sketch

```yaml
winget:
  - name: what-to-run
    publisher: zanza00
    package_identifier: zanza00.WhatToRun
    package_name: What to Run
    short_description: Task picker for the commands you can never remember in a repository
    license: MIT
    license_url: https://github.com/zanza00/what-to-run/blob/main/LICENSE
    homepage: https://github.com/zanza00/what-to-run
    publisher_url: https://github.com/zanza00/what-to-run
    publisher_support_url: https://github.com/zanza00/what-to-run/issues
    author: Simone Picciani
    tags:
      - cli
      - task-runner
      - developer-tools
    release_notes_url: https://github.com/zanza00/what-to-run/releases/tag/{{ .Tag }}
    repository:
      owner: zanza00
      name: winget-pkgs
      branch: what-to-run-{{ .Version }}
      token: "{{ .Env.WINGET_TOKEN }}"
      pull_request:
        enabled: true
        base:
          owner: microsoft
          name: winget-pkgs
          branch: master
    skip_upload: auto
```

`skip_upload: auto` keeps pre-releases out of the package manager. The
`branch` must be unique per version so that a slow review does not stack
changes on the fork's default branch.

### Release workflow changes

- Pass `WINGET_TOKEN` to the GoReleaser step.
- Treat the channel as optional during rollout: if the secret is unset, emit a
  warning like the AUR and apt/RPM steps do, instead of failing the release.
- Add a verification step that checks the pull request exists (or that the
  version is already merged), not that it is live immediately.
- GoReleaser does not fill in `winget-pkgs`' pull request template. Set
  `pull_request.body` to the repository's checklist, or keep the first pull
  request manual and use the automated flow for later versions.

### Verification semantics

- `ok`: the manifest is merged into `microsoft/winget-pkgs`.
- `pending`: the pull request is open and waiting for review.
- `FAIL`: neither the merged path nor an open pull request exists while
  `WINGET_TOKEN` is configured.

The release skill and `verify-release.sh` must distinguish these states.
Treating `pending` as a failure would make every release fail for an external
review that the project does not control.

### Risks

- The first pull request may be delayed or require changes for publisher/URL
  policy. The installer URL is the project's own GitHub release and
  `PublisherUrl` points at the same repository, which is the expected
  discoverable source.
- The pull request is asynchronous. The release must not depend on it.
- A classic PAT with `public_repo` is powerful; store it only as an Actions
  secret and rotate it.
- If a release happens while the previous update PR is still open, a second PR
  may be opened. That is acceptable, but the older one should be merged first
  when possible.

### Documentation

Add a WinGet section to `docs/install.md`:

```powershell
winget install --id zanza00.WhatToRun -e
winget upgrade --id zanza00.WhatToRun -e
winget uninstall --id zanza00.WhatToRun
```

Verify the install with `wtr --version`. Mention that WinGet may lag a release
by a few days while the manifest pull request is reviewed. Do not advertise the
command in `README.md` before the first package is actually available.

### Effort

About four to eight hours of repository work, plus the external review of the
first package.

## Scoop

### Why second

- Lowest implementation cost of the three Windows channels.
- No administrator rights, no account, no review.
- Supports both `amd64` and `arm64` in the same manifest.
- GoReleaser can write the manifest directly to a personal bucket.

### One-time prerequisites

- Create the public repository `zanza00/scoop-bucket`.
- Create a fine-grained personal access token with `Contents: Read and write`
  on that repository, stored as `SCOOP_BUCKET_TOKEN`.
- Create a short `README.md` in the bucket with the install commands.

### Configuration sketch

```yaml
scoops:
  - name: what-to-run
    homepage: https://github.com/zanza00/what-to-run
    description: Task picker for the commands you can never remember in a repository
    license: MIT
    repository:
      owner: zanza00
      name: scoop-bucket
      branch: main
      token: "{{ .Env.SCOOP_BUCKET_TOKEN }}"
    skip_upload: auto
```

GoReleaser writes a manifest with `64bit` and `arm64` entries and a `bin`
array containing `wtr.exe`, so Scoop creates the `wtr` shim. Decide whether the
manifest lives at the bucket root or under `bucket/`, then verify
`scoop bucket list` and `scoop install` against a test bucket.

### User commands

```powershell
scoop bucket add zanza00 https://github.com/zanza00/scoop-bucket
scoop install zanza00/what-to-run
scoop update what-to-run
scoop uninstall what-to-run
```

### Trade-offs

- GoReleaser does not add `checkver`/`autoupdate` to the generated manifest.
  That is fine while every release rewrites the manifest, but it leaves no
  self-healing path if a publisher run fails. A small
  `scripts/publish-scoop.py` with tests, modelled on `publish-homebrew.py`,
  would add `checkver`/`autoupdate`, make recovery local, and give the bucket's
  scheduled updater a way to catch up. This is optional but recommended if the
  project wants the same recovery story as Homebrew and the AUR.
- A personal bucket requires an extra `scoop bucket add` command. Submitting to
  `ScoopInstaller/Extras` would remove it, but that repository is reviewed and
  a new project may not meet its notability criteria. Revisit it if adoption
  grows.

### Verification semantics

The bucket manifest is live as soon as GoReleaser pushes it, so this channel
can be checked strictly: the manifest must contain the release version, both
architecture URLs, and `wtr.exe`. A missing or stale manifest is a `FAIL`, not
a `pending` state.

### Effort

About two to four hours, with immediate publication.

## Chocolatey

### When it is worth it

Chocolatey is the right complement for enterprise users and machines managed
by administrators. It is not the best first channel: the package is
machine-wide, every version is moderated manually, and the Windows package is
x64 only.

### One-time prerequisites

- Create an account on `community.chocolatey.org` and copy the API key into the
  `CHOCOLATEY_API_KEY` Actions secret.
- Reserve the package ID `what-to-run`; it is currently free.
- Publish the first version and follow it through moderation. Review times
  vary, and subsequent versions are moderated too, usually faster.
- Keep the package source in this repository under `packaging/chocolatey/` so
  the `packageSourceUrl` points at a public, readable location.

### Packaging

A Chocolatey package is a `.nupkg` containing:

- `what-to-run.nuspec` with the package metadata and a `$version$` placeholder;
- `tools/chocolateyinstall.ps1`, which downloads the Windows zip with
  `Install-ChocolateyZipPackage` and verifies `Checksum64` from
  `checksums.txt`.

Chocolatey creates shims for every executable in the package's tools directory,
so `wtr` becomes available without extra work.

### Build and publish job

Do **not** add `chocolateys:` to the main GoReleaser configuration. The
release job runs on Linux and the pipe calls `choco pack`, which is not
available there. Use a separate Windows job after the GitHub release exists:

```yaml
publish-chocolatey:
  needs: publish
  runs-on: windows-latest
  if: vars.CHOCOLATEY_ENABLED == 'true'
  steps:
    - uses: actions/checkout@v7
      with:
        ref: ${{ github.sha }}
    - name: Download release assets
      shell: pwsh
      env:
        GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      run: |
        gh release download "$env:GITHUB_REF_NAME" `
          --pattern "what-to-run_*_windows_amd64.zip" `
          --pattern checksums.txt `
          --dir dist --clobber
    - name: Pack, test, push
      shell: pwsh
      env:
        CHOCOLATEY_API_KEY: ${{ secrets.CHOCOLATEY_API_KEY }}
      run: ./scripts/publish-chocolatey.ps1 -Tag "$env:GITHUB_REF_NAME" -Dist dist
```

The script should render the nuspec and install script, run `choco pack` into
`dist`, install from that directory
(`choco install what-to-run --source dist --version <version>`), smoke-test
`wtr --version`, and only then run `choco push`. The first version should be
pushed manually and monitored through moderation; the job can be enabled
afterwards.

### User commands

```powershell
choco install what-to-run
choco upgrade what-to-run
choco uninstall what-to-run
```

### ARM64

Chocolatey's helper functions do not have first-class ARM64 support, and
Chocolatey itself runs as an x64 process under emulation on Windows on ARM.
The package therefore installs the x64 build there. Users who need a native
ARM64 binary should use WinGet or Scoop. A native Chocolatey path is possible
with an explicit `PROCESSOR_ARCHITECTURE`/`PROCESSOR_ARCHITEW6432` branch, but
it needs testing on Windows ARM64 and is not part of the initial scope.

### Verification semantics

- `ok`: the version is visible in the community repository API.
- `pending`: the package was accepted for moderation but is not approved yet.
- `FAIL`: the push failed or the package is missing while the secret is set.

### Effort

About one to two days of repository work, plus account setup and moderation.

## Other channels

### Nix

A flake in the repository would allow `nix run github:zanza00/what-to-run` and
`nix profile install`, without an external review. It is useful for Linux and
macOS users but does not provide a native Windows install (only WSL). A
`buildGoModule` derivation with a `vendorHash` is a small, self-contained
addition. Submitting the package to `nixpkgs` is a separate project with its
own review and maintenance expectations.

### aqua

One YAML file in `aquaproj/aqua-registry` would make the CLI available to aqua
and mise users, including on Windows. The registry's updater keeps versions
current. This is a cheap complement, not a Windows-first channel.

### Alpine `.apk`

`nfpms` can also emit `.apk` packages, but that adds assets to the release and
therefore changes the eleven-asset invariant in `release.yml` and the release
verifier. It is low value unless there is demand from Alpine users; a proper
Alpine repository would be additional work.

### Microsoft Store/MSIX

The Store handles signing and updates, but packaging a CLI as MSIX requires an
app execution alias, a full-trust desktop extension, Partner Center
registration, and certification. It duplicates WinGet's reach with much more
work. Not recommended at this stage.

### Snap and Flatpak

The picker launches the user's shell and runs project tasks, which conflicts
with the confinement models of both formats. Snap's classic confinement
requires manual review; Flatpak is not a good fit for a terminal application
that must execute arbitrary project commands. Not recommended.

### MSYS2

A PKGBUILD in the MSYS2 repositories would target a niche audience; the native
Windows binary already works outside MSYS2. Low value.

### MacPorts

Homebrew already covers macOS. Maintaining a second macOS package manager
port is not justified.

### Ninite, Patch My PC, UniGetUI

Ninite and Patch My PC are closed catalogues without self-service submission.
UniGetUI is a GUI front-end over WinGet, Scoop, and Chocolatey; it requires no
separate packaging once those channels exist.

## Cross-cutting work

### Code signing

None of the three package managers requires an Authenticode signature, but the
binaries remain unsigned and the SmartScreen warning in `docs/install.md`
stays true. Two options are worth evaluating separately:

- **SignPath Foundation** provides free OV-level signing for eligible open
  source projects, with a GitHub Actions integration and a manual approval per
  release.
- **Azure Artifact Signing** (formerly Trusted Signing) costs roughly 10 USD
  per month at the time of writing and is suitable for a fully automated flow.

Signing must happen before the archives are created, or the archives must be
signed and the checksums regenerated. This is a separate change, not a
prerequisite for WinGet, Scoop, or Chocolatey.

### Documentation

- `docs/install.md`: add WinGet, Scoop, and Chocolatey under Windows, with
  install, upgrade, and uninstall commands; keep the zip as the manual
  fallback. The page already explains the two-name upgrade: new channels must
  install only `wtr` and must not resurrect `what-to-run`.
- `README.md`: add the WinGet one-liner and link to the installation page,
  but only once the first package is live.
- `docs/development.md`: document the new secrets, the Windows job, and the
  asynchronous nature of WinGet and Chocolatey.
- `AGENTS.md`: update the release section with the new channels, secrets,
  external repositories, and the fact that the release asset list does not
  change.

### Release workflow and verification

- Pass `WINGET_TOKEN` and `SCOOP_BUCKET_TOKEN` to the GoReleaser step.
- Add the Windows Chocolatey job after `publish`.
- Add explicit checks for the new channels; the GoReleaser pipes continue on
  error and would otherwise hide failures.
- Extend `.agents/skills/release/scripts/verify-release.sh` with `ok`,
  `pending`, `skip`, and `FAIL` states:
  - WinGet: merged manifest or open PR;
  - Scoop: manifest in the bucket with the right version and URLs;
  - Chocolatey: version in the community API, or a pending moderation state.
- Update `.agents/skills/release/SKILL.md` and
  `.agents/skills/release/references/recovery.md` with the new secrets,
  commands, and recovery paths.

### Recovery

The existing recovery model treats publishers as idempotent and re-runnable.
The new channels should follow it:

- **WinGet**: if the automated PR was not opened, run `komac update` (or
  `wingetcreate update`) against the published assets, or re-run a small
  recovery workflow. For the very first package, use `komac new`.
- **Scoop**: regenerate the bucket manifest from the released checksums. A
  Python publisher makes this a local command, like Homebrew and the AUR.
- **Chocolatey**: re-run the Windows packaging workflow with the same tag. If
  a version was rejected during moderation, fix the package and re-push the
  same version; do not bump the version for a rejected upload.

### Testing

- `task all` before committing, as required by the repository; it includes
  `fmt-md`.
- `task check-md` for a check-only Markdown pass.
- `goreleaser check` and `goreleaser release --snapshot --clean`; inspect the
  generated WinGet and Scoop manifests under `dist/`.
- `winget validate --manifest <directory>` on Windows, then install from the
  manifest and check `wtr --version`.
- Install the Scoop manifest from a test bucket and check the `wtr` shim.
- Run `choco pack`, install from the local package, and run `wtr --version`
  before pushing.
- Test upgrades from the previous release, not only clean installs.
- Check that uninstalling through each manager removes the `wtr` command.
- The existing cross-build job covers the Windows binaries; these channels are
  packaging work, so no PTY tests are needed. The Windows job is the end-to-end
  test for Chocolatey.

## Rollout plan

Use small stacked pull requests, as required by `AGENTS.md`. Each layer updates
its own configuration, documentation, and verification; none of them merges
itself.

1. **WinGet** — `.goreleaser.yaml`, `WINGET_TOKEN`, release workflow,
   `docs/install.md`, release skill and verifier. Open the first manifest PR
   manually, wait for the merge, then enable automated updates.
2. **Scoop** — create `zanza00/scoop-bucket`, add the `scoops:` block and
   `SCOOP_BUCKET_TOKEN`, update the docs and verifier.
3. **Chocolatey** — add `packaging/chocolatey/`, the Windows publish job,
   `CHOCOLATEY_API_KEY`, the docs and the recovery path. Push the first
   version manually and follow moderation.
4. **Code signing** (optional) — choose SignPath Foundation or Azure Artifact
   Signing, integrate it before archiving, and update the install docs.
5. **Later, if there is demand** — Nix flake, aqua registry, Alpine `.apk`.

## Risks and non-goals

- **Do not add release assets** for these channels. The eleven-asset check in
  `release.yml` and the release skill is intentional.
- **Do not make WinGet or Chocolatey block the release.** They are
  asynchronous: a pull request and a moderation queue are outside the
  project's control.
- **Do not advertise a channel before it is live.** WinGet needs the first
  manifest merged; Chocolatey needs the first package approved.
- **Do not put `chocolateys:` in the Linux release job.** It calls
  `choco pack`.
- **Do not use `GITHUB_TOKEN` for WinGet.** It cannot open a pull request
  against `microsoft/winget-pkgs`.
- **Do not assume native ARM64 from Chocolatey.** Use WinGet or Scoop for
  that.
- **Do not treat `pending` as success.** A channel that never becomes live
  should be visible in the release verification as pending, not silently
  passed.

## Effort estimate

| Work | Repository effort | External wait |
| --- | --- | --- |
| WinGet | 4–8 hours | 1–3 weeks for the first package review |
| Scoop | 2–4 hours | None |
| Chocolatey | 1–2 days | Account setup and moderation (days to weeks) |
| Code signing | About 1 day plus CI integration | SignPath approval or Azure setup |
| Nix flake | 2–4 hours | None |
| aqua | 1–2 hours | Registry review |
| Store/MSIX | 1–2 weeks | Certification |

## References

Repository files:

- [`docs/install.md`](../install.md)
- [`docs/development.md`](../development.md)
- [`README.md`](../../README.md)
- [`AGENTS.md`](../../AGENTS.md)
- [`.goreleaser.yaml`](../../.goreleaser.yaml)
- [`.github/workflows/release.yml`](../../.github/workflows/release.yml)
- [`.agents/skills/release/SKILL.md`](../../.agents/skills/release/SKILL.md)
- [`scripts/verify-release.sh`](../../.agents/skills/release/scripts/verify-release.sh)

External documentation:

- [WinGet manifest schema](https://github.com/microsoft/winget-pkgs/blob/master/doc/manifest/schema/1.12.0/installer.md)
- [Submitting a WinGet manifest](https://learn.microsoft.com/en-us/windows/package-manager/package/repository)
- [GoReleaser WinGet publisher](https://goreleaser.com/customization/publish/winget/)
- [GoReleaser Scoop publisher](https://goreleaser.com/customization/publish/scoop/)
- [Scoop buckets](https://github.com/ScoopInstaller/Scoop/wiki/Buckets)
- [Chocolatey moderation](https://docs.chocolatey.org/en-us/community-repository/moderation)
- [Chocolatey package creation](https://docs.chocolatey.org/en-us/create/create-packages)
- [SignPath Foundation](https://signpath.org/)
- [Azure Artifact Signing](https://learn.microsoft.com/en-us/azure/trusted-signing/)
