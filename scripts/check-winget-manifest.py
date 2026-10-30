#!/usr/bin/env python3
"""Check the WinGet manifests a release writes into microsoft/winget-pkgs."""

import argparse
from pathlib import Path
import re
import sys

IDENTIFIER = "zanza00.WhatToRun"
FILES = {
    "version": f"{IDENTIFIER}.yaml",
    "installer": f"{IDENTIFIER}.installer.yaml",
    "locale": f"{IDENTIFIER}.locale.en-US.yaml",
}
# The two architectures the release builds for Windows, mapped to the archive
# winget downloads for each of them.
ARCHIVES = {
    "x64": "what-to-run_{version}_windows_amd64.zip",
    "arm64": "what-to-run_{version}_windows_arm64.zip",
}
RELEASE_URL = "https://github.com/zanza00/what-to-run/releases/download/{tag}/{name}"

# Only these keys are read, so a description such as "Runs tasks: fast" can
# never be mistaken for a manifest field.
FIELDS = {
    "PackageIdentifier",
    "PackageVersion",
    "InstallerType",
    "NestedInstallerType",
    "Architecture",
    "RelativeFilePath",
    "PortableCommandAlias",
    "InstallerUrl",
    "InstallerSha256",
}


def read_checksums(path):
    sums = {}
    for line in path.read_text().splitlines():
        if not line.strip():
            continue
        fields = line.split()
        if len(fields) != 2:
            raise ValueError(f"Expected '<sha256>  <name>' in {path}, got {line!r}")
        digest, name = fields
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise ValueError(f"Invalid SHA-256 for {name} in {path}")
        sums[name] = digest
    return sums


def read_fields(text):
    """Return the manifest fields in file order.

    A line reader is enough for these manifests and keeps the checker free of a
    YAML dependency. Indentation is dropped before the key is read, so the
    GoReleaser output and a hand-written manifest parse the same way.
    """
    fields = []
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("- "):
            line = line[2:].strip()
        key, separator, value = line.partition(":")
        if not separator or key not in FIELDS:
            continue
        fields.append((key, value.strip().strip('"')))
    return fields


def parse_installers(fields):
    """Group the installer entries by architecture.

    `NestedInstallerType` is read as the default until an architecture starts:
    GoReleaser repeats it in every entry, a hand-written manifest may set it
    once for the whole list.
    """
    installers = {}
    default_type = None
    current = None
    for key, value in fields:
        if key == "Architecture":
            current = installers.setdefault(value, {})
            current.setdefault("NestedInstallerType", default_type)
        elif key == "NestedInstallerType" and current is None:
            default_type = value
        elif current is not None:
            current[key] = value
    return installers


def check(tag, manifests, sums):
    """Return the reasons the manifests do not install this release."""
    if not re.fullmatch(r"v\d+\.\d+\.\d+", tag):
        raise ValueError(f"Expected a stable vX.Y.Z tag, got {tag!r}")
    version = tag[1:]
    problems = []

    version_fields = dict(read_fields(manifests["version"]))
    installer_fields = read_fields(manifests["installer"])
    installer_scalars = dict(installer_fields)
    locale_fields = dict(read_fields(manifests["locale"]))
    for name, fields in (
        ("version", version_fields),
        ("installer", installer_scalars),
        ("locale", locale_fields),
    ):
        if fields.get("PackageIdentifier") != IDENTIFIER:
            problems.append(
                f"{name} manifest is for {fields.get('PackageIdentifier')!r}, "
                f"expected {IDENTIFIER!r}")
        if fields.get("PackageVersion") != version:
            problems.append(
                f"{name} manifest carries version "
                f"{fields.get('PackageVersion')!r}, expected {version!r}")

    installer_type = installer_scalars.get("InstallerType")
    if installer_type != "zip":
        problems.append(
            f"installer manifest has InstallerType {installer_type!r}, "
            "expected 'zip'")

    installers = parse_installers(installer_fields)
    for arch in sorted(set(installers) - set(ARCHIVES)):
        problems.append(f"manifest carries an unexpected {arch} installer")
    for arch, template in ARCHIVES.items():
        installer = installers.get(arch)
        if installer is None:
            problems.append(f"manifest has no {arch} installer")
            continue
        if installer.get("NestedInstallerType") != "portable":
            problems.append(
                f"{arch} installer type is "
                f"{installer.get('NestedInstallerType')!r}, expected 'portable'")
        name = template.format(version=version)
        url = RELEASE_URL.format(tag=tag, name=name)
        if installer.get("InstallerUrl") != url:
            problems.append(
                f"{arch} downloads {installer.get('InstallerUrl')!r}, "
                f"expected {url!r}")
        if installer.get("RelativeFilePath") != "wtr.exe":
            problems.append(
                f"{arch} installs {installer.get('RelativeFilePath')!r}, "
                "expected 'wtr.exe'")
        if installer.get("PortableCommandAlias") != "wtr":
            problems.append(
                f"{arch} alias is {installer.get('PortableCommandAlias')!r}, "
                "expected 'wtr'")
        digest = sums.get(name)
        if digest is None:
            problems.append(f"checksums.txt has no {name}")
        elif installer.get("InstallerSha256", "").lower() != digest:
            problems.append(
                f"{arch} hash does not match checksums.txt for {name}")
    return problems


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tag", help="release tag, for example v0.8.0")
    parser.add_argument("directory", type=Path,
                        help="directory holding the three manifest files")
    parser.add_argument("checksums", type=Path,
                        help="checksums.txt from the release")
    args = parser.parse_args()
    try:
        sums = read_checksums(args.checksums)
        manifests = {
            name: (args.directory / filename).read_text()
            for name, filename in FILES.items()
        }
        problems = check(args.tag, manifests, sums)
    except (OSError, ValueError) as error:
        print(error, file=sys.stderr)
        return 1
    if problems:
        for problem in problems:
            print(problem, file=sys.stderr)
        return 1
    print(f"WinGet manifest for {args.tag} matches the release")
    return 0


if __name__ == "__main__":
    sys.exit(main())
