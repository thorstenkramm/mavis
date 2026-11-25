package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestServeHandlesRenderedHTTPRequests(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Served by Mavis\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	port := availablePort(t)
	configuration := config.Default()
	configuration.Server.Port = port
	ctx, cancel := context.WithCancel(context.Background())
	serveErrors := make(chan error, 1)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	go func() {
		serveErrors <- Serve(ctx, root, configuration, &stdout, &stderr)
	}()
	waitForTCP(t, port)

	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port)) //nolint:bodyclose // Closed below.
	if err != nil {
		cancel()
		if serveErr := <-serveErrors; serveErr != nil {
			t.Logf("Serve() after request failure = %v", serveErr)
		}
		t.Fatalf("GET served documentation error = %v", err)
	}
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	cancel()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read response: read = %v, close = %v", readErr, closeErr)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `<h1 id="served-by-mavis">Served by Mavis</h1>`) {
		t.Fatalf("response = %d %s", response.StatusCode, body)
	}
	if response.Header.Get("Cache-Control") != "no-store" || response.Header.Get("Content-Security-Policy") == "" {
		t.Fatalf("response headers = %+v", response.Header)
	}
	select {
	case err := <-serveErrors:
		if err != nil {
			t.Fatalf("Serve() shutdown error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not stop after HTTP test")
	}
}
