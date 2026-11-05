#!/usr/bin/env sh
# Cross-compiles Traverse for all supported platforms into dist/ with SHA-256 checksums.
# Requires only Go (https://go.dev/dl/). Usage: scripts/build.sh [version]
set -eu
VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo dev)}"
cd "$(dirname "$0")/.."
rm -rf dist && mkdir -p dist
for target in windows/amd64 windows/arm64 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  os="${target%/*}"; arch="${target#*/}"; ext=""
  [ "$os" = windows ] && ext=".exe"
  out="dist/traverse-${VERSION}-${os}-${arch}${ext}"
  echo "building $out"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -buildvcs=false \
    -ldflags "-s -w -X main.version=${VERSION}" -o "$out" ./cmd/traverse
done
cd dist && (sha256sum traverse-* 2>/dev/null || shasum -a 256 traverse-*) > SHA256SUMS.txt && cat SHA256SUMS.txt
