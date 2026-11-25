package config

import (
	"strings"
	"testing"
)

func TestEditTemplate(t *testing.T) {
	for _, raw := range []string{"", "https://example.org/repo", "http://example.org/{path}", "https://{path}/a", "https:///{path}", "https://example.org/{path}.md", "https://example.org/{path}/{path}", "https://example.org/{other}/{path}", "https://example.org/%7Bpath%7D?q={path}", "https://example.org:99999/{path}", "https://name:secret@example.org/{path}"} {
		_, err := parseEditTemplate(raw)
		if err == nil || !strings.Contains(err.Error(), "edit_url_template") || strings.Contains(err.Error(), "secret") {
			t.Errorf("unexpected error for invalid template: %v", err)
		}
	}
	root := t.TempDir()
	c, err := Load(root)
	if err != nil || c.EditURL.URL("a.md") != "" {
		t.Fatal("expected disabled edit links")
	}
	writeConfig(t, root, `edit_url_template = "https://example.org/edit/{path}?mode=edit#top"`)
	c, err = Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.EditURL.URL("guides/Über #1%.md"); got != "https://example.org/edit/guides/%C3%9Cber%20%231%25.md?mode=edit#top" {
		t.Fatal(got)
	}
	for _, setting := range []string{`edit_url_template = 7`, `edit_url_template = ""`} {
		writeConfig(t, root, setting)
		if _, err := Load(root); err == nil {
			t.Fatal("accepted invalid setting")
		}
	}
}
