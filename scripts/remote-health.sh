#!/usr/bin/env sh
set -eu

controller_url="${LINKBIT_CONTROLLER_URL:?LINKBIT_CONTROLLER_URL is required, for example http://203.0.113.10}"
relay_url="${LINKBIT_RELAY_PUBLIC_URL:-}"
api_key="${LINKBIT_API_KEY:-${LINKBIT_BOOTSTRAP_API_KEY:-}}"

trim_slash() {
  printf '%s' "$1" | sed 's#/*$##'
}

controller_url="$(trim_slash "$controller_url")"
if [ -n "$relay_url" ]; then
  relay_url="$(trim_slash "$relay_url")"
fi

echo "controller health: $controller_url/healthz"
curl --fail --show-error --silent "$controller_url/healthz" >/dev/null

echo "web console: $controller_url/"
curl --fail --show-error --silent "$controller_url/" | grep -q '<title>Linkbit Console'

if [ -n "$api_key" ]; then
  echo "controller overview"
  curl --fail --show-error --silent \
    -H "X-Linkbit-API-Key: $api_key" \
    "$controller_url/api/v1/overview" >/dev/null

  echo "derp map"
  curl --fail --show-error --silent \
    -H "X-Linkbit-API-Key: $api_key" \
    "$controller_url/api/v1/derp-map" >/dev/null
else
  echo "controller overview: skipped, set LINKBIT_API_KEY or LINKBIT_BOOTSTRAP_API_KEY"
fi

if [ -n "$relay_url" ]; then
  echo "relay health: $relay_url/healthz"
  curl --fail --show-error --silent "$relay_url/healthz" >/dev/null
fi

echo "remote health checks passed"
