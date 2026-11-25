package content

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestVisibilityPolicyForDirectRequests(t *testing.T) {
	root := policyFixture(t)
	configuration := config.Default()
	configuration.Exclude = []string{"drafts/**", ".excluded/**"}
	resolver := newResolver(t, root, configuration)

	for _, path := range []string{"/guide.md", "/docs/notes.md"} {
		if _, err := resolver.Resolve(path); err != nil {
			t.Errorf("Resolve(%q) error = %v, want eligible", path, err)
		}
	}
	for _, path := range []string{"/.mavis.conf", "/.hidden.md", "/.private/secret.md", "/drafts/secret.md", "/.excluded/secret.md"} {
		if _, err := resolver.Resolve(path); !errors.Is(err, ErrNotFound) {
			t.Errorf("Resolve(%q) error = %v, want ErrNotFound", path, err)
		}
	}
}

func TestVisibilityPolicyWithHiddenContentIncluded(t *testing.T) {
	root := policyFixture(t)
	configuration := config.Default()
	configuration.IncludeHidden = true
	configuration.Exclude = []string{".excluded/**"}
	resolver := newResolver(t, root, configuration)

	for _, path := range []string{"/.hidden.md", "/.private/secret.md", "/docs/.mavis.conf"} {
		if _, err := resolver.Resolve(path); err != nil {
			t.Errorf("Resolve(%q) error = %v, want eligible", path, err)
		}
	}
	for _, path := range []string{"/.mavis.conf", "/.excluded/secret.md"} {
		if _, err := resolver.Resolve(path); !errors.Is(err, ErrNotFound) {
			t.Errorf("Resolve(%q) error = %v, want ErrNotFound", path, err)
		}
	}
}

func TestListUsesTheSameVisibilityPolicy(t *testing.T) {
	root := policyFixture(t)
	configuration := config.Default()
	configuration.Exclude = []string{"drafts/**"}
	resolver := newResolver(t, root, configuration)

	resources, err := resolver.List("/")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	var names []string
	for _, resource := range resources {
		names = append(names, resource.Info.Name())
	}
	want := "docs guide.md"
	if strings.Join(names, " ") != want {
		t.Fatalf("List() names = %q, want %q", strings.Join(names, " "), want)
	}
}

func TestPolicyHTTPRequests(t *testing.T) {
	root := policyFixture(t)
	configuration := config.Default()
	configuration.Exclude = []string{"drafts/**"}
	resolver := newResolver(t, root, configuration)
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, err := resolver.Resolve(request.URL.EscapedPath())
		if err != nil {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})

	for _, test := range []struct {
		path string
		want int
	}{
		{path: "/guide.md", want: http.StatusOK},
		{path: "/.mavis.conf", want: http.StatusNotFound},
		{path: "/.hidden.md", want: http.StatusNotFound},
		{path: "/drafts/secret.md", want: http.StatusNotFound},
		{path: "/%2e%2e/secret", want: http.StatusNotFound},
	} {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d", response.Code, test.want)
			}
		})
	}
}

func TestPolicyChecksLogicalAndResolvedSymlinkPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}
	root := policyFixture(t)
	configuration := config.Default()
	configuration.Exclude = []string{"drafts/**", "excluded-alias"}
	resolver := newResolver(t, root, configuration)
	links := map[string]string{
		"config-alias":   ".mavis.conf",
		"hidden-alias":   ".hidden.md",
		"draft-alias":    "drafts/secret.md",
		"excluded-alias": "guide.md",
		"visible-alias":  "guide.md",
	}
	for name, target := range links {
		if err := os.Symlink(filepath.Join(root, target), filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"/config-alias", "/hidden-alias", "/draft-alias", "/excluded-alias"} {
		if _, err := resolver.Resolve(path); !errors.Is(err, ErrNotFound) {
			t.Errorf("Resolve(%q) error = %v, want ErrNotFound", path, err)
		}
	}
	if _, err := resolver.Resolve("/visible-alias"); err != nil {
		t.Fatalf("Resolve(visible-alias) error = %v", err)
	}
	resources, err := resolver.List("/")
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range resources {
		name := resource.Info.Name()
		if name != "visible-alias" && strings.HasSuffix(name, "-alias") {
			t.Errorf("List() exposed denied symlink %q", name)
		}
	}
}

func TestPolicyUsesStoredSpellingAndConfigurationIdentity(t *testing.T) {
	root := policyFixture(t)
	configuration := config.Default()
	configuration.IncludeHidden = true
	configuration.Exclude = []string{"drafts/**"}
	r := newResolver(t, root, configuration)
	for _, name := range []string{".MAVIS.CONF", "Drafts/secret.md"} {
		if _, err := r.ResolveLogical(name); !errors.Is(err, ErrNotFound) {
			t.Errorf("ResolveLogical(%q) = %v; want not found", name, err)
		}
	}
	if err := os.Link(filepath.Join(root, config.FileName), filepath.Join(root, "settings.txt")); err != nil {
		t.Skipf("filesystem does not support hard links: %v", err)
	}
	if _, err := r.ResolveLogical("settings.txt"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("configuration identity not protected: %v", err)
	}
}

func policyFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		".mavis.conf":         "include_hidden = true\n",
		".hidden.md":          "hidden",
		".private/secret.md":  "hidden",
		".excluded/secret.md": "excluded",
		"drafts/secret.md":    "excluded",
		"docs/.mavis.conf":    "nested configuration is not privileged\n",
		"docs/notes.md":       "notes",
		"guide.md":            "guide",
	}
	for path, content := range files {
		makeFile(t, filepath.Join(root, path), content)
	}
	return root
}
