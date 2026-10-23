import os
from pathlib import Path
import subprocess
import unittest

SCRIPTS = Path(__file__).resolve().parent
NAMES = ("setup.deb.sh", "setup.rpm.sh")
REPOSITORY = "https://zanza00.github.io/what-to-run"


class SetupScriptsTest(unittest.TestCase):
    def test_scripts_parse(self):
        for name in NAMES:
            with self.subTest(script=name):
                subprocess.run(["sh", "-n", str(SCRIPTS / name)], check=True)

    def test_scripts_use_the_pages_repository(self):
        for name in NAMES:
            with self.subTest(script=name):
                self.assertIn(REPOSITORY, (SCRIPTS / name).read_text())

    @unittest.skipIf(os.geteuid() == 0, "root would pass the root guard")
    def test_scripts_require_root(self):
        for name in NAMES:
            with self.subTest(script=name):
                result = subprocess.run(
                    ["sh", str(SCRIPTS / name)],
                    capture_output=True, text=True,
                )
                self.assertEqual(result.returncode, 1)
                self.assertIn("run this script as root", result.stderr)


if __name__ == "__main__":
    unittest.main()
