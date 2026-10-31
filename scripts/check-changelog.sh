#!/usr/bin/env bash
# Check that a diff which changes user-visible code also updates the
# [Unreleased] section of CHANGELOG.md.
#
# Usage: check-changelog.sh BASE [HEAD]
#
# BASE is the revision the change is based on, for example origin/main. HEAD
# defaults to the checked-out commit. The check passes when the diff does not
# touch internal/, cmd/, npm/ or scripts/, or when the [Unreleased] section of
# CHANGELOG.md changed between BASE and HEAD.
set -euo pipefail

usage() {
  echo "usage: $(basename "$0") BASE [HEAD]" >&2
  exit 2
}

[[ $# -le 2 ]] || usage
[[ $# -ge 1 ]] || usage
base=$(git rev-parse --verify --quiet "${1}^{commit}") || usage
head=$(git rev-parse --verify --quiet "${2:-HEAD}^{commit}") || usage

unreleased_section() {
  # Print the [Unreleased] section of CHANGELOG.md at the given revision. An
  # empty string means the file or the section is missing.
  git show "$1:CHANGELOG.md" 2>/dev/null | awk '
    /^## \[Unreleased\]$/ { in_section = 1 }
    in_section && /^## \[/ && !/^## \[Unreleased\]$/ { exit }
    in_section { print }
  ' || true
}

needs_entry=0
while IFS= read -r file; do
  case $file in
    internal/*|cmd/*|npm/*|scripts/*) needs_entry=1 ;;
  esac
done <<<"$(git diff --name-only "$base...$head")"

if [[ $needs_entry -eq 0 ]]; then
  echo "No user-visible code changed; a changelog entry is not required."
  exit 0
fi

base_section=$(unreleased_section "$base")
head_section=$(unreleased_section "$head")

if [[ -z $head_section ]]; then
  echo "CHANGELOG.md has no ## [Unreleased] section." >&2
  echo "Add the entry for this change under ## [Unreleased]." >&2
  exit 1
fi

if [[ "$head_section" == "$base_section" ]]; then
  echo "This diff changes user-visible code (internal/, cmd/, npm/ or" >&2
  echo "scripts/), but the ## [Unreleased] section of CHANGELOG.md did not" >&2
  echo "change." >&2
  echo "Add the entry under ## [Unreleased], or apply the skip-changelog" >&2
  echo "label when the change is not user-visible." >&2
  exit 1
fi

echo "The ## [Unreleased] section of CHANGELOG.md changed."
