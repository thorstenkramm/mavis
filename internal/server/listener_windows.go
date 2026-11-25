//go:build windows

package server

import (
	"errors"
	"net"
)

// ListenUnix reports that Windows socket mode is unsupported.
func ListenUnix(string) (net.Listener, error) {
	return nil, errors.New("unix-socket listeners are not supported on Windows; use TCP host and port settings")
}
