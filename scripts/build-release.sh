#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-"$ROOT_DIR/dist"}"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

build_one() {
  local goos="$1"
  local goarch="$2"
  local suffix="$3"
  local outdir="$DIST_DIR/baseline_$suffix"

  mkdir -p "$outdir"
  echo "building baseline_$suffix"
  (
    cd "$ROOT_DIR"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags="-s -w" -o "$outdir/baseline" ./cmd/baseline
  )
  tar -czf "$DIST_DIR/baseline_$suffix.tar.gz" -C "$outdir" baseline
}

build_one darwin arm64 Darwin_arm64
build_one darwin amd64 Darwin_x86_64
build_one linux arm64 Linux_arm64
build_one linux amd64 Linux_x86_64

tar -czf "$DIST_DIR/baseline-openclaw-plugin.tgz" -C "$ROOT_DIR/openclaw-plugin" .
tar -czf "$DIST_DIR/baseline-codex-plugin.tgz" -C "$ROOT_DIR/plugins/baseline" .

(
  cd "$DIST_DIR"
  shasum -a 256 baseline_*.tar.gz baseline-openclaw-plugin.tgz baseline-codex-plugin.tgz > checksums.txt
)

echo "release artifacts written to $DIST_DIR"
