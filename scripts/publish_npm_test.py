import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest
from unittest.mock import patch
import zipfile


spec = importlib.util.spec_from_file_location("publisher", Path(__file__).with_name("publish-npm.py"))
publisher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publisher)


def tarball(path, members):
    with tarfile.open(path, "w:gz") as archive:
        for name, content in members.items():
            info = tarfile.TarInfo(name)
            info.size = len(content)
            info.mode = 0o755
            archive.addfile(info, io.BytesIO(content))


def page():
    return '#!/bin/sh\necho "wtr $@"; exit 0\n'.encode()


class PublisherTest(unittest.TestCase):
    def setUp(self):
        self.directory = Path(tempfile.mkdtemp())
        for goos, goarch in publisher.TARGETS:
            name = publisher.asset_name(goos, goarch, "1.2.3")
            path = self.directory / name
            if goos == "windows":
                with zipfile.ZipFile(path, "w") as archive:
                    archive.writestr("wtr.exe", page())
            else:
                # A release archive also carries the docs; the publisher has to
                # find the binary among the other members.
                tarball(path, {"LICENSE": b"MIT\n", "wtr": page()})
        self.checksums = "\n".join(
            f"{index:064x}  {publisher.asset_name(goos, goarch, '1.2.3')}"
            for index, (goos, goarch) in enumerate(publisher.TARGETS, 1)
        )

    def test_render_publishes_one_package_per_platform_plus_the_root(self):
        packages = publisher.render("v1.2.3")
        self.assertEqual(list(packages)[-1], publisher.NAME)
        self.assertEqual(len(packages), len(publisher.TARGETS) + 1)
        root = packages[publisher.NAME]
        self.assertEqual(root["version"], "1.2.3")
        self.assertNotIn("private", root)
        self.assertEqual(root["bin"], {"wtr": "bin/wtr.js"})
        self.assertEqual(
            root["optionalDependencies"],
            {
                "what-to-run-linux-x64": "1.2.3",
                "what-to-run-linux-arm64": "1.2.3",
                "what-to-run-darwin-x64": "1.2.3",
                "what-to-run-darwin-arm64": "1.2.3",
                "what-to-run-windows-x64": "1.2.3",
                "what-to-run-windows-arm64": "1.2.3",
            },
        )
        for name, manifest in packages.items():
            if name == publisher.NAME:
                continue
            self.assertEqual(manifest["version"], "1.2.3")
            self.assertNotIn("bin", manifest)
            self.assertNotIn("private", manifest)
            self.assertEqual(len(manifest["os"]), 1)
            self.assertEqual(len(manifest["cpu"]), 1)
        self.assertEqual(packages["what-to-run-windows-arm64"]["os"], ["win32"])
        self.assertEqual(packages["what-to-run-windows-arm64"]["cpu"], ["arm64"])
        self.assertEqual(packages["what-to-run-darwin-x64"]["cpu"], ["x64"])
        self.assertEqual(
            packages["what-to-run-windows-arm64"]["files"], ["bin/wtr.exe"]
        )

    def test_versions_are_pinned_to_the_release(self):
        root = publisher.render("v9.8.7")[publisher.NAME]
        self.assertEqual(root["version"], "9.8.7")
        self.assertEqual(set(root["optionalDependencies"].values()), {"9.8.7"})

    def test_invalid_tags_fail_before_publication(self):
        for tag in ("v1.2.3-rc1", "1.2.3", "v1.2", 'v1.2.3"'):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                publisher.render(tag)

    def test_checksums_must_describe_every_platform(self):
        publisher.verify_checksums(self.checksums, "1.2.3")
        with self.assertRaises(ValueError):
            publisher.verify_checksums(self.checksums.replace(f"{1:064x}", "invalid"), "1.2.3")
        with self.assertRaises(ValueError):
            publisher.verify_checksums(
                self.checksums + "\n" + self.checksums.splitlines()[0], "1.2.3"
            )
        with self.assertRaises(ValueError):
            publisher.verify_checksums("\n".join(self.checksums.splitlines()[:-1]), "1.2.3")
        with self.assertRaises(ValueError):
            publisher.verify_checksums(self.checksums, "9.9.9")

    def test_stage_writes_the_binaries_and_the_launcher(self):
        packages = publisher.render("v1.2.3")
        staged = publisher.stage(self.directory / "npm", packages, self.directory)
        self.assertEqual(len(staged), len(publisher.TARGETS) + 1)
        for name, manifest in packages.items():
            package = self.directory / "npm" / name
            self.assertEqual(json.loads((package / "package.json").read_text()), manifest)
            self.assertTrue((package / "LICENSE").is_file())
        launcher = self.directory / "npm" / publisher.NAME / "bin" / "wtr.js"
        self.assertTrue(launcher.is_file())
        linux = self.directory / "npm" / "what-to-run-linux-arm64" / "bin" / "wtr"
        self.assertEqual(linux.read_bytes(), page())
        self.assertTrue(linux.stat().st_mode & 0o111)
        windows = self.directory / "npm" / "what-to-run-windows-x64" / "bin" / "wtr.exe"
        self.assertEqual(windows.read_bytes(), page())

    def test_a_release_archive_without_the_binary_is_an_error(self):
        tarball(self.directory / "what-to-run_1.2.3_linux_amd64.tar.gz", {"LICENSE": b"MIT\n"})
        with self.assertRaises(ValueError):
            publisher.stage(self.directory / "npm", publisher.render("v1.2.3"), self.directory)

    def test_publish_skips_versions_already_on_the_registry(self):
        packages = publisher.render("v1.2.3")
        with patch.object(publisher, "published", side_effect=[True, False, False, False, False, False, False]), \
                patch.object(publisher.subprocess, "run") as run:
            publisher.publish(packages, self.directory / "npm", provenance=False)
        self.assertEqual(run.call_count, len(publisher.TARGETS))
        command = run.call_args.args[0]
        self.assertEqual(command[:2], ["npm", "publish"])
        self.assertIn("--access", command)
        self.assertNotIn("--provenance", command)
        self.assertTrue(run.call_args.kwargs["check"])

    def test_provenance_is_only_requested_on_github(self):
        packages = {publisher.NAME: publisher.render("v1.2.3")[publisher.NAME]}
        with patch.object(publisher, "published", return_value=False), \
                patch.object(publisher.subprocess, "run") as run:
            publisher.publish(packages, self.directory / "npm", provenance=True)
        self.assertIn("--provenance", run.call_args.args[0])

    def test_checksums_invocations_only_and_dry_run_does_not_publish(self):
        checksums = self.directory / "checksums.txt"
        checksums.write_text(self.checksums)
        arguments = ["publisher", "--tag", "v1.2.3", "--artifact", str(checksums)]
        with patch("sys.argv", arguments + ["--dry-run"]), \
                patch.object(publisher, "publish") as publish:
            publisher.main()
        publish.assert_not_called()
        self.assertTrue((self.directory / "npm" / publisher.NAME / "package.json").is_file())
        with patch("sys.argv", arguments), patch.object(publisher, "publish") as publish:
            publisher.main()
        publish.assert_called_once()


if __name__ == "__main__":
    unittest.main()
