#!/usr/bin/env bash
# Check that the pacman databases on gh-pages carry a release.
#
# Usage: check-pacman-repository.sh [--arch ARCH] vX.Y.Z
#
# Without --arch both x86_64 and aarch64 must carry the version. The databases
# are read through the GitHub API, so GH_TOKEN must be set. The repositories
# are fetched with curl instead of the gh CLI, because the release workflow
# runs this script inside an Arch Linux container without gh.
set -euo pipefail

usage() {
  echo "usage: $(basename "$0") [--arch ARCH] vX.Y.Z" >&2
  exit 2
}

arch=""
if [[ ${1:-} == --arch ]]; then
  [[ $# -ge 3 ]] || usage
  arch=$2
  shift 2
fi
tag=${1:-}
if [[ ! $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  usage
fi
version=${tag#v}
repo=${GITHUB_REPOSITORY:-zanza00/what-to-run}
token=${GH_TOKEN:?GH_TOKEN must be set}

# The database is a gzip tar with one directory per package, named
# <name>-<version>-<pkgrel>. The release packages always carry pkgrel 1.
fetch_database() {
  curl -fsSL -H "Authorization: token $token" \
    -H "Accept: application/vnd.github.raw" \
    "https://api.github.com/repos/$repo/contents/arch/$1/what-to-run.db?ref=gh-pages"
}

has_version() {
  fetch_database "$1" | tar -tzf - | grep -qxF "what-to-run-$version-1/desc"
}

if [[ -n $arch ]]; then
  arches=("$arch")
else
  arches=(x86_64 aarch64)
fi

failed=0
for candidate in "${arches[@]}"; do
  if has_version "$candidate"; then
    echo "pacman database carries $version for $candidate"
  else
    echo "pacman database does not carry $version for $candidate" >&2
    failed=1
  fi
done
exit "$failed"
