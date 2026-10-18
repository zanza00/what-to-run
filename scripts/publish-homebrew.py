#!/usr/bin/env python3
"""GoReleaser publisher for a macOS/Linux Homebrew formula."""

import argparse
import base64
import json
from pathlib import Path
import re
import subprocess


def render(tag, checksums):
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
    lines = [
        "class WhatToRun < Formula",
        '  desc "Task picker for the commands you can never remember in a repository"',
        '  homepage "https://github.com/zanza00/what-to-run"',
        f'  version "{version}"',
        '  license "MIT"',
    ]
    for platform, os_name in (("macos", "darwin"), ("linux", "linux")):
        lines.extend(["", f"  on_{platform} do"])
        for arch, go_arch in (("arm", "arm64"), ("intel", "amd64")):
            name = f"what-to-run_{version}_{os_name}_{go_arch}.tar.gz"
            url = f"https://github.com/zanza00/what-to-run/releases/download/{tag}/{name}"
            lines.extend([
                f"    on_{arch} do",
                f'      url "{url}"',
                f'      sha256 "{hashes[name]}"',
                "    end",
            ])
        lines.append("  end")
    lines.extend([
        "",
        "  def install",
        '    bin.install "wtr"',
        '    bin.install "what-to-run"',
        "  end",
        "",
        "  test do",
        '    assert_equal version.to_s, shell_output("#{bin}/wtr --version").split.last',
        '    assert_match "classic", shell_output("#{bin}/what-to-run --list-themes")',
        "  end",
        "end",
        "",
    ])
    return "\n".join(lines)


def fetch_existing(endpoint):
    """Return the current formula file, or None when the tap has no formula."""
    try:
        result = subprocess.check_output(
            ["gh", "api", endpoint, "-f", "ref=main", "--method", "GET"],
            text=True, stderr=subprocess.PIPE,
        )
    except subprocess.CalledProcessError as err:
        if "HTTP 404" in (err.stderr or ""):
            return None
        raise
    return json.loads(result)


def publish(tag, formula):
    endpoint = "repos/zanza00/homebrew-tap/contents/Formula/what-to-run.rb"
    existing = fetch_existing(endpoint)
    if existing is not None and base64.b64decode(existing["content"]).decode() == formula:
        print(f"Homebrew formula already matches {tag}")
        return
    payload = {
        "branch": "main",
        "message": f"chore: update what-to-run to {tag}",
        "content": base64.b64encode(formula.encode()).decode(),
    }
    # The Contents API updates an existing file with its blob sha; the first
    # release of a package creates the formula with a plain PUT instead.
    if existing is not None:
        payload["sha"] = existing["sha"]
    subprocess.run(
        ["gh", "api", endpoint, "--method", "PUT", "--input", "-", "--silent"],
        input=json.dumps(payload), text=True, check=True,
    )
    print(f"Published Homebrew formula for {tag}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--artifact", type=Path, required=True)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    # GoReleaser invokes publishers for each archive and the checksum artifact.
    if args.artifact.name != "checksums.txt":
        return
    formula = render(args.tag, args.artifact.read_text())
    destination = args.artifact.parent / "homebrew" / "Formula" / "what-to-run.rb"
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text(formula)
    if args.dry_run:
        print(destination)
    else:
        publish(args.tag, formula)


if __name__ == "__main__":
    main()
