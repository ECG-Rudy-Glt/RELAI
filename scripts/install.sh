#!/bin/sh
# Installs the relai binary for this OS/arch from the latest GitHub release.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/ECG-Rudy-Glt/RELAI/main/scripts/install.sh | sh
#
# Env overrides: RELAI_INSTALL_DIR (default /usr/local/bin; when it is not writable the script stops and
# says what to do, it never runs sudo by itself), RELAI_USE_SUDO=1 (allow `sudo mv` into that directory),
# RELAI_VERSION (default: latest),
# RELAI_REPO, RELAI_BASE_URL (download from somewhere other than this repo's GitHub releases; it must
# also serve checksums.txt), RELAI_SKIP_VERIFY=1 (install without checking the checksum: not advised).
#
# The archive is checked against the release's checksums.txt before anything is installed, and the
# install stops if it does not match. That catches a corrupted or swapped download; because the
# checksums come from the same place as the archive, it does not protect against a compromised release
# (use `gh attestation verify` for that, see SECURITY.md).
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
  base="$RELAI_BASE_URL"
else
  base="https://github.com/$repo/releases/download/v${version}"
fi
url="$base/$archive"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "relai: downloading $url"
curl -fsSL "$url" -o "$tmp/$archive"

if [ "${RELAI_SKIP_VERIFY:-}" = "1" ]; then
  echo "relai: WARNING: skipping checksum verification (RELAI_SKIP_VERIFY=1)" >&2
else
  if command -v sha256sum >/dev/null 2>&1; then
    sha256() { sha256sum "$1" | cut -d' ' -f1; }
  elif command -v shasum >/dev/null 2>&1; then
    sha256() { shasum -a 256 "$1" | cut -d' ' -f1; }
  else
    echo "relai: cannot verify the download: neither sha256sum nor shasum is installed (set RELAI_SKIP_VERIFY=1 to install anyway)" >&2
    exit 1
  fi
  if ! curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"; then
    echo "relai: could not download $base/checksums.txt, so the archive cannot be verified (set RELAI_SKIP_VERIFY=1 to install anyway)" >&2
    exit 1
  fi
  want=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")
  if [ -z "$want" ]; then
    echo "relai: $archive is not listed in checksums.txt: refusing to install" >&2
    exit 1
  fi
  got=$(sha256 "$tmp/$archive")
  if [ "$got" != "$want" ]; then
    echo "relai: CHECKSUM MISMATCH for $archive (expected $want, got $got): refusing to install" >&2
    exit 1
  fi
  echo "relai: checksum verified"
fi

tar -xzf "$tmp/$archive" -C "$tmp" relai

if [ -w "$bin_dir" ] 2>/dev/null; then
  mv "$tmp/relai" "$bin_dir/relai"
elif [ "${RELAI_USE_SUDO:-0}" = "1" ]; then
  echo "relai: $bin_dir is not writable, using sudo (RELAI_USE_SUDO=1)"
  sudo mv "$tmp/relai" "$bin_dir/relai"
else
  echo "relai: $bin_dir is not writable, nothing was installed." >&2
  echo "relai: choose a directory you own:   RELAI_INSTALL_DIR=\$HOME/.local/bin   (and put it on your PATH)" >&2
  echo "relai: or allow sudo for this run:   RELAI_USE_SUDO=1" >&2
  exit 1
fi
chmod +x "$bin_dir/relai"

echo "relai: installed to $bin_dir/relai"
"$bin_dir/relai" version
