#!/usr/bin/env sh
set -eu

GO_BIN="${GO:-.tools/go/bin/go}"
mkdir -p bin

if [ -d web ]; then
  (cd web && npm ci && npm run build)
fi

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 "$GO_BIN" build -o bin/linkbit-controller ./cmd/linkbit-controller
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 "$GO_BIN" build -o bin/linkbit-relay ./cmd/linkbit-relay
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 "$GO_BIN" build -o bin/linkbit-agent ./cmd/linkbit-agent
