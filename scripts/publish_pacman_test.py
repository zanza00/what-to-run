import contextlib
import hashlib
import importlib.util
import io
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("publisher", Path(__file__).with_name("publish-pacman.py"))
publisher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publisher)


class RenderTest(unittest.TestCase):
    def setUp(self):
        self.entries = {
            "what-to-run_1.2.3_linux_amd64.pkg.tar.zst": "1" * 64,
            "what-to-run_1.2.3_linux_arm64.pkg.tar.zst": "2" * 64,
        }
        self.checksums = "\n".join("%s  %s" % (digest, name) for name, digest in self.entries.items())

    def test_render_returns_the_arch_package_hashes(self):
        self.assertEqual(publisher.render("v1.2.3", self.checksums), self.entries)

    def test_invalid_inputs_fail_before_publication(self):
        for tag in ("v1.2.3-rc1", "1.2.3", 'v1.2.3"'):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                publisher.render(tag, self.checksums)
        with self.assertRaises(ValueError):
            publisher.render("v1.2.3", self.checksums.replace("1" * 64, "invalid"))
        with self.assertRaises(ValueError):
            publisher.render("v1.2.3", self.checksums + "\n" + self.checksums.splitlines()[0])
        without_arm = "\n".join(line for line in self.checksums.splitlines() if "arm64" not in line)
        with self.assertRaises(KeyError):
            publisher.render("v1.2.3", without_arm)


class CloneUrlTest(unittest.TestCase):
    def test_github_https_url_embeds_token(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "secret", "GH_TOKEN": ""}):
            url = publisher._clone_url("https://github.com/zanza00/what-to-run.git")
        self.assertEqual(
            url, "https://x-access-token:secret@github.com/zanza00/what-to-run.git"
        )

    def test_url_without_a_token_is_unchanged(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "", "GH_TOKEN": ""}):
            url = publisher._clone_url("https://github.com/zanza00/what-to-run.git")
            ssh = publisher._clone_url("git@github.com:zanza00/what-to-run.git")
        self.assertEqual(url, "https://github.com/zanza00/what-to-run.git")
        self.assertEqual(ssh, "git@github.com:zanza00/what-to-run.git")


class SettleDatabaseTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)

    def test_tarball_with_a_symlink_collapses_to_one_file(self):
        (self.directory / "what-to-run.db.tar.gz").write_text("database")
        os.symlink("what-to-run.db.tar.gz", self.directory / "what-to-run.db")
        publisher._settle_database(self.directory, "what-to-run.db")
        self.assertEqual((self.directory / "what-to-run.db").read_text(), "database")
        self.assertFalse((self.directory / "what-to-run.db.tar.gz").exists())
        self.assertFalse((self.directory / "what-to-run.db").is_symlink())

    def test_real_file_without_a_tarball_stays(self):
        (self.directory / "what-to-run.files").write_text("files")
        publisher._settle_database(self.directory, "what-to-run.files")
        self.assertEqual((self.directory / "what-to-run.files").read_text(), "files")
        self.assertFalse((self.directory / "what-to-run.files.tar.gz").exists())


class PublishTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.key = Path(self.temporary.name) / "apt_key"
        self.key.write_text("key\n")

    def _dist(self, architectures=("amd64", "arm64"), corrupt=False):
        dist = Path(self.temporary.name) / "dist"
        dist.mkdir(exist_ok=True)
        entries = {}
        for architecture in architectures:
            path = dist / ("what-to-run_1.2.3_linux_%s.pkg.tar.zst" % architecture)
            content = ("%s package" % architecture).encode()
            path.write_bytes(content)
            digest = hashlib.sha256(content).hexdigest()
            if corrupt:
                digest = ("f" if digest[0] != "f" else "e") + digest[1:]
            entries[path.name] = digest
        artifact = dist / "checksums.txt"
        artifact.write_text(
            "".join("%s  %s\n" % (digest, name) for name, digest in entries.items())
        )
        return dist, artifact, dict(entries)

    def _recorder(self):
        calls = []

        def record(command, **kwargs):
            calls.append(command)
            return subprocess.CompletedProcess(command, 0, stdout="", stderr="")

        return calls, record

    def test_empty_key_skips_publication(self):
        # The release workflow exports an empty value when the secret is
        # unset; that must skip publishing instead of failing on ".".
        dist, artifact, hashes = self._dist()
        stdout = io.StringIO()
        with patch.dict(os.environ, {"APT_GPG_KEY_FILE": ""}), \
                contextlib.redirect_stdout(stdout):
            publisher.publish("v1.2.3", hashes, dist_dir=dist)
        self.assertIn("Skipping pacman repository publish", stdout.getvalue())

    def test_archive_invocations_do_not_publish(self):
        artifact = Path(self.temporary.name) / "archive.tar.gz"
        artifact.write_text("")
        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", str(artifact)]), \
                patch.object(publisher, "publish") as publish:
            publisher.main()
        publish.assert_not_called()

    def test_missing_key_raises(self):
        dist, artifact, hashes = self._dist()
        with self.assertRaises(ValueError):
            publisher.publish("v1.2.3", hashes, key_file="/nonexistent/apt_key", dist_dir=dist)

    def test_missing_package_raises(self):
        dist, artifact, hashes = self._dist(architectures=("amd64",))
        with self.assertRaisesRegex(ValueError, "arm64"):
            publisher.publish("v1.2.3", hashes, key_file=self.key, dist_dir=dist, dry_run=True)

    def test_digest_mismatch_raises(self):
        dist, artifact, hashes = self._dist(corrupt=True)
        with self.assertRaisesRegex(ValueError, "sha256|SHA-256"):
            publisher.publish("v1.2.3", hashes, key_file=self.key, dist_dir=dist, dry_run=True)

    def test_dry_run_writes_the_repository(self):
        dist, artifact, hashes = self._dist()
        calls, record = self._recorder()

        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", str(artifact),
                                "--gpg-key-file", str(self.key), "--dry-run"]), \
                patch.object(publisher.subprocess, "run", side_effect=record):
            publisher.main()
        repository = dist / "repo"
        for arch, name in (("x86_64", "amd64"), ("aarch64", "arm64")):
            copied = repository / "arch" / arch / ("what-to-run_1.2.3_linux_%s.pkg.tar.zst" % name)
            self.assertTrue(copied.is_file(), copied)
        # The work directory is a temporary one, so the calls are checked by
        # shape rather than by path.
        repo_adds = [command for command in calls if command[0] == "repo-add"]
        self.assertEqual(len(repo_adds), 2)
        for command in repo_adds:
            self.assertEqual(command[1], "--quiet")
            self.assertTrue(command[2].endswith("/arch/x86_64/what-to-run.db.tar.gz") or
                            command[2].endswith("/arch/aarch64/what-to-run.db.tar.gz"), command)
        added = sorted(command[3].rsplit("_", 1)[-1].split(".")[0] for command in repo_adds)
        self.assertEqual(added, ["amd64", "arm64"])
        self.assertEqual(sum("--detach-sign" in " ".join(command) for command in calls), 4)

    def test_dry_run_without_a_key_skips_signing(self):
        dist, artifact, hashes = self._dist()
        calls, record = self._recorder()

        with patch.object(publisher.subprocess, "run", side_effect=record):
            publisher.publish("v1.2.3", hashes, dist_dir=dist, dry_run=True)
        self.assertFalse(any("--detach-sign" in " ".join(command) for command in calls))
        self.assertTrue((dist / "repo" / "arch" / "x86_64").is_dir())


if __name__ == "__main__":
    unittest.main()
