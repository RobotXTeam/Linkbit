#!/usr/bin/env sh
set -eu

out_dir="${1:-./deploy/generated}"
mkdir -p "$out_dir"

controller_url="${LINKBIT_CONTROLLER_URL:?LINKBIT_CONTROLLER_URL is required}"
controller_public_url="${LINKBIT_PUBLIC_URL:-$controller_url}"
relay_public_url="${LINKBIT_RELAY_PUBLIC_URL:?LINKBIT_RELAY_PUBLIC_URL is required}"
controller_listen_addr="${LINKBIT_CONTROLLER_LISTEN_ADDR:-:8080}"
relay_listen_addr="${LINKBIT_RELAY_LISTEN_ADDR:-:8443}"
relay_id="${LINKBIT_RELAY_ID:-relay-1}"
relay_name="${LINKBIT_RELAY_NAME:-Linkbit Relay 1}"
relay_region="${LINKBIT_RELAY_REGION:-default}"
api_key_pepper="${LINKBIT_API_KEY_PEPPER:?LINKBIT_API_KEY_PEPPER is required}"
bootstrap_key="${LINKBIT_BOOTSTRAP_API_KEY:?LINKBIT_BOOTSTRAP_API_KEY is required}"
relay_key="${LINKBIT_API_KEY:?LINKBIT_API_KEY is required}"
log_level="${LINKBIT_LOG_LEVEL:-info}"
hub_wg_enabled="${LINKBIT_HUB_WG_ENABLED:-false}"
hub_wg_interface="${LINKBIT_HUB_WG_INTERFACE:-linkbit-hub}"
hub_wg_ip="${LINKBIT_HUB_WG_IP:-10.88.0.1}"
hub_wg_network="${LINKBIT_HUB_WG_NETWORK:-10.88.0.0/16}"
hub_wg_port="${LINKBIT_HUB_WG_PORT:-41641}"
hub_wg_private_key="${LINKBIT_HUB_WG_PRIVATE_KEY:-}"
hub_wg_endpoint="${LINKBIT_HUB_WG_ENDPOINT:-}"

cat > "$out_dir/controller.env" <<EOF
LINKBIT_LISTEN_ADDR=$controller_listen_addr
LINKBIT_PUBLIC_URL=$controller_public_url
LINKBIT_DATABASE_PATH=/var/lib/linkbit/linkbit.db
LINKBIT_API_KEY_PEPPER=$api_key_pepper
LINKBIT_BOOTSTRAP_API_KEY=$bootstrap_key
LINKBIT_WEB_DIR=/opt/linkbit/web
LINKBIT_LOG_LEVEL=$log_level
LINKBIT_HUB_WG_ENABLED=$hub_wg_enabled
LINKBIT_HUB_WG_INTERFACE=$hub_wg_interface
LINKBIT_HUB_WG_IP=$hub_wg_ip
LINKBIT_HUB_WG_NETWORK=$hub_wg_network
LINKBIT_HUB_WG_PORT=$hub_wg_port
LINKBIT_HUB_WG_PRIVATE_KEY=$hub_wg_private_key
LINKBIT_HUB_WG_ENDPOINT=$hub_wg_endpoint
EOF

cat > "$out_dir/relay.env" <<EOF
LINKBIT_CONTROLLER_URL=$controller_url
LINKBIT_API_KEY=$relay_key
LINKBIT_RELAY_ID=$relay_id
LINKBIT_RELAY_NAME=$relay_name
LINKBIT_RELAY_PUBLIC_URL=$relay_public_url
LINKBIT_RELAY_REGION=$relay_region
LINKBIT_LISTEN_ADDR=$relay_listen_addr
LINKBIT_HEARTBEAT_SECONDS=30
EOF

chmod 600 "$out_dir/controller.env" "$out_dir/relay.env"
echo "wrote $out_dir/controller.env and $out_dir/relay.env"
