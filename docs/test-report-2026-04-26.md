# Linkbit Test Report

Date: 2026-04-26  
Tester role: QA / release validation  
Environment:

- Local workstation: Ubuntu Linux, amd64
- Controller / hub server: public Linux server
- Target device: FriendlyWrt / OpenWrt, ARM64
- Controller URL used in tests: redacted from repository
- Linkbit virtual network: `10.88.0.0/16`

## Executive Summary

Linkbit now has a working controller, web console, Linux system agent, visual desktop client, WireGuard hub, and TCP relay fallback path. The most important regression fix in this run is the TCP relay path: local SSH to the FriendlyWrt target succeeds through the cloud controller even when the WireGuard virtual-IP path is unstable.

The product is usable for controlled beta testing, including local workstation plus FriendlyWrt validation. It is not yet a final commercial release because the UDP/WireGuard path still shows packet loss in this environment, remote desktop depends on an installed target desktop service, and signed GUI installers are not complete.

## Test Matrix

| Area | Test | Result | Evidence |
| --- | --- | --- | --- |
| Go unit tests | `make test` | Pass | All Go packages passed |
| Go race tests | `go test -race ./internal/...` | Pass | Internal packages passed |
| Go static checks | `go vet ./...` | Pass | No vet findings |
| API smoke | `make smoke` | Pass | Controller register/list flow passed |
| API stress | `make stress` | Pass | 10 devices and 2 relays simulated |
| Relay recovery | `make recovery-smoke` | Pass | Relay recovery smoke passed |
| Web typecheck | `npm run typecheck` | Pass | TypeScript no errors |
| Web build | `npm run build` | Pass | Vite production build passed |
| Web npm audit | `npm audit --audit-level=high` | Pass | 0 vulnerabilities |
| Desktop dependency audit | `npm audit --audit-level=high` | Pass | 0 vulnerabilities |
| Desktop package | `npm run pack` | Pass | Linux unpacked app generated |
| Local GUI install | AppImage install | Pass | Installed locally |
| Local Agent install | systemd service | Pass | `linkbit-agent.service` active |
| Controller health | `/healthz` | Pass | Remote health check passed |
| Relay health | `/healthz` | Pass | Remote health check passed |
| Hub interface | `linkbit-hub` | Pass | Server has `10.88.0.1/16` and WG listen port |
| Local interface | `linkbit0` | Pass | Local has `10.88.235.251/32` |
| FriendlyWrt install | OpenWrt init service | Pass | `/usr/local/bin/linkbit-agent` running under `/etc/init.d/linkbit-agent` |
| FriendlyWrt interface | `linkbit0` | Pass | Target has `10.88.92.200/32` |
| TCP relay SSH | Local -> controller -> FriendlyWrt SSH | Pass | 5 real SSH handshakes succeeded |
| TCP relay long-poll | Session wakeup latency | Pass | SSH handshakes after fix: 0.29s to 0.84s |
| WireGuard hub ping | Local -> hub virtual IP | Degraded | Avg 15.5 ms in sample, 50% packet loss |
| End-to-end WireGuard ping | Local -> FriendlyWrt virtual IP | Fail in latest sample | 100% packet loss |
| Remote desktop port | FriendlyWrt RDP/VNC/RustDesk | Not applicable on target | Router has no desktop service listening |

## Live Connectivity Results

### Cloud TCP Relay

The new TCP relay fallback is verified end to end:

```text
local workstation -> Linkbit controller TCP relay -> FriendlyWrt SSH
result: success
```

Five SSH handshakes over the Linkbit relay after long-poll wakeup:

```text
0.40 sec
0.84 sec
0.31 sec
0.33 sec
0.29 sec
```

The target-to-cloud latency sample from FriendlyWrt:

```text
min/avg/max = 6.959/7.294/7.704 ms
loss = 0%
```

This means SSH is usable through Linkbit's cloud relay path. RDP is supported by the same TCP forwarding mechanism when the target device has an RDP service listening on port `3389`.

### WireGuard Hub

The controller server is running the WireGuard hub interface:

```text
linkbit-hub: 10.88.0.1/16
listen port: 443
```

The hub sees both peers:

```text
local workstation: 10.88.235.251/32
FriendlyWrt target: 10.88.92.200/32
```

Latest WireGuard ping samples are degraded:

```text
local -> hub virtual IP: avg 15.5 ms, 50% packet loss
local -> FriendlyWrt virtual IP: 100% packet loss
```

This is why TCP relay is now mandatory and enabled by default. The WireGuard path remains useful when UDP is healthy, but it is not the only path required for product usability.

### FriendlyWrt Target

FriendlyWrt now has a persistent Linkbit service:

```text
/usr/local/bin/linkbit-agent
/etc/init.d/linkbit-agent
/etc/linkbit/agent-state.json
```

The running service loads existing device credentials and polls the controller for TCP relay sessions.

## Functional Coverage

Implemented and passing:

- Controller startup and health.
- SQLite migrations and seeded default user/group.
- API key authentication and scoped relay keys.
- Invitation creation and device enrollment.
- Agent state persistence and automatic WireGuard key generation.
- Device list, delete/revoke, and health updates.
- Relay registration, heartbeat, and DERP map generation.
- Web console dashboard, devices, relays, policies, users, groups, API keys, and settings.
- Release packaging for Linux/macOS/Windows CLI/server binaries.
- Linux visual desktop client build and local install.
- WireGuard hub route distribution.
- TCP relay fallback for SSH/RDP-style TCP traffic.
- Desktop client controls for starting the Agent and local SSH/RDP forwarding.
- FriendlyWrt ARM64 Agent installation as an init service.

Remaining product gaps:

- WireGuard UDP path still needs automatic recovery and better endpoint cleanup.
- Native signed MSI/DMG GUI installers are not complete.
- RustDesk deep integration and bundled installer are not complete.
- HTTPS/Let's Encrypt automation is documented but not fully automated in this deployment.
- Full RBAC, audit logging, and HA/PostgreSQL mode are not complete.

## Security Results

Passed:

- Web frontend audit: 0 high vulnerabilities.
- Desktop audit: 0 high vulnerabilities.
- Go vet: no findings.
- Device relay sessions require device token authentication.
- Relay sessions are policy-gated before stream creation.
- Runtime secrets remain out of repository files.

Risks:

- This test deployment still uses HTTP between agents and the controller.
- Bootstrap/admin keys are manually handled.
- Desktop app is unsigned.
- Router target has no desktop service, so remote desktop cannot be validated there.

## Defects

### P0: WireGuard virtual-IP path unstable

Symptoms:

- Local to FriendlyWrt virtual IP ping failed during final regression.
- Local to hub virtual IP had packet loss.

Impact:

- UDP/WireGuard cannot be the only transport.

Current mitigation:

- TCP relay fallback is implemented, enabled, and verified for SSH.

Required fix:

- Add periodic WireGuard re-apply or endpoint refresh.
- Add peer health telemetry to the UI.
- Add automated stale endpoint cleanup.

### P1: Remote desktop not validated on FriendlyWrt

Symptoms:

- FriendlyWrt has no RDP/VNC/RustDesk desktop service.

Impact:

- Linkbit can forward TCP ports for remote desktop, but this specific router target cannot prove a desktop session.

Required fix:

- Validate RDP on Windows or RustDesk/NoMachine on a desktop Linux/macOS target.
- Keep FriendlyWrt validation focused on SSH and router admin traffic.

### P1: Release GUI assets still need final publishing

Symptoms:

- CLI/server release assets exist, but GUI publishing needs a reliable upload pass.

Impact:

- Local GUI works, but external GUI distribution is incomplete.

Required fix:

- Re-run GUI packaging and upload AppImage/DMG/MSI assets with checksums.

## QA Verdict

Controlled beta ready, not final commercial release.

The blocker that made the product unusable in this environment has been fixed: SSH now works through the cloud TCP relay even while WireGuard UDP is unreliable. The remaining work is hardening, signed GUI installers, RustDesk integration, HTTPS automation, and WireGuard recovery.
