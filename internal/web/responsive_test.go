package web

import (
	"strings"
	"testing"
)

func TestResponsiveNavigationControlsAndOrdinaryHistory(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "README.md", "# Responsive\n")
	handler := newWebHandler(t, root, nilRenderer{}, nil)
	response := requestWeb(handler, "/")
	for _, want := range []string{
		`id="navigation-toggle"`, `aria-controls="file-navigation"`, `aria-expanded="false"`,
		`id="navigation-close"`, `id="navigation-backdrop"`, `href="#main-content"`,
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("page does not contain responsive control %q", want)
		}
	}

	script := readEmbeddedText(t, "static/app.js")
	for _, unwanted := range []string{"pushState", "replaceState", "popstate", "location.href", "fetch("} {
		if strings.Contains(script, unwanted) {
			t.Errorf("JavaScript installs client routing behavior %q", unwanted)
		}
	}
	for _, want := range []string{"Escape", "aria-expanded", "addEventListener"} {
		if !strings.Contains(script, want) {
			t.Errorf("JavaScript does not contain %q", want)
		}
	}
}

func TestResponsiveCSSIncludesMobileDrawerAndOverflowGuards(t *testing.T) {
	stylesheet := readEmbeddedText(t, "static/app.css")
	for _, want := range []string{
		"@media (max-width: 48rem)", "body.nav-open .sidebar", "overflow-x: hidden",
		"max-width: 100%", "overflow-x: auto", ":focus-visible", "prefers-reduced-motion",
		"min-width: 44px", "min-height: 44px",
	} {
		if !strings.Contains(stylesheet, want) {
			t.Errorf("stylesheet does not contain %q", want)
		}
	}
}

func readEmbeddedText(t *testing.T, name string) string {
	t.Helper()
	data, err := embedded.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
