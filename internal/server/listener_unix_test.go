//go:build !windows

package server

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/thorstenkramm/mavis/internal/config"
)

func TestListenUnixPermissionsAndOwnedCleanup(t *testing.T) {
	path := shortSocketPath(t, "mavis.sock")
	listener, err := ListenUnix(path)
	if err != nil {
		t.Fatalf("ListenUnix() error = %v", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("socket permissions = %o, want 600", info.Mode().Perm())
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("socket still exists after Close: %v", err)
	}

	restarted, err := ListenUnix(path)
	if err != nil {
		t.Fatalf("immediate ListenUnix() restart error = %v", err)
	}
	if err := restarted.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestListenUnixRemovesVerifiedStaleSocket(t *testing.T) {
	path := shortSocketPath(t, "stale.sock")
	stale := listenRawUnix(t, path)
	unixListener, ok := stale.(*net.UnixListener)
	if !ok {
		t.Fatalf("listener type = %T", stale)
	}
	unixListener.SetUnlinkOnClose(false)
	if err := stale.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("stale socket was not retained for test: %v", err)
	}

	listener, err := ListenUnix(path)
	if err != nil {
		t.Fatalf("ListenUnix(stale) error = %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestListenUnixRejectsActiveSocket(t *testing.T) {
	path := shortSocketPath(t, "active.sock")
	active := listenRawUnix(t, path)
	unixListener, ok := active.(*net.UnixListener)
	if !ok {
		t.Fatalf("listener type = %T", active)
	}
	unixListener.SetUnlinkOnClose(false)
	defer func() {
		if err := active.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Remove() error = %v", err)
		}
	}()

	_, err := ListenUnix(path)
	if err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("ListenUnix(active) error = %v", err)
	}
}

func TestListenUnixRejectsNonSocketPath(t *testing.T) {
	path := shortSocketPath(t, "ordinary-file")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ListenUnix(path)
	if err == nil || !strings.Contains(err.Error(), "not a socket") {
		t.Fatalf("ListenUnix(file) error = %v", err)
	}
	content, readErr := os.ReadFile(path)
	if readErr != nil || string(content) != "keep" {
		t.Fatalf("ordinary file changed: content = %q, error = %v", content, readErr)
	}
}

func TestUnixSocketCloseDoesNotRemoveReplacement(t *testing.T) {
	path := shortSocketPath(t, "replacement.sock")
	listener, err := ListenUnix(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "replacement" {
		t.Fatalf("replacement was removed: content = %q, error = %v", content, err)
	}
}

func TestServePrintsSocketStartupAndCleansUp(t *testing.T) {
	path := shortSocketPath(t, "serve.sock")
	configuration := config.Default()
	configuration.Server.Socket = path
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Serve(ctx, t.TempDir(), configuration, &stdout, &stderr); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Socket: "+path) || strings.Contains(stdout.String(), "http://") {
		t.Fatalf("stdout = %q, want socket path without TCP URL", stdout.String())
	}
	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("socket remains after Serve: %v", err)
	}
}

func TestListenUnixRejectsEmptyPath(t *testing.T) {
	if _, err := ListenUnix(""); err == nil {
		t.Fatal("ListenUnix(empty) error = nil")
	}
}

func TestGroupSocket(t *testing.T) {
	group, err := user.LookupGroupId(strconv.Itoa(os.Getegid()))
	if err != nil {
		t.Fatal(err)
	}
	path := shortSocketPath(t, "group.sock")
	if _, err := listenUnixWithGroup(path, "mavis-nonexistent-test-group"); err == nil {
		t.Fatal("accepted unknown group")
	}
	for _, fail := range []string{"ownership", "mode", "replacement"} {
		chown := func(p string, uid, gid int) error {
			if uid != -1 || gid != os.Getegid() {
				t.Fatalf("unexpected owner %d:%d", uid, gid)
			}
			if fail == "ownership" {
				return fs.ErrPermission
			}
			return nil
		}
		chmod := func(p string, m fs.FileMode) error {
			if m == 0o660 {
				if fail == "replacement" {
					if err := os.WriteFile(path, []byte("replacement"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				return fs.ErrPermission
			}
			return os.Chmod(p, m)
		}
		if _, err := listenUnix(path, group.Name, chown, chmod); !errors.Is(err, fs.ErrPermission) {
			t.Fatalf("%s error: %v", fail, err)
		}
		if fail == "replacement" {
			if data, err := os.ReadFile(path); err != nil || string(data) != "replacement" {
				t.Fatal("replacement not preserved")
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		} else if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("socket retained: %v", err)
		}
	}
	l, err := listenUnixWithGroup(path, group.Name)
	if err != nil {
		t.Fatal(err)
	}
	defer closeListener(t, l)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("unexpected socket stat type %T", info.Sys())
	}
	if int(st.Uid) != os.Geteuid() || int(st.Gid) != os.Getegid() || info.Mode().Perm() != 0o660 {
		t.Fatalf("unexpected socket metadata: %+v", info)
	}
}

func TestSocketPublishedAfterPermissions(t *testing.T) {
	g, err := user.LookupGroupId(strconv.Itoa(os.Getegid()))
	if err != nil {
		t.Fatal(err)
	}
	path := shortSocketPath(t, "public.sock")
	chmod := func(p string, mode fs.FileMode) error {
		if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("public socket present during setup: %v", err)
		}
		parent, err := os.Stat(filepath.Dir(p))
		if err != nil || parent.Mode().Perm()&0o077 != 0 {
			t.Fatal("staging directory not private")
		}
		return os.Chmod(p, mode)
	}
	l, err := listenUnix(path, g.Name, os.Chown, chmod)
	if err != nil {
		t.Fatal(err)
	}
	defer closeListener(t, l)
	if l.Addr().String() != path {
		t.Fatalf("address %v, want %s", l.Addr(), path)
	}
	c, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPublishSocketNearAddressLimit(t *testing.T) {
	dir := filepath.Dir(shortSocketPath(t, "s"))
	padding := len(syscall.RawSockaddrUnix{}.Path) - len(dir) - len("//s") - 2
	if padding < 1 {
		t.Skip("temporary directory already exceeds socket address limit")
	}
	parent := filepath.Join(dir, strings.Repeat("a", padding))
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "s")
	l, err := ListenUnix(path)
	if err != nil {
		t.Fatal(err)
	}
	defer closeListener(t, l)
	c, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPublishSocketDoesNotOverwriteCompetingPath(t *testing.T) {
	path := shortSocketPath(t, "collision.sock")
	chmod := func(staged string, mode fs.FileMode) error {
		if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
			return err
		}
		return os.Chmod(staged, mode)
	}
	if _, err := listenUnix(path, "", os.Chown, chmod); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("publish error: %v", err)
	}
	if b, err := os.ReadFile(path); err != nil || string(b) != "keep" {
		t.Fatalf("public path changed: %q %v", b, err)
	}
	if files, err := os.ReadDir(filepath.Dir(path)); err != nil || len(files) != 1 {
		t.Fatalf("staging cleanup: %v %v", files, err)
	}
}

func shortSocketPath(t *testing.T, name string) string {
	t.Helper()
	directory, err := os.MkdirTemp("", "mv-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Errorf("RemoveAll() error = %v", err)
		}
	})
	return filepath.Join(directory, name)
}

func listenRawUnix(t *testing.T, path string) net.Listener {
	t.Helper()
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	return listener
}
