package loader

import (
	"fmt"

	"rewproxy/internal/config"
	"rewproxy/internal/rule"
)

func Build(cfgs []config.RuleConfig) (rule.Pipeline, error) {
	var p rule.Pipeline
	for i, rc := range cfgs {
		r, err := buildOne(rc)
		if err != nil {
			return nil, fmt.Errorf("rule[%d]: %w", i, err)
		}
		p = append(p, r)
	}
	return p, nil
}

func buildOne(rc config.RuleConfig) (rule.Rule, error) {
	count := 0
	if rc.HostRewrite != nil {
		count++
	}
	if rc.HeaderSet != nil {
		count++
	}
	if rc.URLRewrite != nil {
		count++
	}
	if rc.QueryRewrite != nil {
		count++
	}
	if count > 1 {
		return nil, fmt.Errorf("exactly one rule type must be set, got %d", count)
	}

	switch {
	case rc.HostRewrite != nil:
		return &rule.HostRewriteRule{From: rc.HostRewrite.From, To: rc.HostRewrite.To}, nil
	case rc.HeaderSet != nil:
		return &rule.HeaderSetRule{Name: rc.HeaderSet.Name, Value: rc.HeaderSet.Value}, nil
	case rc.URLRewrite != nil:
		return &rule.URLRewriteRule{From: rc.URLRewrite.From, To: rc.URLRewrite.To}, nil
	case rc.QueryRewrite != nil:
		return &rule.QueryRewriteRule{Name: rc.QueryRewrite.Name, Value: rc.QueryRewrite.Value}, nil
	default:
		return nil, fmt.Errorf("no recognised rule type")
	}
}
