package server

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/thorstenkramm/mavis/internal/config"
)

// ExportDependencies contains export capabilities resolved once for a serving session.
// An empty PandocExecutable means DOCX export is unavailable.
type ExportDependencies struct {
	PandocExecutable string
}

// ResolveExportDependencies resolves optional external export executables at startup.
func ResolveExportDependencies(downloads config.Downloads) (ExportDependencies, error) {
	return resolveExportDependencies(downloads, exec.LookPath)
}

func resolveExportDependencies(downloads config.Downloads, lookPath func(string) (string, error)) (ExportDependencies, error) {
	if !downloads.ExportDOCX {
		return ExportDependencies{}, nil
	}

	resolved, err := lookPath("pandoc")
	if err != nil {
		//nolint:nilerr // Automatic PATH discovery is optional; a miss disables DOCX.
		return ExportDependencies{}, nil
	}
	absolute, err := filepath.Abs(resolved)
	if err != nil {
		return ExportDependencies{}, fmt.Errorf("resolve pandoc from PATH: %w", err)
	}
	return ExportDependencies{PandocExecutable: absolute}, nil
}
