#!/usr/bin/env python3
"""Publisher for the pacman repository hosted on GitHub Pages.

The release workflow runs the script inside an Arch Linux container, because
`repo-add`, the tool that writes the repository database, only ships with
pacman. Publishing is skipped, with a message, while no GPG key is configured
(`APT_GPG_KEY_FILE` unset or empty): the key that signs the apt repository and
the RPM packages also signs the pacman databases.
"""

import argparse
import hashlib
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

PROJECT = "what-to-run"
MAINTAINER_NAME = "Simone Picciani"
MAINTAINER_EMAIL = "codefloe@simonepicciani.it"
KEY_FINGERPRINT = "D37ADCA12D5035565D1C232291119F65C5CA2B1F"
DEFAULT_REPO_URL = "https://github.com/zanza00/what-to-run.git"
DEFAULT_BRANCH = "gh-pages"
ARCH_DIRECTORY = "arch"
DATABASE = "what-to-run.db"
FILES_DATABASE = "what-to-run.files"
# A pacman database keys its entries by package name and version only, so the
# database of one architecture would overwrite the other. The repository
# serves one database per architecture under arch/<arch>, and the pacman.conf
# server expands $arch, like the official repositories do. The database keeps
# one entry per package name, the newest: repo-add removes the older entry
# when it adds a newer release. The directory keeps every published package,
# like the RPM directory, so a manual pacman -U can still downgrade.
ARCHITECTURES = (("x86_64", "amd64"), ("aarch64", "arm64"))


def package_hashes(tag, checksums):
    """Return the {asset name: SHA-256} map of the release's Arch packages."""
    if not re.fullmatch(r"v\d+\.\d+\.\d+", tag):
        raise ValueError("Expected a stable vX.Y.Z tag")
    hashes = {}
    for line in checksums.splitlines():
        if not line.strip():
            continue
        parts = line.split()
        if len(parts) != 2:
            raise ValueError("Invalid checksum line")
        digest, name = parts
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise ValueError("Invalid SHA-256 for %s" % name)
        if name in hashes:
            raise ValueError("Duplicate checksum for %s" % name)
        hashes[name] = digest
    version = tag[1:]
    expected = [
        "what-to-run_%s_linux_%s.pkg.tar.zst" % (version, go_arch)
        for _, go_arch in ARCHITECTURES
    ]
    for name in expected:
        if name not in hashes:
            raise KeyError(name)
    return {name: hashes[name] for name in expected}


def _clone_url(repo_url):
    url = repo_url or os.environ.get("APT_REPO_URL") or DEFAULT_REPO_URL
    token = os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN")
    if token and url.startswith("https://github.com/"):
        return url.replace("https://", "https://x-access-token:%s@" % token, 1)
    return url


def _key_fingerprint(key_file):
    result = subprocess.run(
        ["gpg", "--batch", "--with-colons", "--import-options", "show-only",
         "--import", str(key_file)],
        capture_output=True, text=True,
    )
    out = result.stdout or result.stderr or ""
    for line in out.splitlines():
        if line.startswith("fpr:"):
            return line.split(":")[9]
    return KEY_FINGERPRINT


def _package_file(dist_dir, go_arch, version):
    """Return the release package for one architecture, requiring it to exist."""
    path = dist_dir / ("what-to-run_%s_linux_%s.pkg.tar.zst" % (version, go_arch))
    if not path.is_file():
        raise ValueError("Missing Arch Linux package: %s" % path.name)
    return path


def _verify_digest(path, digest):
    computed = hashlib.sha256(path.read_bytes()).hexdigest()
    if computed != digest:
        raise ValueError("SHA-256 mismatch for %s" % path.name)


def _materialize_database(directory, name):
    """Serve `name` as a real file, keeping the .tar.gz twin for repo-add.

    repo-add writes `<name>.tar.gz` and links `name` to it, or the other way
    around. The repository serves the file under the name pacman requests,
    while repo-add reads the `.tar.gz` name when the next release appends to
    the database, so both must exist as real files.
    """
    base = directory / name
    tarball = directory / (name + ".tar.gz")
    if base.is_symlink():
        base.unlink()
    if tarball.is_file():
        shutil.copy2(tarball, base)


def _build_arch_repository(arch_dir, package, environment):
    """Copy the package and write the repo-add database for one architecture."""
    arch_dir.mkdir(parents=True, exist_ok=True)
    shutil.copy2(package, arch_dir / package.name)
    try:
        subprocess.run(
            ["repo-add", "--quiet", str(arch_dir / (DATABASE + ".tar.gz")),
             str(arch_dir / package.name)],
            check=True, capture_output=True, text=True, env=environment,
        )
    except FileNotFoundError as error:
        raise SystemExit(
            "repo-add is required to publish the pacman repository; "
            "it ships with pacman, so run the script on Arch Linux"
        ) from error
    for name in (DATABASE, FILES_DATABASE):
        _materialize_database(arch_dir, name)


def _sign(arch_dir, name, environment, fingerprint):
    subprocess.run(
        ["gpg", "--batch", "--yes", "--armor", "--detach-sign",
         "--local-user", fingerprint,
         "--output", str(arch_dir / (name + ".sig")), str(arch_dir / name)],
        check=True, capture_output=True, text=True, env=environment,
    )


def publish(tag, hashes, key_file=None, dist_dir=None, repo_url=None,
            work_dir=None, dry_run=False):
    version = tag[1:]
    # The release workflow exports an empty value when the secret is unset,
    # which means the same thing as not setting the variable.
    if not key_file:
        key_file = os.environ.get("APT_GPG_KEY_FILE") or None
    if key_file is None and not dry_run:
        print("Skipping pacman repository publish: APT_GPG_KEY_FILE is not set")
        return
    if key_file is not None:
        key_path = Path(key_file)
        if not key_path.is_file():
            raise ValueError("GPG key file not found: %s" % key_path)
    else:
        key_path = None
    if dist_dir is None:
        dist_dir = Path("dist")
    else:
        dist_dir = Path(dist_dir)

    packages = {}
    for pacman_arch, go_arch in ARCHITECTURES:
        package = _package_file(dist_dir, go_arch, version)
        _verify_digest(package, hashes[package.name])
        packages[pacman_arch] = package

    if work_dir is None:
        work = Path(tempfile.mkdtemp(prefix="pacman-"))
    else:
        work = Path(work_dir)
    url = _clone_url(repo_url)
    environment = os.environ.copy()

    clone = subprocess.run(
        ["git", "clone", "--quiet", url, str(work)],
        capture_output=True, text=True, env=environment,
    )
    if clone.returncode != 0:
        work.mkdir(parents=True, exist_ok=True)
        subprocess.run(["git", "-C", str(work), "init", "--quiet"], check=True, env=environment)
    else:
        subprocess.run(
            ["git", "-C", str(work), "fetch", "origin", DEFAULT_BRANCH],
            capture_output=True, text=True, env=environment,
        )
    checkout = subprocess.run(
        ["git", "-C", str(work), "checkout", DEFAULT_BRANCH],
        capture_output=True, text=True, env=environment,
    )
    if checkout.returncode != 0:
        subprocess.run(
            ["git", "-C", str(work), "checkout", "--orphan", DEFAULT_BRANCH],
            check=True, env=environment,
        )
        subprocess.run(
            ["git", "-C", str(work), "rm", "-rf", "."],
            capture_output=True, text=True, env=environment,
        )

    fingerprint = None
    if key_path is not None:
        subprocess.run(
            ["gpg", "--batch", "--import", str(key_path)],
            check=True, capture_output=True, text=True, env=environment,
        )
        fingerprint = _key_fingerprint(key_path)

    for pacman_arch, package in packages.items():
        arch_dir = work / ARCH_DIRECTORY / pacman_arch
        _build_arch_repository(arch_dir, package, environment)
        if fingerprint is not None:
            for name in (DATABASE, FILES_DATABASE):
                _sign(arch_dir, name, environment, fingerprint)

    (work / ".nojekyll").write_text("")

    if dry_run:
        destination = dist_dir / "repo"
        shutil.copytree(work, destination, dirs_exist_ok=True)
        print(destination)
        return

    subprocess.run(["git", "-C", str(work), "add", "-A"], check=True, env=environment)
    # repo-add dates the databases and gpg dates the signatures, so a
    # republish always changes the tree: there is no "nothing to commit" case.
    subprocess.run(
        ["git", "-C", str(work), "config", "user.name", MAINTAINER_NAME],
        check=True, env=environment,
    )
    subprocess.run(
        ["git", "-C", str(work), "config", "user.email", MAINTAINER_EMAIL],
        check=True, env=environment,
    )
    subprocess.run(
        ["git", "-C", str(work), "commit", "-m", "Update the pacman repository to %s" % tag],
        check=True, env=environment,
    )
    subprocess.run(
        ["git", "-C", str(work), "push", "origin", DEFAULT_BRANCH],
        check=True, env=environment,
    )
    print("Published pacman repository for %s" % tag)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--artifact", type=Path, required=True)
    parser.add_argument("--gpg-key-file", type=Path)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    key_file = args.gpg_key_file or os.environ.get("APT_GPG_KEY_FILE") or None
    if key_file is None and not args.dry_run:
        print("Skipping pacman repository publish: APT_GPG_KEY_FILE is not set")
        return
    hashes = package_hashes(args.tag, args.artifact.read_text())
    publish(
        args.tag,
        hashes,
        key_file=key_file,
        dist_dir=args.artifact.parent,
        dry_run=args.dry_run,
    )


if __name__ == "__main__":
    main()
