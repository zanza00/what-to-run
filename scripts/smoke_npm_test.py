import importlib.util
from pathlib import Path
import tempfile
import unittest


publisher_spec = importlib.util.spec_from_file_location("publisher", Path(__file__).with_name("publish-npm.py"))
publisher = importlib.util.module_from_spec(publisher_spec)
publisher_spec.loader.exec_module(publisher)

smoke_spec = importlib.util.spec_from_file_location("smoke", Path(__file__).with_name("smoke-npm.py"))
smoke = importlib.util.module_from_spec(smoke_spec)
smoke_spec.loader.exec_module(smoke)


class PackArchivesTest(unittest.TestCase):
    def setUp(self):
        self.directory = Path(tempfile.mkdtemp())
        # A stub binary per target, each with different content. The staged
        # packages must contain this content, one to one.
        self.binaries = {}
        for goos, goarch in publisher.TARGETS:
            binary = self.directory / f"wtr-{goos}-{goarch}"
            binary.write_bytes(f"stub for {goos} {goarch}\n".encode())
            self.binaries[(goos, goarch)] = binary

    def test_the_packed_archives_stage_the_same_binaries(self):
        # The publisher reads the archives and the checksums, so it is the
        # judge of what packing must produce. A smoke test that packed in
        # another shape would publish packages the publisher itself refuses.
        checksums = smoke.pack_archives(self.directory / "dist", self.binaries, "1.2.3")
        packages = publisher.render("v1.2.3")
        publisher.verify_checksums(checksums.read_text(), packages)
        names = [line.split()[1] for line in checksums.read_text().splitlines()]
        self.assertEqual(
            set(names),
            {publisher.asset_name(goos, goarch, "1.2.3") for goos, goarch in publisher.TARGETS},
        )
        publisher.stage(self.directory / "npm", packages, self.directory / "dist")
        for (goos, goarch), binary in self.binaries.items():
            name = "wtr.exe" if goos == "windows" else "wtr"
            package = self.directory / "npm" / publisher.package_name(goos, goarch) / "bin" / name
            self.assertEqual(package.read_bytes(), binary.read_bytes(), f"{goos} {goarch}")


if __name__ == "__main__":
    unittest.main()
