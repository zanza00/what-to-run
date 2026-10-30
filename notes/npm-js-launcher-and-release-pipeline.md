# npm JS launchers and release pipelines for compiled binaries

Follow-up to `npm-as-install-method-for-binaries.md`, 2026-10-30. All claims verified against published
package files (unpkg) and repo sources; URLs inline. Unverified items are marked at the end.

## 1. Launcher mechanics (read from the real shims)

**Resolution.** Every launcher derives the platform package name from `process.platform` + `process.arch` and
uses `require.resolve()`, never a hardcoded `node_modules` path. The @vscode/ripgrep shim is the minimal ESM
example (<https://unpkg.com/@vscode/ripgrep@1.18.0/lib/index.js>):

```js
const require = createRequire(import.meta.url);            // require.resolve inside ESM
const arch = process.env.npm_config_arch || process.arch;  // honors npm --arch override
const binaryName = process.platform === 'win32' ? 'rg.exe' : 'rg';
const platformPkg = `@vscode/ripgrep-${process.platform}-${arch}`;
resolved = require.resolve(`${platformPkg}/bin/${binaryName}`);
// on failure throws: `Could not find ${platformPkg}. Ensure optionalDependencies
//  are installed for this platform (${process.platform}-${arch}).`
```

- lefthook uses a template literal: ``require.resolve(`lefthook-${os}-${arch}/bin/lefthook${extension}`)``
  with `win32`/`cygwin` mapped to `windows` + `.exe` (<https://unpkg.com/lefthook@2.2.2/get-exe.js>).
- deno composes `@deno/` + `platform-arch[-libc]` (<https://unpkg.com/deno@2.9.6/install_api.cjs>); esbuild
  resolves through `generateBinPath()`
  (<https://github.com/evanw/esbuild/blob/main/lib/npm/node-platform.ts>), which first honors the
  `ESBUILD_BINARY_PATH` env override (and ignores the value `/usr/bin/esbuild`, which a broken unofficial
  package set globally), then `require.resolve`s the platform package.
- libc: deno detects musl via code adapted from `detect-libc` and throws "Musl is not supported" (glibc-only
  channel); biome ships separate `-musl` packages instead. A Go static binary needs no libc split (esbuild
  ships a single `linux-x64`).

**Exec.** All real launchers spawn synchronously with `stdio: 'inherit'`, argv passthrough and exit-code
forwarding, no shell:

- deno: `spawnSync(exePath, process.argv.slice(2), { stdio: "inherit" }); process.exit(result.status ?? 1)`
  (<https://unpkg.com/deno@2.9.6/bin.cjs>); esbuild's whole shim is `execFileSync(binPath, argv, {stdio:'inherit'})`
  (<https://github.com/evanw/esbuild/blob/main/lib/npm/node-shim.ts>).
- lefthook: async `spawn(..., {stdio: 'inherit'})` but lossy exit code (`process.exit(1)` on any nonzero close,
  <https://unpkg.com/lefthook@2.2.2/bin/index.js>) — a defect to avoid.
- electron is the reference for signal forwarding: `child.kill(signal)` on SIGINT/SIGTERM/SIGUSR2 and
  `process.exit(child's code)` (<https://unpkg.com/electron@45.1.0/cli.js>). deno/esbuild rely on the shared
  foreground process group for Ctrl+C instead.
- Windows: direct spawn of the `.exe` (no `.cmd` shell hop); electron adds `windowsHide: false`; deno's
  postinstall rewrites `node_modules/.bin/deno.cmd`/`.ps1`/symlink to point at the native binary directly,
  "avoiding Node.js startup overhead on each invocation" (<https://unpkg.com/deno@2.9.6/install_api.cjs>,
  `replaceBinEntry`).

**Missing platform package.** Error messages name the expected package and next step: ripgrep's "Could not
find @vscode/ripgrep-linux-x64. Ensure optionalDependencies are installed for this platform (linux-x64)"; buf
explains the `--no-optional` case verbatim and falls back to `npm install`-ing the platform tarball into a
`downloaded-...` path (<https://unpkg.com/@bufbuild/buf@1.75.0/install.js>); deno throws "Could not find
executable for @deno/linux-x64-glibc at <path>".

**Postinstall as fallback only.** esbuild's `install.js` verifies the binary (sha256 against embedded
`binaryHashes`, `validateBinaryVersion` runs the binary and compares `--version` with the package version) and
only downloads from `registry.npmjs.org` tarballs when the optional package did not install (`--no-optional`,
`--force`) (<https://unpkg.com/esbuild@0.28.2/install.js>,
<https://github.com/evanw/esbuild/blob/main/lib/npm/node-install.ts>). Pure first-run resolution (no
postinstall at all): ripgrep, and goreleaser's generated launcher (`install()` is a no-op when the binary
exists, so it also survives skipped postinstalls; <https://unpkg.com/@go-task/cli@3.54.0/lib.js>).

**Module system and engines.** CJS is the default (esbuild, lefthook, pulumi, buf); deno names its files
`.cjs` explicitly; ripgrep is ESM using `createRequire`; goreleaser generates `"type": "module"`. `engines` is
advisory (npm warns only): esbuild `>=18`, biome `>=14.21.3`
(<https://docs.npmjs.com/cli/v11/configuring-npm/package-json/>).

**Minimal launcher distilled** (ripgrep resolution + deno exec + electron signals):

```js
#!/usr/bin/env node
const { spawn } = require('child_process');
const exe = process.platform === 'win32' ? 'wtr.exe' : 'wtr';
const pkg = `@what-to-run/cli-${process.platform}-${process.arch}`;
const bin = require.resolve(`${pkg}/bin/${exe}`);   // throws with pkg name in message
const child = spawn(bin, process.argv.slice(2), { stdio: 'inherit', windowsHide: false });
for (const sig of ['SIGINT', 'SIGTERM', 'SIGUSR2'])
  process.on(sig, () => child.kill(sig));
child.on('close', (code) => process.exit(code ?? 1));
```

## 2. What a TUI binary needs from the launcher

- `stdio: 'inherit'` is mandatory: the child gets the real stdin/stdout pty, so raw-mode key handling and the
  alternate screen work; never pipe or buffer input.
- Synchronous spawn (`spawnSync`/`execFileSync`) or a promise that never resolves early: the child must own
  the terminal for the whole session.
- Forward argv verbatim, the child's exit code, and SIGINT/SIGTERM (explicit `child.kill` is safer than
  relying on the shared process group, e.g. under supervisors and `timeout`).
- Skip Node on the hot path if possible: deno's postinstall rewrites `.bin` links to the native binary so an
  interactive CLI does not pay Node startup on every invocation
  (<https://unpkg.com/deno@2.9.6/install_api.cjs>) — relevant for a shell widget run per command.
- Keep the launcher dependency-free (pattern 1 needs no tar/unzip/proxy libs; only the pattern-2 downloader
  needs them).

## 3. Release and publish pipelines

| project | trigger | publish mechanism | versioning | provenance | source |
|---|---|---|---|---|---|
| esbuild | push to `main` touching `version.txt` | `publish.yml` with `id-token: write`; `make platform-all` builds all binaries and rewrites every `npm/*/package.json` version; `publish-all` runs `cd npm/@esbuild/<p> && npm publish` platform-first; npm pinned to 11.5.1 ("Trusted publishing requires this specific version of npm"); tag + GitHub release created only after publish | all packages share the version from `version.txt`; `npm/` manifests are the committed source of truth (`rm -fr npm && git checkout npm` before publish) | npm trusted publishing (OIDC); registry manifests carry SLSA attestations | <https://github.com/evanw/esbuild/blob/main/.github/workflows/publish.yml>, <https://github.com/evanw/esbuild/blob/main/Makefile> |
| @biomejs/biome | push to `main` → changesets opens "ci: release" PR | merging the version PR runs `release.yml` (jobs gated on `hasChangesets == 'false'`): builds binaries, attests them, then `npm publish <pkg> --tag latest --access public` for each `packages/@biomejs/*` | changesets, lockstep (e.g. 2.6.4 in every optionalDependency) | `publishConfig: {provenance: true}` in package.json; `id-token: write`; GitHub artifact attestations on binaries | <https://github.com/biomejs/biome/blob/main/.github/workflows/release.yml>, <https://github.com/biomejs/biome/blob/main/packages/@biomejs/biome/package.json> |
| turbo | `workflow_dispatch` with a SemVer increment (canaries hourly) | staging-branch lock → smoke tests → build Rust → sign/notarize macOS → publish JS packages → create git tag only after npm publish succeeds | custom scripts; no `.changeset/` dir in repo | `id-token: write # Required for npm Trusted Publishing using OIDC` and `NPM_CONFIG_PROVENANCE: "true"` | <https://github.com/vercel/turborepo/blob/main/.github/workflows/turborepo-release.yml> |
| @bufbuild/buf | maintainer dispatches `create-release-pr.yaml` with the version; PR merged → tag → GitHub release published | `publish-npm.yaml` (comment: "filename is hardcoded in our trusted publishing setup on npmjs.com") generates packages, smoke-tests on 6 os/arch, then `publish.bash`: platform `.tgz` first, skips already-published (`npm view`), `npm publish --access public`; `environment: npm-release` + `id-token: write` | release PR pins one version for all generated packages | npm trusted publishing via workflow-filename-bound config | <https://github.com/bufbuild/buf/blob/main/.github/workflows/publish-npm.yaml>, <https://github.com/bufbuild/buf/blob/main/packaging/npm/scripts/publish.bash> |
| deno | GitHub release published (or dispatch, dry-run default) | `npm_publish.generated.yml`: `tools/release/npm/build.ts` builds the dist; test job publishes to a local Verdaccio and verifies npm/pnpm installs — local and global, `--ignore-scripts`, Windows cmd/PowerShell, and that `.bin` links point at the native binary; publish job (`id-token: write`) runs `build.ts --publish-only` | version = release tag | trusted publishing; platform pkgs `@deno/*` lockstep with the wrapper | <https://github.com/denoland/deno/blob/main/.github/workflows/npm_publish.generated.yml> |
| @go-task/cli | `git tag v*` push | `release.yml`: `id-token: write` (OIDC), `npm install -g npm@latest`, goreleaser-pro `release --clean`; the `npms:` block in `.goreleaser.yml` (name `@go-task/cli`, `access: public`) generates one package: `type: module`, `bin: run-task.js`, `postinstall: node install.js`, embedded `archives` map of GitHub Releases URLs + sha256 | version = git tag (GoReleaser) | npm trusted publishing inside goreleaser; no NPM_TOKEN secret in env | <https://github.com/go-task/task/blob/main/.github/workflows/release.yml>, <https://github.com/go-task/task/blob/main/.goreleaser.yml> |
| @vscode/ripgrep | none — no CI in repo ("the main branch of vscode-ripgrep does not contain the path .github") | manual: `prepare-binaries.js` downloads rg from microsoft/ripgrep-prebuilt, verifies against `binaries.lock.json` sha256, `sync-packages.js` generates the per-platform packages; `npm publish` is run by hand | all platform packages share the wrapper version | none observed (manual publish) | <https://github.com/microsoft/vscode-ripgrep> |

Common practices: platform packages publish before the wrapper (buf's publish script comments this
explicitly); every publish is idempotent/re-runnable (buf skips published versions, esbuild fails fast if the
tag exists); scoped packages need `access public` (`npm publish --access public` or `npms: access: public`);
esbuild's platform packages set `preferUnplugged: true` so pnpm does not stash them
(<https://github.com/evanw/esbuild/blob/main/npm/@esbuild/linux-x64/package.json>); biome's wrapper lists
`files: ["bin/biome", ...]` explicitly.

npm security posture for CI publishing: trusted publishing lets CI "publish npm packages directly from your
CI/CD workflows using OpenID Connect (OIDC) authentication, eliminating the need for long-lived npm tokens";
requires npm ≥ 11.5.1 and Node ≥ 22.14.0; npm binds the publish to the specific workflow you authorized
(<https://docs.npmjs.com/trusted-publishers>). Provenance statements "publicly establish where a package was
built and who published" (<https://docs.npmjs.com/generating-provenance-statements>) — biome hardcodes
`publishConfig: {provenance: true}`, turbo exports `NPM_CONFIG_PROVENANCE=true`. Long-lived tokens are dead:
"npm classic tokens permanently revoked — they can no longer authenticate, be recreated, or be recovered";
`npm login` now yields two-hour session tokens
(<https://github.blog/changelog/2025-12-09-npm-classic-tokens-revoked-session-based-auth-and-cli-token-management-now-available/>);
new write tokens enforce 2FA by default with a Bypass-2FA opt-in for CI/CD, and npm recommends OIDC trusted
publishing for CI
(<https://github.blog/changelog/2025-11-05-npm-security-update-classic-token-creation-disabled-and-granular-token-changes/>).

## 4. What wtr would need to adopt each pattern

Pattern 1 — platform packages (esbuild/biome/buf/deno style):

- N+1 packages published in lockstep at the tag version: `@what-to-run/cli-linux-x64`, `-darwin-arm64`, ...
  plus the wrapper `@what-to-run/cli`. Platform packages carry `os`, `cpu`, `preferUnplugged: true`; the
  wrapper carries `optionalDependencies` (exact same version), `bin`, `engines`, `files`, and the ~15-line
  launcher above.
- A publish workflow with `id-token: write` and npm ≥ 11.5.1 (trusted publisher configured once per package on
  npmjs.com, optionally pinned to the workflow filename like buf), platform-first publish order, and the
  goreleaser-built binaries dropped into each platform package's `bin/`.
- Smoke tests before publish, ideally deno-style: publish to Verdaccio, `npm i -g`/`pnpm i` (with and without
  `--ignore-scripts`), run the binary, check the version
  (<https://github.com/denoland/deno/blob/main/.github/workflows/npm_publish.generated.yml>).
- Cost: N+1 trusted-publisher registrations and a small generation step, but zero install scripts, no
  script-blocking caveats, and the launcher works even when `postinstall` never runs.

Pattern 2 — GoReleaser `npms:` (@go-task/cli style):

- Add the `npms:` block (name, `access: public`, description, homepage, license, author, keywords) to
  `.goreleaser.yaml`; goreleaser generates one package: `"type": "module"`, `bin: run-task.js`, `postinstall:
  node install.js`, and an `archives` map of your existing GitHub Releases tarball URLs with sha256 checksums
  (<https://goreleaser.com/customization/npm/>, verified in the produced
  <https://unpkg.com/@go-task/cli@3.54.0/package.json>).
- The generated launcher downloads/extracts on postinstall and again lazily at first run if postinstall was
  blocked, then `spawnSync(..., {stdio: 'inherit'})`s the binary
  (<https://unpkg.com/@go-task/cli@3.54.0/lib.js>) — so pnpm ≥ 10 / Bun users still work after approving
  builds or on first run.
- Release-side: `id-token: write` + `setup-node` with `registry-url` + recent npm is all the workflow needs;
  no NPM_TOKEN secret (go-task's workflow is the template).
- Caveats: needs Node at install and at first run; postinstall approval prompts on pnpm/Bun; one package per
  version (no per-platform os/cpu filtering, so the wrapper itself must not declare `os`/`cpu`).

## Unverified

- The `bin/biome` and `bin/turbo` shim sources (generated at build, absent from the repos; 404s, and the
  biome repo exceeds jsdelivr's file API limit) and turbo's exact version-bump scripts (custom, not
  changesets — the repo has no `.changeset/` directory; the workflow exceeds the fetch cap).
- The exact manual publish commands for @vscode/ripgrep (no CI exists), and esbuild's install.js behavior
  under `--force` specifically (the `--no-optional` path is verified via buf's message and esbuild's
  `installUsingNPM`; I did not read the `--force` branch itself).
