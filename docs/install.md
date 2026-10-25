# Installation

The command is `wtr`. Run `wtr --version` to verify an installation.

## Homebrew

On macOS or Linux with [Homebrew](https://brew.sh/):

```sh
brew install zanza00/tap/what-to-run
```

## mise

On macOS or Linux with [mise](https://mise.jdx.dev/):

```sh
mise use -g github:zanza00/what-to-run
```

This installs `wtr` from the GitHub releases.

## Debian and Ubuntu

### apt repository

Configure the project's apt repository, signed with the project's GPG key,
with one command:

```sh
curl -fsSL https://zanza00.github.io/what-to-run/setup.deb.sh | sudo bash
sudo apt install what-to-run
```

The script is `scripts/setup.deb.sh` in this repository, so you can read it
before piping it to a shell. It only adds the key and the source, then
refreshes the apt indexes. The repository supports both `amd64` and `arm64`;
set `WTR_REPOSITORY` to use a mirror, or `WTR_APT_KEYRING` and
`WTR_APT_SOURCES` to change the installed paths.

The same steps by hand:

```sh
curl -fsSL https://zanza00.github.io/what-to-run/key.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/what-to-run.gpg
echo 'deb [signed-by=/usr/share/keyrings/what-to-run.gpg] https://zanza00.github.io/what-to-run stable main' \
  | sudo tee /etc/apt/sources.list.d/what-to-run.list
sudo apt update
sudo apt install what-to-run
```

### .deb package

The `.deb` packages for each version are attached to the
[releases](https://github.com/zanza00/what-to-run/releases). Download the file
for your architecture (`amd64`, or `arm64` on ARM machines) and install it
directly:

```sh
sudo apt install ./what-to-run_<version>_linux_amd64.deb
```

## Fedora, RHEL and derivatives

### dnf/yum repository

Configure the project's RPM repository with one command:

```sh
curl -fsSL https://zanza00.github.io/what-to-run/setup.rpm.sh | sudo bash
sudo dnf install what-to-run
```

The script is `scripts/setup.rpm.sh` in this repository, so you can read it
before piping it to a shell. It imports the repository key, writes
`/etc/yum.repos.d/what-to-run.repo`, and refreshes the dnf metadata. The
repository supports both `x86_64` and `aarch64`; set `WTR_REPOSITORY` to use a
mirror, or `WTR_RPM_KEY` and `WTR_RPM_REPOSITORY_FILE` to change the installed
paths.

The same steps by hand:

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

The `.rpm` packages are attached to the
[releases](https://github.com/zanza00/what-to-run/releases) starting with the
release after v0.7.0. Download the file for your architecture (`amd64`, or
`arm64` on ARM machines) and install it directly:

```sh
sudo dnf install ./what-to-run_<version>_linux_amd64.rpm
```

The file names use the Go architecture names: `amd64` is `x86_64` and `arm64`
is `aarch64`.

The packages are signed with the project's GPG key, the same one that signs the
apt repository. To verify a download, import the key and check the signature:

```sh
sudo rpm --import https://zanza00.github.io/what-to-run/key.asc
rpm --checksig what-to-run_<version>_linux_amd64.rpm
```

## Arch Linux

An AUR package (`what-to-run-bin`) is pending. Until it is published, use a
[release tarball](#prebuilt-tarballs).

## Prebuilt tarballs

> [!WARNING]
> Use this only as a last resort: the CLI has no self-updater, so a tarball
> installed by hand stays at that version until you replace it. Install through
> one of the methods above to receive upgrades from your package manager.

Each release attaches tarballs for macOS and Linux on Intel and ARM64, as an
alternative to the packages above. Extract one and put the binary on your
`PATH`:

```sh
tar -xzf what-to-run_<version>_linux_amd64.tar.gz
sudo install -m 755 wtr /usr/local/bin/
```

Use `what-to-run_<version>_darwin_arm64.tar.gz` on Apple silicon, and so on.

## Windows

### Scoop

With [Scoop](https://scoop.sh/), which installs per user without administrator
rights and has native ARM64 support:

```powershell
scoop bucket add zanza00 https://github.com/zanza00/scoop-bucket
scoop install zanza00/what-to-run
scoop update what-to-run
scoop uninstall what-to-run
```

The manifest installs `wtr` on Intel and ARM64 Windows alike. The bucket
receives the manifest of every release; if it has no manifest for the version
you need yet, use the zip below.

### Zip

Each release attaches a zip for Windows on Intel and ARM64:

- `what-to-run_<version>_windows_amd64.zip`
- `what-to-run_<version>_windows_arm64.zip`

Extract it, put the folder on your `PATH`, and verify the install from
PowerShell:

```powershell
Expand-Archive what-to-run_<version>_windows_amd64.zip -DestinationPath C:\Tools\what-to-run
$env:Path = "C:\Tools\what-to-run;$env:Path"   # or add it permanently
wtr --version
```

The archive contains `wtr.exe`.

The binary is not code-signed, so SmartScreen may block the first run.
`Unblock-File` clears the mark of the web on the extracted files:

```powershell
Get-ChildItem C:\Tools\what-to-run | Unblock-File
```

## Go

With Go installed, `go install` compiles the program locally:

```sh
go install github.com/zanza00/what-to-run/cmd/wtr@latest
```

This requires Go 1.27.2 or later.

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

Upgrading to a release that ships this change removes the second binary; the
configuration in `~/.config/what-to-run/` and the `.what-to-run.toml` project
file keep their names.

## Next steps

To bind the picker to a key in zsh, bash or PowerShell, see
[Widget setup](widget.md).
