package proxy

import (
	"io"
	"log"
	"net"
	"net/http"

	"rewproxy/internal/rule"
)

// splitHostPort splits host:port, returning host and port separately.
// If there is no port, port is empty.
func splitHostPort(hostport string) (host, port string) {
	h, p, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport, ""
	}
	return h, p
}

// hopByHopHeaders are stripped before forwarding.
var hopByHopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// Handler is a forward proxy handler supporting plain HTTP and HTTPS CONNECT tunneling.
type Handler struct {
	Pipeline  rule.Pipeline
	Transport http.RoundTripper // defaults to http.DefaultTransport
	AccessLog bool              // when true, logs one line per request
}

func (h *Handler) transport() http.RoundTripper {
	if h.Transport != nil {
		return h.Transport
	}
	return http.DefaultTransport
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		h.handleTunnel(w, r)
	} else {
		h.handleHTTP(w, r)
	}
}

func (h *Handler) handleHTTP(w http.ResponseWriter, r *http.Request) {
	outReq := r.Clone(r.Context())
	if outReq.URL.Scheme == "" {
		outReq.URL.Scheme = "http"
	}
	if outReq.URL.Host == "" {
		outReq.URL.Host = r.Host
	}

	for _, hdr := range hopByHopHeaders {
		outReq.Header.Del(hdr)
	}
	outReq.RequestURI = ""

	if err := h.Pipeline.Apply(outReq); err != nil {
		log.Printf("rule pipeline error: %v", err)
		http.Error(w, "rule pipeline error: "+err.Error(), http.StatusBadGateway)
		return
	}

	resp, err := h.transport().RoundTrip(outReq)
	if err != nil {
		log.Printf("upstream error: %v", err)
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for _, hdr := range hopByHopHeaders {
		resp.Header.Del(hdr)
	}
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body) //nolint:errcheck

	if h.AccessLog {
		log.Printf("ACCESS method=%s host=%s path=%s status=%d", r.Method, r.Host, r.URL.Path, resp.StatusCode)
	}
}

func (h *Handler) handleTunnel(w http.ResponseWriter, r *http.Request) {
	// r.Host is "host:port" for CONNECT requests.
	// Extract the host part so that HostRewriteRule (which matches bare hostnames)
	// can compare correctly, then reattach the port after applying rules.
	host, port := splitHostPort(r.Host)

	synthetic, err := http.NewRequest("CONNECT", "https://"+host, nil)
	if err != nil {
		log.Printf("tunnel: bad request: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	synthetic.URL.Host = host
	if err := h.Pipeline.Apply(synthetic); err != nil {
		log.Printf("tunnel: rule pipeline error: %v", err)
		http.Error(w, "rule pipeline error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Reattach port (use rewritten host's port if the rule changed it, else original).
	rewrittenHost, rewrittenPort := splitHostPort(synthetic.URL.Host)
	if rewrittenPort != "" {
		port = rewrittenPort
	}
	target := net.JoinHostPort(rewrittenHost, port)

	upstreamConn, err := net.Dial("tcp", target)
	if err != nil {
		log.Printf("tunnel: upstream dial error: %v", err)
		http.Error(w, "upstream dial error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer upstreamConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Printf("tunnel: hijacking not supported")
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Printf("tunnel: hijack error: %v", err)
		http.Error(w, "hijack error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		log.Printf("tunnel: write 200: %v", err)
		return
	}

	if h.AccessLog {
		log.Printf("ACCESS method=CONNECT host=%s status=200", r.Host)
	}

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(upstreamConn, clientConn) //nolint:errcheck
		done <- struct{}{}
	}()
	go func() {
		io.Copy(clientConn, upstreamConn) //nolint:errcheck
		done <- struct{}{}
	}()
	<-done
}
