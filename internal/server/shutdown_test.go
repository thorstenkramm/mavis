package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestServeCancellationReleasesTCPForImmediateRestart(t *testing.T) {
	port := availablePort(t)
	configuration := config.Default()
	configuration.Server.Port = port
	ctx, cancel := context.WithCancel(context.Background())
	serveErrors := make(chan error, 1)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	go func() {
		serveErrors <- Serve(ctx, t.TempDir(), configuration, &stdout, &stderr)
	}()
	waitForTCP(t, port)
	cancel()

	select {
	case err := <-serveErrors:
		if err != nil {
			t.Fatalf("Serve() shutdown error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not stop after cancellation")
	}

	listener, err := ListenTCP("127.0.0.1", port)
	if err != nil {
		t.Fatalf("immediate TCP restart error = %v", err)
	}
	closeListener(t, listener)
}

func TestServeHTTPShutdownIsBounded(t *testing.T) {
	listener, err := ListenTCP("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- serveHTTP(ctx, listener, handler, 50*time.Millisecond)
	}()

	address := "http://" + listener.Addr().String()
	requestErrors := make(chan error, 1)
	go func() {
		response, err := http.Get(address) //nolint:bodyclose // The forced-close test receives no response.
		if err == nil {
			if closeErr := response.Body.Close(); closeErr != nil {
				requestErrors <- closeErr
				return
			}
		}
		requestErrors <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("blocking handler did not start")
	}
	cancel()

	select {
	case err := <-serveErrors:
		if err == nil || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("serveHTTP() error = %v, want deadline", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveHTTP() exceeded its shutdown bound")
	}
	if err := <-requestErrors; err == nil {
		t.Fatal("blocked request unexpectedly completed successfully")
	}
}

func waitForTCP(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 25*time.Millisecond)
		if err == nil {
			if closeErr := connection.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("TCP listener on port %d did not start", port)
}
