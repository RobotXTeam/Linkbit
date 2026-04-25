#!/usr/bin/env sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root"
  exit 1
fi

install_dir="${LINKBIT_INSTALL_DIR:-/opt/linkbit}"
config_dir="${LINKBIT_CONFIG_DIR:-/etc/linkbit}"

mkdir -p "$install_dir" "$config_dir"
install -m 0755 ./bin/linkbit-agent "$install_dir/linkbit-agent"

cat > /etc/systemd/system/linkbit-agent.service <<'SERVICE'
[Unit]
Description=Linkbit Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/linkbit/agent.env
ExecStart=/opt/linkbit/linkbit-agent
Restart=on-failure
RestartSec=3
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/run /tmp

[Install]
WantedBy=multi-user.target
SERVICE

systemctl daemon-reload
systemctl enable --now linkbit-agent

