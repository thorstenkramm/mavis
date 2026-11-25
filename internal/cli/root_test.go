package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestRootRegistersOnlyServe(t *testing.T) {
	command := NewRootCommand()
	if command.Use != "mavis" {
		t.Fatalf("Use = %q, want mavis", command.Use)
	}
	commands := command.Commands()
	if len(commands) != 1 || commands[0].Name() != "serve" {
		t.Fatalf("commands = %v, want only serve", commands)
	}
	if commands[0].Aliases != nil {
		t.Fatalf("serve aliases = %v, want none", commands[0].Aliases)
	}
}

func TestServeFlagsAreLocal(t *testing.T) {
	root := NewRootCommand()
	serve, _, err := root.Find([]string{"serve"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"host", "port", "socket", "exclude", "include-hidden"} {
		if serve.Flags().Lookup(name) == nil {
			t.Errorf("serve flag --%s is not registered", name)
		}
		if root.PersistentFlags().Lookup(name) != nil {
			t.Errorf("serve flag --%s is unexpectedly persistent", name)
		}
	}
}

func TestServeRequiresExactlyOneDirectory(t *testing.T) {
	for _, args := range [][]string{{"serve"}, {"serve", "one", "two"}} {
		called := false
		_, _, err := executeCommand(args, func(context.Context, string, config.Config, io.Writer, io.Writer) error {
			called = true
			return nil
		})
		if err == nil {
			t.Fatalf("Execute(%v) error = nil", args)
		}
		if called {
			t.Fatalf("Execute(%v) called server", args)
		}
	}
}

func TestSocketGroupUsesEffectiveListener(t *testing.T) {
	for _, tc := range []struct {
		name, extraConfig string
		flags             []string
		valid             bool
	}{
		{"socket in file", `socket = "mavis.sock"`, nil, true}, {"CLI socket", "", []string{"--socket", "mavis.sock"}, true},
		{"TCP", "", nil, false}, {"conflicting TCP", `port = 4455`, []string{"--socket", "mavis.sock"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, config.FileName), []byte("[server]\nsocket_group = \"proxy\"\n"+tc.extraConfig), 0o600); err != nil {
				t.Fatal(err)
			}
			called := false
			_, _, err := executeCommand(append([]string{"serve", root}, tc.flags...), func(_ context.Context, _ string, c config.Config, _, _ io.Writer) error {
				called = true
				if c.Server.SocketGroup != "proxy" || c.Server.Socket != "mavis.sock" {
					t.Fatalf("wrong listener: %+v", c.Server)
				}
				return nil
			})
			if (err == nil) != tc.valid || called != tc.valid {
				t.Fatalf("called=%v, err=%v", called, err)
			}
		})
	}
}

func TestFormerServCommandIsRejected(t *testing.T) {
	_, _, err := executeCommand([]string{"serv", "."}, nil)
	if err == nil || !strings.Contains(err.Error(), "serve") {
		t.Fatalf("Execute(serv) error = %v, want serve guidance", err)
	}
}

func TestServeRejectsInvalidRoots(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file.md")
	if err := os.WriteFile(file, []byte("# not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{filepath.Join(t.TempDir(), "missing"), file} {
		called := false
		_, _, err := executeCommand([]string{"serve", root}, func(context.Context, string, config.Config, io.Writer, io.Writer) error {
			called = true
			return nil
		})
		if err == nil || !strings.Contains(err.Error(), "invalid content root") {
			t.Fatalf("Execute(%q) error = %v, want invalid root", root, err)
		}
		if called {
			t.Fatalf("Execute(%q) called server", root)
		}
	}
}

func TestServeUsesAbsoluteRootAndDefaults(t *testing.T) {
	root := t.TempDir()
	var gotRoot string
	var gotConfig config.Config
	_, _, err := executeCommand([]string{"serve", root}, func(_ context.Context, root string, configuration config.Config, _, _ io.Writer) error {
		gotRoot = root
		gotConfig = configuration
		return nil
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !filepath.IsAbs(gotRoot) || gotRoot != root {
		t.Fatalf("root = %q, want absolute %q", gotRoot, root)
	}
	if gotConfig.Server.Host != config.DefaultHost || gotConfig.Server.Port != config.DefaultPort || gotConfig.IncludeHidden {
		t.Fatalf("configuration = %+v, want defaults", gotConfig)
	}
}

func TestServeMergesConfigurationAndFlags(t *testing.T) {
	root := t.TempDir()
	content := []byte("exclude = [\"drafts/**\"]\ninclude_hidden = true\n[server]\nhost = \"example.test\"\nport = 4000\n")
	if err := os.WriteFile(filepath.Join(root, config.FileName), content, 0o600); err != nil {
		t.Fatal(err)
	}
	var got config.Config
	_, _, err := executeCommand([]string{
		"serve", root,
		"--host", "localhost",
		"--port", "5000",
		"--exclude", "*.tmp",
		"--exclude", "private/**",
		"--include-hidden=false",
	}, func(_ context.Context, _ string, configuration config.Config, _, _ io.Writer) error {
		got = configuration
		return nil
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Server.Host != "localhost" || got.Server.Port != 5000 || got.IncludeHidden {
		t.Fatalf("configuration = %+v, want CLI overrides", got)
	}
	if len(got.Exclude) != 3 || got.Exclude[0] != "drafts/**" || got.Exclude[2] != "private/**" {
		t.Fatalf("Exclude = %#v", got.Exclude)
	}
}

func TestServeRejectsInvalidConfigurationBeforeStartup(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte("unknown = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	_, _, err := executeCommand([]string{"serve", root}, func(context.Context, string, config.Config, io.Writer, io.Writer) error {
		called = true
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "unknown") || called {
		t.Fatalf("Execute() error = %v, called = %v", err, called)
	}
}

func TestServePropagatesServerFailure(t *testing.T) {
	want := errors.New("listener failed")
	_, _, err := executeCommand([]string{"serve", t.TempDir()}, func(context.Context, string, config.Config, io.Writer, io.Writer) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("Execute() error = %v, want %v", err, want)
	}
}

func TestDefaultCommandPrintsTCPStartup(t *testing.T) {
	port := freeTCPPort(t)
	root := t.TempDir()
	command := NewRootCommand()
	command.SetArgs([]string{"serve", root, "--port", strconv.Itoa(port)})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := command.ExecuteContext(ctx); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Serving: http://127.0.0.1:") || !strings.Contains(stdout.String(), "Root: "+root) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestDefaultCommandReportsBindFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := listener.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	}()
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("address type = %T", listener.Addr())
	}
	command := NewRootCommand()
	command.SetArgs([]string{"serve", t.TempDir(), "--port", strconv.Itoa(address.Port)})
	var stderr bytes.Buffer
	command.SetErr(&stderr)
	err = command.Execute()
	if err == nil || !strings.Contains(err.Error(), "listen on") {
		t.Fatalf("Execute() error = %v, want bind failure", err)
	}
}

func TestMainExitCodes(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Main(context.Background(), []string{"serv", "."}, &stdout, &stderr); code != 1 {
		t.Fatalf("Main(serv) = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "Error:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main(context.Background(), []string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Main(--help) = %d, want 0; stderr = %q", code, stderr.String())
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("address type = %T", listener.Addr())
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return address.Port
}

func executeCommand(args []string, serve ServeFunc) (string, string, error) {
	if serve == nil {
		serve = func(context.Context, string, config.Config, io.Writer, io.Writer) error { return nil }
	}
	command := NewRootCommandWithServe(serve)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.SetArgs(args)
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	err := command.Execute()
	return stdout.String(), stderr.String(), err
}
