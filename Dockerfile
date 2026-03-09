FROM golang:1.25-trixie

RUN useradd -m -u 1000 -s /bin/bash appuser

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /usr/local/bin/rewproxy .

EXPOSE 8080

USER appuser

ENTRYPOINT ["rewproxy"]
CMD ["--config", "/etc/rewproxy/config.yaml"]
