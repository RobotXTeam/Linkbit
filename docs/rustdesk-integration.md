# RustDesk Integration Plan

Linkbit will package a customized RustDesk client after the VPN control path is stable.

Current repository boundary:

- `internal/agent/rustdesk.go` defines the launcher boundary.
- The agent owns device enrollment, WireGuard setup, and health reporting.
- RustDesk customization should consume Linkbit virtual IPs and device identity after network config is available.

Implementation stages:

1. Vendor or submodule RustDesk source under a separate build workspace.
2. Replace default rendezvous/relay configuration with Linkbit virtual network discovery.
3. Use Linkbit device identity as the second authentication factor.
4. Package RustDesk together with the Linkbit agent installers.

RustDesk binaries and source archives are not committed to this repository.

