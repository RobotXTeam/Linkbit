#!/usr/bin/env sh
set -eu

export LINKBIT_LISTEN_ADDR="${LINKBIT_LISTEN_ADDR:-:8080}"
export LINKBIT_DATABASE_PATH="${LINKBIT_DATABASE_PATH:-./linkbit-dev.db}"
export LINKBIT_API_KEY_PEPPER="${LINKBIT_API_KEY_PEPPER:-dev-only-change-me-32-byte-secret}"
export LINKBIT_BOOTSTRAP_API_KEY="${LINKBIT_BOOTSTRAP_API_KEY:-dev-admin-key}"

GO_BIN="${GO:-.tools/go/bin/go}"
exec "$GO_BIN" run ./cmd/linkbit-controller

