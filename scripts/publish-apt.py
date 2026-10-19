#!/usr/bin/env python3
"""GoReleaser publisher for an APT repository hosted on GitHub Pages."""

import argparse
import os
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
    expected = ["what-to-run_%s_linux_amd64.deb" % version,
                "what-to-run_%s_linux_arm64.deb" % version]
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
        key_file = os.environ.get("APT_GPG_KEY_FILE")
    if dist_dir is None:
        dist_dir = Path("dist")
    else:
        dist_dir = Path(dist_dir)

    version = tag[1:]
    if key_file is None and not dry_run:
        print("Skipping apt publish: APT_GPG_KEY_FILE is not set")
        return
    if key_file is not None:
        key_path = Path(key_file)
        if not key_path.is_file():
            raise ValueError("GPG key file not found: %s" % key_path)
    else:
        key_path = None

    debs = []
    if dist_dir.is_dir():
        for path in sorted(dist_dir.glob("*.deb")):
            if version in path.name:
                debs.append(path)

    if work_dir is None:
        work = Path(tempfile.mkdtemp(prefix="apt-repo-"))
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

    if key_path is not None:
        fingerprint = _key_fingerprint(key_path)
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
        destination = dist_dir / "apt"
        destination.mkdir(parents=True, exist_ok=True)
        subprocess.run(
            ["cp", "-a", "%s/." % work, "%s/" % destination],
            check=True, env=environment,
        )
        print(destination)
        return

    subprocess.run(["git", "-C", str(work), "add", "-A"], check=True, env=environment)
    status = subprocess.run(
        ["git", "-C", str(work), "status", "--porcelain"],
        check=True, capture_output=True, text=True, env=environment,
    )
    if not status.stdout.strip():
        print("APT repository already matches %s" % tag)
        return
    subprocess.run(
        ["git", "-C", str(work), "config", "user.name", MAINTAINER_NAME],
        check=True, env=environment,
    )
    subprocess.run(
        ["git", "-C", str(work), "config", "user.email", MAINTAINER_EMAIL],
        check=True, env=environment,
    )
    subprocess.run(
        ["git", "-C", str(work), "commit", "-m", "Update apt repository to %s" % tag],
        check=True, env=environment,
    )
    subprocess.run(
        ["git", "-C", str(work), "push", "origin", DEFAULT_BRANCH],
        check=True, env=environment,
    )
    print("Published apt repository for %s" % tag)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--artifact", type=Path, required=True)
    parser.add_argument("--gpg-key-file", type=Path)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    if args.artifact.name != "checksums.txt":
        return
    key_file = args.gpg_key_file or os.environ.get("APT_GPG_KEY_FILE")
    if key_file is None and not args.dry_run:
        print("Skipping apt publish: APT_GPG_KEY_FILE is not set")
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
