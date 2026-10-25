#!/usr/bin/env python3
"""Check the Scoop manifest a release writes into the bucket."""

import argparse
import json
from pathlib import Path
import re
import sys

# The two architectures the release builds for Windows, mapped to the archive
# GoReleaser hands Scoop for each of them.
ARCHIVES = {
    "64bit": "what-to-run_{version}_windows_amd64.zip",
    "arm64": "what-to-run_{version}_windows_arm64.zip",
}
RELEASE_URL = "https://github.com/zanza00/what-to-run/releases/download/{tag}/{name}"


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


def check(tag, manifest, sums):
    """Return the reasons the manifest does not install this release."""
    if not re.fullmatch(r"v\d+\.\d+\.\d+", tag):
        raise ValueError(f"Expected a stable vX.Y.Z tag, got {tag!r}")
    if not isinstance(manifest, dict):
        return ["Manifest is not a JSON object"]
    version = tag[1:]
    problems = []
    if manifest.get("version") != version:
        problems.append(f"Manifest carries version {manifest.get('version')!r}, expected {version!r}")
    architecture = manifest.get("architecture")
    if not isinstance(architecture, dict):
        return problems + ["Manifest has no architecture table"]
    for arch in sorted(set(architecture) - set(ARCHIVES)):
        problems.append(f"Manifest carries an unexpected {arch} architecture")
    for arch, template in ARCHIVES.items():
        resource = architecture.get(arch)
        if not isinstance(resource, dict):
            problems.append(f"Manifest has no {arch} architecture")
            continue
        name = template.format(version=version)
        url = RELEASE_URL.format(tag=tag, name=name)
        if resource.get("url") != url:
            problems.append(f"{arch} downloads {resource.get('url')!r}, expected {url!r}")
        if resource.get("bin") != ["wtr.exe"]:
            problems.append(f"{arch} installs {resource.get('bin')!r}, expected ['wtr.exe']")
        digest = sums.get(name)
        if digest is None:
            problems.append(f"checksums.txt has no {name}")
        elif resource.get("hash") != digest:
            problems.append(f"{arch} hash does not match checksums.txt for {name}")
    return problems


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tag", help="release tag, vX.Y.Z")
    parser.add_argument("manifest", type=Path, help="what-to-run.json from the bucket")
    parser.add_argument("checksums", type=Path, help="checksums.txt from the release")
    args = parser.parse_args()
    try:
        problems = check(
            args.tag,
            json.loads(args.manifest.read_text()),
            read_checksums(args.checksums),
        )
    except (OSError, ValueError) as err:
        problems = [str(err)]
    for problem in problems:
        print(problem, file=sys.stderr)
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
