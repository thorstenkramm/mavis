package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
	"github.com/thorstenkramm/mavis/internal/content"
)

func TestLazyNavigationListsOnlyEligibleEntries(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "docs/visible.md", "# Visible\n")
	writeWebFile(t, root, "docs/nested/child.txt", "child\n")
	writeWebFile(t, root, "docs/.hidden.md", "hidden\n")
	writeWebFile(t, root, "drafts/private.md", "private\n")
	writeWebFile(t, root, ".mavis.conf", "private = true\n")
	configuration := config.Default()
	configuration.Exclude = []string{"drafts/**"}
	resolver, err := content.NewResolver(root, configuration)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandlerWithRenderer(resolver, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	response := requestWeb(handler, navigationPath+"?path=docs")
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("navigation response = %d, headers %+v", response.Code, response.Header())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
	var body navigationResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Entries) != 2 {
		t.Fatalf("entries = %+v", body.Entries)
	}
	for _, entry := range body.Entries {
		if entry.Name == ".hidden.md" || strings.Contains(entry.RelativePath, "drafts") || strings.Contains(entry.RelativePath, root) {
			t.Errorf("ineligible or private entry = %+v", entry)
		}
	}
	if body.Entries[0].RelativePath != "docs/nested" || !body.Entries[0].Directory || body.Entries[0].URL != "/docs/nested/" {
		t.Errorf("nested directory = %+v", body.Entries[0])
	}
	if body.Entries[1].RelativePath != "docs/visible.md" || body.Entries[1].Directory || body.Entries[1].URL != "/docs/visible.md" {
		t.Errorf("visible document = %+v", body.Entries[1])
	}
	if strings.Contains(response.Body.String(), root) || strings.Contains(response.Body.String(), ".mavis.conf") {
		t.Fatalf("response exposed filesystem details: %s", response.Body.String())
	}
}

func TestLazyNavigationRejectsInvalidPathsAndMethods(t *testing.T) {
	root := t.TempDir()
	writeWebFile(t, root, "docs/guide.md", "# Guide\n")
	handler := newWebHandler(t, root, nil, nil)
	for _, path := range []string{"../outside", "docs/guide.md", "/absolute"} {
		response := requestWeb(handler, navigationPath+"?path="+path)
		if response.Code != http.StatusNotFound {
			t.Errorf("path %q status = %d", path, response.Code)
		}
	}
	request := httptest.NewRequest(http.MethodPost, navigationPath, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST response = %d, headers %+v", response.Code, response.Header())
	}
}
