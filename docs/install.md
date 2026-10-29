# Installation

The command is `wtr`. Run `wtr --version` to check an installation. It prints
the command name and the version, for example `wtr 0.9.0`.

## Homebrew

With [Homebrew](https://brew.sh/) on macOS or Linux:

```sh
brew install zanza00/tap/what-to-run
```

## mise

With [mise](https://mise.jdx.dev/) on macOS or Linux:

```sh
mise use -g github:zanza00/what-to-run
```

mise installs `wtr` from the GitHub releases.

## npm

With [Node.js](https://nodejs.org/) 22 or later:

```sh
npm install -g what-to-run
```

Publishing is opt-in for this project: releases reach npm only while the
maintainer has enabled the channel, so a version that is missing from the
registry is not a packaging bug — use one of the other methods below.

The command is `wtr`, like every other method, and npm puts it in the global
bin directory (`npm prefix -g` to see where). This is the shortest path on
Windows, where npm needs no separate package manager.

The package runs no install script: the release archive for your platform is
an optional dependency npm picks by itself, so `--ignore-scripts` is safe and
`pnpm`'s build allowlist cannot block it. With `--omit=optional` the launcher
explains what is missing instead of failing silently.

To run it without installing:

```sh
npx what-to-run@latest
```

## Nix

On Linux or Apple silicon macOS with [Nix](https://nixos.org/) and flakes
enabled:

```sh
nix run github:zanza00/what-to-run
```

To install the command into your profile:

```sh
nix profile install github:zanza00/what-to-run
```

The repository is a flake, so Nix builds `wtr` from source. The first build
downloads the Go toolchain and the modules, then Nix keeps the result in the
store. Nixpkgs no longer supports Intel Macs, so use Homebrew on those machines.

On NixOS, add the flake as an input, pass the inputs to the modules with
`specialArgs = { inherit inputs; };`, and add the package to
`environment.systemPackages`:

```nix
inputs.what-to-run.url = "github:zanza00/what-to-run";

environment.systemPackages = [
  inputs.what-to-run.packages.${pkgs.system}.default
];
```

The flake builds the default branch. To pin a release, add the tag to the URL:
`github:zanza00/what-to-run/v0.9.0`. The `version` attribute in `flake.nix`
sets the string that `wtr --version` prints, because a Nix build has no git tag.

## Debian and Ubuntu

### apt repository

Add the project's apt repository and install the package:

```sh
curl -fsSL https://zanza00.github.io/what-to-run/setup.deb.sh | sudo bash
sudo apt install what-to-run
```

The script is `scripts/setup.deb.sh` in this repository, so you can read it
before you pipe it to a shell. It adds the key and the source, then refreshes
the apt indexes. The repository supports `amd64` and `arm64`. Set
`WTR_REPOSITORY` to use a mirror, or `WTR_APT_KEYRING` and `WTR_APT_SOURCES` to
change the installed paths.

To do the same by hand:

```sh
curl -fsSL https://zanza00.github.io/what-to-run/key.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/what-to-run.gpg
echo 'deb [signed-by=/usr/share/keyrings/what-to-run.gpg] https://zanza00.github.io/what-to-run stable main' \
  | sudo tee /etc/apt/sources.list.d/what-to-run.list
sudo apt update
sudo apt install what-to-run
```

### .deb package

Each release attaches the `.deb` packages. Download the file for your
architecture from the
[releases](https://github.com/zanza00/what-to-run/releases) (`amd64`, or `arm64`
on ARM machines) and install it:

```sh
sudo apt install ./what-to-run_<version>_linux_amd64.deb
```

## Fedora, RHEL and derivatives

### dnf/yum repository

Add the project's RPM repository and install the package:

```sh
curl -fsSL https://zanza00.github.io/what-to-run/setup.rpm.sh | sudo bash
sudo dnf install what-to-run
```

The script is `scripts/setup.rpm.sh` in this repository, so you can read it
before you pipe it to a shell. It imports the repository key, writes
`/etc/yum.repos.d/what-to-run.repo`, and refreshes the package manager metadata.
The repository supports `x86_64` and `aarch64`. Set `WTR_REPOSITORY` to use a
mirror, or `WTR_RPM_KEY` and `WTR_RPM_REPOSITORY_FILE` to change the installed
paths.

To configure the repository by hand:

```sh
sudo rpm --import https://zanza00.github.io/what-to-run/key.asc
sudo tee /etc/yum.repos.d/what-to-run.repo >/dev/null <<'EOF'
[what-to-run]
name=what-to-run
baseurl=https://zanza00.github.io/what-to-run/rpm/
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://zanza00.github.io/what-to-run/key.asc
EOF
sudo dnf install what-to-run
```

### .rpm package

Each release from v0.8.0 attaches the `.rpm` packages. Download the file for
your architecture from the
[releases](https://github.com/zanza00/what-to-run/releases) (`amd64`, or `arm64`
on ARM machines) and install it:

```sh
sudo dnf install ./what-to-run_<version>_linux_amd64.rpm
```

The file names use the Go architecture names. An `x86_64` machine downloads
the `amd64` file, and an `aarch64` machine downloads the `arm64` file.

The project's GPG key signs the `.rpm` packages. It is the same key that signs
the apt repository. To verify a download, import the key and check the
signature:

```sh
sudo rpm --import https://zanza00.github.io/what-to-run/key.asc
rpm --checksig what-to-run_<version>_linux_amd64.rpm
```

## Arch Linux

The AUR package `what-to-run-bin` is not published yet. Until it is, use a
[release tarball](#prebuilt-tarballs).

## Prebuilt tarballs

> [!WARNING]
> Use a tarball only as a last resort. The CLI has no self-updater, so a
> tarball installed by hand stays at that version until you replace it. Install
> through one of the methods above to receive upgrades from your package
> manager.

Each release attaches tarballs for macOS and Linux on Intel and ARM64, as an
alternative to the packages above. Extract the archive and put the binary on
your `PATH`:

```sh
tar -xzf what-to-run_<version>_linux_amd64.tar.gz
sudo install -m 755 wtr /usr/local/bin/
```

On Apple silicon, use `what-to-run_<version>_darwin_arm64.tar.gz`. The other
platforms follow the same pattern.

## Windows

### Scoop

[Scoop](https://scoop.sh/) installs per user without administrator rights and
supports ARM64:

```powershell
scoop bucket add zanza00 https://github.com/zanza00/scoop-bucket
scoop install zanza00/what-to-run
scoop update what-to-run
scoop uninstall what-to-run
```

The manifest installs `wtr` on Intel and ARM64 Windows. The bucket receives the
manifest of every release. If the bucket has no manifest for the version you
need yet, use the zip below.

### Zip

> [!WARNING]
> Use a zip only as a last resort. The CLI has no self-updater, so a zip
> installed by hand stays at that version until you replace it. Install through
> Scoop to receive upgrades.

Each release attaches a zip for Windows on Intel and ARM64:

- `what-to-run_<version>_windows_amd64.zip`
- `what-to-run_<version>_windows_arm64.zip`

The zip contains `wtr.exe` at its root. Extract the zip, put the folder on your
`PATH`, and check the install from PowerShell:

```powershell
Expand-Archive what-to-run_<version>_windows_amd64.zip -DestinationPath C:\Tools\what-to-run
$env:Path = "C:\Tools\what-to-run;$env:Path"   # or add it permanently
wtr --version
```

The binary is not code-signed, so SmartScreen may block the first run.
`Unblock-File` clears the mark of the web on the extracted files:

```powershell
Get-ChildItem C:\Tools\what-to-run | Unblock-File
```

## Go

With Go 1.27.2 or later, `go install` compiles the program locally:

```sh
go install github.com/zanza00/what-to-run/cmd/wtr@latest
```

## Upgrading from a two-name release

Earlier releases shipped a second binary, `what-to-run`, built from the same
program. If an rc file still calls it, replace it with `wtr`:

```sh
eval "$(wtr init zsh)"   # was: eval "$(what-to-run init zsh)"
eval "$(wtr init bash)"  # was: eval "$(what-to-run init bash)"
```

```powershell
wtr init powershell | Out-String | Invoke-Expression
# was: what-to-run init powershell | Out-String | Invoke-Expression
```

Releases from v0.8.0 ship only `wtr`. The configuration in
`~/.config/what-to-run/` (or `%AppData%\what-to-run\` on Windows) and the
`.what-to-run.toml` project file keep their names.

## Next steps

To bind the picker to a key in zsh, bash or PowerShell, see
[Widget setup](widget.md).
