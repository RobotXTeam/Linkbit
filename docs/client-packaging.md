# Client Packaging

Targets:

- Linux: systemd service installed by `deploy/install-agent.sh`.
- macOS: launchd service and DMG packaging.
- Windows: Windows service and MSI packaging.

Current implementation:

- Linux agent binary builds with `scripts/build-linux-amd64.sh`.
- WireGuard management is implemented through `ip` and `wg` on Linux.
- The `desktop/` Electron client provides a visual launcher for controller URL, enrollment key, device name, state path, and live agent logs.
- Linux AppImage packaging is included in `scripts/package-release.sh` when building on Linux amd64.
- RustDesk integration is represented by an explicit launcher boundary and can use Linkbit virtual IPs once a desktop service is installed on the target.

Next packaging work:

- Add platform-specific service installers.
- Add signed MSI and DMG desktop artifacts in the release pipeline.
- Add tray status icons around the desktop client.
