#!/bin/sh
set -eu

repo="${BASELINE_REPO:-apollostreetcompany/baseline}"
version="${BASELINE_VERSION:-latest}"
install_dir="${BASELINE_INSTALL_DIR:-$HOME/.local/bin}"

die() {
  echo "baseline install: $*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

need curl
need tar

os="$(uname -s)"
case "$os" in
  Darwin|Linux) ;;
  *) die "unsupported OS: $os" ;;
esac

machine="$(uname -m)"
case "$machine" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64) arch="x86_64" ;;
  *) die "unsupported architecture: $machine" ;;
esac

asset="baseline_${os}_${arch}.tar.gz"
if [ "$version" = "latest" ]; then
  base_url="https://github.com/${repo}/releases/latest/download"
else
  base_url="https://github.com/${repo}/releases/download/${version}"
fi

tmp="$(mktemp -d 2>/dev/null || mktemp -d -t baseline)"
trap 'rm -rf "$tmp"' EXIT INT TERM

archive="$tmp/$asset"
checksums="$tmp/checksums.txt"

echo "Downloading $asset from $repo..."
curl -fsSL "$base_url/$asset" -o "$archive"
curl -fsSL "$base_url/checksums.txt" -o "$checksums"

expected="$(grep " ${asset}\$" "$checksums" | awk '{print $1}' || true)"
[ -n "$expected" ] || die "checksum entry missing for $asset"

if command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$archive" | awk '{print $1}')"
else
  die "missing shasum or sha256sum for checksum verification"
fi

[ "$actual" = "$expected" ] || die "checksum mismatch for $asset"

tar -xzf "$archive" -C "$tmp"
mkdir -p "$install_dir"
cp "$tmp/baseline" "$install_dir/baseline"
chmod 0755 "$install_dir/baseline"

echo "Installed baseline to $install_dir/baseline"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) echo "Add $install_dir to PATH, then run: baseline setup" ;;
esac
"$install_dir/baseline" --version >/dev/null 2>&1 || true
