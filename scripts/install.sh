#!/bin/sh
# Installs the relai binary for this OS/arch from the latest GitHub release.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/ECG-Rudy-Glt/RELAI/main/scripts/install.sh | sh
#
# Env overrides: RELAI_INSTALL_DIR (default /usr/local/bin), RELAI_VERSION (default: latest),
# RELAI_REPO, RELAI_BASE_URL (download from somewhere other than this repo's GitHub releases).
set -eu

repo="${RELAI_REPO:-ECG-Rudy-Glt/RELAI}"
bin_dir="${RELAI_INSTALL_DIR:-/usr/local/bin}"

os=$(uname -s)
case "$os" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) echo "relai: unsupported OS: $os (only Linux and macOS binaries are published)" >&2; exit 1 ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "relai: unsupported architecture: $arch" >&2; exit 1 ;;
esac

version="${RELAI_VERSION:-}"
if [ -z "$version" ]; then
  version=$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" \
    | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"v?([^"]+)".*/\1/')
fi
if [ -z "$version" ]; then
  echo "relai: could not determine the latest release version" >&2
  exit 1
fi

archive="relai_${version}_${os}_${arch}.tar.gz"
if [ -n "${RELAI_BASE_URL:-}" ]; then
  url="$RELAI_BASE_URL/$archive"
else
  url="https://github.com/$repo/releases/download/v${version}/${archive}"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "relai: downloading $url"
curl -fsSL "$url" -o "$tmp/$archive"
tar -xzf "$tmp/$archive" -C "$tmp" relai

if [ -w "$bin_dir" ] 2>/dev/null; then
  mv "$tmp/relai" "$bin_dir/relai"
else
  echo "relai: $bin_dir is not writable, retrying with sudo"
  sudo mv "$tmp/relai" "$bin_dir/relai"
fi
chmod +x "$bin_dir/relai"

echo "relai: installed to $bin_dir/relai"
"$bin_dir/relai" version
