package server

import (
	"net"
	"strings"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestListenTCPAndURL(t *testing.T) {
	listener, err := ListenTCP("127.0.0.1", 0)
	if err != nil {
		t.Fatalf("ListenTCP() error = %v", err)
	}
	defer closeListener(t, listener)

	if got := TCPURL(config.Server{Host: "127.0.0.1", Port: 4000}, listener.Addr()); got != "http://"+listener.Addr().String() {
		t.Fatalf("TCPURL() = %q", got)
	}
	if got := TCPURL(config.Server{Host: config.DefaultHost, Port: config.DefaultPort}, listener.Addr()); got != "http://localhost:3344" {
		t.Fatalf("default TCPURL() = %q", got)
	}
}

func TestListenTCPReportsBindFailure(t *testing.T) {
	listener, err := ListenTCP("127.0.0.1", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer closeListener(t, listener)
	tcpAddress, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address type = %T, want *net.TCPAddr", listener.Addr())
	}
	port := tcpAddress.Port

	_, err = ListenTCP("127.0.0.1", port)
	if err == nil || !strings.Contains(err.Error(), "listen on") {
		t.Fatalf("second ListenTCP() error = %v, want bind diagnostic", err)
	}
}

func TestListenTCPRejectsInvalidPort(t *testing.T) {
	if _, err := ListenTCP("127.0.0.1", 65536); err == nil {
		t.Fatal("ListenTCP() error = nil, want invalid-port error")
	}
}

func TestIsLoopbackHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{host: "127.0.0.1", want: true},
		{host: "::1", want: true},
		{host: "[::1]", want: true},
		{host: "localhost", want: true},
		{host: "0.0.0.0", want: false},
		{host: "::", want: false},
	}
	for _, test := range tests {
		t.Run(test.host, func(t *testing.T) {
			got, err := IsLoopbackHost(test.host)
			if err != nil {
				t.Fatalf("IsLoopbackHost() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("IsLoopbackHost(%q) = %v, want %v", test.host, got, test.want)
			}
		})
	}
}

func closeListener(t *testing.T, listener net.Listener) {
	t.Helper()
	if err := listener.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
