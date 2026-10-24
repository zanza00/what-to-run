#!/usr/bin/env bash
# Check that the RPM repository index on gh-pages carries a release.
#
# Usage: check-rpm-repository.sh [--arch ARCH] vX.Y.Z
#
# Without --arch both x86_64 and aarch64 must carry the version. The index is
# read through the GitHub API, so GH_TOKEN must be set.
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

metadata=$(gh api "repos/$repo/contents/rpm/repodata/primary.xml.gz?ref=gh-pages" \
  -H "Accept: application/vnd.github.raw" | gzip -dc)

has_version() {
  awk -v arch="$1" -v version="$version" '
    /<package / { block = "" }
    { block = block $0 "\n" }
    /<\/package>/ {
      if (block ~ "<name>what-to-run</name>" &&
          block ~ "<arch>" arch "</arch>" &&
          block ~ "ver=\"" version "\"") {
        found = 1
      }
    }
    END { exit !found }
  ' <<<"$metadata"
}

if [[ -n $arch ]]; then
  arches=("$arch")
else
  arches=(x86_64 aarch64)
fi

failed=0
for candidate in "${arches[@]}"; do
  if has_version "$candidate"; then
    echo "RPM metadata carries $version for $candidate"
  else
    echo "RPM metadata does not carry $version for $candidate" >&2
    failed=1
  fi
done
exit "$failed"
