package server

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestResolveExportDependenciesDiscoversPandocInPATH(t *testing.T) {
	tests := []struct {
		name           string
		lookupError    error
		wantExecutable string
	}{
		{name: "found", wantExecutable: "/usr/local/bin/pandoc"},
		{name: "not found", lookupError: exec.ErrNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			dependencies, err := resolveExportDependencies(config.Default().Downloads, func(name string) (string, error) {
				calls++
				if name != "pandoc" {
					t.Fatalf("LookPath name = %q, want pandoc", name)
				}
				return test.wantExecutable, test.lookupError
			})
			if err != nil {
				t.Fatalf("resolveExportDependencies() error = %v", err)
			}
			if calls != 1 {
				t.Fatalf("LookPath calls = %d, want 1", calls)
			}
			wantExecutable := ""
			if test.wantExecutable != "" {
				wantExecutable = expectedAbsolute(t, test.wantExecutable)
			}
			if dependencies.PandocExecutable != wantExecutable {
				t.Fatalf("PandocExecutable = %q, want %q", dependencies.PandocExecutable, wantExecutable)
			}
		})
	}
}

func TestResolveExportDependenciesDisabledSkipsLookup(t *testing.T) {
	dependencies, err := resolveExportDependencies(config.Downloads{
		ExportDOCX: false,
	}, func(string) (string, error) {
		t.Fatal("LookPath called while DOCX is disabled")
		return "", nil
	})
	if err != nil {
		t.Fatalf("resolveExportDependencies() error = %v", err)
	}
	if dependencies.PandocExecutable != "" {
		t.Fatalf("PandocExecutable = %q, want unavailable", dependencies.PandocExecutable)
	}
}

func TestResolvedPandocChangesOnlyOnNextResolution(t *testing.T) {
	availableExecutable := "/first/pandoc"
	lookPath := func(string) (string, error) {
		return availableExecutable, nil
	}
	first, err := resolveExportDependencies(config.Default().Downloads, lookPath)
	if err != nil {
		t.Fatal(err)
	}

	availableExecutable = "/second/pandoc"
	if first.PandocExecutable != expectedAbsolute(t, "/first/pandoc") {
		t.Fatalf("session executable changed to %q", first.PandocExecutable)
	}
	second, err := resolveExportDependencies(config.Default().Downloads, lookPath)
	if err != nil {
		t.Fatal(err)
	}
	if second.PandocExecutable != expectedAbsolute(t, "/second/pandoc") {
		t.Fatalf("executable after restart resolution = %q, want /second/pandoc", second.PandocExecutable)
	}
}

func TestServeWithDOCXDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	configuration := config.Default()
	configuration.Server.Port = availablePort(t)
	configuration.Downloads.ExportDOCX = false
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Serve(ctx, t.TempDir(), configuration, &stdout, &stderr); err != nil {
		t.Fatalf("Serve() error = %v, want disabled DOCX to skip resolution", err)
	}
}

func TestResolveExportDependenciesMakesRelativeResultAbsolute(t *testing.T) {
	dependencies, err := resolveExportDependencies(config.Downloads{
		ExportDOCX: true,
	}, func(string) (string, error) {
		return filepath.Join("tools", "pandoc"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(dependencies.PandocExecutable) {
		t.Fatalf("cached executable is not absolute: %q", dependencies.PandocExecutable)
	}
}

func expectedAbsolute(t *testing.T, filename string) string {
	t.Helper()
	absolute, err := filepath.Abs(filename)
	if err != nil {
		t.Fatal(err)
	}
	return absolute
}
