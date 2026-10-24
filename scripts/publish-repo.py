#!/usr/bin/env python3
"""GoReleaser publisher for the apt and RPM repositories hosted on GitHub Pages."""

import argparse
import os
import shutil
import subprocess
import tempfile
from pathlib import Path
import re

PROJECT = "what-to-run"
HOMEPAGE = "https://github.com/zanza00/what-to-run"
MAINTAINER_NAME = "Simone Picciani"
MAINTAINER_EMAIL = "codefloe@simonepicciani.it"
KEY_FINGERPRINT = "D37ADCA12D5035565D1C232291119F65C5CA2B1F"
DEFAULT_REPO_URL = "https://github.com/zanza00/what-to-run.git"
DEFAULT_BRANCH = "gh-pages"
RPM_DIRECTORY = "rpm"
ARCHITECTURES = ("amd64", "arm64")

DISTRIBUTIONS = """Origin: zanza00
Label: what-to-run
Suite: stable
Codename: stable
Architectures: amd64 arm64
Components: main
Description: Task picker for the commands you can never remember in a repository
SignWith: {fingerprint}
""".format(fingerprint=KEY_FINGERPRINT)


def render(tag, checksums):
    if not re.fullmatch(r"v\d+\.\d+\.\d+", tag):
        raise ValueError("Expected a stable vX.Y.Z tag")
    version = tag[1:]
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
    expected = [
        "what-to-run_%s_linux_%s.deb" % (version, arch)
        for arch in ARCHITECTURES
    ]
    for name in expected:
        if name not in hashes:
            raise KeyError(name)
    return DISTRIBUTIONS


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


def _release_files(dist_dir, suffix, version):
    """Return the release files with the suffix and version in the name."""
    return sorted(
        path for path in dist_dir.glob("*" + suffix) if version in path.name
    )


def _rpm_files(tag, dist_dir):
    """Return the release RPMs, requiring both architectures when any exist."""
    version = tag[1:]
    expected = {
        dist_dir / ("what-to-run_%s_linux_%s.rpm" % (version, arch))
        for arch in ARCHITECTURES
    }
    present = {path for path in expected if path.is_file()}
    if not present:
        return []
    if present != expected:
        missing = sorted(path.name for path in expected - present)
        raise ValueError(
            "Missing RPMs for %s: %s" % (version, ", ".join(missing))
        )
    return sorted(present)


def _build_rpm_repository(tag, rpms, work, environment, fingerprint=None):
    """Stage the RPMs, regenerate the metadata and sign it when a key is given."""
    rpm_dir = work / RPM_DIRECTORY
    rpm_dir.mkdir(parents=True, exist_ok=True)
    for rpm in rpms:
        shutil.copy2(rpm, rpm_dir / rpm.name)
    try:
        subprocess.run(
            # gzip and simple names keep the metadata readable by every
            # dnf/yum version and by the verification steps.
            ["createrepo_c", "--quiet", "--general-compress-type=gz",
             "--simple-md-filenames", "--revision", tag, str(rpm_dir)],
            check=True, capture_output=True, text=True, env=environment,
        )
    except FileNotFoundError as error:
        raise SystemExit(
            "createrepo_c is required to publish the RPM repository"
        ) from error
    if fingerprint is not None:
        subprocess.run(
            ["gpg", "--batch", "--yes", "--armor", "--detach-sign",
             "--local-user", fingerprint,
             "--output", str(rpm_dir / "repodata" / "repomd.xml.asc"),
             str(rpm_dir / "repodata" / "repomd.xml")],
            check=True, capture_output=True, text=True, env=environment,
        )


def publish(tag, *args, **kwargs):
    key_file = kwargs.get("key_file") or kwargs.get("gpg_key_file")
    dist_dir = kwargs.get("dist_dir")
    repo_url = kwargs.get("repo_url")
    work_dir = kwargs.get("work_dir")
    dry_run = kwargs.get("dry_run", False)

    for positional in args:
        if isinstance(positional, (str, Path)):
            path = Path(positional)
            if path.is_dir():
                dist_dir = path
                continue
            key_file = positional
    if key_file is None:
        # The release workflow exports an empty value when the secret is
        # unset, which means the same thing as not setting the variable.
        key_file = os.environ.get("APT_GPG_KEY_FILE") or None
    if dist_dir is None:
        dist_dir = Path("dist")
    else:
        dist_dir = Path(dist_dir)

    version = tag[1:]
    if key_file is None and not dry_run:
        print("Skipping package repository publish: APT_GPG_KEY_FILE is not set")
        return
    if key_file is not None:
        key_path = Path(key_file)
        if not key_path.is_file():
            raise ValueError("GPG key file not found: %s" % key_path)
    else:
        key_path = None

    debs = _release_files(dist_dir, ".deb", version) if dist_dir.is_dir() else []
    rpms = _rpm_files(tag, dist_dir) if dist_dir.is_dir() else []

    if work_dir is None:
        work = Path(tempfile.mkdtemp(prefix="repo-"))
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

    conf_dir = work / "conf"
    conf_dir.mkdir(parents=True, exist_ok=True)
    (conf_dir / "distributions").write_text(DISTRIBUTIONS)

    fingerprint = KEY_FINGERPRINT
    if key_path is not None:
        subprocess.run(
            ["gpg", "--batch", "--import", str(key_path)],
            check=True, capture_output=True, text=True, env=environment,
        )
        fingerprint = _key_fingerprint(key_path)
        (conf_dir / "distributions").write_text(
            DISTRIBUTIONS.replace(KEY_FINGERPRINT, fingerprint)
        )

    for deb in debs:
        subprocess.run(
            ["reprepro", "-b", str(work), "includedeb", "stable", str(deb)],
            check=True, capture_output=True, text=True, env=environment,
        )

    if rpms:
        _build_rpm_repository(
            tag, rpms, work, environment,
            fingerprint if key_path is not None else None,
        )

    if key_path is not None:
        with open(work / "key.asc", "w") as handle:
            subprocess.run(
                ["gpg", "--batch", "--armor", "--export", fingerprint],
                stdout=handle, check=True, env=environment,
            )
        with open(work / "key.gpg", "wb") as handle:
            subprocess.run(
                ["gpg", "--batch", "--export", fingerprint],
                stdout=handle, check=True, env=environment,
            )
    (work / ".nojekyll").write_text("")

    if dry_run:
        destination = dist_dir / "repo"
        shutil.copytree(work, destination, dirs_exist_ok=True)
        print(destination)
        return

    subprocess.run(["git", "-C", str(work), "add", "-A"], check=True, env=environment)
    # reprepro dates the apt Release file and gpg dates the signatures, so a
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
        ["git", "-C", str(work), "commit", "-m", "Update package repositories to %s" % tag],
        check=True, env=environment,
    )
    subprocess.run(
        ["git", "-C", str(work), "push", "origin", DEFAULT_BRANCH],
        check=True, env=environment,
    )
    print("Published package repositories for %s" % tag)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--artifact", type=Path, required=True)
    parser.add_argument("--gpg-key-file", type=Path)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    if args.artifact.name != "checksums.txt":
        return
    key_file = args.gpg_key_file or os.environ.get("APT_GPG_KEY_FILE") or None
    if key_file is None and not args.dry_run:
        print("Skipping package repository publish: APT_GPG_KEY_FILE is not set")
        return
    render(args.tag, args.artifact.read_text())
    publish(
        args.tag,
        key_file=key_file,
        dist_dir=args.artifact.parent,
        dry_run=args.dry_run,
    )


if __name__ == "__main__":
    main()
