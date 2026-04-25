# Linkbit Core Dependency Analysis

## Controller

Primary role: authentication, device registry, policy distribution, relay registry, and admin API.

Planned open-source dependencies:

- Headscale: baseline control-plane behavior, node/user model, Tailscale-compatible coordination patterns.
- Tailscale Go packages: DERP map structures, WireGuard/tailnet protocol compatibility, netcheck concepts.
- Go standard library `net/http`, `crypto`, `database/sql`: portable secure baseline for the first controller API.
- SQLite via `modernc.org/sqlite`: embedded default database without CGO, improving cross-platform builds.
- PostgreSQL via `pgx`: optional production database for larger deployments.
- `golang.org/x/crypto`: Argon2id/password hashing and future certificate/key utilities.
- OpenTelemetry: structured tracing and metrics for controller, relay, and agent.

Current skeleton decision:

- The initial API boundary is implemented with the Go standard library to keep the first step auditable and easy to test.
- Headscale integration will be isolated behind controller service/store interfaces instead of leaking Headscale internals into clients or the web UI.
- API keys are HMAC-hashed with a server-side pepper. Plaintext keys are only shown once at creation.

## Relay Server

Primary role: DERP-compatible fallback relay with controller-managed registration and heartbeat.

Planned open-source dependencies:

- Tailscale `derp`, `derphttp`, and related packages: DERP protocol implementation.
- Go standard library `net/http`: controller registration, heartbeat, and health endpoint.
- OpenTelemetry/Prometheus client: relay load, latency, active sessions, and bandwidth metrics.

Current skeleton decision:

- Relay registration and heartbeat are implemented first.
- DERP serving is represented by an explicit interface so the binary can later swap in the Tailscale DERP server without changing lifecycle code.

## Client Agent

Primary role: device enrollment, WireGuard tunnel lifecycle, health reporting, and desktop integration.

Planned open-source dependencies:

- `golang.zx2c4.com/wireguard/wgctrl`: direct WireGuard device control where supported.
- `wireguard-tools`: fallback command execution through `wg`/`wg-quick`.
- Tailscale/netlink concepts: route and peer health monitoring.
- `getlantern/systray` or platform-specific tray bindings: system tray integration.
- RustDesk source tree: future customized remote-desktop package embedded in installers.

Current skeleton decision:

- The agent owns interfaces for registration, tunnel management, and health checks.
- OS-specific WireGuard and tray code will be added behind build tags.

## Web Management Console

Primary role: operations dashboard for devices, relay nodes, users, ACL/policies, and system settings.

Planned open-source dependencies:

- React + TypeScript + Vite: frontend runtime and build toolchain.
- shadcn/ui-compatible component structure: accessible primitives, local ownership of UI components.
- Radix UI primitives: accessible low-level UI behavior.
- lucide-react: consistent icon set.
- TanStack Query: controller API caching and mutation flow.
- Zod: runtime validation for API payloads.

