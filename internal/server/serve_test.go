package server

import (
	"bytes"
	"context"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestServePrintsCustomTCPStartup(t *testing.T) {
	port := availablePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	configuration := config.Default()
	configuration.Server.Port = port
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	root := t.TempDir()
	if err := Serve(ctx, root, configuration, &stdout, &stderr); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Serving: http://127.0.0.1:") || !strings.Contains(stdout.String(), "Root: "+root) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want no warning", stderr.String())
	}
}

func TestServeWarnsForNonLoopback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	configuration := config.Default()
	configuration.Server.Host = "0.0.0.0"
	configuration.Server.Port = availablePort(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Serve(ctx, t.TempDir(), configuration, &stdout, &stderr); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if !strings.Contains(stderr.String(), "Anyone with network access") || !strings.Contains(stderr.String(), "authentication is not enabled") {
		t.Fatalf("stderr = %q, want exposure warning", stderr.String())
	}
}

func availablePort(t *testing.T) int {
	t.Helper()
	listener, err := ListenTCP("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	closeListener(t, listener)
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatal(err)
	}
	value, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
