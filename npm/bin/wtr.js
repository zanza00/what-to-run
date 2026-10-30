#!/usr/bin/env node
// The command npm links as `wtr`.
//
// what-to-run is written in Go and ships no JavaScript at runtime: npm can
// only link a command declared by the package it installs, so this launcher
// finds the prebuilt binary for the current platform — an optional dependency
// of the package — and runs it as a child.
//
// The child inherits stdin, stdout and stderr on purpose. The picker needs a
// real terminal (`--inline` and the shell widgets) and the exit code of the
// chosen task must reach the shell untouched, so piping anything would break
// both.
"use strict";

const { spawn, spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const NAME = "what-to-run";

// The signals the launcher forwards to the child. A supervisor, a timeout
// wrapper or a shell widget can aim one at the launcher pid alone, and the
// shared process group does not deliver such a signal to the child.
const FORWARDED_SIGNALS = ["SIGINT", "SIGTERM", "SIGUSR2"];

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
    return Promise.resolve(1);
  }

  // Windows has no POSIX signals: SIGUSR2 does not exist there, and a Ctrl-C
  // in the console reaches the child through the console itself. The launcher
  // can therefore wait for the child synchronously, and nothing needs
  // forwarding. spawnSync reports no signal on Windows, so only the exit code
  // decides what the launcher reports.
  if (process.platform === "win32") {
    const result = spawnSync(binary, argv, { stdio: "inherit" });
    if (result.error) {
      console.error(`${NAME}: cannot run ${binary}: ${result.error.message}`);
      return Promise.resolve(1);
    }
    return Promise.resolve(result.status === null ? 1 : result.status);
  }

  return new Promise((resolve) => {
    const child = spawn(binary, argv, { stdio: "inherit" });

    // One listener per signal: the listener must send the same signal it
    // caught, and the listeners alone keep the launcher alive until the child
    // has exited. Sending to an already-exited child is safe: kill() returns
    // false instead of throwing.
    const listeners = [];
    for (const signal of FORWARDED_SIGNALS) {
      const forward = () => child.kill(signal);
      process.on(signal, forward);
      listeners.push([signal, forward]);
    }

    child.on("exit", (status, signal) => {
      // The child is gone, so forwarding can reach nothing anymore. An
      // installed listener would also swallow the re-raised signal below.
      for (const [name, forward] of listeners) {
        process.removeListener(name, forward);
      }
      if (signal) {
        // Dying of the same signal keeps the shell's view of the task intact
        // (Ctrl-C, a closed terminal) instead of turning it into exit code 1.
        process.kill(process.pid, signal);
        return;
      }
      resolve(status === null ? 1 : status);
    });

    child.on("error", (error) => {
      console.error(`${NAME}: cannot run ${binary}: ${error.message}`);
      resolve(1);
    });
  });
}

if (require.main === module) {
  run(process.argv.slice(2)).then((code) => {
    // The exit code, not process.exit(), so a pending write to the terminal is
    // not cut off.
    process.exitCode = code;
  });
}

module.exports = { binaryPath, packageName };
