package server

import (
	"net"
	"net/http"
	"strings"
)

// Compare the HTTP authority with trusted names or literal listener addresses.
// Never resolve a request-supplied hostname, including for a wildcard bind.
func guardTCPAuthority(next http.Handler, configured string, address net.Addr) http.Handler {
	bound, port, addressErr := net.SplitHostPort(address.String())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, requestedPort, err := net.SplitHostPort(r.Host)
		if err != nil {
			host, requestedPort = r.Host, "80"
			if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
				host = host[1 : len(host)-1]
			}
		}
		localHost := bound
		if local, ok := r.Context().Value(http.LocalAddrContextKey).(*net.TCPAddr); ok {
			localHost = local.IP.String()
		}
		ip := net.ParseIP(host)
		localIP := net.ParseIP(localHost)
		valid := strings.EqualFold(host, configured) || strings.EqualFold(host, bound) ||
			(ip != nil && ip.Equal(localIP)) || (localIP.IsLoopback() && strings.EqualFold(host, "localhost"))
		if addressErr != nil || !valid || requestedPort != port {
			http.Error(w, "Unrecognized request authority", http.StatusMisdirectedRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
