# macOS Packaging

Planned artifact: DMG installer for `linkbit-agent`.

Current release pipeline builds the Darwin agent executable through GoReleaser. DMG authoring should be added after launchd and tray integration are implemented.

Required installer behavior:

- Install the agent binary and launchd plist.
- Store configuration under `/Library/Application Support/Linkbit`.
- Do not auto-start without explicit user or administrator choice.

