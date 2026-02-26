#!/bin/sh
set -e

BASE="$1"
if [ -z "$BASE" ]; then
  echo "Usage: curl -fsSL https://YOUR-SITE/install.sh | sh -s https://YOUR-SITE" >&2
  exit 1
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

URL="${BASE}/releases/tsr-${OS}-${ARCH}"
DEST="$HOME/.local/bin/tsr"
mkdir -p "$HOME/.local/bin"

echo "Downloading TSR for ${OS}/${ARCH}..."
curl -fsSL "$URL" -o "$DEST"
chmod +x "$DEST"

echo ""
echo "Installed to $DEST"
echo "Add to PATH if needed:  export PATH=\"\$HOME/.local/bin:\$PATH\""
echo "Then run: tsr run <yourfile.tsr>"
