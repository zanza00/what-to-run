#!/usr/bin/env bash
# Check a published what-to-run release across every distribution channel.
#
# Usage: verify-release.sh vX.Y.Z
#
# Checks the GitHub release and its exact asset set, the Homebrew formula, the
# Scoop manifest in its bucket, the apt index and RPM metadata (including its
# signature) on gh-pages, the AUR package and the npm registry when those
# channels are configured, and a downloaded binary of this machine's platform.
# The script reports a channel the workflow skips (a secret is unset, or the
# npm opt-in variable is off) as skipped, not failed.
set -uo pipefail

repo=zanza00/what-to-run
tap_repo=zanza00/homebrew-tap
scoop_repo=zanza00/scoop-bucket
aur_package=what-to-run-bin
pages_url=https://zanza00.github.io/what-to-run

failures=0
checks=0

usage() {
  echo "usage: $(basename "$0") vX.Y.Z" >&2
  exit 2
}

ok() {
  checks=$((checks + 1))
  printf 'ok    %s\n' "$*"
}

bad() {
  checks=$((checks + 1))
  failures=$((failures + 1))
  printf 'FAIL  %s\n' "$*" >&2
}

skip() {
  printf 'skip  %s\n' "$*"
}

warn() {
  printf 'warn  %s\n' "$*" >&2
}

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing command: $1" >&2
    exit 2
  }
}

tag=${1:-}
if [[ -z $tag || ! $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  usage
fi
version=${tag#v}

need gh
need tar

# The publisher, the launcher and the Scoop checker live in the repository
# this script is part of.
repo_root=$(cd "$(dirname "$0")/../../../.." && pwd)

# The release workflow skips the package repositories and the AUR when their
# secret is unset. Read the names (not the values) to know what this release
# promised.
secrets=""
known_secrets=1
if ! secrets=$(gh api "repos/$repo/actions/secrets" --jq '.secrets[].name' 2>/dev/null); then
  known_secrets=0
  warn "cannot read Actions secrets; repository and AUR problems are reported without failing"
fi
has_secret() {
  [[ $known_secrets == 1 ]] && grep -qx "$1" <<<"$secrets"
}

bad_or_warn() {
  if [[ $known_secrets == 1 ]]; then
    bad "$1"
  else
    warn "$1"
  fi
}

# rpm_has ARCH VERSION: true when the primary metadata has a what-to-run
# package for that architecture with that version.
rpm_has() {
  local arch=$1 version=$2
  awk -v arch="$arch" -v version="$version" '
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

# --- GitHub release -------------------------------------------------------

release_found=0
if flags=$(gh release view "$tag" --repo "$repo" \
  --json isDraft,isPrerelease \
  --jq '"\(.isDraft) \(.isPrerelease)"' 2>/dev/null); then
  release_found=1
  if [[ $flags == "false false" ]]; then
    ok "GitHub release $tag is published, neither draft nor prerelease"
  else
    bad "GitHub release $tag is a draft or prerelease (draft/prerelease: $flags)"
  fi
else
  bad "GitHub release $tag not found"
fi

if [[ $release_found == 1 ]]; then
  expected=$(printf '%s\n' \
    checksums.txt \
    "what-to-run_${version}_darwin_amd64.tar.gz" \
    "what-to-run_${version}_darwin_arm64.tar.gz" \
    "what-to-run_${version}_linux_amd64.deb" \
    "what-to-run_${version}_linux_amd64.rpm" \
    "what-to-run_${version}_linux_amd64.tar.gz" \
    "what-to-run_${version}_linux_arm64.deb" \
    "what-to-run_${version}_linux_arm64.rpm" \
    "what-to-run_${version}_linux_arm64.tar.gz" \
    "what-to-run_${version}_windows_amd64.zip" \
    "what-to-run_${version}_windows_arm64.zip" | sort)
  actual=$(gh release view "$tag" --repo "$repo" \
    --json assets --jq '.assets[].name' 2>/dev/null | sort)
  if [[ $actual == "$expected" ]]; then
    ok "release carries exactly the 11 expected assets"
  else
    bad "release assets differ from the expected 11"
    while IFS= read -r name; do
      [[ -z $name ]] && continue
      grep -qxF "$name" <<<"$actual" ||
        printf '        missing: %s\n' "$name" >&2
    done <<<"$expected"
    while IFS= read -r name; do
      [[ -z $name ]] && continue
      grep -qxF "$name" <<<"$expected" ||
        printf '        unexpected: %s\n' "$name" >&2
    done <<<"$actual"
  fi
fi

# --- Homebrew -------------------------------------------------------------

if formula=$(gh api "repos/$tap_repo/contents/Formula/what-to-run.rb" \
  -H "Accept: application/vnd.github.raw" 2>/dev/null); then
  if grep -qx "  version \"$version\"" <<<"$formula" &&
    grep -q "/download/$tag/" <<<"$formula"; then
    ok "Homebrew formula in $tap_repo points at $tag"
  else
    bad "Homebrew formula in $tap_repo does not point at $tag"
  fi
else
  bad "cannot read Formula/what-to-run.rb from $tap_repo"
fi

# --- Scoop ----------------------------------------------------------------

if [[ $known_secrets == 1 ]] && ! has_secret SCOOP_BUCKET_TOKEN; then
  skip "Scoop: SCOOP_BUCKET_TOKEN is not configured"
elif ! command -v python3 >/dev/null 2>&1; then
  warn "python3 not available: cannot check the Scoop manifest"
else
  scoop_tmp=$(mktemp -d)
  manifest=$(gh api "repos/$scoop_repo/contents/bucket/what-to-run.json" \
    -H "Accept: application/vnd.github.raw" 2>/dev/null) || manifest=""
  checker="$repo_root/scripts/check-scoop-manifest.py"
  if [[ -z $manifest ]]; then
    bad_or_warn "cannot read bucket/what-to-run.json from $scoop_repo"
  elif [[ ! -f $checker ]]; then
    bad_or_warn "cannot find scripts/check-scoop-manifest.py under $repo_root"
  elif ! gh release download "$tag" --repo "$repo" \
      --pattern checksums.txt --dir "$scoop_tmp" --clobber >/dev/null 2>&1; then
    bad_or_warn "cannot download checksums.txt to check the Scoop manifest"
  else
    printf '%s\n' "$manifest" >"$scoop_tmp/what-to-run.json"
    if problems=$(python3 "$checker" "$tag" "$scoop_tmp/what-to-run.json" \
      "$scoop_tmp/checksums.txt" 2>&1); then
      ok "Scoop manifest in $scoop_repo is at $tag with matching hashes"
    else
      bad_or_warn "Scoop manifest in $scoop_repo is wrong"
      while IFS= read -r problem; do
        [[ -z $problem ]] && continue
        printf '        %s\n' "$problem" >&2
      done <<<"$problems"
    fi
  fi
  rm -rf "$scoop_tmp"
fi

# --- apt and RPM repositories ---------------------------------------------

repositories_expected=1
if [[ $known_secrets == 1 ]] && ! has_secret APT_GPG_PRIVATE_KEY; then
  repositories_expected=0
  skip "apt and RPM repositories: APT_GPG_PRIVATE_KEY is not configured"
fi
if [[ $repositories_expected == 1 ]]; then
  for arch in amd64 arm64; do
    packages=$(gh api \
      "repos/$repo/contents/dists/stable/main/binary-$arch/Packages?ref=gh-pages" \
      -H "Accept: application/vnd.github.raw" 2>/dev/null) || packages=""
    if [[ -z $packages ]]; then
      bad_or_warn "cannot read the apt index for $arch from gh-pages"
    elif grep -A1 -x "Package: what-to-run" <<<"$packages" |
      grep -qx "Version: $version"; then
      ok "apt index for $arch carries $version"
    else
      bad_or_warn "apt index for $arch does not carry $version"
    fi
  done

  if ! metadata=$(gh api \
    "repos/$repo/contents/rpm/repodata/primary.xml.gz?ref=gh-pages" \
    -H "Accept: application/vnd.github.raw" 2>/dev/null | gzip -dc); then
    metadata=""
  fi
  if [[ -z $metadata ]]; then
    bad_or_warn "cannot read the RPM metadata from gh-pages"
  else
    for arch in x86_64 aarch64; do
      if rpm_has "$arch" "$version"; then
        ok "RPM metadata carries $version for $arch"
      else
        bad_or_warn "RPM metadata does not carry $version for $arch"
      fi
    done

    need gpg
    keyring=$(mktemp -d)
    signature=$(mktemp)
    repomd=$(mktemp)
    if ! gh api "repos/$repo/contents/key.asc?ref=gh-pages" \
        -H "Accept: application/vnd.github.raw" >"$keyring/key.asc" 2>/dev/null ||
      ! gh api "repos/$repo/contents/rpm/repodata/repomd.xml.asc?ref=gh-pages" \
        -H "Accept: application/vnd.github.raw" >"$signature" 2>/dev/null ||
      ! gh api "repos/$repo/contents/rpm/repodata/repomd.xml?ref=gh-pages" \
        -H "Accept: application/vnd.github.raw" >"$repomd" 2>/dev/null; then
      bad_or_warn "cannot read the RPM metadata signature from gh-pages"
    elif GNUPGHOME="$keyring" gpg --batch --quiet --import "$keyring/key.asc" 2>/dev/null &&
      GNUPGHOME="$keyring" gpg --batch --quiet --verify "$signature" "$repomd" 2>/dev/null; then
      ok "RPM metadata signature verifies with the repository key"
    else
      bad_or_warn "RPM metadata signature does not verify"
    fi
    rm -rf "$keyring" "$signature" "$repomd"
  fi

  for script in setup.deb.sh setup.rpm.sh; do
    published=""
    published=$(gh api "repos/$repo/contents/$script?ref=gh-pages" \
      -H "Accept: application/vnd.github.raw" 2>/dev/null) || published=""
    if [[ -z $published ]]; then
      bad_or_warn "setup script $script is missing from gh-pages"
    elif ! grep -qF "$pages_url" <<<"$published"; then
      bad_or_warn "setup script $script does not point at $pages_url"
    else
      ok "setup script $script is published on gh-pages"
    fi
  done
fi

# --- AUR ------------------------------------------------------------------

if [[ $known_secrets == 1 ]] && ! has_secret AUR_SSH_PRIVATE_KEY; then
  skip "AUR: AUR_SSH_PRIVATE_KEY is not configured"
elif ! command -v curl >/dev/null 2>&1; then
  warn "curl not available: cannot check the AUR package"
else
  pkgbuild=$(curl -fsSL --max-time 20 \
    "https://aur.archlinux.org/cgit/aur.git/plain/PKGBUILD?h=$aur_package" \
    2>/dev/null) || pkgbuild=""
  if [[ -z $pkgbuild ]]; then
    if [[ $known_secrets == 1 ]]; then
      bad "cannot read the AUR PKGBUILD for $aur_package"
    else
      warn "cannot read the AUR PKGBUILD for $aur_package"
    fi
  elif grep -qx "pkgver=$version" <<<"$pkgbuild"; then
    ok "AUR $aur_package carries $version"
  else
    if [[ $known_secrets == 1 ]]; then
      bad "AUR $aur_package does not carry $version"
    else
      warn "AUR $aur_package does not carry $version"
    fi
  fi
fi

# --- npm --------------------------------------------------------------------

# The channel is opt-in: the workflow publishes it only when the NPM_PUBLISH
# repository variable is "true", and skips it with a warning otherwise. When
# the variable list is unreadable, npm problems are reported without failing,
# like the channels behind optional secrets.
npm_publish=""
known_variables=1
if ! npm_publish=$(gh api "repos/$repo/actions/variables" \
  --jq '.variables[] | select(.name == "NPM_PUBLISH") | .value' 2>/dev/null); then
  known_variables=0
  warn "cannot read the repository variables; npm problems are reported without failing"
fi

if [[ $known_variables == 1 && $npm_publish != "true" ]]; then
  skip 'npm: NPM_PUBLISH is not set to "true"'
elif ! command -v npm >/dev/null 2>&1; then
  warn "npm not available: cannot check the npm packages"
elif ! command -v python3 >/dev/null 2>&1; then
  warn "python3 not available: cannot list the npm packages"
else
  # The publisher owns the platform table. Read the six package names from it
  # instead of copying them here.
  platform_packages=$(python3 -c '
import importlib.util
import sys

spec = importlib.util.spec_from_file_location("publish_npm", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
for goos, goarch in module.TARGETS:
    print(module.package_name(goos, goarch))
' "$repo_root/scripts/publish-npm.py") || platform_packages=""
  if [[ -z $platform_packages ]]; then
    bad_or_warn "cannot list the platform packages from scripts/publish-npm.py"
  else
    for package in what-to-run $platform_packages; do
      if got=$(npm view "$package@$version" version 2>/dev/null) &&
        [[ $got == "$version" ]]; then
        ok "npm package $package carries $version"
      else
        bad_or_warn "npm package $package does not carry $version"
      fi
    done

    # The launcher's own table says which platform package this machine
    # installs, so an uncovered platform skips the wrapper run instead of
    # failing it.
    machine_package=$(node -e '
const wtr = require(process.argv[1]);
console.log(wtr.packageName() || "");
' "$repo_root/npm/bin/wtr.js" 2>/dev/null) || machine_package=""
    if [[ -z $machine_package ]]; then
      skip "npm wrapper run: no platform package for $(uname -s) $(uname -m)"
    elif got=$(npx --yes "what-to-run@$version" --version 2>/dev/null) &&
      [[ $got == "wtr $version" ]]; then
      ok "npx what-to-run@$version reports '$got'"
    else
      bad_or_warn "npx what-to-run@$version does not report 'wtr $version'"
    fi
  fi
fi

# --- Version reported by a released binary --------------------------------

archive=""
case "$(uname -s):$(uname -m)" in
  Linux:x86_64 | Linux:amd64) archive="what-to-run_${version}_linux_amd64.tar.gz" ;;
  Linux:aarch64 | Linux:arm64) archive="what-to-run_${version}_linux_arm64.tar.gz" ;;
  Darwin:x86_64) archive="what-to-run_${version}_darwin_amd64.tar.gz" ;;
  Darwin:arm64 | Darwin:aarch64) archive="what-to-run_${version}_darwin_arm64.tar.gz" ;;
esac

if [[ -z $archive ]]; then
  skip "version smoke test: no archive for $(uname -s) $(uname -m)"
elif [[ $release_found == 0 ]]; then
  skip "version smoke test: release not found"
else
  tmp=$(mktemp -d) || {
    echo "cannot create a temporary directory" >&2
    exit 2
  }
  trap 'rm -rf "$tmp"' EXIT
  if ! gh release download "$tag" --repo "$repo" \
    --pattern "$archive" --pattern checksums.txt \
    --dir "$tmp" --clobber >/dev/null 2>&1; then
    bad "cannot download $archive and checksums.txt"
  else
    expected_sum=$(awk -v file="$archive" '$2 == file {print $1}' \
      "$tmp/checksums.txt")
    actual_sum=""
    if command -v sha256sum >/dev/null 2>&1; then
      actual_sum=$(sha256sum "$tmp/$archive" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      actual_sum=$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')
    fi
    if [[ -z $actual_sum ]]; then
      warn "no sha256 tool available: cannot verify $archive"
    elif [[ $expected_sum == "$actual_sum" ]]; then
      ok "$archive matches checksums.txt"
    else
      bad "$archive does not match checksums.txt"
    fi

    if ! tar -xzf "$tmp/$archive" -C "$tmp" 2>/dev/null; then
      bad "cannot extract $archive"
    else
      got=$("$tmp/wtr" --version 2>/dev/null) || got=""
      if [[ $got == "wtr $version" ]]; then
        ok "downloaded wtr reports '$got'"
      else
        bad "downloaded wtr reports '${got:-nothing}' instead of 'wtr $version'"
      fi
    fi
  fi
fi

# --- Summary --------------------------------------------------------------

printf '\n%d checks, %d failed\n' "$checks" "$failures"
if ((failures > 0)); then
  exit 1
fi
