import contextlib
import importlib.util
import io
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("publisher", Path(__file__).with_name("publish-apt.py"))
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


class PublishTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.key = Path(self.temporary.name) / "apt_key"
        self.key.write_text("key\n")

    def test_empty_key_skips_publication(self):
        # The release workflow exports an empty value when the secret is
        # unset; that must skip publishing instead of failing on ".".
        artifact = Path(self.temporary.name) / "checksums.txt"
        artifact.write_text("a" * 64 + "  what-to-run_1.2.3_linux_amd64.deb\n")
        stdout = io.StringIO()
        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", str(artifact)]), \
                patch.dict(os.environ, {"APT_GPG_KEY_FILE": ""}), \
                patch.object(publisher, "publish") as publish, \
                contextlib.redirect_stdout(stdout):
            publisher.main()
        publish.assert_not_called()
        self.assertIn("Skipping apt publish", stdout.getvalue())

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

    def test_dry_run_writes_repository(self):
        dist = Path(self.temporary.name) / "dist"
        dist.mkdir()
        (dist / "what-to-run_1.2.3_linux_amd64.deb").write_text("")
        (dist / "what-to-run_1.2.3_linux_arm64.deb").write_text("")
        artifact = dist / "checksums.txt"
        artifact.write_text("a" * 64 + "  what-to-run_1.2.3_linux_amd64.deb\n" + "b" * 64 + "  what-to-run_1.2.3_linux_arm64.deb\n")
        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", str(artifact),
                                "--gpg-key-file", str(self.key), "--dry-run"]):
            with patch.object(publisher.subprocess, "run", return_value=subprocess.CompletedProcess([], 0)):
                publisher.main()
        self.assertTrue((dist / "apt").exists())


if __name__ == "__main__":
    unittest.main()
