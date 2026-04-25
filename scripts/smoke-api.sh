#!/usr/bin/env sh
set -eu

BASE="${LINKBIT_SMOKE_BASE:-http://127.0.0.1:18080}"
ADMIN_KEY="${LINKBIT_BOOTSTRAP_API_KEY:-smoke-admin-key}"
DB_PATH="${LINKBIT_DATABASE_PATH:-./linkbit-smoke.db}"
PEPPER="${LINKBIT_API_KEY_PEPPER:-smoke-pepper-change-me}"

cleanup() {
  if [ -n "${SERVER_PID:-}" ]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
  fi
  rm -f "$DB_PATH" "$DB_PATH-shm" "$DB_PATH-wal"
}
trap cleanup EXIT INT TERM

rm -f "$DB_PATH" "$DB_PATH-shm" "$DB_PATH-wal"
LINKBIT_LISTEN_ADDR=127.0.0.1:18080 \
LINKBIT_DATABASE_PATH="$DB_PATH" \
LINKBIT_API_KEY_PEPPER="$PEPPER" \
LINKBIT_BOOTSTRAP_API_KEY="$ADMIN_KEY" \
"${GO:-.tools/go/bin/go}" run ./cmd/linkbit-controller &
SERVER_PID=$!

for _ in $(seq 1 60); do
  if curl -fsS "$BASE/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
done

curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
  -d '{"id":"default-user","name":"Default User","role":"member"}' \
  "$BASE/api/v1/users" >/dev/null

curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
  -d '{"id":"default","name":"Default Group"}' \
  "$BASE/api/v1/groups" >/dev/null

curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
  -d '{"id":"relay-smoke","name":"Smoke Relay","region":"default","publicUrl":"http://127.0.0.1:8443"}' \
  "$BASE/api/v1/relays/register" >/dev/null

INVITE=$(curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
  -d '{"userId":"default-user","groupId":"default","expiresInSeconds":3600}' \
  "$BASE/api/v1/invitations" | node -pe 'JSON.parse(require("fs").readFileSync(0,"utf8")).token')

REG=$(curl -fsS -H 'Content-Type: application/json' \
  -d "{\"enrollmentKey\":\"$INVITE\",\"name\":\"smoke-device\",\"publicKey\":\"smoke-public-key\",\"fingerprint\":\"smoke-fp\"}" \
  "$BASE/api/v1/devices/register")

DEVICE_ID=$(printf '%s' "$REG" | node -pe 'JSON.parse(require("fs").readFileSync(0,"utf8")).device.id')
DEVICE_TOKEN=$(printf '%s' "$REG" | node -pe 'JSON.parse(require("fs").readFileSync(0,"utf8")).device.deviceToken')

curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
  -d "{\"id\":\"allow-default\",\"name\":\"Allow Default\",\"sourceId\":\"default\",\"targetId\":\"default\",\"ports\":[\"*\"],\"protocol\":\"tcp\",\"enabled\":true}" \
  "$BASE/api/v1/policies" >/dev/null

curl -fsS -H "X-Linkbit-Device-Token: $DEVICE_TOKEN" -H 'Content-Type: application/json' \
  -d '{"status":"online","latencyMs":5,"peersReachable":0,"peersTotal":0}' \
  "$BASE/api/v1/devices/$DEVICE_ID/health" >/dev/null

curl -fsS -H "X-Linkbit-Device-Token: $DEVICE_TOKEN" \
  "$BASE/api/v1/devices/$DEVICE_ID/network-config" >/dev/null

curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" "$BASE/api/v1/derp-map" >/dev/null
curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" "$BASE/api/v1/overview" >/dev/null

echo "smoke-api ok"
