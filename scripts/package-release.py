"""Build the CPA store ZIP and checksums.txt from a tested Linux amd64 library."""
import argparse
import hashlib
import pathlib
import zipfile

parser = argparse.ArgumentParser()
parser.add_argument('--library', required=True, type=pathlib.Path)
parser.add_argument('--version', default='0.10.0')
parser.add_argument('--output', default='dist/release', type=pathlib.Path)
args = parser.parse_args()
root = pathlib.Path(__file__).resolve().parent.parent
args.output.mkdir(parents=True, exist_ok=True)
name = 'claude-cache-keepalive_' + args.version + '_linux_amd64.zip'
archive = args.output / name
with zipfile.ZipFile(archive, 'w', zipfile.ZIP_DEFLATED) as package:
    for source, target, mode in [
        (args.library, 'claude-cache-keepalive.so', 0o755),
        (root / 'LICENSE', 'LICENSE', 0o644),
        (root / 'NOTICE', 'NOTICE', 0o644),
    ]:
        entry = zipfile.ZipInfo(target, (2026, 11, 8, 0, 0, 0))
        entry.create_system = 3
        entry.external_attr = (0o100000 | mode) << 16
        entry.compress_type = zipfile.ZIP_DEFLATED
        package.writestr(entry, source.read_bytes())
digest = hashlib.sha256(archive.read_bytes()).hexdigest()
(args.output / 'checksums.txt').write_text(digest + '  ' + name + '\n', encoding='utf-8')
with zipfile.ZipFile(archive) as package:
    assert package.namelist() == ['claude-cache-keepalive.so', 'LICENSE', 'NOTICE']
    assert package.testzip() is None
    assert package.read('claude-cache-keepalive.so') == args.library.read_bytes()
print(str(archive))
print(digest)
