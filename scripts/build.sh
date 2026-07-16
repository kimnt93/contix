#!/usr/bin/env bash
# Cross-compile contix release binaries for Linux, macOS and Windows.
#
#   ./scripts/build.sh [version]
#
# Output goes to ./dist/ with a SHA256SUMS checksum file.
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
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
  out="contix-${VERSION}-${os}-${arch}"
  bin="contix"
  [ "$os" = "windows" ] && bin="contix.exe"

  workdir="${DIST}/${out}"
  mkdir -p "$workdir"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "$LDFLAGS" -o "${workdir}/${bin}" .
  cp README.md "$workdir/" 2>/dev/null || true

  # Archive: .zip for Windows, .tar.gz otherwise.
  if [ "$os" = "windows" ]; then
    (cd "$DIST" && zip -qr "${out}.zip" "$out")
    archive="${out}.zip"
  else
    (cd "$DIST" && tar czf "${out}.tar.gz" "$out")
    archive="${out}.tar.gz"
  fi
  rm -rf "$workdir"
  echo "  ${archive}"
done

# Checksums.
(cd "$DIST" && sha256sum ./* > SHA256SUMS 2>/dev/null || shasum -a 256 ./* > SHA256SUMS)
echo "Done. Artifacts in ${DIST}/"
