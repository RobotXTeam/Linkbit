#!/usr/bin/env sh
set -eu

out_dir="${LINKBIT_ARTIFACT_DIR:-artifacts/release}"
version="${LINKBIT_VERSION:-$(git describe --tags --always --dirty 2>/dev/null || printf '0.1.0-dev')}"

mkdir -p "$out_dir"
.tools/go/bin/go build -o bin/linkbit-agent ./cmd/linkbit-agent

(cd desktop && npm ci && npm run dist -- --linux AppImage)
cp desktop/dist/*.AppImage "$out_dir/linkbit-desktop_${version}_linux_amd64.AppImage"

echo "$out_dir/linkbit-desktop_${version}_linux_amd64.AppImage"
