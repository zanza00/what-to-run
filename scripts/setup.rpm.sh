#!/bin/sh
# Configure the what-to-run RPM repository on Fedora, RHEL and derivatives.
#
#   curl -fsSL https://zanza00.github.io/what-to-run/setup.rpm.sh | sudo bash
#
# The script imports the repository key, writes the dnf/yum repository, and
# refreshes the metadata. Install the package with: sudo dnf install what-to-run
set -eu

REPOSITORY="${WTR_REPOSITORY:-https://zanza00.github.io/what-to-run}"
REPOSITORY="${REPOSITORY%/}"
KEY="${WTR_RPM_KEY:-/etc/pki/rpm-gpg/RPM-GPG-KEY-what-to-run}"
REPOSITORY_FILE="${WTR_RPM_REPOSITORY_FILE:-/etc/yum.repos.d/what-to-run.repo}"

if [ "$(id -u)" -ne 0 ]; then
    echo "error: run this script as root, for example:" >&2
    echo "  curl -fsSL $REPOSITORY/setup.rpm.sh | sudo bash" >&2
    exit 1
fi
for tool in curl rpm; do
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
install -Dm644 "$temporary" "$KEY"
rpm --import "$KEY"

cat > "$REPOSITORY_FILE" <<EOF
[what-to-run]
name=what-to-run
baseurl=$REPOSITORY/rpm/
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=file://$KEY
EOF

if command -v dnf >/dev/null 2>&1; then
    dnf -q makecache
elif command -v yum >/dev/null 2>&1; then
    yum -q makecache
else
    echo "warning: neither dnf nor yum was found; the repository is configured but its metadata was not refreshed" >&2
fi

printf '\nwhat-to-run repository configured. Install it with:\n  sudo dnf install what-to-run\n'
