# Linkbit

Linkbit is a self-hosted, security-first low-latency communication platform. It is planned as three decoupled components:

- `linkbit-controller`: public control plane for users, devices, policies, API keys, and relay registry.
- `linkbit-relay`: independently deployable DERP-compatible relay node with controller registration and heartbeat.
- `linkbit-agent`: cross-platform endpoint service for registration, WireGuard tunnel management, health checks, and desktop integration.

This repository currently contains the first-step project skeleton, dependency analysis, secure API-key primitives, basic controller APIs, and a React management console scaffold.

## Current Status

- SQLite-backed controller migrations and persistence.
- Relay registration and heartbeat APIs.
- Device enrollment invitations and registration API.
- HMAC-backed API key and enrollment token handling.
- Agent HTTP registration client.
- React management console wired to controller overview, devices, relays, policies, and invitation creation.
- CI workflow for Go tests and web checks.
- goreleaser skeleton for controller, relay, and agent binaries.
- Controller can serve the built web console directly.
- Docker/Compose and Caddy templates are included.
- Linux agent has a WireGuard command manager; tray and RustDesk boundaries are in place.
- Remote deployment helpers build binaries, upload the web console, and verify controller/relay health.

## Repository Layout

```text
cmd/
  linkbit-controller/   Controller entrypoint
  linkbit-relay/        Relay node entrypoint
  linkbit-agent/        Client agent entrypoint
internal/
  auth/                 API key generation, hashing, verification
  config/               Environment-backed configuration
  controller/           Controller HTTP API skeleton
  relay/                Relay registration and heartbeat loop
  agent/                Client registration, tunnel and health abstractions
  models/               Shared domain models
  store/                Storage adapters
pkg/linkbitapi/         Public API constants shared by binaries
web/                    React + TypeScript management console
docs/                   Architecture and dependency notes
deploy/                 Deployment scripts
scripts/                Local development helpers
```

## Local Toolchain

This workspace can use either a system Go toolchain or the project-local `.tools/go/bin/go` toolchain. Run:

```bash
make test
```

The web scaffold can be installed and checked with:

```bash
cd web
npm install
npm run typecheck
```

## Security Defaults

- API keys are generated from cryptographically secure random bytes.
- API keys must be stored as HMAC-SHA256 digests, never plaintext.
- Controller/relay/agent communication is designed to require TLS in production.
- Runtime secrets are read from environment variables and excluded from Git.

## Development Quick Start

```bash
make check
make test
./scripts/dev-controller.sh
```

Open the web console:

```bash
cd web
npm install
npm run dev
```

Use the default development admin key `dev-admin-key` only for local testing.

## Remote Deployment

For a direct server install, use public `80` for the controller and `443` for the relay. For a reverse-proxy install, keep the services on internal ports such as `8080/8443` and expose the proxy.

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

Then run `./scripts/remote-install.sh`, copy the generated env files to `/etc/linkbit/`, run the install scripts on the server, and verify with `make remote-health`.
