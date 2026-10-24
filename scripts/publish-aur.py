#!/usr/bin/env python3
"""GoReleaser publisher for an Arch User Repository package.

Publishing is skipped, with a message, while no SSH key is configured
(`AUR_SSH_KEY_FILE` unset or empty) so releases keep working before the AUR
package exists. A configured but unusable key is an error.
"""

import argparse
import os
from pathlib import Path
import re
import subprocess
import tempfile


PACKAGE = "what-to-run-bin"
UPSTREAM = "what-to-run"
DESCRIPTION = "Task picker for the commands you can never remember in a repository"
HOMEPAGE = "https://github.com/zanza00/what-to-run"
MAINTAINER_NAME = "Simone Picciani"
MAINTAINER_EMAIL = "codefloe@simonepicciani.it"
AUR_URL = f"ssh://aur@aur.archlinux.org/{PACKAGE}.git"


def render(tag, checksums):
    """Return the PKGBUILD and .SRCINFO for a stable release tag."""
    if not re.fullmatch(r"v\d+\.\d+\.\d+", tag):
        raise ValueError("Expected a stable vX.Y.Z tag")
    version = tag[1:]
    hashes = {}
    for line in checksums.splitlines():
        digest, name = line.split()
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise ValueError(f"Invalid SHA-256 for {name}")
        if name in hashes:
            raise ValueError(f"Duplicate checksum for {name}")
        hashes[name] = digest
    sources = {}
    for arch, go_arch in (("x86_64", "amd64"), ("aarch64", "arm64")):
        name = f"{UPSTREAM}_{version}_linux_{go_arch}.tar.gz"
        sources[arch] = (f"{HOMEPAGE}/releases/download/{tag}/{name}", hashes[name])

    pkgbuild = [
        f"# Maintainer: {MAINTAINER_NAME} <{MAINTAINER_EMAIL}>",
        f"pkgname={PACKAGE}",
        f"pkgver={version}",
        "pkgrel=1",
        f'pkgdesc="{DESCRIPTION}"',
        "arch=('x86_64' 'aarch64')",
        f'url="{HOMEPAGE}"',
        f"license=('MIT')",
        # The package is what-to-run-bin and provides/conflicts with the
        # upstream package name; there is no second binary called what-to-run.
        f"provides=('{UPSTREAM}')",
        f"conflicts=('{UPSTREAM}')",
        "options=('!strip')",
    ]
    for arch, (url, _) in sources.items():
        pkgbuild.append(f"source_{arch}=('{url}')")
    for arch, (_, digest) in sources.items():
        pkgbuild.append(f"sha256sums_{arch}=('{digest}')")
    pkgbuild.extend([
        "",
        "package() {",
        '  install -Dm755 "${srcdir}/wtr" "${pkgdir}/usr/bin/wtr"',
        "}",
        "",
    ])

    # Field order follows `makepkg --printsrcinfo`, which the AUR validates.
    srcinfo = [
        f"pkgbase = {PACKAGE}",
        f"\tpkgdesc = {DESCRIPTION}",
        f"\tpkgver = {version}",
        "\tpkgrel = 1",
        f"\turl = {HOMEPAGE}",
        "\tarch = x86_64",
        "\tarch = aarch64",
        "\tlicense = MIT",
        f"\tprovides = {UPSTREAM}",
        f"\tconflicts = {UPSTREAM}",
        "\toptions = !strip",
    ]
    for arch, (url, _) in sources.items():
        srcinfo.append(f"\tsource_{arch} = {url}")
    for arch, (_, digest) in sources.items():
        srcinfo.append(f"\tsha256sums_{arch} = {digest}")
    srcinfo.extend(["", f"pkgname = {PACKAGE}", ""])

    return "\n".join(pkgbuild), "\n".join(srcinfo)


def publish(tag, pkgbuild, srcinfo, key_file):
    """Push the package to the AUR from a fresh clone of its repository."""
    key = Path(key_file)
    if not key.is_file():
        raise ValueError(f"AUR SSH key file not found: {key}")
    environment = os.environ.copy()
    environment["GIT_SSH_COMMAND"] = (
        f"ssh -i {key} -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new"
    )
    with tempfile.TemporaryDirectory() as directory:
        subprocess.run(
            ["git", "clone", "--quiet", AUR_URL, directory],
            check=True, env=environment,
        )
        repository = Path(directory)
        (repository / "PKGBUILD").write_text(pkgbuild)
        (repository / ".SRCINFO").write_text(srcinfo)
        status = subprocess.run(
            ["git", "-C", directory, "status", "--porcelain"],
            check=True, capture_output=True, text=True, env=environment,
        )
        if not status.stdout.strip():
            print(f"AUR package already matches {tag}")
            return
        subprocess.run(
            ["git", "-C", directory, "add", "PKGBUILD", ".SRCINFO"],
            check=True, env=environment,
        )
        subprocess.run(
            [
                "git", "-C", directory,
                "-c", f"user.name={MAINTAINER_NAME}",
                "-c", f"user.email={MAINTAINER_EMAIL}",
                "commit", "--quiet", "-m", f"Update to {tag}",
            ],
            check=True, env=environment,
        )
        subprocess.run(
            ["git", "-C", directory, "push", "--quiet", "origin", "master"],
            check=True, env=environment,
        )
    print(f"Published AUR package for {tag}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--artifact", type=Path, required=True)
    parser.add_argument("--ssh-key-file", type=Path)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    # GoReleaser invokes publishers for each archive and the checksum artifact.
    if args.artifact.name != "checksums.txt":
        return
    key_file = args.ssh_key_file or os.environ.get("AUR_SSH_KEY_FILE") or None
    if key_file is None and not args.dry_run:
        print("Skipping AUR publish: AUR_SSH_KEY_FILE is not set")
        return
    pkgbuild, srcinfo = render(args.tag, args.artifact.read_text())
    destination = args.artifact.parent / "aur" / PACKAGE
    destination.mkdir(parents=True, exist_ok=True)
    (destination / "PKGBUILD").write_text(pkgbuild)
    (destination / ".SRCINFO").write_text(srcinfo)
    if args.dry_run:
        print(destination)
    else:
        publish(args.tag, pkgbuild, srcinfo, key_file)


if __name__ == "__main__":
    main()
