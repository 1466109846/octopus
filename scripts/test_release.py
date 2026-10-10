import gzip
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
import zipfile

MODULE = Path(__file__).with_name('release.py')
spec = importlib.util.spec_from_file_location('release_build', MODULE)
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class ReleaseBuildTests(unittest.TestCase):
    def test_version_comes_from_source_not_previous_git_tag(self):
        with tempfile.TemporaryDirectory() as root:
            Path(root, 'main.go').write_text('package main\n// Version v0.13.10.2\n', encoding='utf-8')
            self.assertEqual(release.source_version(Path(root)), 'v0.13.10.2')

    def test_missing_or_duplicate_version_is_rejected(self):
        for content in ['package main\n', '// Version v1.2.3\n// Version v1.2.4\n']:
            with self.subTest(content=content), tempfile.TemporaryDirectory() as root:
                Path(root, 'main.go').write_text(content, encoding='utf-8')
                with self.assertRaises(ValueError):
                    release.source_version(Path(root))

    def test_git_remote_becomes_canonical_public_url(self):
        for remote in ['https://github.com/1466109846/octopus.git', 'git@github.com:1466109846/octopus.git']:
            self.assertEqual(release.canonical_repository(remote), 'https://github.com/1466109846/octopus')
        with self.assertRaises(ValueError):
            release.canonical_repository('https://token@github.com/1466109846/octopus.git')

    def test_frontend_metadata_rejects_stale_version_or_repository(self):
        expected = {'version': 'v0.13.10.2', 'repository': 'https://github.com/1466109846/octopus', 'commit': 'abc1234'}
        with tempfile.TemporaryDirectory() as root:
            out = Path(root)
            with gzip.open(out / 'build-info.json.gz', 'wt', encoding='utf-8') as f:
                json.dump(expected, f)
            with gzip.open(out / 'sw.js.gz', 'wt', encoding='utf-8') as f:
                f.write("const CACHE_VERSION = 'v0.13.10.2-abc1234';")
            self.assertEqual(release.verify_frontend(out, expected), expected)
            for field, stale in [('version', 'v0.13.10'), ('repository', 'https://github.com/xiaocqaq/octopus'), ('commit', 'old')]:
                with self.subTest(field=field):
                    bad = {**expected, field: stale}
                    with self.assertRaises(ValueError):
                        release.verify_frontend(out, bad)

    def test_unreplaced_worker_placeholder_is_rejected(self):
        metadata = {'version': 'v0.13.10.2', 'repository': 'https://github.com/1466109846/octopus', 'commit': 'abc1234'}
        with tempfile.TemporaryDirectory() as root:
            out = Path(root)
            (out / 'build-info.json').write_text(json.dumps(metadata), encoding='utf-8')
            (out / 'sw.js').write_text("const CACHE_VERSION = '__APP_VERSION__';", encoding='utf-8')
            with self.assertRaises(ValueError):
                release.verify_frontend(out, metadata)

    def test_package_contains_docs_licenses_metadata_and_executable_mode(self):
        with tempfile.TemporaryDirectory() as root:
            base = Path(root)
            exe = base / 'octopus-linux-amd64'
            exe.write_bytes(b'\x7fELF-test')
            files = []
            for name in ['README.md', 'README_zh.md', 'LICENSE', 'THIRD_PARTY_LICENSES.csv', 'build-info.json']:
                item = base / name
                item.write_text('{}' if name.endswith('.json') else name, encoding='utf-8')
                files.append(item)
            archive = base / 'octopus-linux-amd64.zip'
            release.package_binary(archive, exe, 'octopus', files)
            with zipfile.ZipFile(archive) as z:
                self.assertIsNone(z.testzip())
                self.assertEqual(z.read('octopus'), exe.read_bytes())
                self.assertEqual(z.getinfo('octopus').external_attr >> 16 & 0o777, 0o755)
                self.assertIn('THIRD_PARTY_LICENSES.csv', z.namelist())
                self.assertIn('build-info.json', z.namelist())


if __name__ == '__main__':
    unittest.main()
