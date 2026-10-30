#!/usr/bin/env python3
"""Release publisher for the npm packages.

npm installs programs written in any language: a package only has to declare a
`bin` entry, and npm links it into the global bin directory. what-to-run
publishes one small launcher (`npm/bin/wtr.js`) plus one package per platform
with the Go binary, wired through optional dependencies. npm keeps the package
that matches the machine and drops the others, so the install runs no script
at all: `--ignore-scripts` (and pnpm's build allowlist) cannot break it, and
`npm --omit=optional` fails in the launcher with a clear message.

Publishing is a step of its own in the release workflow rather than a
GoReleaser publisher: GoReleaser passes a publisher only the environment
variables listed in `.goreleaser.yaml`, and trusted publishing needs the
runner's OIDC token (`ACTIONS_ID_TOKEN_REQUEST_*`) in the environment. The
workflow keeps the channel opt-in with the `NPM_PUBLISH` repository variable,
until the packages exist on the registry.
"""

import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import zipfile


NAME = "what-to-run"
# The six goos/goarch pairs .goreleaser.yaml builds for.
TARGETS = (
    ("linux", "amd64"),
    ("linux", "arm64"),
    ("darwin", "amd64"),
    ("darwin", "arm64"),
    ("windows", "amd64"),
    ("windows", "arm64"),
)
# npm's spelling of the same platforms, for the `os` and `cpu` fields npm uses
# to decide which optional dependency belongs on the machine.
NPM_OS = {"linux": "linux", "darwin": "darwin", "windows": "win32"}
NPM_CPU = {"amd64": "x64", "arm64": "arm64"}

ROOT = Path(__file__).resolve().parent.parent
TEMPLATE = ROOT / "npm" / "package.json"
LAUNCHER = ROOT / "npm" / "bin" / "wtr.js"
LICENSE = ROOT / "LICENSE"


def version_of(tag):
    match = re.fullmatch(r"v(\d+\.\d+\.\d+)", tag)
    if match is None:
        raise ValueError("Expected a stable vX.Y.Z tag")
    return match.group(1)


def package_name(goos, goarch):
    return f"{NAME}-{goos}-{NPM_CPU[goarch]}"


# The reverse lookup `stage` needs, built once instead of scanned per package.
BY_PACKAGE = {package_name(goos, goarch): (goos, goarch) for goos, goarch in TARGETS}


def asset_name(goos, goarch, version):
    extension = "zip" if goos == "windows" else "tar.gz"
    return f"{NAME}_{version}_{goos}_{goarch}.{extension}"


def verify_checksums(checksums, version):
    """Fail before staging when a release asset this publisher needs is absent."""
    available = set()
    for line in checksums.splitlines():
        digest, name = line.split()
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise ValueError(f"Invalid SHA-256 for {name}")
        if name in available:
            raise ValueError(f"Duplicate checksum for {name}")
        available.add(name)
    for goos, goarch in TARGETS:
        name = asset_name(goos, goarch, version)
        if name not in available:
            raise ValueError(f"Missing release asset: {name}")


def render(tag):
    """Return every package to publish, the platform ones before the root one.

    The root manifest comes from `npm/package.json`, which is marked private so
    it cannot be published by accident; only the version and the optional
    dependencies are filled in here, so the metadata has one home.
    """
    version = version_of(tag)
    template = json.loads(TEMPLATE.read_text())
    packages = {}
    for goos, goarch in TARGETS:
        binary = "wtr.exe" if goos == "windows" else "wtr"
        packages[package_name(goos, goarch)] = {
            "name": package_name(goos, goarch),
            "version": version,
            "description": f"The {NAME} binary for {goos} {goarch}",
            "homepage": template["homepage"],
            "repository": template["repository"],
            "license": template["license"],
            "os": [NPM_OS[goos]],
            "cpu": [NPM_CPU[goarch]],
            "files": [f"bin/{binary}"],
            # Yarn PnP must unpack the binary, which the launcher spawns by
            # path; npm and pnpm ignore the field.
            "preferUnplugged": True,
        }
    root = {key: value for key, value in template.items() if key != "private"}
    root["version"] = version
    root["optionalDependencies"] = {
        package_name(goos, goarch): version for goos, goarch in TARGETS
    }
    packages[NAME] = root
    return packages


def extract_binary(archive, binary):
    """Return the binary stored in a release archive, whatever else it holds."""
    if archive.name.endswith(".zip"):
        with zipfile.ZipFile(archive) as bundle:
            for info in bundle.infolist():
                if not info.is_dir() and Path(info.filename).name == binary:
                    return bundle.read(info)
    else:
        with tarfile.open(archive, "r:gz") as bundle:
            for member in bundle.getmembers():
                if member.isfile() and Path(member.name).name == binary:
                    return bundle.extractfile(member).read()
    raise ValueError(f"{archive} does not contain {binary}")


def stage(directory, packages, assets):
    """Write one publishable directory per package under `directory`."""
    staged = []
    for name, manifest in packages.items():
        package = directory / name
        (package / "bin").mkdir(parents=True, exist_ok=True)
        (package / "package.json").write_text(json.dumps(manifest, indent=2) + "\n")
        shutil.copy(LICENSE, package / "LICENSE")
        if name == NAME:
            shutil.copy(LAUNCHER, package / "bin" / "wtr.js")
            # npm renders the package README on the registry, and always
            # includes it regardless of `files`.
            shutil.copy(ROOT / "README.md", package / "README.md")
        else:
            goos, goarch = BY_PACKAGE[name]
            binary = "wtr.exe" if goos == "windows" else "wtr"
            content = extract_binary(assets / asset_name(goos, goarch, manifest["version"]), binary)
            path = package / "bin" / binary
            path.write_bytes(content)
            path.chmod(0o755)
        staged.append(package)
    return staged


def published(name, version):
    """True when that exact version already exists on the registry."""
    result = subprocess.run(
        ["npm", "view", f"{name}@{version}", "version"],
        capture_output=True, text=True,
    )
    return result.returncode == 0 and result.stdout.strip() == version


def publish(packages, directory, provenance):
    for name, manifest in packages.items():
        version = manifest["version"]
        if published(name, version):
            print(f"{name} {version} is already on the registry")
            continue
        command = ["npm", "publish", str(directory / name), "--access", "public"]
        if provenance:
            # The runner's OIDC token also proves the tarball was built here.
            command.append("--provenance")
        result = subprocess.run(command)
        if result.returncode != 0:
            raise SystemExit(
                f"npm publish {name}@{version} failed with exit code {result.returncode}"
            )
        print(f"Published {name} {version}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument(
        "--artifact",
        type=Path,
        required=True,
        help="checksums.txt of the release; the archives are read from its directory",
    )
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    version = version_of(args.tag)
    verify_checksums(args.artifact.read_text(), version)
    packages = render(args.tag)
    destination = args.artifact.parent / "npm"
    stage(destination, packages, args.artifact.parent)
    if args.dry_run:
        print(destination)
    else:
        publish(
            packages, destination, provenance=os.environ.get("GITHUB_ACTIONS") == "true"
        )


if __name__ == "__main__":
    main()
