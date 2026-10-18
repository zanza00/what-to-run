import base64
import importlib.util
import json
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location("publisher", Path(__file__).with_name("publish-homebrew.py"))
publisher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publisher)


class PublisherTest(unittest.TestCase):
    def setUp(self):
        self.entries = {
            f"what-to-run_1.2.3_{system}_{arch}.tar.gz": str(index) * 64
            for index, (system, arch) in enumerate(
                (("darwin", "arm64"), ("darwin", "amd64"), ("linux", "arm64"), ("linux", "amd64")), 1
            )
        }
        self.checksums = "\n".join(f"{digest}  {name}" for name, digest in self.entries.items())

    def test_formula_covers_all_platforms_with_matching_checksums(self):
        formula = publisher.render("v1.2.3", self.checksums)
        self.assertIn('version "1.2.3"', formula)
        self.assertIn('license "MIT"', formula)
        self.assertIn('bin.install "wtr"', formula)
        self.assertIn('bin.install "what-to-run"', formula)
        self.assertIn("  test do", formula)
        self.assertEqual(formula.count("      url "), 4)
        for name, digest in self.entries.items():
            self.assertIn(f'/{name}"\n      sha256 "{digest}"', formula)

    def test_invalid_inputs_fail_before_publication(self):
        for tag in ("v1.2.3-rc1", "1.2.3", 'v1.2.3"'):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                publisher.render(tag, self.checksums)
        with self.assertRaises(ValueError):
            publisher.render("v1.2.3", self.checksums.replace("1" * 64, "invalid"))
        with self.assertRaises(ValueError):
            publisher.render("v1.2.3", self.checksums + "\n" + self.checksums.splitlines()[0])
        with self.assertRaises(KeyError):
            publisher.render("v1.2.3", "\n".join(self.checksums.splitlines()[:-1]))

    def test_identical_formula_does_not_write(self):
        existing = json.dumps({"sha": "old-sha", "content": base64.b64encode(b"formula").decode()})
        with patch.object(publisher.subprocess, "check_output", return_value=existing), \
                patch.object(publisher.subprocess, "run") as write:
            publisher.publish("v1.2.3", "formula")
        write.assert_not_called()

    def test_update_uses_existing_sha_and_exact_content(self):
        existing = json.dumps({"sha": "old-sha", "content": base64.b64encode(b"old").decode()})
        with patch.object(publisher.subprocess, "check_output", return_value=existing), \
                patch.object(publisher.subprocess, "run") as write:
            publisher.publish("v1.2.3", "new\n")
        payload = json.loads(write.call_args.kwargs["input"])
        self.assertEqual(payload["sha"], "old-sha")
        self.assertEqual(payload["branch"], "main")
        self.assertEqual(base64.b64decode(payload["content"]), b"new\n")
        self.assertTrue(write.call_args.kwargs["check"])

    def test_api_failure_propagates(self):
        with patch.object(publisher.subprocess, "check_output", side_effect=subprocess.CalledProcessError(1, "gh")):
            with self.assertRaises(subprocess.CalledProcessError):
                publisher.publish("v1.2.3", "formula")

    def test_archive_invocations_do_not_publish(self):
        with patch("sys.argv", ["publisher", "--tag", "v1.2.3", "--artifact", "missing-archive.tar.gz"]), \
                patch.object(publisher, "publish") as publish:
            publisher.main()
        publish.assert_not_called()


if __name__ == "__main__":
    unittest.main()
