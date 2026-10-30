# npm as an install channel for compiled binaries

Research note, 2026-10-30, verified against primary sources (npm registry, official docs, published package
files). Weekly downloads come from npm's API and include transitive installs.

## Summary

- npm is a mainstream channel for non-JavaScript CLIs. Go binaries shipping there today: esbuild (~399M weekly
  downloads), turbo (~42M), lefthook (~8.7M), @bufbuild/buf (~3.5M), pulumi (~132k), @go-task/cli (~53k). Also
  Rust (@biomejs/biome ~25M, oxlint ~40M, deno, @sentry/cli, lightningcss, @swc/core ~59M), Zig (bun ~6.2M), and
  C++/Rust (workerd).
- Two patterns dominate: (1) one npm package per platform as `optionalDependencies` plus a tiny JS launcher in
  `bin` (esbuild style) — no install script, survives script-blocking package managers; (2) a `postinstall`
  script that downloads the binary from GitHub Releases or a CDN (puppeteer style) — now blocked by default in
  pnpm >= 10 and Bun, and being abandoned even by its biggest users (electron 42 dropped its postinstall
  download).
- Relevant here: GoReleaser has a built-in npm publisher that generates pattern (2)
  (`https://goreleaser.com/customization/npm/`); @go-task/cli uses it.

## Distribution patterns

### 1. Platform packages + JS launcher (esbuild style)

- The wrapper package has `"bin"` pointing at a JS shim and lists every platform binary as an
  `optionalDependencies` entry at the same version (esbuild 0.28.2: `@esbuild/linux-x64`,
  `@esbuild/darwin-arm64`, ... — <https://registry.npmjs.org/esbuild/latest>). biome, turbo, buf, deno, lefthook
  and workerd use the same layout (<https://registry.npmjs.org/@biomejs/biome/latest>,
  <https://registry.npmjs.org/turbo/latest>, <https://registry.npmjs.org/workerd/latest>).
- Each platform package declares `"os"` and `"cpu"` — `{"os":["linux"],"cpu":["x64"]}` for @esbuild/linux-x64 —
  so exactly the matching one installs (<https://registry.npmjs.org/@esbuild/linux-x64/latest>). npm skips
  optional dependencies whose os/cpu/libc do not match the host: "Optional dependencies that cannot be installed
  on the current platform or engine (a non-matching `os`, `cpu`, or `libc`) are not flagged"
  (<https://docs.npmjs.com/cli/v11/commands/npm-install/>). The fields support allow lists and `"!blocked"`
  entries; host = `process.platform`/`process.arch`; `engines` is advisory unless `engine-strict`
  (<https://docs.npmjs.com/cli/v11/configuring-npm/package-json/>).
- Launcher mechanics (from the published packages): map `process.platform`/`process.arch` to the platform
  package name, `require.resolve()` its binary, then `child_process.spawn` forwarding argv, exit code and
  signals (verified in lefthook <https://unpkg.com/lefthook@2.2.2/get-exe.js>, deno
  <https://unpkg.com/deno@2.9.6/install_api.cjs>, buf <https://unpkg.com/@bufbuild/buf@1.75.0/install.js>,
  pulumi <https://unpkg.com/pulumi@3.273.0/run.js>; esbuild adds an `ESBUILD_BINARY_PATH` env override,
  <https://github.com/evanw/esbuild/blob/main/lib/npm/node-platform.ts>). esbuild, buf, bun and deno also ship a
  fallback that, at `postinstall` or first run, downloads the platform tarball from the npm registry only if the
  optional package did not install (`--no-optional`, `--force`), with an embedded sha256 check (esbuild
  `downloadDirectlyFromNPM` + `binaryHashes`: <https://unpkg.com/esbuild@0.28.2/install.js>); esbuild's
  postinstall is a fallback, not the primary install path.

### 2. postinstall download (puppeteer style) — legacy, blocked by default

- Still used by puppeteer (`node install.mjs` downloads Chrome, <https://registry.npmjs.org/puppeteer/latest>),
  prisma (`@prisma/engines` postinstall + `@prisma/fetch-engine`,
  <https://registry.npmjs.org/@prisma/engines/latest>), @sentry/cli
  (<https://registry.npmjs.org/@sentry/cli/latest>) and @go-task/cli, whose package embeds an `archives` map of
  GitHub Releases URLs + sha256 and downloads/extracts at postinstall
  (<https://unpkg.com/@go-task/cli@3.54.0/package.json>). Tradeoff: arbitrary code with network access at
  install time.
- Package-manager policies (official docs):
  - npm: lifecycle scripts run by default; `--ignore-scripts` opts out
    (<https://docs.npmjs.com/cli/v11/using-npm/scripts/>). npm 11 adds an `allowScripts` policy that flags
    unreviewed dependency scripts with a notice; `strict-allow-scripts` makes it a hard error
    (<https://docs.npmjs.com/cli/v11/commands/npm-install/>).
  - pnpm >= 10: "Lifecycle scripts of dependencies are not executed during installation by default!" — they must
    be allow-listed in `pnpm.onlyBuiltDependencies` (<https://github.com/pnpm/pnpm/releases/tag/v10.0.0>); v11
    renames the mechanism to `allowBuilds` (<https://pnpm.io/settings/build>).
  - Bun: dependency lifecycle scripts do not run unless the package is in `trustedDependencies`
    (<https://bun.sh/docs/cli/install/>).
- Even electron abandoned the pattern, citing "supply chain security attacks against the npm ecosystem with
  postinstall scripts as a common attack vector": "Electron will now download itself dynamically the first time
  that its main bin script is run" (<https://www.electronjs.org/blog/electron-42-0>); the package now ships no
  scripts at all (<https://unpkg.com/electron@45.1.0/package.json>). @vscode/ripgrep moved the same way:
  per-platform packages and "There is no postinstall step and no runtime network access"
  (<https://github.com/microsoft/vscode-ripgrep>).
- Provenance: most surveyed packages publish with SLSA provenance attestations via GitHub OIDC trusted
  publishing (esbuild's manifest shows `trustedPublisher` + `dist.attestations.provenance`,
  <https://registry.npmjs.org/esbuild/latest>).

### 3. napi-rs platform packages (native modules, not standalone CLIs)

napi-rs generates "a small root package plus one optional package per platform" (`@pkg-darwin-x64`,
`@pkg-linux-x64-gnu`/`-musl`, `@pkg-win32-x64-msvc`) whose `index.js` loads the addon matching os/cpu/libc
(<https://napi.rs/docs/introduction/getting-started>). The binary is a `.node` library loaded by JS, not exec'd,
so it is not the pattern for a standalone CLI (users: @swc/core, lightningcss, @tailwindcss/oxide, rspack,
rolldown, oxlint).

## Examples

| package | binary | role of npm channel | pattern | source |
|---|---|---|---|---|
| esbuild | Go | primary: "The recommended way to install esbuild is to install the native executable using npm" (local devDependency, not -g) | platform pkgs + registry fallback | <https://esbuild.github.io/getting-started/> |
| @biomejs/biome | Rust | primary; npm is the first tab, standalone curl script is the fallback | platform pkgs (@biomejs/cli-*) | <https://biomejs.dev/guides/getting-started/> |
| turbo | Go | primary: the CLI is added as a root `devDependency` (npm/pnpm/yarn/bun) | platform pkgs (@turbo/linux-64 etc.) | <https://turborepo.dev/docs/getting-started/installation> |
| @bufbuild/buf | Go | side channel: docs file npm under "JavaScript toolchain (project-local)"; brew/scoop first | platform pkgs + download fallback | <https://buf.build/docs/installation/> |
| pulumi | Go | undocumented side channel (zero npm mentions on the install page) | JS launcher + first-run download into a cache dir | <https://www.pulumi.com/docs/install/>, <https://unpkg.com/pulumi@3.273.0/lib/resolve.js> |
| deno | Rust | first-class option: `npm install -g deno` listed with brew/curl | platform pkgs (@deno/linux-x64-glibc; musl unsupported) + copy at install | <https://docs.deno.com/runtime/getting_started/installation/> |
| bun | Zig | first-class option: `npm install -g bun` | platform pkgs (@oven/bun-*) + npm-install/registry fallbacks | <https://bun.com/docs/installation> |
| oxlint | Rust | primary; recommended as a devDependency | JS CLI + @oxlint/binding-* (napi-rs names) | <https://oxc.rs/docs/guide/usage/linter> |
| wrangler + workerd | JS + C++ | wrangler is npm-only, recommended local per project; workerd ships as its dependency | JS CLI + platform pkgs | <https://developers.cloudflare.com/workers/wrangler/install-and-update/> |
| lefthook | Go | first-listed install in the README (npm, then Go/Ruby/Python/apt/brew/winget) | platform pkgs + postinstall runs `lefthook install` (git hooks) | <https://unpkg.com/lefthook@2.2.2/README.md>, <https://unpkg.com/lefthook@2.2.2/postinstall.js> |
| @go-task/cli | Go | listed install channel: `npm install -g @go-task/cli` | GoReleaser npm publisher: postinstall download from GitHub Releases | <https://taskfile.dev/docs/installation> |
| @tailwindcss/oxide, @swc/core, lightningcss | Rust | library dependencies, not CLIs | napi-rs platform pkgs | <https://registry.npmjs.org/@tailwindcss/oxide/latest> |
| rspack / rolldown | Rust | bundler libraries/CLIs | napi-rs binding pkgs + JS CLI | <https://registry.npmjs.org/@rspack/binding/latest> |
| electron | C++ | primary | was postinstall download; since v42 lazy download on first bin run, no scripts | <https://www.electronjs.org/blog/electron-42-0> |
| @vscode/ripgrep | Rust | VS Code dependency | was postinstall download; now platform pkgs | <https://github.com/microsoft/vscode-ripgrep> |
| puppeteer, @prisma/engines, @sentry/cli | Chrome / Rust | primary | postinstall downloads | npm registry manifests above |

Weekly downloads (2026-10-30, npm API): esbuild 399M, @tailwindcss/oxide 113M, @swc/core 59M, turbo 42M, oxlint
40M, wrangler 38M, prisma 26M, @biomejs/biome 25M, puppeteer 15M, electron 11M, lefthook 8.7M, bun 6.2M,
@bufbuild/buf 3.5M, deno 376k, pulumi 132k, @go-task/cli 53k.

## Practical notes for a Go binary considering npm

- Naming: scoped `@scope/pkg` is the norm for org projects (@bufbuild/buf, @go-task/cli, @biomejs/biome);
  unscoped when the brand predates npm (esbuild, deno, bun, lightningcss). Platform packages:
  `@scope/pkg-<os>-<arch>` (esbuild, lefthook, @deno/linux-x64-glibc) or `...-gnu`/`...-musl` when splitting
  libc (biome); turbo shortens to @turbo/linux-64.
- Versioning: wrapper and all platform packages publish at the same version as the release tag (esbuild 0.28.2,
  biome 2.6.4, turbo 2.11.10, buf 1.75.0 — registry manifests above).
- Global vs local: esbuild, biome, buf and oxlint docs push local devDependencies run through
  `node_modules/.bin`/npx; deno, bun and Task document `npm i -g`. A shell-widget picker like wtr matches the
  `-g` style.
- Node dependency: the shim and any installer run under Node/npm; a Go binary still needs this thin JS layer
  (bun solves it by copying a real binary into `bin/bun.exe`).
- GoReleaser path: `npms:` in .goreleaser.yaml yields pattern (2) today — one package, postinstall download of
  the existing GitHub Releases archives with checksum verification
  (<https://goreleaser.com/customization/npm/>). Pattern (1) needs custom CI publishing N platform packages plus
  the wrapper, as the esbuild/biome/buf pipelines do. With a postinstall download, expect pnpm/Bun users to see
  blocked-script warnings until they approve the build; publish with trusted publishing + provenance as the
  peers do.

## Unverified

- The internals of the `bin/turbo` and `bin/biome` launcher shims (served as octet-stream; verified from
  manifests, not shim source), and the musl story for turbo and @tailwindcss/oxide on linux (biome/deno/oxlint
  musl variants verified; turbo's list has none).
