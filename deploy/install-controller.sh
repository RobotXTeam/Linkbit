#!/usr/bin/env sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root"
  exit 1
fi

install_dir="${LINKBIT_INSTALL_DIR:-/opt/linkbit}"
config_dir="${LINKBIT_CONFIG_DIR:-/etc/linkbit}"

mkdir -p "$install_dir" "$config_dir"
install -m 0755 ./bin/linkbit-controller "$install_dir/linkbit-controller"

cat > /etc/systemd/system/linkbit-controller.service <<'SERVICE'
[Unit]
Description=Linkbit Controller
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/linkbit/controller.env
ExecStart=/opt/linkbit/linkbit-controller
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/linkbit

[Install]
WantedBy=multi-user.target
SERVICE

systemctl daemon-reload
systemctl enable --now linkbit-controller

