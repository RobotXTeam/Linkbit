#!/usr/bin/env sh
set -eu

host="${LINKBIT_REMOTE_HOST:?LINKBIT_REMOTE_HOST is required, for example root@example.com}"
ssh_opts="${LINKBIT_SSH_OPTS:-}"
remote_dir="${LINKBIT_REMOTE_DIR:-/root/linkbit-install}"

./scripts/build-linux-amd64.sh

ssh $ssh_opts "$host" "mkdir -p '$remote_dir'/bin '$remote_dir'/deploy '$remote_dir'/web/dist"
scp $ssh_opts bin/linkbit-controller bin/linkbit-relay "$host:$remote_dir/bin/"
scp $ssh_opts deploy/install-controller.sh deploy/install-relay.sh "$host:$remote_dir/deploy/"
tar -C web/dist -czf - . | ssh $ssh_opts "$host" "rm -rf '$remote_dir'/web/dist && mkdir -p '$remote_dir'/web/dist && tar -C '$remote_dir'/web/dist -xzf -"

echo "uploaded artifacts to $host:$remote_dir"
echo "copy generated env files to /etc/linkbit on the server, then run install scripts as root"
