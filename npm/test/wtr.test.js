// Tests for the npm launcher.
//
// They run on Linux and macOS only. The fixtures are shell scripts, which is
// what the launcher spawns there. Windows is covered by the smoke test the
// release workflow runs against the published package (`npx what-to-run`).
// The shims npm generates for `.cmd` and PowerShell are npm's code, not this
// repository's.
"use strict";

const { spawn, spawnSync } = require("node:child_process");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { binaryPath, packageName } = require("../bin/wtr.js");

const launcher = path.join(__dirname, "..", "bin", "wtr.js");
const posix = { skip: process.platform === "win32" ? "needs a POSIX shell" : false };

// A global install of what-to-run, as npm lays it out. The package with the
// launcher sits at the root. The platform package with the binary sits next
// to it. Leave `binary` out to skip the optional dependency, the way
// `--omit=optional` does.
function installTree(binary, mode = 0o755) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "wtr-npm-"));
  const entry = path.join(root, "bin", "wtr.js");
  fs.mkdirSync(path.dirname(entry), { recursive: true });
  fs.copyFileSync(launcher, entry);
  const pkg = packageName();
  if (binary !== undefined && pkg !== null) {
    const directory = path.join(root, "node_modules", pkg);
    fs.mkdirSync(path.join(directory, "bin"), { recursive: true });
    fs.writeFileSync(path.join(directory, "package.json"), JSON.stringify({ name: pkg, version: "1.2.3" }));
    fs.writeFileSync(path.join(directory, "bin", "wtr"), binary, { mode });
  }
  return entry;
}

// Resolves with what the shell sees when the launcher exits: the status, the
// signal, or both.
function exitOf(proc) {
  return new Promise((resolve) => {
    proc.on("exit", (status, signal) => resolve({ status, signal }));
  });
}

// Resolves with the first line a launcher's stream prints, so a test can wait
// for the child inside it to be ready.
function firstLine(stream) {
  return new Promise((resolve, reject) => {
    let buffered = "";
    stream.on("data", (chunk) => {
      buffered += chunk;
      const end = buffered.indexOf("\n");
      if (end !== -1) {
        resolve(buffered.slice(0, end));
      }
    });
    stream.on("error", reject);
  });
}

// Resolves when no process answers for pid anymore. A signal aimed at the
// launcher must take the child with it. This is how a test sees that.
function gone(pid) {
  return new Promise((resolve, reject) => {
    const deadline = Date.now() + 5000;
    (function poll() {
      try {
        process.kill(pid, 0);
      } catch {
        resolve();
        return;
      }
      if (Date.now() > deadline) {
        reject(new Error(`pid ${pid} is still running after the launcher exited`));
        return;
      }
      setTimeout(poll, 25);
    })();
  });
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
  const entry = installTree('#!/bin/sh\nprintf "args=%s\\n" "$*"\nexit 7\n');
  const result = spawnSync(process.execPath, [entry, "one two", "three"], { encoding: "utf8" });
  assert.equal(result.status, 7);
  assert.equal(result.stdout, "args=one two three\n");
  assert.equal(result.stderr, "");
});

test("the launcher fails with a message when the platform package is missing", posix, () => {
  const entry = installTree(undefined);
  const result = spawnSync(process.execPath, [entry, "--version"], { encoding: "utf8" });
  assert.equal(result.status, 1);
  assert.match(result.stderr, new RegExp(`what-to-run: ${packageName()} is not installed`));
});

test("the launcher leaves stdin attached", posix, () => {
  const entry = installTree('#!/bin/sh\nread line\nprintf "stdin=%s\\n" "$line"\n');
  const result = spawnSync(process.execPath, [entry], { encoding: "utf8", input: "hello\n" });
  assert.equal(result.status, 0);
  assert.equal(result.stdout, "stdin=hello\n");
});

test("a binary that cannot be executed is reported, not thrown", posix, () => {
  const entry = installTree('#!/bin/sh\nexit 0\n', 0o644);
  const result = spawnSync(process.execPath, [entry], { encoding: "utf8" });
  assert.equal(result.status, 1);
  assert.match(result.stderr, /^what-to-run: cannot run /);
  assert.doesNotMatch(result.stderr, /\n\s+at /);
});

test("the launcher dies of the same signal as the task", posix, () => {
  const entry = installTree("#!/bin/sh\nkill -TERM $$\n");
  const result = spawnSync(process.execPath, [entry], { encoding: "utf8" });
  assert.equal(result.signal, "SIGTERM");
  assert.equal(result.status, null);
});

test("a signal aimed at the launcher takes the child with it, and the shell sees the signal", posix, async (t) => {
  for (const signal of ["SIGINT", "SIGTERM", "SIGUSR2"]) {
    await t.test(signal, async () => {
      const entry = installTree("#!/bin/sh\necho $$\nexec sleep 30\n");
      const proc = spawn(process.execPath, [entry], { stdio: ["ignore", "pipe", "pipe"] });
      let diagnostics = "";
      proc.stderr.setEncoding("utf8");
      proc.stderr.on("data", (chunk) => {
        diagnostics += chunk;
      });
      const child = Number.parseInt(await firstLine(proc.stdout), 10);
      assert.ok(Number.isInteger(child), `the child printed no pid: ${child}`);

      process.kill(proc.pid, signal);

      assert.deepEqual(await exitOf(proc), { status: null, signal });
      await gone(child);
      assert.equal(diagnostics, "");
    });
  }
});

test("the launcher stays alive until a child that traps the signal exits", posix, async () => {
  const marker = path.join(fs.mkdtempSync(path.join(os.tmpdir(), "wtr-npm-sig-")), "seen");
  const entry = installTree(
    "#!/bin/sh\n" +
      'trap \'touch "$WTR_TEST_SIGNAL_MARKER"; exit 5\' USR2\n' +
      "echo ready\n" +
      'i=0\nwhile [ "$i" -lt 100 ]; do sleep 0.1; i=$((i+1)); done\n',
  );
  const proc = spawn(process.execPath, [entry], {
    stdio: ["ignore", "pipe", "pipe"],
    env: { ...process.env, WTR_TEST_SIGNAL_MARKER: marker },
  });
  let diagnostics = "";
  proc.stderr.setEncoding("utf8");
  proc.stderr.on("data", (chunk) => {
    diagnostics += chunk;
  });
  // The trap is installed before the child says it is ready.
  await firstLine(proc.stdout);

  process.kill(proc.pid, "SIGUSR2");

  assert.deepEqual(await exitOf(proc), { status: 5, signal: null });
  assert.ok(fs.existsSync(marker), "the child saw the signal");
  assert.equal(diagnostics, "");
});
