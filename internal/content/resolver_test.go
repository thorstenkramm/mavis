package content

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestResolverResolvesRootDirectoriesAndFiles(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "guide", "setup.md"), "# Setup")
	resolver := newResolver(t, root, config.Default())

	tests := []struct {
		request  string
		relative string
		dir      bool
	}{
		{request: "", relative: "", dir: true},
		{request: "/", relative: "", dir: true},
		{request: "/guide/", relative: "guide", dir: true},
		{request: "guide/setup.md", relative: "guide/setup.md", dir: false},
		{request: "/guide/setup.md", relative: "guide/setup.md", dir: false},
	}
	for _, test := range tests {
		t.Run(test.request, func(t *testing.T) {
			resource, err := resolver.Resolve(test.request)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if resource.RelativePath != test.relative || resource.Info.IsDir() != test.dir {
				t.Fatalf("Resource = %+v, want relative %q dir %v", resource, test.relative, test.dir)
			}
			wantTarget := test.relative
			if wantTarget == "" {
				wantTarget = "."
			}
			if resource.TargetPath != wantTarget {
				t.Fatalf("TargetPath = %q, want %q", resource.TargetPath, wantTarget)
			}
		})
	}
}

func TestResolverRejectsAdversarialPaths(t *testing.T) {
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "docs", "guide.md"), "# Guide")
	resolver := newResolver(t, root, config.Default())

	paths := []string{
		"..",
		"../secret",
		"/../secret",
		"/..%2Fsecret",
		"/%2e%2e/secret",
		"docs/%2E%2E/secret",
		`docs\..\secret`,
		`\\server\share`,
		"//etc/passwd",
		"C:/Windows/win.ini",
		"/C:/Windows/win.ini",
		"%zz",
		"/%00",
		"docs/%2fetc/passwd",
		"docs//guide.md",
		"/./docs/guide.md",
		"/missing.md",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			_, err := resolver.Resolve(path)
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("Resolve(%q) error = %v, want ErrNotFound", path, err)
			}
		})
	}
}

func TestResolverSymlinkContainment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}
	root := t.TempDir()
	makeFile(t, filepath.Join(root, "docs", "guide.md"), "# Guide")
	outside := t.TempDir()
	makeFile(t, filepath.Join(outside, "secret.md"), "secret")
	if err := os.Symlink(filepath.Join(root, "docs", "guide.md"), filepath.Join(root, "internal-link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "external-link")); err != nil {
		t.Fatal(err)
	}
	resolver := newResolver(t, root, config.Default())

	resource, err := resolver.Resolve("/internal-link.md")
	if err != nil {
		t.Fatalf("internal symlink error = %v", err)
	}
	if resource.TargetPath != "docs/guide.md" {
		t.Fatalf("TargetPath = %q, want docs/guide.md", resource.TargetPath)
	}
	if _, err := resolver.Resolve("/external-link/secret.md"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("external symlink error = %v, want ErrNotFound", err)
	}
}

func TestNewResolverRejectsFileRoot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.md")
	makeFile(t, path, "# File")
	if _, err := NewResolver(path, config.Default()); err == nil {
		t.Fatal("NewResolver(file) error = nil")
	}
}

func newResolver(t *testing.T, root string, configuration config.Config) *Resolver {
	t.Helper()
	resolver, err := NewResolver(root, configuration)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}
	return resolver
}

func makeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
