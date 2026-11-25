package web

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/render"
)

func TestIconsAndKindsDependOnStructureOnly(t *testing.T) {
	tests := []struct {
		name      string
		directory bool
		icon      string
		kind      string
	}{
		{name: "docs", directory: true, icon: iconFolder, kind: "Folder"},
		{name: "notes.md", directory: true, icon: iconFolder, kind: "Folder"},
		{name: "README.md", icon: iconMarkdown, kind: "Markdown"},
		{name: "GUIDE.MARKDOWN", icon: iconMarkdown, kind: "Markdown"},
		{name: "data.yaml", icon: iconYAML, kind: "YAML"},
		{name: "data.YML", icon: iconYAML, kind: "YML"},
		{name: "tool.py", icon: iconPython, kind: "PY"},
		{name: "archive.tar.gz", icon: iconFile, kind: "GZ"},
		{name: "Makefile", icon: iconFile, kind: "File"},
		{name: ".env", icon: iconFile, kind: "File"},
	}
	for _, test := range tests {
		if icon := iconFor(test.name, test.directory); icon != test.icon {
			t.Errorf("iconFor(%q, %v) = %q, want %q", test.name, test.directory, icon, test.icon)
		}
		if kind := kindFor(test.name, test.directory); kind != test.kind {
			t.Errorf("kindFor(%q, %v) = %q, want %q", test.name, test.directory, kind, test.kind)
		}
		if _, err := embedded.ReadFile("static/icons/" + test.icon + ".svg"); err != nil {
			t.Errorf("icon %q is not embedded: %v", test.icon, err)
		}
	}
}

func TestListingNavigationAndBreadcrumbsUseIcons(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "area/topic/page.md", "# Page\n")
	writeWebFile(t, root, "area/topic/data.yaml", "a: 1\n")
	writeWebFile(t, root, "area/topic/tool.py", "print(1)\n")
	writeWebFile(t, root, "area/topic/sub/note.txt", "note\n")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)

	listing := requestWeb(handler, "/area/topic/").Body.String()
	for _, want := range []string{
		`<a href="/area/">area</a>`, `<span aria-current="location">topic</span>`,
		`<td><a href="/area/topic/sub/"><img class="navigation-icon" src="/_mavis/assets/icons/folder.svg" alt="" aria-hidden="true"><span>sub</span></a></td><td>Folder</td>`,
		`src="/_mavis/assets/icons/yaml.svg" alt="" aria-hidden="true"><span>data.yaml</span></a></td><td>YAML</td>`,
		`src="/_mavis/assets/icons/python.svg" alt="" aria-hidden="true"><span>tool.py</span></a></td><td>PY</td>`,
		`<td>Markdown</td>`,
	} {
		if !strings.Contains(listing, want) {
			t.Errorf("listing does not contain %q: %s", want, listing)
		}
	}
	if root := requestWeb(handler, "/").Body.String(); strings.Contains(root, `class="breadcrumbs"`) {
		t.Fatalf("content root has a breadcrumb trail: %s", root)
	}
	viewer := requestWeb(handler, "/area/topic/data.yaml").Body.String()
	if !strings.Contains(viewer, `<a href="/area/topic/">topic</a>`) || !strings.Contains(viewer, `<span aria-current="location">data.yaml</span>`) {
		t.Fatalf("viewer breadcrumbs missing: %s", viewer)
	}

	var response navigationResponse
	if err := json.Unmarshal(requestWeb(handler, "/_mavis/navigation?path=area/topic").Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	icons := make(map[string]string)
	for _, entry := range response.Entries {
		icons[entry.Name] = entry.Icon
	}
	for name, want := range map[string]string{"sub": iconFolder, "page.md": iconMarkdown, "data.yaml": iconYAML, "tool.py": iconPython} {
		if icons[name] != want {
			t.Errorf("navigation icon for %q = %q, want %q", name, icons[name], want)
		}
	}
}

func TestThemeAssetsAreServedWithoutInlineCode(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "README.md", "# Root\n")
	handler := newWebHandler(t, root, render.NewRenderer(), nil)
	page := requestWeb(handler, "/").Body.String()
	for _, want := range []string{
		`<script src="/_mavis/assets/theme.js"></script>`, `id="theme-toggle"`, `aria-label="Switch color theme"`,
		`<svg class="brand-mark"`, `href="/_mavis/assets/icons/mavis.svg"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page does not contain %q", want)
		}
	}
	if strings.Contains(page, "<style") || strings.Contains(page, ` style="`) {
		t.Fatal("page uses inline styles that the content security policy blocks")
	}
	for _, asset := range []string{"theme.js", "icons/mavis.svg", "icons/NOTICE.md"} {
		if response := requestWeb(handler, "/_mavis/assets/"+asset); response.Code != 200 {
			t.Errorf("asset %q status = %d", asset, response.Code)
		}
	}
	if theme := readEmbeddedText(t, "static/theme.js"); !strings.Contains(theme, "mavis.theme.v1") || !strings.Contains(theme, "prefers-color-scheme: dark") {
		t.Fatal("theme script does not honor the saved and system preference")
	}
	if script := readEmbeddedText(t, "static/app.js"); !strings.Contains(script, `writePreference("mavis.theme.v1"`) {
		t.Fatal("theme toggle does not persist its choice")
	}
}
