"""Verify GoReleaser split checksums and make them sha256sum-compatible."""

import argparse
import hashlib
from pathlib import Path
import re


def format_checksums(root, check_only=False, archive=None):
    if archive:
        checksums = [root / f"{archive}.sha256"]
        if not checksums[0].is_file():
            raise SystemExit(f"Missing checksum file: {checksums[0]}")
    else:
        checksums = sorted(root.glob("*.sha256"))
    if not checksums:
        raise SystemExit(f"No split checksum files found in {root}")

    for checksum_path in checksums:
        archive_path = checksum_path.with_suffix("")
        if not archive_path.is_file():
            raise SystemExit(f"Missing archive for {checksum_path}")

        text = checksum_path.read_text(encoding="utf-8")
        fields = text.split()
        if not fields or not re.fullmatch(r"[0-9a-f]{64}", fields[0]):
            raise SystemExit(f"Invalid SHA-256 file: {checksum_path}")
        if len(fields) > 2 or (len(fields) == 2 and fields[1] != archive_path.name):
            raise SystemExit(f"Unexpected checksum entry in {checksum_path}")

        digest = hashlib.sha256(archive_path.read_bytes()).hexdigest()
        if fields[0] != digest:
            raise SystemExit(f"Checksum mismatch for {archive_path}")
        formatted = f"{digest}  {archive_path.name}\n"
        if check_only:
            if text != formatted:
                raise SystemExit(f"Checksum file is not in standard format: {checksum_path}")
        elif text != formatted:
            checksum_path.write_text(formatted, encoding="utf-8")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="verify without changing files")
    parser.add_argument("--archive", help="process only this archive's checksum file")
    parser.add_argument("root", nargs="?", type=Path, default=Path("dist"))
    arguments = parser.parse_args()
    format_checksums(arguments.root, arguments.check, arguments.archive)
