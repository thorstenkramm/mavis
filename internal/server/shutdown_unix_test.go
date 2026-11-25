//go:build !windows

package server

import (
	"bytes"
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestServeCancellationReleasesUnixSocketForImmediateRestart(t *testing.T) {
	path := shortSocketPath(t, "restart.sock")
	configuration := config.Default()
	configuration.Server.Socket = path
	ctx, cancel := context.WithCancel(context.Background())
	serveErrors := make(chan error, 1)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	go func() {
		serveErrors <- Serve(ctx, t.TempDir(), configuration, &stdout, &stderr)
	}()
	waitForUnix(t, path)
	cancel()

	select {
	case err := <-serveErrors:
		if err != nil {
			t.Fatalf("Serve() shutdown error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not stop after cancellation")
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("socket remains after shutdown: %v", err)
	}
	listener, err := ListenUnix(path)
	if err != nil {
		t.Fatalf("immediate unix-socket restart error = %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
}

func waitForUnix(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("unix", path, 25*time.Millisecond)
		if err == nil {
			if closeErr := connection.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("unix listener at %q did not start", path)
}
