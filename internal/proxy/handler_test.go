package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"rewproxy/internal/proxy"
	"rewproxy/internal/rule"
)

// newProxyServer starts a proxy with the given pipeline and returns its URL.
func newProxyServer(t *testing.T, pipeline rule.Pipeline) *httptest.Server {
	t.Helper()
	h := &proxy.Handler{Pipeline: pipeline}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

// newUpstream starts a fake upstream that records the Host and headers of each request.
func newUpstream(t *testing.T) (*httptest.Server, *http.Request) {
	t.Helper()
	var captured http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = *r
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, &captured
}

func clientViaProxy(proxyURL string) *http.Client {
	u, _ := url.Parse(proxyURL)
	return &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(u)},
	}
}

func TestHandleHTTP_noRules(t *testing.T) {
	upstream, captured := newUpstream(t)
	proxySrv := newProxyServer(t, nil)

	client := clientViaProxy(proxySrv.URL)
	resp, err := client.Get(upstream.URL + "/hello")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if captured.URL.Path != "/hello" {
		t.Errorf("path = %q, want /hello", captured.URL.Path)
	}
}

func TestHandleHTTP_headerSet(t *testing.T) {
	upstream, captured := newUpstream(t)
	pipeline := rule.Pipeline{
		&rule.HeaderSetRule{Name: "X-Test", Value: "rewproxy"},
	}
	proxySrv := newProxyServer(t, pipeline)

	client := clientViaProxy(proxySrv.URL)
	resp, err := client.Get(upstream.URL + "/")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if got := captured.Header.Get("X-Test"); got != "rewproxy" {
		t.Errorf("X-Test = %q, want %q", got, "rewproxy")
	}
}

func TestHandleHTTP_hostRewrite(t *testing.T) {
	upstream, captured := newUpstream(t)

	// Parse the upstream host so we can rewrite to it.
	upstreamURL, _ := url.Parse(upstream.URL)
	upstreamHost := upstreamURL.Host

	pipeline := rule.Pipeline{
		&rule.HostRewriteRule{From: "old.example.com", To: upstreamHost},
	}
	proxySrv := newProxyServer(t, pipeline)

	// Send a request to old.example.com via the proxy; the proxy should
	// rewrite the host to the real upstream before dialing.
	proxyURL, _ := url.Parse(proxySrv.URL)
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}
	req, _ := http.NewRequest("GET", "http://old.example.com/rewritten", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if captured.URL.Path != "/rewritten" {
		t.Errorf("path = %q, want /rewritten", captured.URL.Path)
	}
}

func TestHandleTunnel_connect(t *testing.T) {
	// Fake TLS upstream — client will CONNECT through the proxy to reach it.
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	proxySrv := newProxyServer(t, nil)

	proxyURL, _ := url.Parse(proxySrv.URL)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: upstream.Client().Transport.(*http.Transport).TLSClientConfig,
		},
	}

	resp, err := client.Get(upstream.URL + "/tunnel")
	if err != nil {
		t.Fatalf("GET via CONNECT: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHandleTunnel_hostRewrite(t *testing.T) {
	// Fake TLS upstream.
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	upstreamURL, _ := url.Parse(upstream.URL)
	upstreamHost := upstreamURL.Hostname()

	pipeline := rule.Pipeline{
		&rule.HostRewriteRule{From: "old.example.com", To: upstreamHost},
	}
	proxySrv := newProxyServer(t, pipeline)

	proxyURL, _ := url.Parse(proxySrv.URL)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: upstream.Client().Transport.(*http.Transport).TLSClientConfig,
		},
	}

	// Request goes to old.example.com:port; proxy rewrites host to upstreamHost, keeps port.
	targetURL := "https://old.example.com:" + upstreamURL.Port() + "/rewritten"
	resp, err := client.Get(targetURL)
	if err != nil {
		t.Fatalf("GET via CONNECT with host rewrite: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
