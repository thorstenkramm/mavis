package server

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

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

	executable := downloads.Pandoc
	if executable == "" {
		executable = "pandoc"
		resolved, err := lookPath(executable)
		if err != nil {
			//nolint:nilerr // Automatic PATH discovery is optional; a miss disables DOCX.
			return ExportDependencies{}, nil
		}
		absolute, absErr := filepath.Abs(resolved)
		if absErr != nil {
			return ExportDependencies{}, fmt.Errorf("resolve downloads.pandoc %q: %w", resolved, absErr)
		}
		return ExportDependencies{PandocExecutable: absolute}, nil
	}
	if strings.TrimSpace(executable) == "" {
		return ExportDependencies{}, fmt.Errorf("downloads.pandoc must not be empty")
	}

	resolved, err := lookPath(executable)
	if err != nil {
		return ExportDependencies{}, fmt.Errorf("resolve downloads.pandoc %q: %w", executable, err)
	}
	absolute, err := filepath.Abs(resolved)
	if err != nil {
		return ExportDependencies{}, fmt.Errorf("resolve downloads.pandoc %q: %w", resolved, err)
	}
	return ExportDependencies{PandocExecutable: absolute}, nil
}
