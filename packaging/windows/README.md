# Windows Packaging

Planned artifact: MSI installer for `linkbit-agent`.

Current release pipeline builds the Windows agent executable through GoReleaser. MSI authoring should be added with WiX after the Windows service wrapper is implemented.

Required installer behavior:

- Install `linkbit-agent.exe`.
- Register a Windows service.
- Store configuration outside the installation directory.
- Do not auto-start without explicit user or administrator choice.

