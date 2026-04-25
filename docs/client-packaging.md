# Client Packaging

Targets:

- Linux: systemd service installed by `deploy/install-agent.sh`.
- macOS: launchd service and DMG packaging.
- Windows: Windows service and MSI packaging.

Current implementation:

- Linux agent binary builds with `scripts/build-linux-amd64.sh`.
- WireGuard management is implemented through `ip` and `wg` on Linux.
- Tray and RustDesk integrations are represented by explicit agent boundaries.

Next packaging work:

- Add platform-specific service installers.
- Add tray implementations behind build tags.
- Add MSI and DMG release artifacts in the release pipeline.

