package rule

import (
	"net"
	"net/http"
	"strings"
)

type HostRewriteRule struct{ From, To string }

func (r *HostRewriteRule) Apply(req *http.Request) error {
	host, port := splitHostPort(req.URL.Host)
	if hostRewriteMatch(host, r.From) {
		// Rewrite always replaces the full hostname with To.
		// Subdomains are not preserved (e.g. api.example.com -> example.net).
		rewritten := r.To
		if _, toPort := splitHostPort(rewritten); toPort == "" && port != "" {
			rewritten = net.JoinHostPort(rewritten, port)
		}
		req.URL.Host = rewritten
		req.Host = rewritten
	}
	return nil
}

func hostRewriteMatch(host, from string) bool {
	if from == "" {
		return false
	}
	if host == from {
		return true
	}
	// Suffix match with label boundary:
	// "lavender.5ch.net" matches "5ch.net", but "evil5ch.net" does not.
	return strings.HasSuffix(host, "."+from)
}

func splitHostPort(hostport string) (host, port string) {
	h, p, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport, ""
	}
	return h, p
}
