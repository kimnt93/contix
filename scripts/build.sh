#!/usr/bin/env bash
# Cross-compile contix binaries for Linux, macOS and Windows.
#
#   ./scripts/build.sh [version]
#
# Output goes to ./dist/ as stable-named raw binaries (no version in the
# filename) so `make install` can pick the right one for the host platform
# without rebuilding. A SHA256SUMS file is written alongside them.
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${1:-$(tr -d '\r\n' < release/VERSION 2>/dev/null || echo dev)}"
LDFLAGS="-s -w -X contix/internal/cli.Version=${VERSION}"
DIST="dist"

# GOOS/GOARCH pairs to build.
TARGETS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
  "windows/arm64"
)

rm -rf "$DIST"
mkdir -p "$DIST"

echo "Building contix ${VERSION}"
for t in "${TARGETS[@]}"; do
  os="${t%/*}"; arch="${t#*/}"
  out="contix-${os}-${arch}"
  [ "$os" = "windows" ] && out="${out}.exe"

  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "$LDFLAGS" -o "${DIST}/${out}" .
  echo "  ${out}"
done

# Checksums.
(cd "$DIST" && sha256sum ./contix-* > SHA256SUMS 2>/dev/null || shasum -a 256 ./contix-* > SHA256SUMS)
echo "Done. Binaries in ${DIST}/"
