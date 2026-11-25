package server

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/thorstenkramm/mavis/internal/config"
)

// ListenTCP creates a TCP listener. Port zero is allowed for isolated tests.
func ListenTCP(host string, port int) (net.Listener, error) {
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("invalid TCP port %d", port)
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", address, err)
	}
	return listener, nil
}

// IsLoopbackHost reports whether every resolved address is loopback.
func IsLoopbackHost(host string) (bool, error) {
	plainHost := strings.Trim(host, "[]")
	if zone := strings.LastIndex(plainHost, "%"); zone >= 0 {
		plainHost = plainHost[:zone]
	}
	if address := net.ParseIP(plainHost); address != nil {
		return address.IsLoopback(), nil
	}
	addresses, err := net.LookupIP(host)
	if err != nil {
		return false, fmt.Errorf("resolve server.host %q: %w", host, err)
	}
	if len(addresses) == 0 {
		return false, fmt.Errorf("resolve server.host %q: no addresses", host)
	}
	for _, address := range addresses {
		if !address.IsLoopback() {
			return false, nil
		}
	}
	return true, nil
}

// TCPURL returns the user-facing URL for a bound TCP listener.
func TCPURL(configuration config.Server, address net.Addr) string {
	if configuration.Host == config.DefaultHost && configuration.Port == config.DefaultPort {
		return "http://localhost:3344"
	}
	return "http://" + address.String()
}
