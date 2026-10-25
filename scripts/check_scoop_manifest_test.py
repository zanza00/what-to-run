import contextlib
import importlib.util
import io
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location(
    "checker", Path(__file__).with_name("check-scoop-manifest.py"))
checker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(checker)

TAG = "v1.2.3"
VERSION = "1.2.3"
AMD64 = f"what-to-run_{VERSION}_windows_amd64.zip"
ARM64 = f"what-to-run_{VERSION}_windows_arm64.zip"
HASHES = {AMD64: "a" * 64, ARM64: "b" * 64}


def download_url(name):
    return f"https://github.com/zanza00/what-to-run/releases/download/{TAG}/{name}"


def manifest():
    return {
        "version": VERSION,
        "architecture": {
            "64bit": {"url": download_url(AMD64), "bin": ["wtr.exe"], "hash": HASHES[AMD64]},
            "arm64": {"url": download_url(ARM64), "bin": ["wtr.exe"], "hash": HASHES[ARM64]},
        },
        "homepage": "https://github.com/zanza00/what-to-run",
        "license": "MIT",
    }


def checksums(sums):
    return "\n".join(f"{digest}  {name}" for name, digest in sums.items()) + "\n"


class CheckTest(unittest.TestCase):
    def setUp(self):
        self.manifest = manifest()
        self.sums = dict(HASHES)

    def test_valid_manifest_has_no_problems(self):
        self.assertEqual(checker.check(TAG, self.manifest, self.sums), [])

    def test_version_must_match_the_tag(self):
        self.manifest["version"] = "1.2.2"
        problems = checker.check(TAG, self.manifest, self.sums)
        self.assertEqual(len(problems), 1)
        self.assertIn("1.2.2", problems[0])

    def test_swapped_archives_are_reported_per_architecture(self):
        architecture = self.manifest["architecture"]
        architecture["64bit"]["url"], architecture["arm64"]["url"] = (
            architecture["arm64"]["url"], architecture["64bit"]["url"])
        problems = checker.check(TAG, self.manifest, self.sums)
        self.assertEqual(len(problems), 2)
        self.assertTrue(any(problem.startswith("64bit downloads") for problem in problems))
        self.assertTrue(any(problem.startswith("arm64 downloads") for problem in problems))

    def test_hashes_come_from_checksums_txt(self):
        self.sums[AMD64] = "c" * 64
        problems = checker.check(TAG, self.manifest, self.sums)
        self.assertEqual(problems, [f"64bit hash does not match checksums.txt for {AMD64}"])

    def test_missing_checksum_is_reported(self):
        del self.sums[ARM64]
        problems = checker.check(TAG, self.manifest, self.sums)
        self.assertEqual(problems, [f"checksums.txt has no {ARM64}"])

    def test_binary_and_architecture_are_checked(self):
        self.manifest["architecture"]["64bit"]["bin"] = []
        del self.manifest["architecture"]["arm64"]
        self.manifest["architecture"]["386"] = {"url": download_url("other.zip"), "bin": [], "hash": ""}
        problems = checker.check(TAG, self.manifest, self.sums)
        self.assertIn("64bit installs [], expected ['wtr.exe']", problems)
        self.assertIn("Manifest has no arm64 architecture", problems)
        self.assertIn("Manifest carries an unexpected 386 architecture", problems)

    def test_tag_must_be_stable(self):
        for tag in ("1.2.3", "v1.2.3-rc1", "v1.2"):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                checker.check(tag, self.manifest, self.sums)

    def test_shape_errors_are_reported(self):
        self.assertEqual(checker.check(TAG, [], self.sums), ["Manifest is not a JSON object"])
        self.assertEqual(
            checker.check(TAG, {"version": VERSION}, self.sums),
            ["Manifest has no architecture table"],
        )


class MainTest(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.directory = Path(directory.name)

    def run_main(self, tag, manifest_text, checksums_text):
        manifest_path = self.directory / "what-to-run.json"
        checksums_path = self.directory / "checksums.txt"
        manifest_path.write_text(manifest_text)
        checksums_path.write_text(checksums_text)
        stderr = io.StringIO()
        argv = ["check-scoop-manifest.py", tag, str(manifest_path), str(checksums_path)]
        with patch.object(sys, "argv", argv), contextlib.redirect_stderr(stderr):
            status = checker.main()
        return status, stderr.getvalue()

    def test_a_matching_manifest_passes(self):
        status, stderr = self.run_main(TAG, json.dumps(manifest()), checksums(HASHES))
        self.assertEqual((status, stderr), (0, ""))

    def test_every_problem_is_printed(self):
        broken = manifest()
        broken["version"] = "9.9.9"
        broken["architecture"]["64bit"]["hash"] = "d" * 64
        status, stderr = self.run_main(TAG, json.dumps(broken), checksums(HASHES))
        self.assertEqual(status, 1)
        self.assertIn("expected '1.2.3'", stderr)
        self.assertIn("64bit hash does not match checksums.txt", stderr)

    def test_invalid_inputs_fail_with_a_message(self):
        for text in ("not json", json.dumps([])):
            with self.subTest(text=text):
                status, stderr = self.run_main(TAG, text, checksums(HASHES))
                self.assertEqual(status, 1)
                self.assertTrue(stderr)
        status, stderr = self.run_main(TAG, json.dumps(manifest()), "invalid\n")
        self.assertEqual(status, 1)
        self.assertIn("Expected '<sha256>  <name>'", stderr)


if __name__ == "__main__":
    unittest.main()
