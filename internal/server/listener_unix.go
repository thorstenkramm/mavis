//go:build !windows

package server

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

type unixSocket struct {
	net.Listener
	path      string
	info      os.FileInfo
	closeOnce sync.Once
	closeErr  error
}

// ListenUnix creates a local-only Unix-socket listener.
func ListenUnix(path string) (net.Listener, error) {
	if path == "" {
		return nil, errors.New("unix-socket path must not be empty")
	}
	if err := removeStaleSocket(path); err != nil {
		return nil, err
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on unix socket %q: %w", path, err)
	}
	unixListener, ok := listener.(*net.UnixListener)
	if !ok {
		if closeErr := listener.Close(); closeErr != nil {
			return nil, errors.Join(errors.New("unix listener has unexpected type"), closeErr)
		}
		return nil, errors.New("unix listener has unexpected type")
	}
	// Ownership-aware Close performs unlinking after checking the path identity.
	unixListener.SetUnlinkOnClose(false)
	info, err := os.Lstat(path)
	if err != nil {
		if closeErr := listener.Close(); closeErr != nil {
			return nil, errors.Join(fmt.Errorf("inspect unix socket %q: %w", path, err), closeErr)
		}
		return nil, fmt.Errorf("inspect unix socket %q: %w", path, err)
	}
	socket := &unixSocket{Listener: listener, path: path, info: info}
	if err := os.Chmod(path, 0o600); err != nil {
		if closeErr := socket.Close(); closeErr != nil {
			return nil, errors.Join(fmt.Errorf("secure unix socket %q: %w", path, err), closeErr)
		}
		return nil, fmt.Errorf("secure unix socket %q: %w", path, err)
	}
	return socket, nil
}

func (socket *unixSocket) Close() error {
	socket.closeOnce.Do(func() {
		listenerErr := socket.Listener.Close()
		current, err := os.Lstat(socket.path)
		if errors.Is(err, fs.ErrNotExist) {
			socket.closeErr = listenerErr
			return
		}
		if err != nil {
			socket.closeErr = errors.Join(listenerErr, fmt.Errorf("inspect owned unix socket: %w", err))
			return
		}
		if current.Mode()&os.ModeSocket == 0 || !os.SameFile(socket.info, current) {
			socket.closeErr = listenerErr
			return
		}
		if err := os.Remove(socket.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			socket.closeErr = errors.Join(listenerErr, fmt.Errorf("remove owned unix socket: %w", err))
			return
		}
		socket.closeErr = listenerErr
	})
	return socket.closeErr
}

func removeStaleSocket(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect unix socket %q: %w", path, err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("unix-socket path %q already exists and is not a socket", path)
	}

	connection, err := net.DialTimeout("unix", path, 250*time.Millisecond)
	if err == nil {
		if closeErr := connection.Close(); closeErr != nil {
			return fmt.Errorf("close unix-socket probe %q: %w", path, closeErr)
		}
		return fmt.Errorf("unix socket %q is already in use", path)
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		return fmt.Errorf("cannot verify whether unix socket %q is stale: %w", path, err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove stale unix socket %q: %w", path, err)
	}
	return nil
}
