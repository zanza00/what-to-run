from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parent / "check-changelog.sh"

CHANGELOG = """# Changelog

## [Unreleased]

### Added

- old entry

## [0.1.0] - 2026-01-01

### Added

- released entry
"""


class CheckChangelogTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.repo = Path(self.temporary.name)
        self.git("init", "-q")
        self.git("config", "user.email", "test@example.com")
        self.git("config", "user.name", "Test")
        self.write("CHANGELOG.md", CHANGELOG)
        self.write("internal/task.go", "package internal\n")
        self.commit("base")

    def git(self, *args):
        return subprocess.run(
            ["git", *args], cwd=self.repo, check=True,
            capture_output=True, text=True,
        )

    def write(self, path, text):
        target = self.repo / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(text)

    def commit(self, message="change"):
        self.git("add", "-A")
        self.git("commit", "-qm", message)

    def run_script(self, *args):
        return subprocess.run(
            ["bash", str(SCRIPT), *args], cwd=self.repo,
            capture_output=True, text=True,
        )

    def test_code_change_without_entry_fails(self):
        self.write("internal/task.go", "package internal\n// changed\n")
        self.commit()
        result = self.run_script("HEAD~1", "HEAD")
        self.assertEqual(result.returncode, 1)
        self.assertIn("gained no new entry", result.stderr)

    def test_code_change_with_new_entry_passes(self):
        self.write("internal/task.go", "package internal\n// changed\n")
        self.write(
            "CHANGELOG.md",
            CHANGELOG.replace("- old entry", "- old entry\n- new entry"),
        )
        self.commit()
        result = self.run_script("HEAD~1", "HEAD")
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_empty_heading_does_not_count(self):
        self.write("internal/task.go", "package internal\n// changed\n")
        self.write(
            "CHANGELOG.md",
            CHANGELOG.replace(
                "### Added\n\n- old entry",
                "### Added\n\n- old entry\n\n### Fixed",
            ),
        )
        self.commit()
        result = self.run_script("HEAD~1", "HEAD")
        self.assertEqual(result.returncode, 1)
        self.assertIn("gained no new entry", result.stderr)

    def test_reworded_entry_does_not_count(self):
        self.write("internal/task.go", "package internal\n// changed\n")
        self.write(
            "CHANGELOG.md",
            CHANGELOG.replace("- old entry", "- reworded entry"),
        )
        self.commit()
        result = self.run_script("HEAD~1", "HEAD")
        self.assertEqual(result.returncode, 1)
        self.assertIn("gained no new entry", result.stderr)

    def test_docs_only_change_passes(self):
        self.write("docs/readme.md", "docs\n")
        self.commit()
        result = self.run_script("HEAD~1", "HEAD")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("No watched path changed", result.stdout)

    def test_goreleaser_change_needs_entry(self):
        self.write(".goreleaser.yaml", "version: 2\n")
        self.commit()
        result = self.run_script("HEAD~1", "HEAD")
        self.assertEqual(result.returncode, 1)
        self.assertIn("gained no new entry", result.stderr)

    def test_release_workflow_change_needs_entry(self):
        self.write(".github/workflows/release.yml", "name: release\n")
        self.commit()
        result = self.run_script("HEAD~1", "HEAD")
        self.assertEqual(result.returncode, 1)
        self.assertIn("gained no new entry", result.stderr)

    def test_checker_only_change_passes(self):
        self.write("scripts/check-changelog.sh", "#!/usr/bin/env bash\n")
        self.commit()
        result = self.run_script("HEAD~1", "HEAD")
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_base_without_changelog_passes(self):
        # The base branch predates CHANGELOG.md; the head adds it.
        self.git("rm", "-q", "CHANGELOG.md")
        self.commit("remove changelog")
        self.write("CHANGELOG.md", CHANGELOG)
        self.write("internal/task.go", "package internal\n// changed\n")
        self.commit()
        result = self.run_script("HEAD~1", "HEAD")
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_missing_unreleased_section_fails(self):
        self.write("internal/task.go", "package internal\n// changed\n")
        self.write(
            "CHANGELOG.md",
            "# Changelog\n\n## [0.1.0] - 2026-01-01\n\n- released entry\n",
        )
        self.commit()
        result = self.run_script("HEAD~1", "HEAD")
        self.assertEqual(result.returncode, 1)
        self.assertIn("no ## [Unreleased] section", result.stderr)

    def test_unknown_revision_exits_2(self):
        result = self.run_script("does-not-exist", "HEAD")
        self.assertEqual(result.returncode, 2)
        self.assertIn("unknown revision", result.stderr)

    def test_no_diff_passes(self):
        result = self.run_script("HEAD", "HEAD")
        self.assertEqual(result.returncode, 0, result.stderr)
