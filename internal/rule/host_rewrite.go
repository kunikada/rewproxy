package rule

import (
	"net"
	"net/http"
	"strings"
)

type HostRewriteRule struct {
	From              string
	To                string
	PreserveSubdomain bool
}

func (r *HostRewriteRule) Apply(req *http.Request) error {
	host, port := splitHostPort(req.URL.Host)
	if hostRewriteMatch(host, r.From) {
		rewritten := r.To
		// If PreserveSubdomain is set and host has a subdomain prefix, prepend it to To.
		// e.g. "sub.domain-a.com" -> "sub.domain-b.com" when From="domain-a.com".
		if r.PreserveSubdomain && host != r.From {
			prefix := strings.TrimSuffix(host, "."+r.From)
			rewritten = prefix + "." + r.To
		}
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
