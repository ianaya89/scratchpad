#!/bin/sh
# pad installer — downloads the right release tarball and installs the binary.
#
#   curl -fsSL https://raw.githubusercontent.com/ianaya89/scratchpad/main/install.sh | sh
#
# Env overrides:
#   PAD_VERSION       tag to install (default: latest release)
#   PAD_INSTALL_DIR   install directory (default: ~/.local/bin)
set -eu

REPO="ianaya89/scratchpad"
BINARY="pad"
INSTALL_DIR="${PAD_INSTALL_DIR:-$HOME/.local/bin}"

die() { echo "error: $*" >&2; exit 1; }

command -v curl >/dev/null 2>&1 || die "curl is required"
command -v tar >/dev/null 2>&1 || die "tar is required"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) die "unsupported architecture: $arch" ;;
esac
case "$os" in
  linux | darwin) ;;
  *) die "unsupported OS: $os" ;;
esac

version="${PAD_VERSION:-}"
if [ -z "$version" ]; then
  # resolve latest via the releases/latest redirect (no API, no rate limit)
  version=$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
    "https://github.com/$REPO/releases/latest" | sed -E 's#.*/tag/##')
fi
[ -n "$version" ] || die "could not determine latest version"

ver="${version#v}"
tarball="${BINARY}_${ver}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$version"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $tarball ($version)..."
curl -fsSL "$base/$tarball" -o "$tmp/$tarball" || die "download failed: $base/$tarball"

# verify checksum when available (non-fatal if tools/file are missing)
if curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt" 2>/dev/null; then
  sum=""
  if command -v sha256sum >/dev/null 2>&1; then sum="sha256sum -c"; fi
  if [ -z "$sum" ] && command -v shasum >/dev/null 2>&1; then sum="shasum -a 256 -c"; fi
  if [ -n "$sum" ]; then
    ( cd "$tmp" && grep " ${tarball}\$" checksums.txt | $sum - >/dev/null 2>&1 ) \
      && echo "Checksum OK" || die "checksum verification failed"
  fi
fi

tar -xzf "$tmp/$tarball" -C "$tmp"
[ -f "$tmp/$BINARY" ] || die "binary not found in archive"

mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmp/$BINARY" "$INSTALL_DIR/$BINARY"
echo "Installed $BINARY $version -> $INSTALL_DIR/$BINARY"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "Note: $INSTALL_DIR is not on your PATH — add it to use '$BINARY'." ;;
esac
