#!/usr/bin/env sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root"
  exit 1
fi

install_dir="${LINKBIT_INSTALL_DIR:-/opt/linkbit}"
config_dir="${LINKBIT_CONFIG_DIR:-/etc/linkbit}"

mkdir -p "$install_dir" "$config_dir"
install -m 0755 ./bin/linkbit-relay "$install_dir/linkbit-relay"

cat > /etc/systemd/system/linkbit-relay.service <<'SERVICE'
[Unit]
Description=Linkbit Relay
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/linkbit/relay.env
ExecStart=/opt/linkbit/linkbit-relay
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true

[Install]
WantedBy=multi-user.target
SERVICE

systemctl daemon-reload
systemctl enable linkbit-relay
systemctl restart linkbit-relay
