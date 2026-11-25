package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestMergePrecedence(t *testing.T) {
	trueValue := true
	falseValue := false
	host := "localhost"
	port := 5000

	tests := []struct {
		name            string
		file            string
		overrides       Overrides
		wantHost        string
		wantPort        int
		wantHidden      bool
		wantExclusions  []string
		wantSocket      string
		wantSourceState [3]bool
	}{
		{
			name:            "file values override defaults",
			file:            "include_hidden = true\n[server]\nhost = \"example.test\"\nport = 4000\n",
			wantHost:        "example.test",
			wantPort:        4000,
			wantHidden:      true,
			wantSourceState: [3]bool{true, true, false},
		},
		{
			name:            "CLI server values override file",
			file:            "[server]\nhost = \"example.test\"\nport = 4000\n",
			overrides:       Overrides{Host: &host, Port: &port},
			wantHost:        "localhost",
			wantPort:        5000,
			wantSourceState: [3]bool{true, true, false},
		},
		{
			name:       "CLI enables hidden files",
			file:       "include_hidden = false\n",
			overrides:  Overrides{IncludeHidden: &trueValue},
			wantHost:   DefaultHost,
			wantPort:   DefaultPort,
			wantHidden: true,
		},
		{
			name:       "CLI disables hidden files",
			file:       "include_hidden = true\n",
			overrides:  Overrides{IncludeHidden: &falseValue},
			wantHost:   DefaultHost,
			wantPort:   DefaultPort,
			wantHidden: false,
		},
		{
			name:       "absent CLI preserves hidden files",
			file:       "include_hidden = true\n",
			wantHost:   DefaultHost,
			wantPort:   DefaultPort,
			wantHidden: true,
		},
		{
			name:           "exclusions are combined",
			file:           "exclude = [\"drafts/**\"]\n",
			overrides:      Overrides{Exclude: []string{"*.tmp"}},
			wantHost:       DefaultHost,
			wantPort:       DefaultPort,
			wantExclusions: []string{"drafts/**", "*.tmp"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base := loadForTest(t, test.file)
			merged, err := Merge(base, test.overrides)
			if err != nil {
				t.Fatalf("Merge() error = %v", err)
			}
			if merged.Server.Host != test.wantHost || merged.Server.Port != test.wantPort || merged.Server.Socket != test.wantSocket {
				t.Fatalf("Server = %+v, want %s:%d socket %q", merged.Server, test.wantHost, test.wantPort, test.wantSocket)
			}
			if merged.IncludeHidden != test.wantHidden {
				t.Fatalf("IncludeHidden = %v, want %v", merged.IncludeHidden, test.wantHidden)
			}
			if !reflect.DeepEqual(merged.Exclude, test.wantExclusions) {
				t.Fatalf("Exclude = %#v, want %#v", merged.Exclude, test.wantExclusions)
			}
			gotSourceState := [3]bool{merged.hostSet, merged.portSet, merged.socketSet}
			if gotSourceState != test.wantSourceState {
				t.Fatalf("source state = %v, want %v", gotSourceState, test.wantSourceState)
			}
		})
	}
}

func TestMergeDoesNotAliasExclusions(t *testing.T) {
	base := Default()
	base.Exclude = []string{"drafts/**"}
	merged, err := Merge(base, Overrides{Exclude: []string{"*.tmp"}})
	if err != nil {
		t.Fatal(err)
	}
	merged.Exclude[0] = "changed"
	if base.Exclude[0] != "drafts/**" {
		t.Fatalf("base exclusions changed: %#v", base.Exclude)
	}
}

func TestListenerConflicts(t *testing.T) {
	host := "localhost"
	port := 5000
	socket := "/tmp/mavis-test.sock"

	tests := []struct {
		name      string
		file      string
		overrides Overrides
		want      []string
	}{
		{name: "file socket and host", file: "[server]\nhost = \"localhost\"\nsocket = \"/tmp/mavis-test.sock\"\n", want: []string{"server.socket", "server.host"}},
		{name: "file socket and port", file: "[server]\nport = 4000\nsocket = \"/tmp/mavis-test.sock\"\n", want: []string{"server.socket", "server.port"}},
		{name: "CLI host with file socket", file: "[server]\nsocket = \"/tmp/mavis-test.sock\"\n", overrides: Overrides{Host: &host}, want: []string{"server.socket", "server.host"}},
		{name: "CLI port with file socket", file: "[server]\nsocket = \"/tmp/mavis-test.sock\"\n", overrides: Overrides{Port: &port}, want: []string{"server.socket", "server.port"}},
		{name: "CLI socket with file host", file: "[server]\nhost = \"localhost\"\n", overrides: Overrides{Socket: &socket}, want: []string{"server.socket", "server.host"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base := loadForTest(t, test.file)
			_, err := Merge(base, test.overrides)
			if err == nil {
				t.Fatal("Merge() error = nil, want listener conflict")
			}
			for _, want := range test.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("Merge() error = %v, want substring %q", err, want)
				}
			}
		})
	}
}

func TestSocketWithOnlyDefaultsIsValid(t *testing.T) {
	socket := "/tmp/mavis-test.sock"
	merged, err := Merge(Default(), Overrides{Socket: &socket})
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
	if merged.Server.Socket != socket {
		t.Fatalf("Socket = %q, want %q", merged.Server.Socket, socket)
	}
}

func loadForTest(t *testing.T, content string) Config {
	t.Helper()
	root := t.TempDir()
	if content != "" {
		writeConfig(t, root, content)
	}
	configuration, err := Load(root)
	if err != nil && !strings.Contains(err.Error(), "conflicting listeners") {
		t.Fatalf("Load() error = %v", err)
	}
	if err != nil {
		// Rebuild file-backed source presence for conflict tests that Load rejects.
		return Config{
			Server:    Server{Host: DefaultHost, Port: DefaultPort, Socket: "/tmp/mavis-test.sock"},
			hostSet:   strings.Contains(content, "host ="),
			portSet:   strings.Contains(content, "port ="),
			socketSet: true,
		}
	}
	return configuration
}
