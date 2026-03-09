package main

import (
	"flag"
	"log"
	"net/http"

	"rewproxy/internal/config"
	"rewproxy/internal/loader"
	"rewproxy/internal/proxy"
)

var version = "1.1.0"

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pipeline, err := loader.Build(cfg.Rules)
	if err != nil {
		log.Fatalf("build rules: %v", err)
	}

	h := &proxy.Handler{Pipeline: pipeline, AccessLog: cfg.AccessLog, FollowRedirects: cfg.FollowRedirects}

	log.Printf("rewproxy v%s listening on %s", version, cfg.Listen)
	if err := http.ListenAndServe(cfg.Listen, h); err != nil {
		log.Fatalf("server: %v", err)
	}
}
