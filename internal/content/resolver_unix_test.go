//go:build !windows

package content

import (
	"errors"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestResolverRejectsSpecialFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pipe")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	resolver := newResolver(t, root, config.Default())
	if _, err := resolver.Resolve("/pipe"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Resolve(pipe) error = %v, want ErrNotFound", err)
	}
}
