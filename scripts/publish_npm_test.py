import importlib.util
import io
import json
import os
from pathlib import Path
import re
import tarfile
import tempfile
import unittest
from unittest.mock import MagicMock, patch
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


def fake_binary():
    return '#!/bin/sh\necho "wtr $@"; exit 0\n'.encode()


class PublisherTest(unittest.TestCase):
    def setUp(self):
        self.directory = Path(tempfile.mkdtemp())
        for goos, goarch in publisher.TARGETS:
            name = publisher.asset_name(goos, goarch, "1.2.3")
            path = self.directory / name
            if goos == "windows":
                with zipfile.ZipFile(path, "w") as archive:
                    archive.writestr("wtr.exe", fake_binary())
            else:
                # A release archive also carries the docs; the publisher has to
                # find the binary among the other members.
                tarball(path, {"LICENSE": b"MIT\n", "wtr": fake_binary()})
        self.checksums = "\n".join(
            f"{index:064x}  {publisher.asset_name(goos, goarch, '1.2.3')}"
            for index, (goos, goarch) in enumerate(publisher.TARGETS, 1)
        )

    def test_render_publishes_one_package_per_platform_plus_the_root(self):
        packages = publisher.render("v1.2.3")
        self.assertEqual(list(packages)[-1], publisher.NAME)
        self.assertEqual(len(packages), len(publisher.TARGETS) + 1)
        root = packages[publisher.NAME].manifest
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
        for name, package in packages.items():
            if name == publisher.NAME:
                continue
            manifest = package.manifest
            self.assertEqual(manifest["version"], "1.2.3")
            self.assertNotIn("bin", manifest)
            self.assertNotIn("private", manifest)
            self.assertEqual(len(manifest["os"]), 1)
            self.assertEqual(len(manifest["cpu"]), 1)
        self.assertEqual(packages["what-to-run-windows-arm64"].manifest["os"], ["win32"])
        self.assertEqual(packages["what-to-run-windows-arm64"].manifest["cpu"], ["arm64"])
        self.assertEqual(packages["what-to-run-darwin-x64"].manifest["cpu"], ["x64"])
        self.assertEqual(
            packages["what-to-run-windows-arm64"].manifest["files"], ["bin/wtr.exe"]
        )

    def test_versions_are_pinned_to_the_release(self):
        root = publisher.render("v9.8.7")[publisher.NAME].manifest
        self.assertEqual(root["version"], "9.8.7")
        self.assertEqual(set(root["optionalDependencies"].values()), {"9.8.7"})

    def test_invalid_tags_fail_before_publication(self):
        for tag in ("v1.2.3-rc1", "1.2.3", "v1.2", 'v1.2.3"'):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                publisher.render(tag)

    def test_checksums_must_describe_every_platform(self):
        packages = publisher.render("v1.2.3")
        publisher.verify_checksums(self.checksums, packages)
        with self.assertRaises(ValueError):
            publisher.verify_checksums(self.checksums.replace(f"{1:064x}", "invalid"), packages)
        with self.assertRaises(ValueError):
            publisher.verify_checksums(
                self.checksums + "\n" + self.checksums.splitlines()[0], packages
            )
        with self.assertRaises(ValueError):
            publisher.verify_checksums("\n".join(self.checksums.splitlines()[:-1]), packages)
        with self.assertRaises(ValueError):
            publisher.verify_checksums(self.checksums, publisher.render("v9.9.9"))

    def test_stage_writes_the_binaries_and_the_launcher(self):
        packages = publisher.render("v1.2.3")
        staged = publisher.stage(self.directory / "npm", packages, self.directory)
        self.assertEqual(len(staged), len(publisher.TARGETS) + 1)
        for name, package in packages.items():
            target = self.directory / "npm" / name
            self.assertEqual(json.loads((target / "package.json").read_text()), package.manifest)
            self.assertTrue((target / "LICENSE").is_file())
        launcher = self.directory / "npm" / publisher.NAME / "bin" / "wtr.js"
        self.assertTrue(launcher.is_file())
        linux = self.directory / "npm" / "what-to-run-linux-arm64" / "bin" / "wtr"
        self.assertEqual(linux.read_bytes(), fake_binary())
        self.assertTrue(linux.stat().st_mode & 0o111)
        windows = self.directory / "npm" / "what-to-run-windows-x64" / "bin" / "wtr.exe"
        self.assertEqual(windows.read_bytes(), fake_binary())

    def test_a_release_archive_without_the_binary_is_an_error(self):
        tarball(self.directory / "what-to-run_1.2.3_linux_amd64.tar.gz", {"LICENSE": b"MIT\n"})
        with self.assertRaises(ValueError):
            publisher.stage(self.directory / "npm", publisher.render("v1.2.3"), self.directory)

    def test_the_root_package_ships_the_readme(self):
        publisher.stage(self.directory / "npm", publisher.render("v1.2.3"), self.directory)
        readme = Path(__file__).resolve().parent.parent / "README.md"
        root = self.directory / "npm" / publisher.NAME / "README.md"
        self.assertEqual(root.read_bytes(), readme.read_bytes())
        self.assertFalse((self.directory / "npm" / "what-to-run-linux-x64" / "README.md").exists())

    def test_the_platform_matrix_matches_goreleaser(self):
        config = (Path(__file__).resolve().parent.parent / ".goreleaser.yaml").read_text()
        # Only the builds block lists the matrix in brackets: the archives block
        # spells a bare `goos: windows`, which this pattern must not pick up.
        lists = re.findall(r"^\s+go(?:os|arch): \[([^\]]*)\]", config, re.MULTILINE)
        self.assertEqual(len(lists), 2)
        platforms = [value.strip() for value in lists[0].split(",")]
        architectures = [value.strip() for value in lists[1].split(",")]
        self.assertEqual(
            {(platform, architecture) for platform in platforms for architecture in architectures},
            set(publisher.TARGETS),
        )

    def test_the_launcher_platform_table_matches_the_publisher(self):
        # The launcher carries the same mapping in its own table, spelled
        # the way Node reports the machine: the keys join `process.platform`
        # and `process.arch`, and the values are the package name suffixes.
        # Parse that table and hold it against the publisher's spellings,
        # so a drift between the two files fails here, not on a machine
        # the release does not cover.
        source = (Path(__file__).resolve().parent.parent / "npm" / "bin" / "wtr.js").read_text()
        table = re.search(r"const SUFFIXES = \{[^}]*\}", source)
        self.assertIsNotNone(table, "the launcher no longer declares SUFFIXES")
        entries = dict(re.findall(r'"([^"]+)":\s*"([^"]+)"', table.group(0)))
        named = {key: f"{publisher.NAME}-{suffix}" for key, suffix in entries.items()}
        expected = {
            f"{publisher.NPM_OS[goos]}-{publisher.NPM_CPU[goarch]}": publisher.package_name(goos, goarch)
            for goos, goarch in publisher.TARGETS
        }
        self.assertEqual(named, expected)

    def test_publish_skips_versions_already_on_the_registry(self):
        packages = publisher.render("v1.2.3")
        with patch.object(publisher, "published", side_effect=[True, False, False, False, False, False, False]), \
                patch.object(
                    publisher.subprocess, "run", return_value=MagicMock(returncode=0)
                ) as run:
            publisher.publish(packages, self.directory / "npm", provenance=False)
        self.assertEqual(run.call_count, len(publisher.TARGETS))
        command = run.call_args.args[0]
        self.assertEqual(command[:2], ["npm", "publish"])
        self.assertIn("--access", command)
        self.assertNotIn("--provenance", command)

    def test_a_failed_publish_names_the_package(self):
        packages = {publisher.NAME: publisher.render("v1.2.3")[publisher.NAME]}
        with patch.object(publisher, "published", return_value=False), \
                patch.object(
                    publisher.subprocess, "run", return_value=MagicMock(returncode=1)
                ):
            with self.assertRaises(SystemExit) as raised:
                publisher.publish(packages, self.directory / "npm", provenance=False)
        self.assertIn("what-to-run@1.2.3", str(raised.exception))
        self.assertIn("exit code 1", str(raised.exception))

    def test_main_requests_provenance_only_on_github(self):
        checksums = self.directory / "checksums.txt"
        checksums.write_text(self.checksums)
        packages = {publisher.NAME: publisher.render("v1.2.3")[publisher.NAME]}
        arguments = ["publisher", "--tag", "v1.2.3", "--artifact", str(checksums)]

        def provenance_seen(environment, absent):
            with patch.object(publisher, "verify_checksums"), \
                    patch.object(publisher, "render", return_value=packages), \
                    patch.object(publisher, "stage", return_value=[]), \
                    patch.object(publisher, "publish") as publish, \
                    patch("sys.argv", arguments), \
                    patch.dict(os.environ, environment, clear=False):
                if absent:
                    os.environ.pop("GITHUB_ACTIONS", None)
                publisher.main()
            publish.assert_called_once()
            return publish.call_args.kwargs["provenance"]

        self.assertIs(provenance_seen({"GITHUB_ACTIONS": "true"}, False), True)
        self.assertIs(provenance_seen({}, True), False)

    def test_dry_run_stages_without_publishing(self):
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
