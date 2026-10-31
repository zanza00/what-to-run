#!/usr/bin/env bash
# Check that a diff which changes user-visible files also adds an entry to the
# [Unreleased] section of CHANGELOG.md.
#
# Usage: check-changelog.sh BASE [HEAD]
#
# BASE is the revision the change is based on, for example origin/main. HEAD
# defaults to the checked-out commit and exists for local checks. The check
# passes when the diff does not touch a watched path, or when the [Unreleased]
# section of CHANGELOG.md gains at least one entry between the merge base and
# HEAD.
set -euo pipefail

# A change to one of these paths ships to users: the CLI and its domain, the
# npm channel, the setup and publish scripts, the release packaging and the
# dependencies.
watched_paths=(
  internal
  cmd
  npm
  scripts
  .goreleaser.yaml
  .github/workflows/release.yml
  go.mod
  go.sum
)

usage() {
  echo "usage: $(basename "$0") BASE [HEAD]" >&2
  exit 2
}

die() {
  echo "$(basename "$0"): $*" >&2
  exit 2
}

[[ $# -le 2 ]] || usage
[[ $# -ge 1 ]] || usage
base=$(git rev-parse --verify --quiet "${1}^{commit}") || die "unknown revision: $1"
head=$(git rev-parse --verify --quiet "${2:-HEAD}^{commit}") || die "unknown revision: ${2:-HEAD}"

merge_base=$(git merge-base "$base" "$head") ||
  die "no merge base for $base and $head"

unreleased_section() {
  # Print the [Unreleased] section of CHANGELOG.md at the given revision. An
  # empty string means the file or the section is missing.
  git show "$1:CHANGELOG.md" 2>/dev/null | awk '
    /^## \[Unreleased\]$/ { in_section = 1 }
    in_section && /^## \[/ && !/^## \[Unreleased\]$/ { exit }
    in_section { print }
  ' || true
}

unreleased_entry_count() {
  # Count the top-level entries in the [Unreleased] section.
  unreleased_section "$1" | awk '/^- / { count++ } END { print count + 0 }'
}

is_watched() {
  local prefix
  for prefix in "${watched_paths[@]}"; do
    if [[ $1 == "$prefix" || $1 == "$prefix"/* ]]; then
      return 0
    fi
  done
  return 1
}

changed_files=$(git diff --name-only "$merge_base" "$head") ||
  die "cannot diff $merge_base..$head"

needs_entry=0
while IFS= read -r file; do
  # The checker itself is not user-facing, so a change to it alone does not
  # need an entry.
  if [[ $file == scripts/check-changelog.sh ]]; then
    continue
  fi
  if is_watched "$file"; then
    needs_entry=1
  fi
done <<<"$changed_files"

if [[ $needs_entry -eq 0 ]]; then
  echo "No watched path changed; a changelog entry is not required."
  exit 0
fi

head_section=$(unreleased_section "$head")

if [[ -z $head_section ]]; then
  echo "CHANGELOG.md has no ## [Unreleased] section." >&2
  echo "Add the entry for this change under ## [Unreleased]." >&2
  exit 1
fi

base_count=$(unreleased_entry_count "$merge_base")
head_count=$(unreleased_entry_count "$head")

if (( head_count <= base_count )); then
  echo "This diff changes a watched path, but the ## [Unreleased] section of" >&2
  echo "CHANGELOG.md gained no new entry." >&2
  echo "Add at least one entry under ## [Unreleased], or apply the" >&2
  echo "skip-changelog label when the change is not user-visible." >&2
  exit 1
fi

echo "The ## [Unreleased] section of CHANGELOG.md gained an entry."
