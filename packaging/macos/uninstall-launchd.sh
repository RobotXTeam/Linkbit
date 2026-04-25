#!/usr/bin/env sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root"
  exit 1
fi

plist="/Library/LaunchDaemons/com.linkbit.agent.plist"
launchctl bootout system "$plist" 2>/dev/null || true
rm -f "$plist"
rm -f /usr/local/linkbit/linkbit-agent
