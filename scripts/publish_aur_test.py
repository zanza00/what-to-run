import contextlib
import importlib.util
import io
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location("publisher", Path(__file__).with_name("publish-aur.py"))
publisher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publisher)


class RenderTest(unittest.TestCase):
    def setUp(self):
        self.entries = {
            f"what-to-run_1.2.3_{system}_{arch}.tar.gz": str(index) * 64
            for index, (system, arch) in enumerate(
                (("darwin", "arm64"), ("darwin", "amd64"), ("linux", "arm64"), ("linux", "amd64")), 1
            )
        }
        self.checksums = "\n".join(f"{digest}  {name}" for name, digest in self.entries.items())

    def test_package_covers_both_architectures_with_matching_checksums(self):
        pkgbuild, srcinfo = publisher.render("v1.2.3", self.checksums)
        self.assertIn("pkgname=what-to-run-bin", pkgbuild)
        self.assertIn("pkgver=1.2.3", pkgbuild)
        self.assertIn("pkgrel=1", pkgbuild)
        self.assertIn("provides=('what-to-run')", pkgbuild)
        self.assertIn("conflicts=('what-to-run')", pkgbuild)
        self.assertIn('install -Dm755 "${srcdir}/wtr" "${pkgdir}/usr/bin/wtr"', pkgbuild)
        for arch, go_arch in (("x86_64", "amd64"), ("aarch64", "arm64")):
            name = f"what-to-run_1.2.3_linux_{go_arch}.tar.gz"
            url = f"https://github.com/zanza00/what-to-run/releases/download/v1.2.3/{name}"
            self.assertIn(f"source_{arch}=('{url}')", pkgbuild)
            self.assertIn(f"sha256sums_{arch}=('{self.entries[name]}')", pkgbuild)
            self.assertIn(f"\tsource_{arch} = {url}", srcinfo)
            self.assertIn(f"\tsha256sums_{arch} = {self.entries[name]}", srcinfo)
        self.assertIn("pkgbase = what-to-run-bin", srcinfo)
        self.assertIn("\tpkgver = 1.2.3", srcinfo)
        self.assertIn("\tarch = x86_64", srcinfo)
        self.assertIn("\tarch = aarch64", srcinfo)
        self.assertIn("\n\npkgname = what-to-run-bin\n", srcinfo)

    def test_invalid_inputs_fail_before_publication(self):
        for tag in ("v1.2.3-rc1", "1.2.3", 'v1.2.3"'):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                publisher.render(tag, self.checksums)
        with self.assertRaises(ValueError):
            publisher.render("v1.2.3", self.checksums.replace("1" * 64, "invalid"))
        with self.assertRaises(ValueError):
            publisher.render("v1.2.3", self.checksums + "\n" + self.checksums.splitlines()[0])
        without_linux = "\n".join(line for line in self.checksums.splitlines() if "linux" not in line)
        with self.assertRaises(KeyError):
            publisher.render("v1.2.3", without_linux)


class PublishTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.key = Path(self.temporary.name) / "aur_key"
        self.key.write_text("key\n")
        self.pkgbuild = "pkgbuild\n"
        self.srcinfo = "srcinfo\n"

    def test_publish_clones_commits_and_pushes(self):
        calls = []
        captured = {}

        def fake_run(command, **environment):
            calls.append(command)
            if "status" in command:
                repository = Path(command[command.index("-C") + 1])
                captured["PKGBUILD"] = (repository / "PKGBUILD").read_text()
                captured[".SRCINFO"] = (repository / ".SRCINFO").read_text()
                captured["environment"] = environment["env"]
                return subprocess.CompletedProcess(command, 0, stdout=" M PKGBUILD\n")
            return subprocess.CompletedProcess(command, 0, stdout="")

        with patch.object(publisher.subprocess, "run", side_effect=fake_run):
            publisher.publish("v1.2.3", self.pkgbuild, self.srcinfo, self.key)

        self.assertEqual(captured["PKGBUILD"], self.pkgbuild)
        self.assertEqual(captured[".SRCINFO"], self.srcinfo)
        self.assertIn(f"ssh -i {self.key}", captured["environment"]["GIT_SSH_COMMAND"])
        self.assertIn("IdentitiesOnly=yes", captured["environment"]["GIT_SSH_COMMAND"])
        self.assertEqual(calls[0][:3], ["git", "clone", "--quiet"])
        self.assertEqual(calls[0][3], publisher.AUR_URL)
        self.assertEqual(calls[2][3:], ["add", "PKGBUILD", ".SRCINFO"])
        self.assertIn("commit", calls[3])
        self.assertIn("Update to v1.2.3", calls[3])
        self.assertIn(f"user.name={publisher.MAINTAINER_NAME}", calls[3])
        self.assertIn(f"user.email={publisher.MAINTAINER_EMAIL}", calls[3])
        self.assertEqual(calls[4][3:], ["push", "--quiet", "origin", "master"])

    def test_unchanged_package_is_not_pushed(self):
        commands = []

        def fake_run(command, **environment):
            commands.append(command)
            return subprocess.CompletedProcess(command, 0, stdout="")

        with patch.object(publisher.subprocess, "run", side_effect=fake_run):
            publisher.publish("v1.2.3", self.pkgbuild, self.srcinfo, self.key)
        self.assertEqual(len(commands), 2)

    def test_missing_key_stops_publication(self):
        with patch.object(publisher.subprocess, "run") as run, self.assertRaises(ValueError):
            publisher.publish("v1.2.3", self.pkgbuild, self.srcinfo, "/nonexistent/aur_key")
        run.assert_not_called()

    def test_archive_invocations_do_not_publish(self):
        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", "missing-archive.tar.gz"]), \
                patch.object(publisher, "publish") as publish:
            publisher.main()
        publish.assert_not_called()

    def checksums(self):
        artifact = Path(self.temporary.name) / "checksums.txt"
        artifact.write_text("\n".join(
            f"{str(index) * 64}  what-to-run_1.2.3_linux_{arch}.tar.gz"
            for index, arch in ((1, "amd64"), (2, "arm64"))
        ))
        return artifact

    def test_dry_run_writes_package_files(self):
        artifact = self.checksums()
        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", str(artifact), "--dry-run"]):
            publisher.main()
        destination = artifact.parent / "aur" / "what-to-run-bin"
        self.assertIn("pkgver=1.2.3", (destination / "PKGBUILD").read_text())
        self.assertIn("pkgbase = what-to-run-bin", (destination / ".SRCINFO").read_text())

    def test_unconfigured_key_skips_publication(self):
        artifact = self.checksums()
        stdout = io.StringIO()
        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", str(artifact)]), \
                patch.dict(os.environ, {"AUR_SSH_KEY_FILE": ""}), \
                patch.object(publisher, "publish") as publish, \
                contextlib.redirect_stdout(stdout):
            publisher.main()
        publish.assert_not_called()
        self.assertIn("Skipping AUR publish", stdout.getvalue())

    def test_explicit_missing_key_file_is_an_error(self):
        artifact = self.checksums()
        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", str(artifact),
                                "--ssh-key-file", "/nonexistent/aur_key"]), \
                patch.dict(os.environ, {"AUR_SSH_KEY_FILE": ""}), \
                self.assertRaises(ValueError):
            publisher.main()


if __name__ == "__main__":
    unittest.main()
