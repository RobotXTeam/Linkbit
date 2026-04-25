#!/usr/bin/env sh
set -eu

BASE="${LINKBIT_RECOVERY_BASE:-http://127.0.0.1:18082}"
ADMIN_KEY="${LINKBIT_BOOTSTRAP_API_KEY:-recovery-admin-key}"
DB_PATH="${LINKBIT_DATABASE_PATH:-./linkbit-recovery.db}"
PEPPER="${LINKBIT_API_KEY_PEPPER:-recovery-pepper-change-me}"

cleanup() {
  if [ -n "${SERVER_PID:-}" ]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
  fi
  rm -f "$DB_PATH" "$DB_PATH-shm" "$DB_PATH-wal"
}
trap cleanup EXIT INT TERM

rm -f "$DB_PATH" "$DB_PATH-shm" "$DB_PATH-wal"
LINKBIT_LISTEN_ADDR=127.0.0.1:18082 \
LINKBIT_DATABASE_PATH="$DB_PATH" \
LINKBIT_API_KEY_PEPPER="$PEPPER" \
LINKBIT_BOOTSTRAP_API_KEY="$ADMIN_KEY" \
"${GO:-.tools/go/bin/go}" run ./cmd/linkbit-controller >/tmp/linkbit-recovery.log 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 80); do
  if curl -fsS "$BASE/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
done

curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
  -d '{"id":"relay-a","name":"Relay A","region":"recovery","publicUrl":"http://198.51.100.1:8443"}' \
  "$BASE/api/v1/relays/register" >/dev/null

BEFORE=$(curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" "$BASE/api/v1/derp-map")
curl -fsS -X DELETE -H "X-Linkbit-API-Key: $ADMIN_KEY" "$BASE/api/v1/relays/relay-a" >/dev/null
curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
  -d '{"id":"relay-b","name":"Relay B","region":"recovery","publicUrl":"http://203.0.113.1:8443"}' \
  "$BASE/api/v1/relays/register" >/dev/null
AFTER=$(curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" "$BASE/api/v1/derp-map")

if printf '%s' "$BEFORE" | grep -q '198.51.100.1' && printf '%s' "$AFTER" | grep -q '203.0.113.1'; then
  echo "relay-recovery ok"
else
  echo "relay-recovery failed"
  exit 1
fi

