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
import dataclasses
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


def binary_name(goos):
    return "wtr.exe" if goos == "windows" else "wtr"


def asset_name(goos, goarch, version):
    extension = "zip" if goos == "windows" else "tar.gz"
    return f"{NAME}_{version}_{goos}_{goarch}.{extension}"


@dataclasses.dataclass(frozen=True)
class Package:
    """One npm package of a release.

    The helpers below take this concept instead of a (goos, goarch) pair
    for the platform and a (name, version) pair for the registry. The
    root package is the one npm links as `wtr`: it has no platform, and
    it ships the launcher instead of a binary.
    """

    name: str
    version: str
    manifest: dict
    goos: str = ""
    goarch: str = ""

    @classmethod
    def for_platform(cls, goos, goarch, version, template):
        """The optional dependency that ships the binary of one target."""
        name = package_name(goos, goarch)
        binary = binary_name(goos)
        return cls(
            name,
            version,
            {
                "name": name,
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
            },
            goos,
            goarch,
        )

    @classmethod
    def root(cls, version, template):
        """The package npm links as `wtr`.

        The manifest comes from `npm/package.json`, which is marked
        private so it cannot be published by accident; only the version
        and the optional dependencies are filled in here, so the
        metadata has one home.
        """
        manifest = {key: value for key, value in template.items() if key != "private"}
        manifest["version"] = version
        manifest["optionalDependencies"] = {
            package_name(goos, goarch): version for goos, goarch in TARGETS
        }
        return cls(NAME, version, manifest)

    @property
    def is_root(self):
        return not self.goarch

    @property
    def binary(self):
        """The Go binary file the package ships, or None for the root."""
        return None if self.is_root else binary_name(self.goos)

    @property
    def asset_name(self):
        """The release archive the binary is extracted from."""
        return asset_name(self.goos, self.goarch, self.version)


def verify_checksums(checksums, packages):
    """Fail before staging when a release asset this publisher needs is absent."""
    available = set()
    for line in checksums.splitlines():
        digest, name = line.split()
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise ValueError(f"Invalid SHA-256 for {name}")
        if name in available:
            raise ValueError(f"Duplicate checksum for {name}")
        available.add(name)
    for package in packages.values():
        if not package.is_root and package.asset_name not in available:
            raise ValueError(f"Missing release asset: {package.asset_name}")


def render(tag):
    """Return every package to publish, the platform ones before the root one."""
    version = version_of(tag)
    template = json.loads(TEMPLATE.read_text())
    packages = {}
    for goos, goarch in TARGETS:
        package = Package.for_platform(goos, goarch, version, template)
        packages[package.name] = package
    package = Package.root(version, template)
    packages[package.name] = package
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
    for package in packages.values():
        target = directory / package.name
        (target / "bin").mkdir(parents=True, exist_ok=True)
        (target / "package.json").write_text(json.dumps(package.manifest, indent=2) + "\n")
        shutil.copy(LICENSE, target / "LICENSE")
        if package.is_root:
            shutil.copy(LAUNCHER, target / "bin" / "wtr.js")
            # npm renders the package README on the registry, and always
            # includes it regardless of `files`.
            shutil.copy(ROOT / "README.md", target / "README.md")
        else:
            content = extract_binary(assets / package.asset_name, package.binary)
            path = target / "bin" / package.binary
            path.write_bytes(content)
            path.chmod(0o755)
        staged.append(target)
    return staged


def published(package):
    """True when that exact version already exists on the registry."""
    result = subprocess.run(
        ["npm", "view", f"{package.name}@{package.version}", "version"],
        capture_output=True, text=True,
    )
    return result.returncode == 0 and result.stdout.strip() == package.version


def publish(packages, directory, provenance):
    for package in packages.values():
        if published(package):
            print(f"{package.name} {package.version} is already on the registry")
            continue
        command = ["npm", "publish", str(directory / package.name), "--access", "public"]
        if provenance:
            # The runner's OIDC token also proves the tarball was built here.
            command.append("--provenance")
        result = subprocess.run(command)
        if result.returncode != 0:
            raise SystemExit(
                f"npm publish {package.name}@{package.version} failed with exit code {result.returncode}"
            )
        print(f"Published {package.name} {package.version}")


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
    packages = render(args.tag)
    verify_checksums(args.artifact.read_text(), packages)
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
