#!/bin/sh
# Configure the what-to-run apt repository on Debian and Ubuntu.
#
#   curl -fsSL https://zanza00.github.io/what-to-run/setup.deb.sh | sudo bash
#
# The script installs the repository key and source, then refreshes the apt
# indexes. Install the package with: sudo apt install what-to-run
#
# WTR_REPOSITORY overrides the repository URL; WTR_APT_KEYRING and
# WTR_APT_SOURCES override the installed paths.
set -eu

REPOSITORY="${WTR_REPOSITORY:-https://zanza00.github.io/what-to-run}"
REPOSITORY="${REPOSITORY%/}"
KEYRING="${WTR_APT_KEYRING:-/usr/share/keyrings/what-to-run.gpg}"
SOURCES="${WTR_APT_SOURCES:-/etc/apt/sources.list.d/what-to-run.list}"

if [ "$(id -u)" -ne 0 ]; then
    echo "error: run this script as root, for example:" >&2
    echo "  curl -fsSL $REPOSITORY/setup.deb.sh | sudo bash" >&2
    exit 1
fi
for tool in curl apt-get; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        echo "error: $tool is required" >&2
        exit 1
    fi
done

temporary="$(mktemp)"
trap 'rm -f "$temporary"' EXIT
curl -fsSL "$REPOSITORY/key.gpg" -o "$temporary"
if [ ! -s "$temporary" ]; then
    echo "error: could not download the repository key" >&2
    exit 1
fi
install -Dm644 "$temporary" "$KEYRING"
printf 'deb [signed-by=%s] %s stable main\n' "$KEYRING" "$REPOSITORY" > "$SOURCES"
apt-get update

printf '\nwhat-to-run repository configured. Install it with:\n  sudo apt install what-to-run\n'
