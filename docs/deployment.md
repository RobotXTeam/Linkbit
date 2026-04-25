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

## Relay

Required environment:

- `LINKBIT_CONTROLLER_URL`
- `LINKBIT_API_KEY`
- `LINKBIT_RELAY_ID`
- `LINKBIT_RELAY_PUBLIC_URL`

The relay currently performs controller registration and heartbeat. The DERP protocol server is isolated behind `internal/relay.DERPService` for the next integration step.

## Agent

Required environment:

- `LINKBIT_CONTROLLER_URL`
- `LINKBIT_ENROLLMENT_KEY`
- `LINKBIT_WG_PUBLIC_KEY`

The agent currently performs controller registration. WireGuard tunnel control and tray integration are isolated behind interfaces for OS-specific implementations.

