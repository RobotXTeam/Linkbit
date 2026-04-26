#!/usr/bin/env sh
set -eu

GO_BIN="${GO:-.tools/go/bin/go}"
version="${LINKBIT_VERSION:-$(git describe --tags --always --dirty 2>/dev/null || printf '0.1.0-dev')}"
commit="${LINKBIT_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || printf 'unknown')}"
date="${LINKBIT_BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
out_dir="${LINKBIT_ARTIFACT_DIR:-artifacts/release}"

ldflags="-s -w -X github.com/linkbit/linkbit/internal/version.Version=$version -X github.com/linkbit/linkbit/internal/version.Commit=$commit -X github.com/linkbit/linkbit/internal/version.Date=$date"

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$1 is required" >&2
    exit 1
  fi
}

require tar
require zip

rm -rf "$out_dir"
mkdir -p "$out_dir"

(cd web && npm ci && npm run build)

build_platform() {
  goos="$1"
  goarch="$2"
  ext=""
  if [ "$goos" = "windows" ]; then
    ext=".exe"
  fi
  build_dir="$out_dir/build/linkbit_${version}_${goos}_${goarch}"
  pkg_dir="$build_dir/linkbit"
  mkdir -p "$pkg_dir/bin" "$pkg_dir/web" "$pkg_dir/deploy" "$pkg_dir/docs" "$pkg_dir/packaging"

  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" "$GO_BIN" build -ldflags "$ldflags" -o "$pkg_dir/bin/linkbit-controller$ext" ./cmd/linkbit-controller
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" "$GO_BIN" build -ldflags "$ldflags" -o "$pkg_dir/bin/linkbit-relay$ext" ./cmd/linkbit-relay
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" "$GO_BIN" build -ldflags "$ldflags" -o "$pkg_dir/bin/linkbit-agent$ext" ./cmd/linkbit-agent

  cp README.md "$pkg_dir/"
  cp README.zh-CN.md "$pkg_dir/" 2>/dev/null || true
  cp -R assets "$pkg_dir/" 2>/dev/null || true
  cp -R web/dist/. "$pkg_dir/web/"
  cp deploy/*.example "$pkg_dir/deploy/"
  cp deploy/install-controller.sh deploy/install-relay.sh deploy/install-agent.sh "$pkg_dir/deploy/" 2>/dev/null || true
  cp docs/deployment.md docs/api-skeleton.md docs/openapi.yaml "$pkg_dir/docs/"

  case "$goos" in
    linux)
      cp -R packaging/linux "$pkg_dir/packaging/" 2>/dev/null || true
      (cd "$build_dir" && tar -czf "../../linkbit_${version}_${goos}_${goarch}.tar.gz" linkbit)
      ;;
    darwin)
      cp -R packaging/macos "$pkg_dir/packaging/"
      (cd "$build_dir" && tar -czf "../../linkbit_${version}_${goos}_${goarch}.tar.gz" linkbit)
      ;;
    windows)
      cp -R packaging/windows "$pkg_dir/packaging/"
      (cd "$build_dir" && zip -qr "../../linkbit_${version}_${goos}_${goarch}.zip" linkbit)
      ;;
  esac
}

build_platform linux amd64
build_platform linux arm64
build_platform darwin amd64
build_platform darwin arm64
build_platform windows amd64

if [ "${LINKBIT_BUILD_DESKTOP:-1}" = "1" ] && [ "$(uname -s)" = "Linux" ] && [ "$(uname -m)" = "x86_64" ] && [ -d desktop ]; then
  (cd desktop && npm ci && npm run dist -- --linux AppImage)
  cp desktop/dist/*.AppImage "$out_dir/linkbit-desktop_${version}_linux_amd64.AppImage"
fi

(cd "$out_dir" && sha256sum linkbit_"$version"_* linkbit-desktop_"$version"_* 2>/dev/null > checksums.txt)

echo "release artifacts:"
find "$out_dir" -maxdepth 1 -type f -print | sort
