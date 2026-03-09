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

Rewrite requests targeting `*.legacy.example` to `gateway.example`.

rules:
  - host_rewrite:
      from: legacy.example
      to: gateway.example

Request:

https://api.legacy.example/v1/resource

Forwarded as:

https://gateway.example/v1/resource

`host_rewrite` matching rules:
- Matches exact host (`legacy.example`) or subdomains by suffix (`api.legacy.example` matches `legacy.example`).
- Replaces the full host with `to` (subdomain labels are not preserved).
- If the original request had an explicit port and `to` does not include one, the original port is kept.


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

See [config.yaml](config.yaml).


## License

MIT
