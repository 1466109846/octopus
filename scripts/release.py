#!/usr/bin/env python3
"""Build and package a release locally; this script never publishes anything."""
import argparse
import gzip
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
from datetime import datetime, timezone
import zipfile

ROOT = Path(__file__).resolve().parents[1]
TARGETS = ('linux-amd64', 'linux-arm64', 'linux-arm', 'linux-386',
           'windows-amd64', 'darwin-arm64', 'darwin-amd64')
FALLBACK_REPO = 'https://github.com/1466109846/octopus'


def source_version(root):
    versions = re.findall(r'^// Version (v\d+\.\d+\.\d+(?:\.\d+)?)\s*$',
                          (root / 'main.go').read_text(encoding='utf-8'), re.MULTILINE)
    if len(versions) != 1:
        raise ValueError('main.go must contain exactly one // Version vX.Y.Z[.N] marker')
    return versions[0]


def canonical_repository(remote):
    remote = remote.strip()
    if remote.startswith('git@github.com:'):
        remote = 'https://github.com/' + remote.split(':', 1)[1]
    remote = remote.removesuffix('/').removesuffix('.git')
    if not re.fullmatch(r'https://github\.com/[\w.-]+/[\w.-]+', remote):
        raise ValueError('Repository must be a public GitHub URL without embedded credentials')
    return remote


def read_asset(out, name):
    direct = out / name
    if direct.exists():
        return direct.read_text(encoding='utf-8')
    with gzip.open(out / (name + '.gz'), 'rt', encoding='utf-8') as f:
        return f.read()


def verify_frontend(out, expected):
    actual = json.loads(read_asset(out, 'build-info.json'))
    for key in ('version', 'repository', 'commit'):
        if actual.get(key) != expected[key]:
            raise ValueError(f'Stale frontend {key}: {actual.get(key)!r} != {expected[key]!r}')
    worker = read_asset(out, 'sw.js')
    if '__APP_VERSION__' in worker or f"{expected['version']}-{expected['commit']}" not in worker:
        raise ValueError('Service Worker cache version does not match this build')
    return actual


def package_binary(archive, binary, executable_name, extra_files):
    with zipfile.ZipFile(archive, 'w', zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for file, name, mode in [(binary, executable_name, 0o755),
                                 *[(p, p.name, 0o644) for p in extra_files]]:
            info = zipfile.ZipInfo.from_file(file, arcname=name)
            info.create_system = 3
            info.external_attr = (0o100000 | mode) << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            z.writestr(info, file.read_bytes())
    with zipfile.ZipFile(archive) as z:
        if z.testzip() is not None:
            raise ValueError(f'Archive integrity check failed: {archive.name}')


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT, text=True).strip()


def run(command, *, env, cwd=ROOT, stdout=None):
    print('Running:', ' '.join(str(c) for c in command[:4]), flush=True)
    subprocess.run(command, cwd=cwd, env=env, stdout=stdout, check=True)


def write_checksums(assets, checksums):
    # newline='' 抑制 Windows 的 \n -> \r\n 翻译: SHA256SUMS 会被 Linux/macOS 的
    # sha256sum -c 直接消费, 行尾带 \r 时那边把 '\r' 拼进文件名, 七个包全部校验失败。
    with (assets / 'SHA256SUMS').open('w', encoding='ascii', newline='') as f:
        f.write('\n'.join(sorted(checksums)) + '\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--targets', nargs='+', choices=TARGETS, default=list(TARGETS))
    parser.add_argument('--repo', help='Override origin with a public GitHub repository URL')
    parser.add_argument('--out', type=Path, help='New build directory (must not already exist)')
    parser.add_argument('--allow-dirty', action='store_true', help='Local validation only; metadata gets -dirty')
    args = parser.parse_args()

    version = source_version(ROOT)
    dirty = bool(git('status', '--porcelain', '--untracked-files=normal'))
    if dirty and not args.allow_dirty:
        raise SystemExit('Commit the release source before building, or use --allow-dirty for local validation')
    commit = git('rev-parse', '--short', 'HEAD') + ('-dirty' if dirty else '')
    try:
        remote = args.repo or git('remote', 'get-url', 'origin')
    except subprocess.CalledProcessError:
        remote = FALLBACK_REPO
    repository = canonical_repository(remote)
    metadata = {'version': version, 'repository': repository, 'commit': commit,
                'build_time': datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')}
    out = args.out.resolve() if args.out else ROOT / 'build' / f'{version}-{commit}'
    if out.exists():
        raise SystemExit(f'Refusing to overwrite existing build directory: {out}')
    bins, assets = out / 'bin', out / 'assets'
    bins.mkdir(parents=True)
    assets.mkdir()
    metadata_file = out / 'build-info.json'
    metadata_file.write_text(json.dumps(metadata, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')

    env = os.environ.copy()
    env.update({'VITE_APP_VERSION': version, 'VITE_GITHUB_REPO': repository, 'VITE_APP_COMMIT': commit})
    pnpm = shutil.which('pnpm.cmd' if os.name == 'nt' else 'pnpm')
    if not pnpm:
        raise SystemExit('pnpm is required to build the embedded frontend')
    placeholder = ROOT / 'static' / 'out' / 'README.md'
    original = placeholder.read_bytes() if placeholder.exists() else None
    try:
        run([pnpm, 'install', '--frozen-lockfile'], env=env, cwd=ROOT / 'web')
        run([pnpm, 'run', 'build'], env=env, cwd=ROOT / 'web')
    finally:
        if original is not None:
            placeholder.parent.mkdir(parents=True, exist_ok=True)
            placeholder.write_bytes(original)
    verify_frontend(ROOT / 'static' / 'out', metadata)
    print('Frontend version, repository, commit and Service Worker verified:', metadata, flush=True)

    flags = ' '.join(f'-X github.com/bestruirui/octopus/internal/conf.{key}={value}' for key, value in {
        'Version': version, 'Repo': repository, 'Commit': commit,
        'BuildTime': metadata['build_time'], 'Author': 'bestrui',
    }.items()) + ' -s -w'
    for target in args.targets:
        system, arch = target.split('-')
        target_env = {**env, 'GOOS': system, 'GOARCH': arch, 'CGO_ENABLED': '0'}
        target_env.pop('GOARM', None)
        if arch == 'arm':
            target_env['GOARM'] = '7'
        binary = bins / ('octopus-' + target + ('.exe' if system == 'windows' else ''))
        run(['go', 'build', '-trimpath', '-tags=jsoniter', '-ldflags=' + flags, '-o', str(binary), '.'], env=target_env)
        print('Built:', target, binary.stat().st_size, 'bytes', flush=True)
        if os.name == 'nt' and target == 'windows-amd64':
            printed = subprocess.check_output([str(binary), 'version'], text=True)
            if f'Version: {version}' not in printed or f'Commit ID: {commit}' not in printed:
                raise ValueError('Windows binary version does not match build metadata')

    licenses = out / 'THIRD_PARTY_LICENSES.csv'
    license_env = {**env, 'GOFLAGS': '-tags=jsoniter'}
    with licenses.open('w', encoding='utf-8', newline='') as f:
        run(['go', 'run', 'github.com/google/go-licenses/v2@v2.0.1', 'report', '.',
             '--ignore', 'github.com/bestruirui/octopus'], env=license_env, stdout=f)
    if licenses.stat().st_size == 0:
        raise ValueError('Third-party license report is empty')
    extra = [ROOT / 'README.md', ROOT / 'README_zh.md', ROOT / 'LICENSE', licenses, metadata_file]
    checksums = []
    for target in args.targets:
        windows = target.startswith('windows-')
        binary = bins / ('octopus-' + target + ('.exe' if windows else ''))
        archive = assets / ('octopus-' + target + '.zip')
        package_binary(archive, binary, 'octopus.exe' if windows else 'octopus', extra)
        digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        checksums.append(f'{digest}  {archive.name}')
        print('Packaged:', archive.name, flush=True)
    write_checksums(assets, checksums)
    shutil.copyfile(metadata_file, assets / metadata_file.name)
    print('Release assets verified:', assets, flush=True)


if __name__ == '__main__':
    main()
