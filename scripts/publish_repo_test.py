import contextlib
import importlib.util
import io
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("publisher", Path(__file__).with_name("publish-repo.py"))
publisher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publisher)


class RenderTest(unittest.TestCase):
    def setUp(self):
        self.entries = {
            "what-to-run_1.2.3_linux_amd64.deb": "1" * 64,
            "what-to-run_1.2.3_linux_arm64.deb": "2" * 64,
        }
        self.checksums = "\n".join("%s  %s" % (digest, name) for name, digest in self.entries.items())

    def test_render_contains_repository_configuration(self):
        result = publisher.render("v1.2.3", self.checksums)
        self.assertIn("Codename: stable", result)
        self.assertIn("Architectures: amd64 arm64", result)
        self.assertIn("SignWith:", result)

    def test_invalid_inputs_fail_before_publication(self):
        for tag in ("v1.2.3-rc1", "1.2.3", 'v1.2.3"'):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                publisher.render(tag, self.checksums)
        with self.assertRaises(ValueError):
            publisher.render("v1.2.3", self.checksums.replace("1" * 64, "invalid"))
        with self.assertRaises(ValueError):
            publisher.render("v1.2.3", self.checksums + "\n" + self.checksums.splitlines()[0])
        without_linux = "\n".join(line for line in self.checksums.splitlines() if "amd64" not in line)
        with self.assertRaises(KeyError):
            publisher.render("v1.2.3", without_linux)


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


class PublishTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.key = Path(self.temporary.name) / "apt_key"
        self.key.write_text("key\n")

    def _dist(self, rpm_architectures=("amd64", "arm64")):
        dist = Path(self.temporary.name) / "dist"
        dist.mkdir(exist_ok=True)
        for architecture in ("amd64", "arm64"):
            (dist / ("what-to-run_1.2.3_linux_%s.deb" % architecture)).write_text("")
        for architecture in rpm_architectures:
            (dist / ("what-to-run_1.2.3_linux_%s.rpm" % architecture)).write_text("")
        artifact = dist / "checksums.txt"
        artifact.write_text("a" * 64 + "  what-to-run_1.2.3_linux_amd64.deb\n" +
                            "b" * 64 + "  what-to-run_1.2.3_linux_arm64.deb\n")
        return dist, artifact

    def _recorder(self):
        calls = []

        def record(command, **kwargs):
            calls.append(command)
            return subprocess.CompletedProcess(command, 0, stdout="", stderr="")

        return calls, record

    def test_empty_key_skips_publication(self):
        # The release workflow exports an empty value when the secret is
        # unset; that must skip publishing instead of failing on ".".
        dist, artifact = self._dist(rpm_architectures=())
        stdout = io.StringIO()
        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", str(artifact)]), \
                patch.dict(os.environ, {"APT_GPG_KEY_FILE": ""}), \
                patch.object(publisher, "publish") as publish, \
                contextlib.redirect_stdout(stdout):
            publisher.main()
        publish.assert_not_called()
        self.assertIn("Skipping package repository publish", stdout.getvalue())

    def test_archive_invocations_do_not_publish(self):
        artifact = Path(self.temporary.name) / "archive.tar.gz"
        artifact.write_text("")
        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", str(artifact)]), \
                patch.object(publisher, "publish") as publish:
            publisher.main()
        publish.assert_not_called()

    def test_missing_key_raises(self):
        with self.assertRaises(ValueError):
            publisher.publish("v1.2.3", "/nonexistent/apt_key")

    def test_incomplete_rpm_set_raises(self):
        dist, _ = self._dist(rpm_architectures=("amd64",))
        with self.assertRaisesRegex(ValueError, "arm64"):
            publisher.publish("v1.2.3", key_file=self.key, dist_dir=dist, dry_run=True)

    def test_dry_run_writes_repository(self):
        dist, artifact = self._dist()
        calls, record = self._recorder()

        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", str(artifact),
                                "--gpg-key-file", str(self.key), "--dry-run"]), \
                patch.object(publisher.subprocess, "run", side_effect=record):
            publisher.main()
        repository = dist / "repo"
        self.assertTrue((repository / "conf" / "distributions").exists())
        self.assertTrue((repository / "rpm" / "what-to-run_1.2.3_linux_amd64.rpm").exists())
        self.assertTrue((repository / "rpm" / "what-to-run_1.2.3_linux_arm64.rpm").exists())
        self.assertTrue((repository / "setup.deb.sh").exists())
        self.assertTrue((repository / "setup.rpm.sh").exists())
        self.assertTrue(any("createrepo_c" in command for command in calls))
        self.assertTrue(any("--detach-sign" in command for command in calls))

    def test_rpm_repository_is_skipped_without_rpms(self):
        dist, _ = self._dist(rpm_architectures=())
        calls, record = self._recorder()

        with patch.object(publisher.subprocess, "run", side_effect=record):
            publisher.publish("v1.2.3", key_file=self.key, dist_dir=dist, dry_run=True)
        self.assertFalse(any("createrepo_c" in command for command in calls))
        self.assertFalse((dist / "repo" / "rpm").exists())
        self.assertTrue((dist / "repo" / "setup.deb.sh").exists())
        self.assertTrue((dist / "repo" / "setup.rpm.sh").exists())


if __name__ == "__main__":
    unittest.main()
