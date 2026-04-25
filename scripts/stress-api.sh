#!/usr/bin/env sh
set -eu

BASE="${LINKBIT_STRESS_BASE:-http://127.0.0.1:18081}"
ADMIN_KEY="${LINKBIT_BOOTSTRAP_API_KEY:-stress-admin-key}"
DB_PATH="${LINKBIT_DATABASE_PATH:-./linkbit-stress.db}"
PEPPER="${LINKBIT_API_KEY_PEPPER:-stress-pepper-change-me}"
DEVICE_COUNT="${LINKBIT_STRESS_DEVICES:-10}"

cleanup() {
  if [ -n "${SERVER_PID:-}" ]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
  fi
  rm -f "$DB_PATH" "$DB_PATH-shm" "$DB_PATH-wal"
}
trap cleanup EXIT INT TERM

rm -f "$DB_PATH" "$DB_PATH-shm" "$DB_PATH-wal"
LINKBIT_LISTEN_ADDR=127.0.0.1:18081 \
LINKBIT_DATABASE_PATH="$DB_PATH" \
LINKBIT_API_KEY_PEPPER="$PEPPER" \
LINKBIT_BOOTSTRAP_API_KEY="$ADMIN_KEY" \
"${GO:-.tools/go/bin/go}" run ./cmd/linkbit-controller >/tmp/linkbit-stress.log 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 80); do
  if curl -fsS "$BASE/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
done

curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
  -d '{"id":"load-user","name":"Load User","role":"member"}' "$BASE/api/v1/users" >/dev/null
curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
  -d '{"id":"load-group","name":"Load Group"}' "$BASE/api/v1/groups" >/dev/null

for relay in 1 2; do
  curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
    -d "{\"id\":\"relay-$relay\",\"name\":\"Relay $relay\",\"region\":\"stress\",\"publicUrl\":\"http://127.0.0.1:84$relay\"}" \
    "$BASE/api/v1/relays/register" >/dev/null
done

i=1
while [ "$i" -le "$DEVICE_COUNT" ]; do
  INVITE=$(curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
    -d '{"userId":"load-user","groupId":"load-group","expiresInSeconds":3600}' \
    "$BASE/api/v1/invitations" | node -pe 'JSON.parse(require("fs").readFileSync(0,"utf8")).token')
  REG=$(curl -fsS -H 'Content-Type: application/json' \
    -d "{\"enrollmentKey\":\"$INVITE\",\"name\":\"device-$i\",\"publicKey\":\"pub-$i\",\"fingerprint\":\"fp-$i\"}" \
    "$BASE/api/v1/devices/register")
  DEVICE_ID=$(printf '%s' "$REG" | node -pe 'JSON.parse(require("fs").readFileSync(0,"utf8")).device.id')
  DEVICE_TOKEN=$(printf '%s' "$REG" | node -pe 'JSON.parse(require("fs").readFileSync(0,"utf8")).device.deviceToken')
  curl -fsS -H "X-Linkbit-Device-Token: $DEVICE_TOKEN" -H 'Content-Type: application/json' \
    -d '{"status":"online","latencyMs":7,"peersReachable":0,"peersTotal":0}' \
    "$BASE/api/v1/devices/$DEVICE_ID/health" >/dev/null
  i=$((i + 1))
done

curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" -H 'Content-Type: application/json' \
  -d '{"id":"allow-load","name":"Allow Load Group","sourceId":"load-group","targetId":"load-group","ports":["*"],"protocol":"tcp","enabled":true}' \
  "$BASE/api/v1/policies" >/dev/null

OVERVIEW=$(curl -fsS -H "X-Linkbit-API-Key: $ADMIN_KEY" "$BASE/api/v1/overview")
ONLINE=$(printf '%s' "$OVERVIEW" | node -pe 'JSON.parse(require("fs").readFileSync(0,"utf8")).onlineDevices')
RELAYS=$(printf '%s' "$OVERVIEW" | node -pe 'JSON.parse(require("fs").readFileSync(0,"utf8")).relayNodes')

if [ "$ONLINE" -ne "$DEVICE_COUNT" ] || [ "$RELAYS" -ne 2 ]; then
  echo "stress failed: $OVERVIEW"
  exit 1
fi

echo "stress-api ok: devices=$ONLINE relays=$RELAYS"

