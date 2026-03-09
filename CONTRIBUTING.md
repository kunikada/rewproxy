# Contributing to rewproxy

## Architecture Overview

rewproxy is a forward HTTP proxy. Clients configure rewproxy as their HTTP proxy, and rewproxy runs each request through an ordered pipeline of rewrite rules before forwarding it to the origin server.

```
Client
  │  HTTP CONNECT or plain HTTP request
  ▼
[Listener :8080]
  │
  ▼
[Request Handler]
  │  clone & mutate *http.Request
  ▼
[Rule Pipeline]  ← rules applied in config order
  │   rule 1: host_rewrite
  │   rule 2: header_set
  │   rule N: ...
  ▼
[HTTP Transport / upstream dial]
  │
  ▼
Origin Server
  │  response streamed back verbatim
  ▼
Client
```

Key design decisions:

- Rules are applied sequentially in the order they appear in config.
- A rule mutates the request in-place; returning a non-nil error aborts the pipeline and sends 502 to the client.
- HTTPS tunneling uses the `CONNECT` method with byte-stream proxying — no TLS termination.
- The proxy is not transparent; clients must be explicitly configured (e.g. `HTTP_PROXY=http://localhost:8080`).
- Error conditions are always logged via `log.Printf` before sending an error response to the client.
- Access logging (one line per request) is opt-in via `access_log: true` in config; disabled by default.


## Directory and Package Layout

```
rewproxy/
├── main.go                   # package main — CLI flags, wiring, ListenAndServe
├── go.mod                    # module rewproxy
├── go.sum
├── config.yaml               # local dev config example
├── Dockerfile
├── compose.yaml
├── README.md
├── CONTRIBUTING.md
└── internal/
    ├── config/
    │   └── config.go         # Config and RuleConfig structs; YAML loading
    ├── proxy/
    │   └── handler.go        # net/http Handler; CONNECT tunnel + plain HTTP
    ├── rule/
    │   ├── rule.go           # Rule interface + Pipeline type
    │   ├── host_rewrite.go   # HostRewriteRule
    │   ├── header_set.go     # HeaderSetRule
    │   ├── url_rewrite.go    # URLRewriteRule
    │   └── query_rewrite.go  # QueryRewriteRule
    └── loader/
        └── loader.go         # factory: []RuleConfig → rule.Pipeline
```

> Note: `Dockerfile` sets `WORKDIR /src` inside the container — this is not the `src/` subdirectory of the repo. `main.go` lives at the project root.


## Core Interfaces

### Rule (`internal/rule/rule.go`)

```go
package rule

import "net/http"

// Rule is the core interface every rewrite rule must satisfy.
// Apply mutates the outbound request. A non-nil error aborts the pipeline.
type Rule interface {
    Apply(req *http.Request) error
}

// Pipeline is an ordered slice of Rules applied left to right.
type Pipeline []Rule

func (p Pipeline) Apply(req *http.Request) error {
    for _, r := range p {
        if err := r.Apply(req); err != nil {
            return err
        }
    }
    return nil
}
```

### Config structs (`internal/config/config.go`)

`RuleConfig` uses a discriminated-union pattern: exactly one pointer field is non-nil after YAML unmarshaling.

```go
package config

type Config struct {
    Listen    string       `yaml:"listen"`
    AccessLog bool         `yaml:"access_log"`
    Rules     []RuleConfig `yaml:"rules"`
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

func Load(path string) (*Config, error) { ... }
```

YAML library: `gopkg.in/yaml.v3`.

### Rule implementations

**`internal/rule/host_rewrite.go`**

```go
type HostRewriteRule struct{ From, To string }

func (r *HostRewriteRule) Apply(req *http.Request) error {
    if req.URL.Host == r.From {
        req.URL.Host = r.To
        req.Host = r.To
    }
    return nil
}
```

**`internal/rule/header_set.go`**

```go
type HeaderSetRule struct{ Name, Value string }

func (r *HeaderSetRule) Apply(req *http.Request) error {
    req.Header.Set(r.Name, r.Value)
    return nil
}
```

**`internal/rule/url_rewrite.go`**

Replaces a path prefix. If `req.URL.Path` starts with `From`, the matching prefix is replaced with `To`. No-op if the prefix does not match.

```go
type URLRewriteRule struct{ From, To string }

func (r *URLRewriteRule) Apply(req *http.Request) error {
    if strings.HasPrefix(req.URL.Path, r.From) {
        req.URL.Path = r.To + req.URL.Path[len(r.From):]
    }
    return nil
}
```

Config key: `url_rewrite` with fields `from` and `to`.

**`internal/rule/query_rewrite.go`**

Sets (or overwrites) a single query parameter. Equivalent to `header_set` but for query parameters.

```go
type QueryRewriteRule struct{ Name, Value string }

func (r *QueryRewriteRule) Apply(req *http.Request) error {
    q := req.URL.Query()
    q.Set(r.Name, r.Value)
    req.URL.RawQuery = q.Encode()
    return nil
}
```

Config key: `query_rewrite` with fields `name` and `value`.

### Loader (`internal/loader/loader.go`)

Converts `[]config.RuleConfig` into a `rule.Pipeline`.

```go
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
    switch {
    case rc.HostRewrite != nil:
        return &rule.HostRewriteRule{From: rc.HostRewrite.From, To: rc.HostRewrite.To}, nil
    case rc.HeaderSet != nil:
        return &rule.HeaderSetRule{Name: rc.HeaderSet.Name, Value: rc.HeaderSet.Value}, nil
    default:
        return nil, fmt.Errorf("no recognised rule type")
    }
}
```

### Proxy Handler (`internal/proxy/handler.go`)

The handler supports two request modes:

- **Plain HTTP** — client sends a full `http://host/path` request. Strip hop-by-hop headers, apply the rule pipeline, forward via `http.DefaultTransport`, copy response back.
- **HTTPS CONNECT tunnel** — client sends `CONNECT host:443 HTTP/1.1`. Apply rules to the target host before dialing, respond `200 Connection Established`, then copy bytes bidirectionally.

```go
type Handler struct {
    Pipeline  rule.Pipeline
    Transport http.RoundTripper // defaults to http.DefaultTransport
    AccessLog bool              // when true, logs one line per request
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodConnect {
        h.handleTunnel(w, r)
    } else {
        h.handleHTTP(w, r)
    }
}
```

**Error logging**: every code path that returns a 4xx/5xx to the client calls `log.Printf` first, so errors are always visible in the process log.

**Access logging**: when `Handler.AccessLog` is true, one line is emitted per request after the response is sent:

```
// plain HTTP
ACCESS method=GET host=example.com path=/foo status=200

// CONNECT tunnel
ACCESS method=CONNECT host=example.com:443 status=200
```

Hop-by-hop headers to strip before forwarding:
`Connection`, `Proxy-Connection`, `Keep-Alive`, `Proxy-Authenticate`, `Proxy-Authorization`, `Te`, `Trailers`, `Transfer-Encoding`, `Upgrade`.


## Adding a New Rule Type

1. Add a `*FooConfig` field to `RuleConfig` in `internal/config/config.go`.
2. Create `internal/rule/foo.go` with `type FooRule struct` implementing `Apply(*http.Request) error`.
3. Add a `case rc.Foo != nil:` branch to `buildOne` in `internal/loader/loader.go`.
4. Add unit tests in `internal/rule/rule_test.go`.
5. Document the new config key in `README.md`.

No other files need to change.


## Testing

**Unit tests** (`internal/rule/rule_test.go`): test each rule's `Apply` with a synthetic `*http.Request` via `http.NewRequest`. No network required.

**Integration tests** (`internal/proxy/handler_test.go`): use `net/http/httptest` to spin up a fake upstream and a fake proxy, then configure an `http.Client` with `http.ProxyURL` to send requests through it. Assert the upstream received the expected rewritten request.

**Config tests** (`internal/config/config_test.go`): use `os.CreateTemp` with inline YAML to test `config.Load` and `loader.Build` for each rule type.

**Run all tests:**

```sh
go test -race ./...
```


## Code Style and Conventions

Use `goimports` and `go vet` before committing.
