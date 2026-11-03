# Installation

Pick a package manager for your platform, then run `wtr --version` to check the
installation.

| Method | Platform | Install |
| --- | --- | --- |
| [Homebrew](#homebrew) | macOS, Linux | `brew install zanza00/tap/what-to-run` |
| [mise](#mise) | macOS, Linux | `mise use -g github:zanza00/what-to-run` |
| [npm](#npm) | any OS with Node.js 22+ | `npm install -g what-to-run` |
| [Nix](#nix) | Linux, Apple silicon macOS | `nix profile install github:zanza00/what-to-run` |
| [apt](#apt) | Debian, Ubuntu | see [apt](#apt) |
| [dnf](#dnf) | Fedora, RHEL and derivatives | see [dnf](#dnf) |
| [Scoop](#scoop) | Windows | see [Scoop](#scoop) |

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

npm puts `wtr` in the global bin directory. Run `npm prefix -g` to see where.
npm is the shortest path on Windows, because it needs no other package manager.

To run `wtr` without installing it:

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
  inputs.what-to-run.packages.${pkgs.stdenv.hostPlatform.system}.default
];
```

The flake builds the default branch. To pin a release, use a tag that contains
the flake: `github:zanza00/what-to-run/<tag>`. The `version` attribute in
`flake.nix` sets the string that `wtr --version` prints, because a Nix build
has no git tag.

## apt

On Debian or Ubuntu, add the project's apt repository and install the package:

```sh
curl -fsSL https://zanza00.github.io/what-to-run/setup.deb.sh | sudo bash
sudo apt install what-to-run
```

The script is `scripts/setup.deb.sh` in this repository, so you can read it
before you pipe it to a shell. It adds the key and the source, then refreshes
the apt indexes. The repository supports `amd64` and `arm64`.

To use a mirror, set `WTR_REPOSITORY`. To change the installed paths, set
`WTR_APT_KEYRING` and `WTR_APT_SOURCES`.

To add the repository by hand:

```sh
curl -fsSL https://zanza00.github.io/what-to-run/key.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/what-to-run.gpg
echo 'deb [signed-by=/usr/share/keyrings/what-to-run.gpg] https://zanza00.github.io/what-to-run stable main' \
  | sudo tee /etc/apt/sources.list.d/what-to-run.list
sudo apt update
sudo apt install what-to-run
```

## dnf

On Fedora, RHEL or a derivative, add the project's RPM repository and install
the package:

```sh
curl -fsSL https://zanza00.github.io/what-to-run/setup.rpm.sh | sudo bash
sudo dnf install what-to-run
```

The script is `scripts/setup.rpm.sh` in this repository, so you can read it
before you pipe it to a shell. It imports the repository key, writes
`/etc/yum.repos.d/what-to-run.repo`, and refreshes the package manager
metadata. The repository supports `x86_64` and `aarch64`.

To use a mirror, set `WTR_REPOSITORY`. To change the installed paths, set
`WTR_RPM_KEY` and `WTR_RPM_REPOSITORY_FILE`.

To add the repository by hand:

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

## Arch Linux

On Arch Linux, add the project's pacman repository and install the package.
The repository key is the same one that signs the apt repository, so import it
and mark it trusted:

```sh
curl -fsSL -o /tmp/what-to-run.asc https://zanza00.github.io/what-to-run/key.asc
sudo pacman-key --add /tmp/what-to-run.asc
sudo pacman-key --lsign-key apt@zanza00.github.io
```

Then add the repository to `/etc/pacman.conf` and install:

```sh
sudo tee -a /etc/pacman.conf >/dev/null <<'EOF'

[what-to-run]
Server = https://zanza00.github.io/what-to-run/arch/$arch
SigLevel = PackageOptional DatabaseRequired
EOF
sudo pacman -Syu what-to-run
```

The repository supports `x86_64` and `aarch64`. It serves one database per
architecture, like the official repositories, so pacman expands `$arch`.

For a one-time install without the repository, download the `.pkg.tar.zst`
from the [releases page](https://github.com/zanza00/what-to-run/releases) and
install it:

```sh
sudo pacman -U what-to-run_<version>_linux_amd64.pkg.tar.zst
```

Download the `amd64` file on an `x86_64` machine and the `arm64` file on an
`aarch64` machine. Updates stay manual: repeat the download for each new
release.

## Scoop

On Windows, [Scoop](https://scoop.sh/) installs per user without administrator
rights and supports ARM64:

```powershell
scoop bucket add zanza00 https://github.com/zanza00/scoop-bucket
scoop install zanza00/what-to-run
```

## Release files

Every [release](https://github.com/zanza00/what-to-run/releases) attaches one
package or archive per platform. File names follow
`what-to-run_<version>_<os>_<arch>.<ext>`, for example
`what-to-run_0.9.0_linux_amd64.tar.gz`. The architecture names are the Go
names `amd64` and `arm64`. On Linux, `uname -m` prints `x86_64` or `aarch64`
instead, so download the `amd64` file on an Intel machine and the `arm64`
file on an ARM machine.

## Next steps

To bind the picker to a key in zsh, bash or PowerShell, see
[Widget setup](widget.md).
