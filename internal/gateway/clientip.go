package gateway

import (
	"net"
	"net/http"
	"strings"
)

func getClientIP(r *http.Request, headers []string) net.IP {
	for _, header := range headers {
		parts := strings.Split(r.Header.Get(header), ",")
		if len(parts) > 0 {
			ip := net.ParseIP(strings.TrimSpace(parts[0]))
			if ip != nil {
				return ip
			}
		}
	}

	// As a last resort, use RemoteAddr.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil
	}

	return net.ParseIP(host)
}
