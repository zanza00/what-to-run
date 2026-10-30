#!/usr/bin/env node
// The command npm links as `wtr`.
//
// what-to-run is written in Go and ships no JavaScript at runtime: npm can
// only link a command declared by the package it installs, so this launcher
// finds the prebuilt binary for the current platform — an optional dependency
// of the package — and replaces itself with it.
//
// The child inherits stdin, stdout and stderr on purpose. The picker needs a
// real terminal (`--inline` and the shell widgets) and the exit code of the
// chosen task must reach the shell untouched, so piping anything would break
// both.
"use strict";

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const NAME = "what-to-run";

// Mirrors the goos/goarch matrix in .goreleaser.yaml: the binary is built
// with CGO_ENABLED=0, so it is static and one package per OS and architecture
// is enough (no glibc/musl split).
const SUFFIXES = {
  "linux-x64": "linux-x64",
  "linux-arm64": "linux-arm64",
  "darwin-x64": "darwin-x64",
  "darwin-arm64": "darwin-arm64",
  "win32-x64": "windows-x64",
  "win32-arm64": "windows-arm64",
};

function key(platform, arch) {
  return `${platform}-${arch}`;
}

// The optional dependency that carries the binary, or null when the release
// has no build for this platform.
function packageName(platform = process.platform, arch = process.arch) {
  const suffix = SUFFIXES[key(platform, arch)];
  return suffix ? `${NAME}-${suffix}` : null;
}

// The absolute path of the binary inside that package.
function binaryPath(platform = process.platform, arch = process.arch, resolve = (request) => require.resolve(request)) {
  const pkg = packageName(platform, arch);
  if (!pkg) {
    throw new Error(
      `no prebuilt binary for ${key(platform, arch)}; ` +
        `this release covers ${Object.keys(SUFFIXES).join(", ")}`,
    );
  }

  let manifest;
  try {
    manifest = resolve(`${pkg}/package.json`);
  } catch {
    // npm drops an optional dependency it cannot install, and pnpm asks
    // before running the launcher it links, so name the flag that removes it.
    throw new Error(
      `${pkg} is not installed; reinstall it without omitting optional dependencies: ` +
        `npm install -g ${NAME}`,
    );
  }

  const binary = path.join(
    path.dirname(manifest),
    "bin",
    platform === "win32" ? "wtr.exe" : "wtr",
  );
  if (!fs.existsSync(binary)) {
    throw new Error(`${pkg} does not contain ${path.basename(binary)}; reinstall ${NAME}`);
  }
  return binary;
}

function run(argv) {
  let binary;
  try {
    binary = binaryPath();
  } catch (error) {
    console.error(`${NAME}: ${error.message}`);
    return 1;
  }

  const result = spawnSync(binary, argv, { stdio: "inherit" });
  if (result.error) {
    console.error(`${NAME}: cannot run ${binary}: ${result.error.message}`);
    return 1;
  }
  if (result.signal) {
    // Dying of the same signal keeps the shell's view of the task intact
    // (Ctrl-C, a closed terminal) instead of turning it into exit code 1.
    // Windows has no POSIX signals, so spawnSync reports none there and the
    // exit-code path below is what runs.
    process.kill(process.pid, result.signal);
    return 1;
  }
  return result.status === null ? 1 : result.status;
}

if (require.main === module) {
  // The exit code, not process.exit(), so a pending write to the terminal is
  // not cut off.
  process.exitCode = run(process.argv.slice(2));
}

module.exports = { binaryPath, packageName };
