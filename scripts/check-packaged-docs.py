"""Check the README and guide links in an extracted Mavis release."""

from pathlib import Path
import re
import sys
from urllib.parse import unquote, urlsplit


def check(root):
    content = root / "docs/content"
    required = (root / "README.md", content / "getting-started/installation.md",
                content / "reference/configuration.md", root / "testdata/docs/README.md")
    for document in required:
        if not document.is_file():
            raise SystemExit(f"Missing packaged document: {document}")

    for document in [root / "README.md", *content.rglob("*.md")]:
        text = re.sub(r"(?ms)^```.*?^```[^\n]*", "", document.read_text(encoding="utf-8"))
        for reference in re.findall(r"\]\(([^()\s]+)\)", text):
            url = urlsplit(reference)
            if url.scheme or url.netloc or not url.path.endswith(".md"):
                continue
            target = (document.parent / unquote(url.path)).resolve()
            if not target.is_relative_to(root) or not target.is_file():
                raise SystemExit(f"Broken packaged link in {document}: {reference}")


if __name__ == "__main__":
    check(Path(sys.argv[1]).resolve())
