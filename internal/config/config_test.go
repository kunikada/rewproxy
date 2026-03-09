package config_test

import (
	"os"
	"testing"

	"rewproxy/internal/config"
	"rewproxy/internal/loader"
	"rewproxy/internal/rule"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestLoad_defaults(t *testing.T) {
	path := writeTemp(t, "rules: []\n")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Listen != ":8080" {
		t.Errorf("Listen = %q, want %q", cfg.Listen, ":8080")
	}
}

func TestLoad_listen(t *testing.T) {
	path := writeTemp(t, "listen: \":9090\"\nrules: []\n")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Listen != ":9090" {
		t.Errorf("Listen = %q, want %q", cfg.Listen, ":9090")
	}
}

func TestLoad_hostRewrite(t *testing.T) {
	yaml := `
rules:
  - host_rewrite:
      from: old.example.com
      to: example.com
`
	path := writeTemp(t, yaml)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1", len(cfg.Rules))
	}
	rc := cfg.Rules[0]
	if rc.HostRewrite == nil {
		t.Fatal("HostRewrite is nil")
	}
	if rc.HostRewrite.From != "old.example.com" || rc.HostRewrite.To != "example.com" {
		t.Errorf("HostRewrite = %+v", rc.HostRewrite)
	}
}

func TestBuild_hostRewrite(t *testing.T) {
	cfgs := []config.RuleConfig{
		{HostRewrite: &config.HostRewriteConfig{From: "old.example.com", To: "example.com"}},
	}
	p, err := loader.Build(cfgs)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(p) != 1 {
		t.Fatalf("len(pipeline) = %d, want 1", len(p))
	}
}

func TestBuild_headerSet(t *testing.T) {
	cfgs := []config.RuleConfig{
		{HeaderSet: &config.HeaderSetConfig{Name: "User-Agent", Value: "rewproxy"}},
	}
	p, err := loader.Build(cfgs)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(p) != 1 {
		t.Fatalf("len(pipeline) = %d, want 1", len(p))
	}
	_ = p[0].(*rule.HeaderSetRule)
}

func TestBuild_urlRewrite(t *testing.T) {
	cfgs := []config.RuleConfig{
		{URLRewrite: &config.URLRewriteConfig{From: "/api/v1", To: "/v2"}},
	}
	p, err := loader.Build(cfgs)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(p) != 1 {
		t.Fatalf("len(pipeline) = %d, want 1", len(p))
	}
	_ = p[0].(*rule.URLRewriteRule)
}

func TestBuild_queryRewrite(t *testing.T) {
	cfgs := []config.RuleConfig{
		{QueryRewrite: &config.QueryRewriteConfig{Name: "env", Value: "staging"}},
	}
	p, err := loader.Build(cfgs)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(p) != 1 {
		t.Fatalf("len(pipeline) = %d, want 1", len(p))
	}
	_ = p[0].(*rule.QueryRewriteRule)
}

func TestBuild_unknownRule(t *testing.T) {
	cfgs := []config.RuleConfig{{}}
	_, err := loader.Build(cfgs)
	if err == nil {
		t.Fatal("expected error for unknown rule type")
	}
}

func TestBuild_multipleRuleTypes(t *testing.T) {
	cfgs := []config.RuleConfig{
		{
			HostRewrite: &config.HostRewriteConfig{From: "old.example.com", To: "example.com"},
			HeaderSet:   &config.HeaderSetConfig{Name: "X-Test", Value: "v"},
		},
	}
	_, err := loader.Build(cfgs)
	if err == nil {
		t.Fatal("expected error for multiple rule types in one entry")
	}
}
