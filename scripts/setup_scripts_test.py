import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

SCRIPTS = Path(__file__).resolve().parent
NAMES = ("setup.deb.sh", "setup.rpm.sh")
REPOSITORY = "https://zanza00.github.io/what-to-run"

# The scripts write outside the working tree unless the WTR_* variables point
# elsewhere, and they refuse to run without root: the behaviour tests stub the
# tools, redirect every path, and take root from a user namespace when needed.
TOOLS = ("sh", "id", "install", "mktemp", "rm", "cat")
STUBS = {
    "curl": """#!/bin/sh
out=""
while [ $# -gt 0 ]; do
    case $1 in
        -o) out=$2; shift 2 ;;
        *) shift ;;
    esac
done
printf 'repository key' > "$out"
""",
    "apt-get": """#!/bin/sh
printf 'apt-get %s\\n' "$*" >> "$LOGFILE"
""",
    "rpm": """#!/bin/sh
printf 'rpm %s\\n' "$*" >> "$LOGFILE"
""",
    "dnf": """#!/bin/sh
printf 'dnf %s\\n' "$*" >> "$LOGFILE"
""",
    "yum": """#!/bin/sh
printf 'yum %s\\n' "$*" >> "$LOGFILE"
""",
}


def _root_prefix():
    if os.geteuid() == 0:
        return []
    unshare = shutil.which("unshare")
    if unshare is None:
        return None
    if subprocess.run([unshare, "-r", "true"], capture_output=True).returncode != 0:
        return None
    return [unshare, "-r"]


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


class SetupScriptsBehaviourTest(unittest.TestCase):
    def setUp(self):
        self.prefix = _root_prefix()
        if self.prefix is None:
            self.skipTest("needs root or unprivileged user namespaces")
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        self.bin = self.directory / "bin"
        self.bin.mkdir()
        self.log = self.directory / "log"
        for name, source in STUBS.items():
            stub = self.bin / name
            stub.write_text(source)
            stub.chmod(0o755)
        for name in TOOLS:
            (self.bin / name).symlink_to(shutil.which(name))

    def run_script(self, name, **environment):
        variables = {"PATH": str(self.bin), "LOGFILE": str(self.log)}
        variables.update(environment)
        return subprocess.run(
            [*self.prefix, str(self.bin / "sh"), str(SCRIPTS / name)],
            capture_output=True, text=True, env=variables,
        )

    def test_deb_script_configures_the_repository(self):
        keyring = self.directory / "keyring.gpg"
        sources = self.directory / "what-to-run.list"
        result = self.run_script(
            "setup.deb.sh",
            WTR_APT_KEYRING=str(keyring), WTR_APT_SOURCES=str(sources),
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(keyring.read_text(), "repository key")
        self.assertEqual(
            sources.read_text(),
            "deb [signed-by=%s] %s stable main\n" % (keyring, REPOSITORY),
        )
        self.assertEqual(self.log.read_text(), "apt-get update\n")

    def test_rpm_script_configures_the_repository(self):
        result, key, repository_file = self.run_rpm()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(key.read_text(), "repository key")
        self.assertEqual(
            repository_file.read_text(),
            "[what-to-run]\n"
            "name=what-to-run\n"
            "baseurl=%s/rpm/\n"
            "enabled=1\n"
            "gpgcheck=1\n"
            "repo_gpgcheck=1\n"
            "gpgkey=file://%s\n" % (REPOSITORY, key),
        )
        self.assertEqual(
            self.log.read_text(),
            "rpm --import %s\ndnf -q makecache\n" % key,
        )
        self.assertIn("sudo dnf install what-to-run", result.stdout)

    def test_rpm_script_falls_back_to_yum(self):
        (self.bin / "dnf").unlink()
        result, key, _ = self.run_rpm()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(
            self.log.read_text(),
            "rpm --import %s\nyum -q makecache\n" % key,
        )
        self.assertIn("sudo yum install what-to-run", result.stdout)

    def test_rpm_script_warns_without_a_package_manager(self):
        (self.bin / "dnf").unlink()
        (self.bin / "yum").unlink()
        result, _, _ = self.run_rpm()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("neither dnf nor yum", result.stderr)
        self.assertIn("distribution package manager", result.stdout)

    def run_rpm(self):
        key = self.directory / "key.asc"
        repository_file = self.directory / "what-to-run.repo"
        result = self.run_script(
            "setup.rpm.sh",
            WTR_RPM_KEY=str(key), WTR_RPM_REPOSITORY_FILE=str(repository_file),
        )
        return result, key, repository_file


if __name__ == "__main__":
    unittest.main()
