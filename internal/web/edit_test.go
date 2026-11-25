package web

import (
	"html"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
	"github.com/thorstenkramm/mavis/internal/render"
)

func TestRepositoryEditControl(t *testing.T) {
	root := t.TempDir()
	const special = "Über #1% &.txt"
	for name, body := range map[string]string{".mavis.conf": `edit_url_template = "https://git.example/edit/main/{path}"`, "sub/README.md": "# Hello", "sub/" + special: "example", "picture.png": "\x89PNG\r\n\x1a\n", "manual.pdf": "%PDF-1.7", "listing/item.bin": "\xff"} {
		writeWebFile(t, root, name, body)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	r, err := content.NewResolver(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewHandlerWithExporter(r, render.NewRenderer(), nil, nil, cfg.EditURL)
	if err != nil {
		t.Fatal(err)
	}
	wantLinks := map[string]string{"/sub/README.md": "sub/README.md", "/sub/": "sub/README.md", "/sub/" + url.PathEscape(special): "sub/" + url.PathEscape(special)}
	if os.Symlink(filepath.Join(root, "sub", "README.md"), filepath.Join(root, "alias.md")) == nil {
		wantLinks["/alias.md"] = "alias.md"
	}
	for target, want := range wantLinks {
		body := html.UnescapeString(requestWeb(h, target).Body.String())
		if !strings.Contains(body, `class="edit-page" href="https://git.example/edit/main/`+want+`"`) {
			t.Errorf("wrong edit link on %s", target)
		}
	}
	for _, target := range []string{"/picture.png", "/manual.pdf", "/listing/", "/missing", "/_mavis/search?q=Hello", "/sub/README.md?download=markdown", "/sub/README.md?download=pdf", "/sub/" + url.PathEscape(special) + "?raw=1"} {
		if strings.Contains(requestWeb(h, target).Body.String(), `class="edit-page"`) {
			t.Errorf("unexpected edit link on %s", target)
		}
	}
}
