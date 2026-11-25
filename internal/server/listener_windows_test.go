//go:build windows

package server

import (
	"strings"
	"testing"
)

func TestListenUnixUnsupportedOnWindows(t *testing.T) {
	_, err := ListenUnix("mavis.sock")
	if err == nil || !strings.Contains(err.Error(), "not supported on Windows") || !strings.Contains(err.Error(), "TCP") {
		t.Fatalf("ListenUnix() error = %v, want actionable unsupported-platform error", err)
	}
}
