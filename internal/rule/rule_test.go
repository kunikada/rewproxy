package rule_test

import (
	"net/http"
	"testing"

	"rewproxy/internal/rule"
)

func TestHostRewriteRule_match(t *testing.T) {
	r := &rule.HostRewriteRule{From: "old.example.com", To: "example.com"}
	req, _ := http.NewRequest("GET", "http://old.example.com/path", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Host; got != "example.com" {
		t.Errorf("URL.Host = %q, want %q", got, "example.com")
	}
	if got := req.Host; got != "example.com" {
		t.Errorf("Host = %q, want %q", got, "example.com")
	}
}

func TestHostRewriteRule_nomatch(t *testing.T) {
	r := &rule.HostRewriteRule{From: "old.example.com", To: "example.com"}
	req, _ := http.NewRequest("GET", "http://other.example.com/path", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Host; got != "other.example.com" {
		t.Errorf("URL.Host = %q, want %q", got, "other.example.com")
	}
}

func TestHostRewriteRule_suffixMatch(t *testing.T) {
	r := &rule.HostRewriteRule{From: "5ch.net", To: "5ch.io"}
	req, _ := http.NewRequest("GET", "http://lavender.5ch.net/path", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Host; got != "5ch.io" {
		t.Errorf("URL.Host = %q, want %q", got, "5ch.io")
	}
	if got := req.Host; got != "5ch.io" {
		t.Errorf("Host = %q, want %q", got, "5ch.io")
	}
}

func TestHostRewriteRule_suffixBoundary(t *testing.T) {
	r := &rule.HostRewriteRule{From: "5ch.net", To: "5ch.io"}
	req, _ := http.NewRequest("GET", "http://evil5ch.net/path", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Host; got != "evil5ch.net" {
		t.Errorf("URL.Host = %q, want %q", got, "evil5ch.net")
	}
}

func TestHostRewriteRule_withPort_suffixMatch(t *testing.T) {
	r := &rule.HostRewriteRule{From: "example.com", To: "example.net"}
	req, _ := http.NewRequest("GET", "http://api.example.com:8080/path", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Host; got != "example.net:8080" {
		t.Errorf("URL.Host = %q, want %q", got, "example.net:8080")
	}
	if got := req.Host; got != "example.net:8080" {
		t.Errorf("Host = %q, want %q", got, "example.net:8080")
	}
}

func TestHostRewriteRule_withPort_toHasPort(t *testing.T) {
	r := &rule.HostRewriteRule{From: "example.com", To: "example.net:8443"}
	req, _ := http.NewRequest("GET", "http://api.example.com:8080/path", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Host; got != "example.net:8443" {
		t.Errorf("URL.Host = %q, want %q", got, "example.net:8443")
	}
}

func TestHostRewriteRule_emptyFrom_noMatch(t *testing.T) {
	r := &rule.HostRewriteRule{From: "", To: "example.net"}
	req, _ := http.NewRequest("GET", "http://example.com/path", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Host; got != "example.com" {
		t.Errorf("URL.Host = %q, want %q", got, "example.com")
	}
}

func TestHeaderSetRule(t *testing.T) {
	r := &rule.HeaderSetRule{Name: "User-Agent", Value: "rewproxy"}
	req, _ := http.NewRequest("GET", "http://example.com/", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.Header.Get("User-Agent"); got != "rewproxy" {
		t.Errorf("User-Agent = %q, want %q", got, "rewproxy")
	}
}

func TestURLRewriteRule_match(t *testing.T) {
	r := &rule.URLRewriteRule{From: "/api/v1", To: "/v2"}
	req, _ := http.NewRequest("GET", "http://example.com/api/v1/users", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Path; got != "/v2/users" {
		t.Errorf("Path = %q, want %q", got, "/v2/users")
	}
}

func TestURLRewriteRule_nomatch(t *testing.T) {
	r := &rule.URLRewriteRule{From: "/api/v1", To: "/v2"}
	req, _ := http.NewRequest("GET", "http://example.com/other/path", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Path; got != "/other/path" {
		t.Errorf("Path = %q, want %q", got, "/other/path")
	}
}

func TestQueryRewriteRule_set(t *testing.T) {
	r := &rule.QueryRewriteRule{Name: "env", Value: "staging"}
	req, _ := http.NewRequest("GET", "http://example.com/path", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Query().Get("env"); got != "staging" {
		t.Errorf("env = %q, want %q", got, "staging")
	}
}

func TestQueryRewriteRule_overwrite(t *testing.T) {
	r := &rule.QueryRewriteRule{Name: "env", Value: "staging"}
	req, _ := http.NewRequest("GET", "http://example.com/path?env=prod", nil)

	if err := r.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Query().Get("env"); got != "staging" {
		t.Errorf("env = %q, want %q", got, "staging")
	}
}

func TestPipeline(t *testing.T) {
	p := rule.Pipeline{
		&rule.HostRewriteRule{From: "old.example.com", To: "example.com"},
		&rule.HeaderSetRule{Name: "X-Forwarded-By", Value: "rewproxy"},
	}
	req, _ := http.NewRequest("GET", "http://old.example.com/path", nil)

	if err := p.Apply(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := req.URL.Host; got != "example.com" {
		t.Errorf("URL.Host = %q, want %q", got, "example.com")
	}
	if got := req.Header.Get("X-Forwarded-By"); got != "rewproxy" {
		t.Errorf("X-Forwarded-By = %q, want %q", got, "rewproxy")
	}
}
