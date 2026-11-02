#!/usr/bin/env bash
# Prove that the tree at REF installs through the Nix channel on every
# platform docs/install.md ships it for. The workflow builds the flake from
# a checkout and then repeats the install the way a user does it: fetch the
# flake by ref, nix profile install, run the installed binary.
#
# Usage: verify-nix-install.sh REF
#
# REF is a branch or a tag, for example chore/cut-changelog-v0.10.0 or
# v0.10.0. The script exits non-zero while the run is not green.
set -euo pipefail

ref=${1:?usage: verify-nix-install.sh REF}
repo=zanza00/what-to-run
workflow=nix-install.yml

die() {
  echo "verify-nix-install: $*" >&2
  exit 1
}

# Print the newest run of the workflow whose head is the resolved commit.
# The branch filter keeps the search narrow; the sha filter keeps it exact.
run_on_ref() {
  gh run list --workflow "$workflow" --branch "$ref" --limit 15 \
    --json databaseId,headSha \
    --jq ".[] | select(.headSha == \"$sha\") | .databaseId" | head -n1
}

sha=$(gh api "repos/$repo/commits/$ref" --jq .sha) ||
  die "cannot resolve ref $ref"

run_id=$(run_on_ref)

if [ -z "$run_id" ]; then
  gh workflow run "$workflow" --ref "$ref" ||
    die "cannot dispatch $workflow on $ref"
  # A dispatched run takes a few seconds to appear in the list.
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12; do
    run_id=$(run_on_ref)
    if [ -n "$run_id" ]; then
      break
    fi
    sleep 5
  done
  if [ -z "$run_id" ]; then
    die "the dispatched run did not appear"
  fi
fi

attempts=0
while :; do
  if gh run watch "$run_id" --exit-status; then
    echo "Nix install verified on $ref."
    exit 0
  fi
  attempts=$((attempts + 1))
  if [ "$attempts" -ge 3 ]; then
    break
  fi
  # Retry only when the pool cancelled jobs before a runner took them. A real
  # failure needs a fix on the ref, and a rerun cannot repair that.
  queued_only=$(gh api "repos/$repo/actions/runs/$run_id/jobs" \
    --jq '[.jobs[] | select(.conclusion == "cancelled" and (.runner_name == null or .runner_name == ""))] | length')
  if [ "$queued_only" -eq 0 ]; then
    break
  fi
  echo "The pool cancelled the run before a runner took it; rerunning the failed jobs (attempt $attempts of 2)."
  gh run rerun "$run_id" --failed ||
    die "cannot rerun run $run_id"
done

die "run $run_id is not green: https://github.com/$repo/actions/runs/$run_id"
