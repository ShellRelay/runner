#!/usr/bin/env bash
# Install shellrelay runner — downloads pre-built binary for current platform
set -euo pipefail

REPO="ShellRelay/runner"
BINARY="shellrelay"

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Choose install directory: /usr/local/bin if writable, else ~/.local/bin
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ] 2>/dev/null; then
  INSTALL_DIR="$HOME/.local/bin"
fi

# Fetch latest release tag
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
ASSET_NAME="shellrelay-${OS}-${ARCH}"
URL="https://github.com/$REPO/releases/download/$LATEST/${ASSET_NAME}"
CHECKSUMS_URL="https://github.com/$REPO/releases/download/$LATEST/checksums.txt"

mkdir -p "$INSTALL_DIR"
echo "Downloading $BINARY $LATEST ($OS/$ARCH) ..."

# Download to a temp file first
TMP=$(mktemp "${INSTALL_DIR}/${BINARY}.XXXXXX")
trap 'rm -f "$TMP"' EXIT

curl -fsSL -o "$TMP" "$URL"

# Verify checksum if checksums file is available
if curl -fsSL -o /dev/null --head "$CHECKSUMS_URL" 2>/dev/null; then
  CHECKSUMS=$(curl -fsSL "$CHECKSUMS_URL")
  EXPECTED=$(echo "$CHECKSUMS" | grep "$ASSET_NAME" | awk '{print $1}')
  if [ -n "$EXPECTED" ]; then
    # Compute SHA256 of downloaded binary
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL=$(sha256sum "$TMP" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL=$(shasum -a 256 "$TMP" | awk '{print $1}')
    else
      echo "Warning: no sha256sum or shasum available — skipping checksum verification"
      ACTUAL="$EXPECTED"
    fi
    if [ "$ACTUAL" != "$EXPECTED" ]; then
      echo "Checksum verification FAILED!"
      echo "  Expected: $EXPECTED"
      echo "  Actual:   $ACTUAL"
      exit 1
    fi
    echo "Checksum verified."
  fi
fi

chmod +x "$TMP"
mv "$TMP" "$INSTALL_DIR/$BINARY"
trap - EXIT  # clear trap since mv succeeded

echo ""
echo "Installed to $INSTALL_DIR/$BINARY"

# Check PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "Add $INSTALL_DIR to your PATH:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    echo ""
    echo "Then add it to your shell profile (~/.zshrc or ~/.bashrc)"
    ;;
esac

echo ""
echo "Get started:"
echo "  shellrelay start <server-id> <server-token>"
echo ""
echo "Other commands: stop, restart, status, rotate"
