# Installation

Releases ship two binaries with the same picker: `what-to-run` (the explicit
name) and `wtr` (the short alias). The methods below install both unless noted
otherwise. Run `wtr --version` or `what-to-run --version` to verify the
installation.

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

This installs both binaries from the GitHub releases.

## Debian and Ubuntu (apt repository)

Add the project's apt repository, signed with the project's GPG key, and
install from it:

```sh
curl -fsSL https://zanza00.github.io/what-to-run/key.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/what-to-run.gpg
echo 'deb [signed-by=/usr/share/keyrings/what-to-run.gpg] https://zanza00.github.io/what-to-run stable main' \
  | sudo tee /etc/apt/sources.list.d/what-to-run.list
sudo apt update
sudo apt install what-to-run
```

The repository supports both `amd64` and `arm64`.

## Debian and Ubuntu (.deb package)

The `.deb` packages for each version are attached to the
[releases](https://github.com/zanza00/what-to-run/releases). Download the file
for your architecture (`amd64`, or `arm64` on ARM machines) and install it
directly:

```sh
sudo apt install ./what-to-run_<version>_linux_amd64.deb
```

## Fedora, RHEL and derivatives (.rpm package)

The `.rpm` packages for each version are attached to the
[releases](https://github.com/zanza00/what-to-run/releases). Download the file
for your architecture (`amd64`, or `arm64` on ARM machines) and install it
directly:

```sh
sudo dnf install ./what-to-run_<version>_linux_amd64.rpm
```

## Arch Linux

An AUR package (`what-to-run-bin`) is pending. Until it is published, use a
release tarball.

## Prebuilt tarballs

Each release attaches tarballs for macOS and Linux on Intel and ARM64, as an
alternative to the packages above. Every archive contains both binaries.
Extract it and put them on your `PATH`:

```sh
tar -xzf what-to-run_<version>_linux_amd64.tar.gz
sudo install -m 755 wtr what-to-run /usr/local/bin/
```

Use `what-to-run_<version>_darwin_arm64.tar.gz` on Apple silicon, and so on.

## Windows

Each release attaches a zip for Windows on Intel and ARM64:

- `what-to-run_<version>_windows_amd64.zip`
- `what-to-run_<version>_windows_arm64.zip`

Extract it, put the folder on your `PATH`, and verify the install from
PowerShell:

```powershell
Expand-Archive what-to-run_<version>_windows_amd64.zip -DestinationPath C:\Tools\what-to-run
$env:Path = "C:\Tools\what-to-run;$env:Path"   # or add it permanently
wtr --version
what-to-run --version
```

The archive contains both `wtr.exe` and `what-to-run.exe`: the invoked name is
the one shown in usage, messages and the picker title.

The binaries are not code-signed, so SmartScreen may block the first run.
`Unblock-File` clears the mark of the web on the extracted files:

```powershell
Get-ChildItem C:\Tools\what-to-run | Unblock-File
```

## Go

With Go installed, `go install` compiles the program locally (it installs only
`wtr`):

```sh
go install github.com/zanza00/what-to-run/cmd/wtr@latest
```

This requires Go 1.27.2 or later.

## Next steps

To bind the picker to a key in zsh, bash or PowerShell, see
[Widget setup](widget.md).
