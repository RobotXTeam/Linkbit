#!/usr/bin/env sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root"
  exit 1
fi

install_dir="/usr/local/linkbit"
config_dir="/Library/Application Support/Linkbit"
plist="/Library/LaunchDaemons/com.linkbit.agent.plist"

mkdir -p "$install_dir" "$config_dir"
install -m 0755 ./bin/linkbit-agent "$install_dir/linkbit-agent"
install -m 0644 ./packaging/macos/com.linkbit.agent.plist "$plist"

echo "edit $config_dir/agent.env before loading the service"
echo "load with: launchctl bootstrap system $plist"
