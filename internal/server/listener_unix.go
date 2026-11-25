//go:build !windows

package server

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
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

func listenUnixWithGroup(path, group string) (net.Listener, error) {
	return listenUnix(path, group, os.Chown, os.Chmod)
}

func listenUnix(path, group string, chown func(string, int, int) error, chmod func(string, fs.FileMode) error) (net.Listener, error) {
	if path == "" {
		return nil, errors.New("unix-socket path must not be empty")
	}
	gid := -1
	if group != "" {
		g, err := user.LookupGroup(group)
		if err != nil {
			return nil, fmt.Errorf("resolve server.socket_group %q: %w", group, err)
		}
		gid, err = strconv.Atoi(g.Gid)
		if err != nil || gid < 0 {
			return nil, fmt.Errorf("invalid GID for server.socket_group %q", group)
		}
	}
	if err := removeStaleSocket(path); err != nil {
		return nil, err
	}
	// Bind in an owner-only directory on the destination filesystem. This
	// keeps even queued connections out until the final mode and GID apply,
	// without changing the process-wide umask.
	stage, err := os.MkdirTemp(filepath.Dir(path), ".mavis-socket-")
	if err != nil {
		return nil, fmt.Errorf("stage unix socket %q: %w", path, err)
	}
	defer removeSocketStage(stage)
	staged := filepath.Join(stage, "s")
	listener, err := bindStagedSocket(staged, path)
	if err != nil {
		return nil, fmt.Errorf("listen on unix socket %q: %w", path, err)
	}
	listener.SetUnlinkOnClose(false)
	defer removeSocketStage(staged)
	info, err := os.Lstat(staged)
	if err != nil {
		if closeErr := listener.Close(); closeErr != nil {
			return nil, errors.Join(fmt.Errorf("inspect unix socket %q: %w", path, err), closeErr)
		}
		return nil, fmt.Errorf("inspect unix socket %q: %w", path, err)
	}
	socket := &unixSocket{Listener: listener, path: path, info: info}
	err = chmod(staged, 0o600)
	if err == nil && gid >= 0 {
		if err = chown(staged, -1, gid); err != nil {
			err = fmt.Errorf("assign socket group %q: %w", group, err)
		} else if err = chmod(staged, 0o660); err != nil {
			err = fmt.Errorf("set group socket permissions: %w", err)
		}
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("secure unix socket %q: %w", path, err), listener.Close())
	}
	// Publish atomically without overwriting a competing listener or file.
	if err := os.Link(staged, path); err != nil {
		return nil, errors.Join(fmt.Errorf("publish unix socket %q: %w", path, err), listener.Close())
	}
	return socket, nil
}

func bindStagedSocket(staged, public string) (*net.UnixListener, error) {
	limit := len(syscall.RawSockaddrUnix{}.Path)
	if len(public) >= limit {
		return nil, syscall.ENAMETOOLONG
	}
	if len(staged) >= limit {
		// A valid public address may have a parent too long for the extra
		// staging component. Bind through a short, private directory alias.
		alias, err := os.MkdirTemp("/tmp", "mv-")
		if err != nil {
			return nil, err
		}
		defer removeSocketStage(alias)
		dir, err := filepath.Abs(filepath.Dir(staged))
		if err != nil {
			return nil, err
		}
		if err := os.Symlink(dir, filepath.Join(alias, "d")); err != nil {
			return nil, err
		}
		defer removeSocketStage(filepath.Join(alias, "d"))
		staged = filepath.Join(alias, "d", filepath.Base(staged))
	}
	return net.ListenUnix("unix", &net.UnixAddr{Net: "unix", Name: staged})
}

func removeSocketStage(path string) {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Printf("clean up temporary Unix socket path: %v", err)
	}
}

func (socket *unixSocket) Addr() net.Addr { return &net.UnixAddr{Net: "unix", Name: socket.path} }

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
