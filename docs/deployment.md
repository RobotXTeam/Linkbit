# Linkbit Deployment

## Controller

Required environment:

- `LINKBIT_LISTEN_ADDR`
- `LINKBIT_DATABASE_PATH`
- `LINKBIT_API_KEY_PEPPER`
- `LINKBIT_BOOTSTRAP_API_KEY`

Local development:

```bash
./scripts/dev-controller.sh
```

Production systemd install expects a built binary at `./bin/linkbit-controller`:

```bash
make test
mkdir -p bin
.tools/go/bin/go build -o bin/linkbit-controller ./cmd/linkbit-controller
sudo ./deploy/install-controller.sh
```

Create `/etc/linkbit/controller.env` with secret values before starting the service.
Use `deploy/controller.env.example` as the non-secret template.

The controller can serve the built web console directly when `LINKBIT_WEB_DIR` points to the web build directory.
For a direct single-server deployment without a reverse proxy, set `LINKBIT_LISTEN_ADDR=:80` and `LINKBIT_PUBLIC_URL=http://<server-ip>`.
For a reverse-proxy deployment, keep the controller on an internal port such as `:8080` and expose only the proxy on `80/443`.

## Relay

Required environment:

- `LINKBIT_CONTROLLER_URL`
- `LINKBIT_API_KEY`
- `LINKBIT_RELAY_ID`
- `LINKBIT_RELAY_PUBLIC_URL`

The relay performs controller registration, heartbeat, and runs a DERP-compatible HTTP service mounted at `/derp`.
Use `deploy/relay.env.example` as the non-secret template.
For a direct single-server deployment, set `LINKBIT_LISTEN_ADDR=:443` and `LINKBIT_RELAY_PUBLIC_URL=http://<server-ip>:443`.
For a reverse-proxy deployment, keep the relay on an internal port such as `:8443` and proxy public HTTPS traffic to it.

Install:

```bash
./scripts/build-linux-amd64.sh
sudo ./deploy/install-relay.sh
```

## Docker Compose

Copy templates and fill secrets:

```bash
cp deploy/controller.env.example deploy/controller.env
cp deploy/relay.env.example deploy/relay.env
docker compose -f deploy/compose.yml up -d --build
```

## TLS

`deploy/Caddyfile.example` contains a Caddy reverse-proxy template for automatic HTTPS.

## Verification

Local API checks:

```bash
make smoke
make stress
make recovery-smoke
```

Prometheus-compatible metrics are exposed at `/metrics` and require `X-Linkbit-API-Key`.

## Remote Install Helper

Generate environment files locally without committing secrets:

```bash
LINKBIT_CONTROLLER_URL=https://controller.example.com \
LINKBIT_RELAY_PUBLIC_URL=https://relay.example.com \
LINKBIT_API_KEY_PEPPER=replace-me \
LINKBIT_BOOTSTRAP_API_KEY=replace-me \
LINKBIT_API_KEY=replace-me \
./scripts/render-deploy-env.sh
```

Direct public-IP install using already-open `80/443`:

```bash
LINKBIT_CONTROLLER_URL=http://203.0.113.10 \
LINKBIT_PUBLIC_URL=http://203.0.113.10 \
LINKBIT_RELAY_PUBLIC_URL=http://203.0.113.10:443 \
LINKBIT_CONTROLLER_LISTEN_ADDR=:80 \
LINKBIT_RELAY_LISTEN_ADDR=:443 \
LINKBIT_API_KEY_PEPPER=replace-me \
LINKBIT_BOOTSTRAP_API_KEY=replace-me \
LINKBIT_API_KEY=replace-me \
./scripts/render-deploy-env.sh
```

Upload binaries and install scripts:

```bash
LINKBIT_REMOTE_HOST=root@example.com ./scripts/remote-install.sh
```

After copying `deploy/generated/controller.env` and `deploy/generated/relay.env` into `/etc/linkbit/` on the server and running the install scripts, verify the public surface:

```bash
LINKBIT_CONTROLLER_URL=http://203.0.113.10 \
LINKBIT_RELAY_PUBLIC_URL=http://203.0.113.10:443 \
LINKBIT_API_KEY=replace-with-admin-or-bootstrap-key \
make remote-health
```

The remote installer uploads the built console from `web/dist` and the controller install script accepts both `./web/dist` and already-flattened `./web` layouts.

## Agent

Required environment:

- `LINKBIT_CONTROLLER_URL`
- `LINKBIT_ENROLLMENT_KEY` only for first enrollment

The Linux agent performs controller registration, creates a WireGuard interface through `ip` and `wg`, and reports health back to the controller with a device-scoped token.
Use `deploy/agent.env.example` as the non-secret template.
The agent generates a WireGuard keypair when none is supplied, then stores the keypair and device token in `LINKBIT_STATE_PATH`.
After first enrollment, later restarts no longer need the one-time enrollment key.

One-shot enrollment check without changing network interfaces:

```bash
./linkbit-agent --controller http://203.0.113.10 --enrollment-key <token> --dry-run --once
```

Long-running agent with real WireGuard changes:

```bash
sudo ./linkbit-agent --controller http://203.0.113.10 --enrollment-key <token>
```

Install:

```bash
./scripts/build-linux-amd64.sh
sudo ./deploy/install-agent.sh
```

## First Device

Fresh controllers seed a `default-user` user and a `default` device group so the web console can generate an invitation immediately after you enter an admin API key.
The invitation panel shows both the raw token and a ready-to-run agent command.
