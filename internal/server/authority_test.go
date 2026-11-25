package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTCPAuthority(t *testing.T) {
	for _, tc := range []struct {
		configured, bound, local, host string
		want                           int
	}{
		{"127.0.0.1", "127.0.0.1", "127.0.0.1", "localhost:3344", 204},
		{"::1", "::1", "::1", "[::1]:3344", 204},
		{"127.0.0.1", "127.0.0.1", "127.0.0.1", "unrecognized.test:3344", 421},
		{"127.0.0.1", "127.0.0.1", "127.0.0.1", "localhost:9000", 421},
		{"0.0.0.0", "0.0.0.0", "192.0.2.1", "192.0.2.1:3344", 204},
		{"docs.test", "192.0.2.1", "192.0.2.1", "docs.test:3344", 204},
		{"0.0.0.0", "0.0.0.0", "192.0.2.1", "unrecognized.test", 421},
	} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = tc.host
		r = r.WithContext(context.WithValue(r.Context(), http.LocalAddrContextKey, &net.TCPAddr{IP: net.ParseIP(tc.local), Port: 3344}))
		w := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
		guardTCPAuthority(next, tc.configured, &net.TCPAddr{IP: net.ParseIP(tc.bound), Port: 3344}).ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Errorf("%s on %s: status %d, want %d", tc.host, tc.configured, w.Code, tc.want)
		}
	}
}
