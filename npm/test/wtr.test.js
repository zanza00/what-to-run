// Tests for the npm launcher.
//
// They run on Linux and macOS only: the fixtures are shell scripts, which is
// what the launcher spawns there. Windows is covered by the smoke test the
// release workflow runs against the published package (`npx what-to-run`),
// because the shims npm generates for `.cmd` and PowerShell are npm's code,
// not this repository's.
"use strict";

const { spawnSync } = require("node:child_process");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { binaryPath, packageName } = require("../bin/wtr.js");

const launcher = path.join(__dirname, "..", "bin", "wtr.js");
const posix = { skip: process.platform === "win32" ? "needs a POSIX shell" : false };

// A global install of what-to-run as npm lays it out: the package with the
// launcher, and next to it the platform package with the binary. `binary`
// omitted leaves the optional dependency out, the way `--omit=optional` does.
function stage(binary) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "wtr-npm-"));
  const entry = path.join(root, "bin", "wtr.js");
  fs.mkdirSync(path.dirname(entry), { recursive: true });
  fs.copyFileSync(launcher, entry);
  const pkg = packageName();
  if (binary !== undefined && pkg !== null) {
    const directory = path.join(root, "node_modules", pkg);
    fs.mkdirSync(path.join(directory, "bin"), { recursive: true });
    fs.writeFileSync(path.join(directory, "package.json"), JSON.stringify({ name: pkg, version: "1.2.3" }));
    fs.writeFileSync(path.join(directory, "bin", "wtr"), binary, { mode: 0o755 });
  }
  return entry;
}

test("packageName covers every released platform", () => {
  assert.equal(packageName("linux", "x64"), "what-to-run-linux-x64");
  assert.equal(packageName("linux", "arm64"), "what-to-run-linux-arm64");
  assert.equal(packageName("darwin", "x64"), "what-to-run-darwin-x64");
  assert.equal(packageName("darwin", "arm64"), "what-to-run-darwin-arm64");
  assert.equal(packageName("win32", "x64"), "what-to-run-windows-x64");
  assert.equal(packageName("win32", "arm64"), "what-to-run-windows-arm64");
});

test("packageName has no answer for a platform without a build", () => {
  for (const [platform, arch] of [["freebsd", "x64"], ["linux", "ia32"], ["sunos", "x64"]]) {
    assert.equal(packageName(platform, arch), null, `${platform}-${arch}`);
  }
});

test("the error names the platform that has no build", () => {
  assert.throws(() => binaryPath("freebsd", "x64"), /no prebuilt binary for freebsd-x64/);
});

test("a missing optional dependency is explained", () => {
  const resolve = () => {
    throw new Error("MODULE_NOT_FOUND");
  };
  assert.throws(
    () => binaryPath("linux", "x64", resolve),
    /what-to-run-linux-x64 is not installed.*npm install -g what-to-run/,
  );
});

test("binaryPath points at the binary inside the platform package", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "wtr-npm-pkg-"));
  fs.mkdirSync(path.join(root, "bin"));
  fs.writeFileSync(path.join(root, "bin", "wtr"), "");
  assert.equal(
    binaryPath("linux", "x64", () => path.join(root, "package.json")),
    path.join(root, "bin", "wtr"),
  );
});

test("an empty platform package is not silently accepted", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "wtr-npm-empty-"));
  assert.throws(
    () => binaryPath("linux", "x64", () => path.join(root, "package.json")),
    /does not contain wtr; reinstall what-to-run/,
  );
});

test("the launcher forwards arguments, output and the exit code", posix, () => {
  const entry = stage('#!/bin/sh\nprintf "args=%s\\n" "$*"\nexit 7\n');
  const result = spawnSync(process.execPath, [entry, "one two", "three"], { encoding: "utf8" });
  assert.equal(result.status, 7);
  assert.equal(result.stdout, "args=one two three\n");
  assert.equal(result.stderr, "");
});

test("the launcher fails with a message when the platform package is missing", posix, () => {
  const entry = stage(undefined);
  const result = spawnSync(process.execPath, [entry, "--version"], { encoding: "utf8" });
  assert.equal(result.status, 1);
  assert.match(result.stderr, new RegExp(`what-to-run: ${packageName()} is not installed`));
});
