#!/usr/bin/env sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root"
  exit 1
fi

install_dir="${LINKBIT_INSTALL_DIR:-/opt/linkbit}"
config_dir="${LINKBIT_CONFIG_DIR:-/etc/linkbit}"

mkdir -p "$install_dir" "$config_dir"
mkdir -p /var/lib/linkbit
install -m 0755 ./bin/linkbit-controller "$install_dir/linkbit-controller"

# Accept both repository layout (./web/dist) and remote upload layout (./web).
# This keeps the install script portable across local installs and scp-based deploys.
web_source=""
if [ -f ./web/dist/index.html ]; then
  web_source="./web/dist"
elif [ -f ./web/index.html ]; then
  web_source="./web"
fi
if [ -n "$web_source" ]; then
  rm -rf "$install_dir/web"
  mkdir -p "$install_dir/web"
  cp -R "$web_source"/. "$install_dir/web/"
fi

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
