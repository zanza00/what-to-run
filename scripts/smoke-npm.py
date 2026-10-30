#!/usr/bin/env python3
"""Smoke test the npm channel against a registry that runs on the machine.

The gate assembles the same packages the publisher stages, publishes them to
a private Verdaccio registry, and proves the install three ways: a global npm
install runs `wtr --version` and prints the release version, pnpm installs
under `--ignore-scripts` because the packages declare no scripts, and npm with
`--omit=optional` fails with the launcher error that names the missing
platform package. A release runs it before the real publish, so a broken
channel stops the release before anything reaches npmjs.com.

Nothing reaches npmjs.com. The registry is a local process with no uplink, and
it never serves a package the gate did not publish to it. Only the Verdaccio
server itself is downloaded from npmjs.com.

With `--artifact` the gate stages the packages from real release archives, the
way the release workflow does with `dist/checksums.txt`. Without it, the gate
builds the binaries from the checkout and packs the archives itself.
"""

import argparse
import hashlib
import importlib.util
import os
from pathlib import Path
import platform
import shutil
import signal
import socket
import subprocess
import tarfile
import tempfile
import time
import urllib.request
import zipfile


publisher_spec = importlib.util.spec_from_file_location("publisher", Path(__file__).with_name("publish-npm.py"))
publisher = importlib.util.module_from_spec(publisher_spec)
publisher_spec.loader.exec_module(publisher)

# Verdaccio is pinned to an exact version, so a release cannot depend on what
# the registry happens to resolve that day.
VERDACCIO = "6.11.1"
# The local registry accepts anonymous publishes, so the npm token is a
# placeholder: npm refuses to publish without one.
TOKEN = "smoke-test"
# Every command the gate waits on gets a generous ceiling: a hang must fail
# the release, not hold it.
COMMAND_SECONDS = 300
BUILD_SECONDS = 600
REGISTRY_SECONDS = 240

ROOT = Path(__file__).resolve().parent.parent


def npm_target():
    """Return the goos and goarch of this machine, in Go's spelling."""
    system = platform.system().lower()
    if system == "windows":
        # The command npm links on Windows runs through shims (.cmd and
        # .ps1) this gate cannot run. The release workflow's verify job
        # checks the published package there.
        raise SystemExit("The npm smoke test runs on macOS and Linux only.")
    machine = platform.machine().lower()
    goarch = {"x86_64": "amd64", "aarch64": "arm64"}.get(machine, machine)
    if (system, goarch) not in publisher.TARGETS:
        raise SystemExit(f"The npm channel has no package for {system} {goarch}.")
    return system, goarch


def preflight(build):
    """Fail before any work when a tool the gate needs is missing."""
    tools = ["npm", "npx", "pnpm"] + (["go"] if build else [])
    missing = [tool for tool in tools if shutil.which(tool) is None]
    if missing:
        raise SystemExit(f"The smoke test needs {', '.join(missing)} on the PATH.")


def try_run(command, seconds=COMMAND_SECONDS, **kwargs):
    """Run a command, capture its transcript, and never treat a failure as
    the end of the gate: the caller decides which outcome it expects."""
    try:
        return subprocess.run(
            command, capture_output=True, text=True, timeout=seconds, **kwargs
        )
    except subprocess.TimeoutExpired:
        raise SystemExit(f"{' '.join(map(str, command))} did not finish in {seconds} seconds.")


def run(command, **kwargs):
    """Run a command the gate needs to succeed, and fail with its transcript."""
    result = try_run(command, **kwargs)
    if result.returncode != 0:
        raise SystemExit(
            f"{' '.join(map(str, command))} failed with exit code {result.returncode}\n"
            f"{result.stdout}{result.stderr}"
        )
    return result


def build_binaries(out, version):
    """Build one binary per release target, with the release flags."""
    binaries = {}
    for goos, goarch in publisher.TARGETS:
        name = "wtr.exe" if goos == "windows" else "wtr"
        binary = out / f"{goos}-{goarch}" / name
        binary.parent.mkdir(parents=True, exist_ok=True)
        # The same flags .goreleaser.yaml passes, so --version prints what
        # the released binary prints: the tag without the leading v.
        run(
            ["go", "build", "-ldflags", f"-s -w -X main.version={version}", "-o", str(binary), "./cmd/wtr"],
            seconds=BUILD_SECONDS,
            cwd=ROOT,
            env={**os.environ, "CGO_ENABLED": "0", "GOOS": goos, "GOARCH": goarch},
        )
        binaries[(goos, goarch)] = binary
    return binaries


def pack_archives(directory, binaries, version):
    """Write the release archives and checksums.txt, in GoReleaser's shape.

    The publisher reads the checksums and extracts the binary from each
    archive, so the asset names and the archive formats must match its
    reader. The test beside the publisher checks the same rule from its side.
    """
    directory.mkdir(parents=True, exist_ok=True)
    for (goos, goarch), binary in binaries.items():
        name = "wtr.exe" if goos == "windows" else "wtr"
        archive_name = publisher.asset_name(goos, goarch, version)
        # The files sit under a directory named like the archive, the way
        # GoReleaser packs them. The publisher finds the binary by name,
        # whatever else sits next to it.
        member = f"{publisher.NAME}_{version}_{goos}_{goarch}/{name}"
        if goos == "windows":
            with zipfile.ZipFile(directory / archive_name, "w") as archive:
                archive.write(binary, member)
        else:
            with tarfile.open(directory / archive_name, "w:gz") as archive:
                archive.add(binary, arcname=member)
    checksums = []
    for goos, goarch in publisher.TARGETS:
        name = publisher.asset_name(goos, goarch, version)
        digest = hashlib.sha256((directory / name).read_bytes()).hexdigest()
        checksums.append(f"{digest}  {name}")
    checksums_file = directory / "checksums.txt"
    checksums_file.write_text("\n".join(checksums) + "\n")
    return checksums_file


def start_registry(directory, log):
    """Start Verdaccio on a private port, and wait until it answers."""
    with socket.socket() as probe:
        probe.bind(("127.0.0.1", 0))
        port = probe.getsockname()[1]
    registry = f"http://127.0.0.1:{port}/"
    (directory / "storage").mkdir()
    (directory / "plugins").mkdir()
    # The config declares no uplink, so the registry can serve nothing the
    # gate did not publish to it. Publishing stays open to any caller,
    # because the gate owns the whole registry for the run.
    (directory / "config.yaml").write_text(
        f"storage: {directory / 'storage'}\n"
        f"plugins: {directory / 'plugins'}\n"
        "auth:\n"
        "  htpasswd:\n"
        f"    file: {directory / 'htpasswd'}\n"
        "packages:\n"
        "  '@*/*':\n"
        "    access: $all\n"
        "    publish: $anonymous\n"
        "  '**':\n"
        "    access: $all\n"
        "    publish: $anonymous\n"
    )
    # npx downloads Verdaccio from npmjs.com, so the server must not see the
    # registry configuration the clients of the gate read.
    environment = {
        key: value for key, value in os.environ.items() if not key.startswith("NPM_CONFIG")
    }
    server = subprocess.Popen(
        ["npx", "--yes", f"verdaccio@{VERDACCIO}", "--config", str(directory / "config.yaml"), "--listen", f"127.0.0.1:{port}"],
        stdout=log,
        stderr=subprocess.STDOUT,
        env=environment,
        start_new_session=True,
    )
    deadline = time.monotonic() + REGISTRY_SECONDS
    while time.monotonic() < deadline:
        if server.poll() is not None:
            break
        try:
            with urllib.request.urlopen(registry, timeout=2):
                return server, registry
        except OSError:
            time.sleep(0.5)
    if server.poll() is None:
        server.kill()
    log.flush()
    log.seek(0)
    tail = "".join(log.read().splitlines(keepends=True)[-20:])
    raise SystemExit(f"Verdaccio did not answer on {registry}:\n{tail}")


def stop_registry(server):
    """Stop the registry, and leave no process behind."""
    if server.poll() is not None:
        return
    # The server runs in its own session, so a signal to the session reaches
    # npx and Verdaccio together. The server can die between the check above
    # and the signal, and the gate must not fail on the way out.
    try:
        os.killpg(server.pid, signal.SIGTERM)
    except ProcessLookupError:
        return
    try:
        server.wait(10)
    except subprocess.TimeoutExpired:
        os.killpg(server.pid, signal.SIGKILL)
        server.wait()


def write_npmrc(path, registry):
    """Write the npm configuration every client of the gate reads."""
    # npm writes the token under a key built from the registry address, so
    # the key must match the registry the clients talk to.
    host = registry.split("//", 1)[1]
    path.write_text(
        f"registry={registry}\n"
        f"//{host}:_authToken={TOKEN}\n"
        "audit=false\n"
        "fund=false\n"
    )


def check_the_registry_is_private():
    """Fail when the registry serves a package the gate did not publish."""
    # npm exists on npmjs.com and the gate never publishes it. An answer
    # here would mean the registry proxies the public registry, and the
    # smoke test would install pieces of the real one.
    result = try_run(["npm", "view", "npm", "version"])
    if result.returncode == 0:
        raise SystemExit(
            "The local registry served npm, so it proxies npmjs.com; "
            "the smoke test refuses to install from it."
        )
    print("The registry serves no package it was not given.")


def expect_wtr_version(command, version, label):
    """Run the installed wtr, and fail when it does not print the version."""
    result = run(command)
    if result.stdout.strip() != f"wtr {version}":
        raise SystemExit(
            f"wtr --version printed {result.stdout.strip()!r}, expected 'wtr {version}':\n"
            f"{result.stdout}{result.stderr}"
        )
    print(f"{label}: wtr --version printed \"{result.stdout.strip()}\"")


def check_npm_global_install(work, version):
    """Prove that a global npm install links and runs the binary."""
    prefix = work / "npm-global"
    run(["npm", "install", "--global", "--prefix", str(prefix), f"{publisher.NAME}@{version}"])
    expect_wtr_version([str(prefix / "bin" / "wtr"), "--version"], version, "npm install --global")


def check_pnpm_ignores_scripts(work, version):
    """Prove that pnpm needs no build approval under --ignore-scripts."""
    project = work / "pnpm-project"
    project.mkdir()
    result = run(["pnpm", "add", f"{publisher.NAME}@{version}", "--ignore-scripts"], cwd=project)
    # pnpm names the packages whose scripts it skipped, and asks the user
    # to approve them. The channel declares no scripts, so the warning must
    # not appear: it would mean the packages need approval after all.
    if "Ignored build scripts" in result.stdout + result.stderr:
        raise SystemExit(
            "pnpm asked for build approval, but the npm channel declares no scripts:\n"
            f"{result.stdout}{result.stderr}"
        )
    expect_wtr_version(
        [str(project / "node_modules" / ".bin" / "wtr"), "--version"],
        version,
        "pnpm add --ignore-scripts",
    )


def check_omitting_optional(work, version, goos, goarch):
    """Prove that --omit=optional fails in the launcher, with the package named."""
    # A project install drops the optional dependencies, so it is the install
    # that reaches the launcher error. A global install keeps them on every
    # npm version the gate has seen, whatever --omit=optional says.
    project = work / "npm-omit"
    project.mkdir()
    run(["npm", "install", "--omit=optional", f"{publisher.NAME}@{version}"], cwd=project)
    result = try_run([str(project / "node_modules" / ".bin" / "wtr"), "--version"])
    expected = publisher.package_name(goos, goarch)
    if result.returncode == 0 or f"{expected} is not installed" not in result.stderr:
        raise SystemExit(
            "wtr did not fail with the launcher error after --omit=optional:\n"
            f"exit code {result.returncode}\n{result.stdout}{result.stderr}"
        )
    print(f"npm install --omit=optional: wtr fails with \"{expected} is not installed\"")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--tag",
        default="v0.0.0",
        help="the release tag the packages carry; local runs keep the default",
    )
    parser.add_argument(
        "--artifact",
        type=Path,
        help="checksums.txt of a release; the archives are read from its directory",
    )
    args = parser.parse_args()
    version = publisher.version_of(args.tag)
    goos, goarch = npm_target()
    preflight(build=args.artifact is None)

    with tempfile.TemporaryDirectory(prefix="wtr-npm-smoke.") as name:
        work = Path(name)
        if args.artifact is None:
            print(f"Building the release binaries for version {version}")
            binaries = build_binaries(work / "binaries", version)
            print("Packing the release archives")
            assets = work / "dist"
            checksums = pack_archives(assets, binaries, version)
        else:
            # The same input the real publish reads, so the gate proves the
            # packages the release would publish.
            assets = args.artifact.parent
            checksums = args.artifact
        packages = publisher.render(args.tag)
        publisher.verify_checksums(checksums.read_text(), packages)
        destination = assets / "npm"
        publisher.stage(destination, packages, assets)
        print(f"Staged {len(packages)} packages in {destination}")

        log = (work / "verdaccio.log").open("w+")
        server = None
        try:
            server, registry = start_registry(work, log)
            print(f"Verdaccio answers on {registry}")
            # Every later npm and pnpm call reads this file, so all of them
            # talk to the local registry.
            write_npmrc(work / "npmrc", registry)
            os.environ["NPM_CONFIG_USERCONFIG"] = str(work / "npmrc")
            check_the_registry_is_private()
            publisher.publish(packages, destination, provenance=False)
            check_npm_global_install(work, version)
            check_pnpm_ignores_scripts(work, version)
            check_omitting_optional(work, version, goos, goarch)
            print("The npm channel smoke test passed.")
        finally:
            if server is not None:
                stop_registry(server)
            log.close()


if __name__ == "__main__":
    main()
