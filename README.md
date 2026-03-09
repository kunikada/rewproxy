# rewproxy

rewproxy is a programmable forward proxy written in Go.

It allows rewriting outbound HTTP requests such as hostnames,
URLs, headers, and query parameters before they are sent to
the destination server.

rewproxy is designed as a compatibility layer when upstream
services change but modifying client code is difficult.

client → rewproxy → internet


## Features

- Forward HTTP proxy
- Host rewriting
- URL rewriting
- Header modification
- Query parameter rewriting
- Configurable rewrite rules
- Lightweight single binary


## Example

Rewrite requests from `old.example.com` to `example.com`.

rules:
  - host_rewrite:
      from: old.example.com
      to: example.com

Request:

https://old.example.com/path/to/resource

Forwarded as:

https://example.com/path/to/resource


## Usage

### 1) Single Binary

./rewproxy --config config.yaml

Configure the client to use:

http://localhost:8080

### 2) Docker Command

docker run --rm -p 8080:8080 \
  -v $(pwd)/config.yaml:/etc/rewproxy/config.yaml:ro \
  ghcr.io/your-org/rewproxy:latest \
  --config /etc/rewproxy/config.yaml

### 3) Docker Compose

```yaml
services:
  rewproxy:
    image: ghcr.io/your-org/rewproxy:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/etc/rewproxy/config.yaml:ro
    command: ["--config", "/etc/rewproxy/config.yaml"]
```

Start:

docker compose up -d


## Configuration Example

listen: ":8080"

rules:

  - host_rewrite:
      from: "old.example.com"
      to: "example.com"

  - header_set:
      name: "User-Agent"
      value: "rewproxy"


## Motivation

Sometimes external services change domains or endpoints.
Updating all client code can be slow or impossible.

rewproxy provides a small proxy layer that transparently
redirects requests to the new destination.


## Status

Experimental


## License

MIT
