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

## Relay

Required environment:

- `LINKBIT_CONTROLLER_URL`
- `LINKBIT_API_KEY`
- `LINKBIT_RELAY_ID`
- `LINKBIT_RELAY_PUBLIC_URL`

The relay performs controller registration, heartbeat, and runs a DERP-compatible HTTP service mounted at `/derp`.
Use `deploy/relay.env.example` as the non-secret template.

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

## Agent

Required environment:

- `LINKBIT_CONTROLLER_URL`
- `LINKBIT_ENROLLMENT_KEY`
- `LINKBIT_WG_PUBLIC_KEY`
- `LINKBIT_WG_PRIVATE_KEY`

The Linux agent performs controller registration, creates a WireGuard interface through `ip` and `wg`, and reports health back to the controller with a device-scoped token.
Use `deploy/agent.env.example` as the non-secret template.

Install:

```bash
./scripts/build-linux-amd64.sh
sudo ./deploy/install-agent.sh
```
