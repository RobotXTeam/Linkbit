<p align="center">
  <img src="assets/logo.svg" width="132" alt="Linkbit logo" />
</p>

<h1 align="center">Linkbit</h1>

<p align="center">
  Self-hosted secure networking for low-latency SSH, desktop access, service sharing, and private device operations.
</p>

<p align="center">
  <a href="README.zh-CN.md">中文</a>
  ·
  <a href="docs/deployment.md">Deployment Guide</a>
  ·
  <a href="docs/openapi.yaml">OpenAPI</a>
  ·
  <a href="https://github.com/RobotXTeam/Linkbit/releases">Releases</a>
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white">
  <img alt="TypeScript" src="https://img.shields.io/badge/TypeScript-React-3178C6?logo=typescript&logoColor=white">
  <img alt="WireGuard" src="https://img.shields.io/badge/WireGuard-data%20plane-88171A?logo=wireguard&logoColor=white">
  <img alt="Self Hosted" src="https://img.shields.io/badge/Self--Hosted-ready-0F766E">
  <img alt="License" src="https://img.shields.io/badge/License-TBD-lightgrey">
</p>

---

## What Is Linkbit?

Linkbit is a self-hosted VPN and application integration platform designed for teams, homelabs, and private infrastructure operators who need secure device-to-device communication without handing the control plane to a third party.

It combines a public controller, a relay-capable network hub, endpoint agents, and a web console. Devices enroll with invitation tokens, receive virtual IPs, and can communicate through a cloud server hub when direct connectivity is not available.

## Product Capabilities

- Secure device enrollment with one-time invitation tokens and device-scoped credentials.
- Controller-managed virtual network with WireGuard-based data plane.
- Cloud server hub routing for NAT-heavy environments.
- Web console for devices, users, groups, relays, API keys, policies, and settings.
- Linux endpoint agent with automatic WireGuard identity generation and persistent state.
- Visual desktop client for launching the agent without CLI-only workflows.
- Release packaging for Linux, macOS, Windows, plus Linux AppImage desktop packaging.
- Real-world validation between a local workstation and an ARM64 OpenWrt/FriendlyWrt target.

## Architecture

```text
┌──────────────────────┐
│  Linkbit Controller  │
│  API · Web · SQLite  │
│  WireGuard Hub       │
└──────────┬───────────┘
           │ Cloud relay path
    ┌──────┴──────┐
    │             │
┌───▼───┐     ┌───▼───┐
│Agent A│     │Agent B│
│10.88.x│     │10.88.x│
└───────┘     └───────┘
```

Core components:

- `linkbit-controller`: authentication, device registry, policy API, relay registry, web console, and optional WireGuard hub.
- `linkbit-relay`: DERP-style relay service with controller registration and heartbeat.
- `linkbit-agent`: endpoint enrollment, WireGuard interface management, health reporting, and desktop integration boundary.
- `desktop/`: Electron visual client for non-CLI operation.
- `web/`: React + TypeScript management console.

## Verified Status

The current development deployment has been verified with:

- Controller and web console health checks.
- Relay health checks.
- API smoke tests, stress tests, and relay recovery tests.
- WireGuard hub route installation.
- Local workstation to ARM64 target communication through Linkbit virtual IPs.
- SSH over Linkbit virtual IP.

Observed Linkbit tunnel latency in the tested environment:

```text
local workstation -> cloud hub -> ARM64 target
average ICMP RTT: about 30 ms
SSH over Linkbit virtual IP: working
```

Remote desktop depends on the target device running a desktop service such as RDP, VNC, NoMachine, or RustDesk. Linkbit provides the private network path; the desktop protocol service must still be installed on the target OS.

## Quick Start

### 1. Build and test

```bash
make test
make smoke
make stress
make recovery-smoke
```

### 2. Build release packages

```bash
LINKBIT_VERSION=v0.2.0 ./scripts/package-release.sh
```

Artifacts are written to:

```text
artifacts/release/
```

### 3. Run a controller

Create controller configuration from the example:

```bash
cp deploy/controller.env.example /etc/linkbit/controller.env
```

Important production settings:

```env
LINKBIT_LISTEN_ADDR=:80
LINKBIT_PUBLIC_URL=https://controller.example.com
LINKBIT_API_KEY_PEPPER=replace-with-random-secret
LINKBIT_BOOTSTRAP_API_KEY=replace-with-admin-key
LINKBIT_HUB_WG_ENABLED=true
LINKBIT_HUB_WG_INTERFACE=linkbit-hub
LINKBIT_HUB_WG_IP=10.88.0.1
LINKBIT_HUB_WG_NETWORK=10.88.0.0/16
LINKBIT_HUB_WG_PORT=443
LINKBIT_HUB_WG_PRIVATE_KEY=replace-with-wireguard-private-key
LINKBIT_HUB_WG_ENDPOINT=controller.example.com:443
```

Install:

```bash
./deploy/install-controller.sh
./deploy/install-relay.sh
```

### 4. Enroll a device

Create an invitation from the web console, then run:

```bash
sudo ./linkbit-agent \
  --controller https://controller.example.com \
  --enrollment-key <token> \
  --name laptop \
  --interface linkbit0
```

For visual operation, use the Linkbit desktop client AppImage and enter the same controller URL and enrollment token.

## Repository Layout

```text
cmd/                    Go binary entrypoints
internal/controller/    Controller API and WireGuard hub
internal/agent/         Agent, WireGuard manager, state, health
internal/relay/         Relay node runtime
internal/store/         SQLite storage adapter
web/                    React management console
desktop/                Electron desktop client
deploy/                 Systemd install scripts and env examples
scripts/                Build, test, packaging, and deployment helpers
docs/                   Architecture, API, packaging, deployment notes
assets/                 Branding assets
```

## Security Model

- API keys and enrollment tokens are generated from cryptographically secure random bytes.
- Tokens are stored as HMAC-SHA256 digests, never plaintext.
- Device credentials are scoped to device APIs only.
- The controller can reject malformed WireGuard endpoints before distributing network config.
- Production deployments should use HTTPS for the controller and restrict admin API key access.
- Runtime secrets are loaded from environment files and excluded from Git.

## Roadmap

- Signed Windows MSI and macOS DMG desktop installers.
- Tray status icon and native service management.
- TCP fallback relay for environments where UDP is blocked by cloud ingress.
- RustDesk packaging and deep Linkbit identity integration.
- Multi-tenant RBAC and audit logs.
- High availability controller and external PostgreSQL backend.

## Commercial Positioning

Linkbit is intended to become a polished, self-hosted alternative for secure operations teams that need:

- Private infrastructure access without exposing SSH/RDP to the public internet.
- Predictable cloud-server relay behavior.
- Low operational overhead for adding devices.
- A control plane that can be owned, inspected, and deployed anywhere.

## License

License is not finalized yet. Add a project license before public commercial distribution.
