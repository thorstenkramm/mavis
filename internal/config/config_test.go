package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAbsentUsesDefaults(t *testing.T) {
	configuration, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if configuration.Server.Host != DefaultHost || configuration.Server.Port != DefaultPort {
		t.Fatalf("Server = %+v, want %s:%d", configuration.Server, DefaultHost, DefaultPort)
	}
	if configuration.Server.Socket != "" || configuration.IncludeHidden || len(configuration.Exclude) != 0 {
		t.Fatalf("Load() = %+v, want safe defaults", configuration)
	}
}

func TestLoadValidConfiguration(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `exclude = ["drafts/**", "*.tmp"]
include_hidden = true

[server]
host = "localhost"
port = 4000
`)

	configuration, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !configuration.IncludeHidden {
		t.Fatal("IncludeHidden = false, want true")
	}
	if configuration.Server.Host != "localhost" || configuration.Server.Port != 4000 {
		t.Fatalf("Server = %+v, want localhost:4000", configuration.Server)
	}
	if len(configuration.Exclude) != 2 || configuration.Exclude[0] != "drafts/**" {
		t.Fatalf("Exclude = %#v", configuration.Exclude)
	}
	if !configuration.hostSet || !configuration.portSet || configuration.socketSet {
		t.Fatalf("source presence = host:%v port:%v socket:%v", configuration.hostSet, configuration.portSet, configuration.socketSet)
	}
}

func TestLoadSocketDoesNotConflictWithDefaults(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "[server]\nsocket = \"/tmp/mavis-test.sock\"\n")
	configuration, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if configuration.Server.Socket != "/tmp/mavis-test.sock" || !configuration.socketSet {
		t.Fatalf("Server = %+v, socketSet = %v", configuration.Server, configuration.socketSet)
	}
}

func TestLoadRejectsInvalidTOML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "malformed", content: "[server\n", want: "parse .mavis.conf"},
		{name: "unknown top-level key", content: "unknown = true\n", want: "unknown"},
		{name: "unknown server key", content: "[server]\nunknown = true\n", want: "unknown"},
		{name: "wrong exclude type", content: "exclude = true\n", want: "exclude"},
		{name: "wrong hidden type", content: "include_hidden = \"yes\"\n", want: "include_hidden"},
		{name: "wrong port type", content: "[server]\nport = \"3344\"\n", want: "port"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeConfig(t, root, test.content)
			_, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Load() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "empty host", content: "[server]\nhost = \"\"\n", want: "server.host"},
		{name: "blank host", content: "[server]\nhost = \"  \"\n", want: "server.host"},
		{name: "zero port", content: "[server]\nport = 0\n", want: "server.port"},
		{name: "negative port", content: "[server]\nport = -1\n", want: "server.port"},
		{name: "large port", content: "[server]\nport = 65536\n", want: "server.port"},
		{name: "empty socket", content: "[server]\nsocket = \"\"\n", want: "server.socket"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeConfig(t, root, test.content)
			_, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Load() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestLoadReportsReadFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, FileName), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "read .mavis.conf") {
		t.Fatalf("Load() error = %v, want read error", err)
	}
}

func writeConfig(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
