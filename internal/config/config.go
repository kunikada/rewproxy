package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen          string       `yaml:"listen"`
	AccessLog       bool         `yaml:"access_log"`
	FollowRedirects bool         `yaml:"follow_redirects"`
	Rules           []RuleConfig `yaml:"rules"`
}

// RuleConfig holds exactly one rule type.
type RuleConfig struct {
	HostRewrite  *HostRewriteConfig  `yaml:"host_rewrite,omitempty"`
	HeaderSet    *HeaderSetConfig    `yaml:"header_set,omitempty"`
	URLRewrite   *URLRewriteConfig   `yaml:"url_rewrite,omitempty"`
	QueryRewrite *QueryRewriteConfig `yaml:"query_rewrite,omitempty"`
}

type HostRewriteConfig struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type HeaderSetConfig struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type URLRewriteConfig struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type QueryRewriteConfig struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	return &cfg, nil
}
