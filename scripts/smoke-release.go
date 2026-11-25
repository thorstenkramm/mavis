//go:build ignore

// smoke-release verifies a native Mavis binary through its public CLI and HTTP interface.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	binary := flag.String("binary", "", "path to the native Mavis binary")
	flag.Parse()
	if err := smokeTest(*binary); err != nil {
		fmt.Fprintln(os.Stderr, "release smoke test:", err)
		os.Exit(1)
	}
	fmt.Println("Release smoke test passed:", *binary)
}

func smokeTest(binary string) (result error) {
	if binary == "" {
		return fmt.Errorf("-binary is required")
	}
	root, err := os.MkdirTemp("", "mavis-release-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	writeFixture(root, "README.md", "# Release Smoke\n\n- [x] Packaged\n\n| Feature | Status |\n| --- | --- |\n| Rendering | Works |\n")
	writeFixture(root, "public.txt", "public content\n")
	writeFixture(root, ".secret.md", "secret\n")
	writeFixture(root, "drafts/private.md", "private\n")
	writeFixture(root, ".mavis.conf", "exclude = [\"drafts/**\"]\ninclude_hidden = false\n\n[server]\nhost = \"127.0.0.1\"\nport = 1\n")

	port, err := availablePort()
	if err != nil {
		return err
	}
	command := exec.Command(binary, "serve", root, "--port", strconv.Itoa(port))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return fmt.Errorf("start binary: %w", err)
	}
	done := make(chan error, 1)
	lifetime, exited := context.WithCancel(context.Background())
	go func() {
		done <- command.Wait()
		exited()
	}()
	defer func() {
		if lifetime.Err() == nil {
			_ = command.Process.Kill()
		}
		<-lifetime.Done() // Wait has joined the os/exec output-copy goroutines.
		if result != nil {
			result = fmt.Errorf("%w; stdout=%q stderr=%q", result, stdout.String(), stderr.String())
		} else if !strings.Contains(stdout.String(), "Serving:") || !strings.Contains(stdout.String(), "Root: "+root) {
			result = fmt.Errorf("startup output is incomplete: %q", stdout.String())
		}
	}()

	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	if err := waitForServer(lifetime, baseURL); err != nil {
		return fmt.Errorf("startup failed: %w", err)
	}
	body, err := request(lifetime, baseURL+"/", http.StatusOK)
	if err != nil {
		return err
	}
	for _, expected := range []string{`<h1 id="release-smoke">Release Smoke</h1>`, "<table>", `aria-label="Packaged"`} {
		if !strings.Contains(body, expected) {
			return fmt.Errorf("rendered README does not contain %q", expected)
		}
	}
	body, err = request(lifetime, baseURL+"/public.txt", http.StatusOK)
	if err != nil || body != "public content\n" {
		return fmt.Errorf("public file response = %q, %v", body, err)
	}
	for _, path := range []string{"/.mavis.conf", "/.secret.md", "/drafts/private.md", "/%2e%2e/outside.txt"} {
		if _, err := request(lifetime, baseURL+path, http.StatusNotFound); err != nil {
			return fmt.Errorf("content boundary %s: %w", path, err)
		}
	}
	if runtime.GOOS == "windows" {
		if err := checkWindowsSocket(binary); err != nil {
			return err
		}
	}
	return stopServer(command, done)
}

func writeFixture(root, name, content string) {
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		panic(err)
	}
}

func availablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("allocate smoke-test port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return 0, fmt.Errorf("release smoke-test port: %w", err)
	}
	return port, nil
}

func waitForServer(parent context.Context, baseURL string) error {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	for {
		if _, err := request(ctx, baseURL+"/", http.StatusOK); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func request(parent context.Context, url string, expectedStatus int) (string, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	closeErr := response.Body.Close()
	if readErr != nil {
		return "", readErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if response.StatusCode != expectedStatus {
		return string(body), fmt.Errorf("GET %s returned %d, want %d", url, response.StatusCode, expectedStatus)
	}
	return string(body), nil
}

func stopServer(command *exec.Cmd, done <-chan error) error {
	if runtime.GOOS == "windows" {
		if err := command.Process.Kill(); err != nil {
			return fmt.Errorf("stop Windows smoke server: %w", err)
		}
		<-done
		return nil
	}
	if err := command.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("interrupt smoke server: %w", err)
	}
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		return nil
	case <-time.After(10 * time.Second):
		_ = command.Process.Kill()
		return fmt.Errorf("graceful shutdown timed out")
	}
}

func checkWindowsSocket(binary string) error {
	root, err := os.MkdirTemp("", "mavis-windows-socket-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, "serve", root, "--socket", filepath.Join(root, "mavis.sock")).CombinedOutput()
	if err == nil {
		return fmt.Errorf("Windows socket mode unexpectedly started")
	}
	if !strings.Contains(string(output), "unix-socket listeners are not supported on Windows") {
		return fmt.Errorf("Windows socket error is not actionable: %q", output)
	}
	return nil
}
