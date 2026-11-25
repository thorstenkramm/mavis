package server

import (
	"bytes"
	"context"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestResolveExportDependenciesUsesExplicitExecutable(t *testing.T) {
	for _, executable := range []string{"/opt/pandoc/bin/pandoc", "pandoc-custom"} {
		t.Run(executable, func(t *testing.T) {
			calls := 0
			dependencies, err := resolveExportDependencies(config.Downloads{
				ExportDOCX: true,
				Pandoc:     executable,
			}, func(name string) (string, error) {
				calls++
				if name != executable {
					t.Fatalf("LookPath name = %q, want %q", name, executable)
				}
				return "/resolved/pandoc", nil
			})
			if err != nil {
				t.Fatalf("resolveExportDependencies() error = %v", err)
			}
			if calls != 1 || dependencies.PandocExecutable != expectedAbsolute(t, "/resolved/pandoc") {
				t.Fatalf("dependencies = %+v, LookPath calls = %d", dependencies, calls)
			}
		})
	}
}

func TestResolveExportDependenciesRejectsInvalidExplicitExecutableWithoutFallback(t *testing.T) {
	calls := 0
	_, err := resolveExportDependencies(config.Downloads{
		ExportDOCX: true,
		Pandoc:     "missing-custom-pandoc",
	}, func(name string) (string, error) {
		calls++
		if name != "missing-custom-pandoc" {
			t.Fatalf("LookPath name = %q, want explicit executable", name)
		}
		return "", exec.ErrNotFound
	})
	if err == nil || !strings.Contains(err.Error(), "downloads.pandoc") {
		t.Fatalf("resolveExportDependencies() error = %v, want downloads.pandoc error", err)
	}
	if calls != 1 {
		t.Fatalf("LookPath calls = %d, want no PATH fallback", calls)
	}
}

func TestResolveExportDependenciesDisabledSkipsLookup(t *testing.T) {
	dependencies, err := resolveExportDependencies(config.Downloads{
		ExportDOCX: false,
		Pandoc:     "/configured/but/disabled/pandoc",
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

func TestServeRejectsInvalidExplicitPandocBeforeListening(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer closeListener(t, listener)
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address type = %T", listener.Addr())
	}

	configuration := config.Default()
	configuration.Server.Port = address.Port
	configuration.Downloads.Pandoc = filepath.Join(t.TempDir(), "missing-pandoc")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = Serve(context.Background(), t.TempDir(), configuration, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "downloads.pandoc") {
		t.Fatalf("Serve() error = %v, want downloads.pandoc resolution failure", err)
	}
	if strings.Contains(err.Error(), "listen on") {
		t.Fatalf("Serve() opened listener before resolving pandoc: %v", err)
	}
}

func TestServeDisabledDOCXSkipsInvalidConfiguredExecutable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	configuration := config.Default()
	configuration.Server.Port = availablePort(t)
	configuration.Downloads.ExportDOCX = false
	configuration.Downloads.Pandoc = filepath.Join(t.TempDir(), "missing-pandoc")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Serve(ctx, t.TempDir(), configuration, &stdout, &stderr); err != nil {
		t.Fatalf("Serve() error = %v, want disabled DOCX to skip resolution", err)
	}
}

func TestResolveExportDependenciesMakesRelativeResultAbsolute(t *testing.T) {
	dependencies, err := resolveExportDependencies(config.Downloads{
		ExportDOCX: true,
		Pandoc:     filepath.Join("tools", "pandoc"),
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

func TestResolveExportDependenciesRejectsBlankExplicitExecutable(t *testing.T) {
	_, err := resolveExportDependencies(config.Downloads{
		ExportDOCX: true,
		Pandoc:     " \t ",
	}, func(string) (string, error) {
		t.Fatal("LookPath called for blank explicit executable")
		return "", nil
	})
	if err == nil || !strings.Contains(err.Error(), "downloads.pandoc") {
		t.Fatalf("resolveExportDependencies() error = %v, want downloads.pandoc error", err)
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
