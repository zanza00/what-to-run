#!/bin/sh
# Configure the what-to-run pacman repository on Arch Linux.
#
#   curl -fsSL https://zanza00.github.io/what-to-run/setup.arch.sh | sudo bash
#
# The script imports the repository key, adds the [what-to-run] repository to
# the pacman configuration, and refreshes the databases. Install the package
# with: sudo pacman -Syu what-to-run
#
# WTR_REPOSITORY overrides the repository URL; WTR_PACMAN_CONF overrides the
# pacman configuration file.
set -eu

REPOSITORY="${WTR_REPOSITORY:-https://zanza00.github.io/what-to-run}"
REPOSITORY="${REPOSITORY%/}"
PACMAN_CONF="${WTR_PACMAN_CONF:-/etc/pacman.conf}"

if [ "$(id -u)" -ne 0 ]; then
    echo "error: run this script as root, for example:" >&2
    echo "  curl -fsSL $REPOSITORY/setup.arch.sh | sudo bash" >&2
    exit 1
fi
for tool in curl pacman-key pacman; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        echo "error: $tool is required" >&2
        exit 1
    fi
done

temporary="$(mktemp)"
trap 'rm -f "$temporary"' EXIT
curl -fsSL "$REPOSITORY/key.asc" -o "$temporary"
if [ ! -s "$temporary" ]; then
    echo "error: could not download the repository key" >&2
    exit 1
fi
pacman-key --add "$temporary"
pacman-key --lsign-key apt@zanza00.github.io

# The escaped dollar keeps $arch literal, because pacman expands the variable
# itself, like it does for the official repositories.
if grep -q '^\[what-to-run\]' "$PACMAN_CONF"; then
    echo "what-to-run repository already in $PACMAN_CONF; skipped the append"
else
    cat >> "$PACMAN_CONF" <<EOF

[what-to-run]
Server = $REPOSITORY/arch/\$arch
SigLevel = PackageOptional DatabaseRequired
EOF
fi

pacman -Sy

printf '\nwhat-to-run repository configured. Install it with:\n  sudo pacman -Syu what-to-run\n'
