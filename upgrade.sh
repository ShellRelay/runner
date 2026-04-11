#!/bin/bash
# Build and install the shellrelay runner from the local source tree.
#
# Usage:
#   cd runner && ./upgrade.sh
#
# What it does:
#   1. Reads VERSION from runner/VERSION
#   2. Builds the Go binary with the version baked in
#   3. Installs to ~/.local/bin/shellrelay
#   4. Restarts the daemon if it was running
#   5. Strips stale SHELLRELAY_URL from ~/.shellrelay/config (if present)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION=$(cat "$SCRIPT_DIR/VERSION" | tr -d '[:space:]')
INSTALL_DIR="$HOME/.local/bin"
INSTALL_PATH="$INSTALL_DIR/shellrelay"
CONFIG_FILE="$HOME/.shellrelay/config"

echo "Building shellrelay v${VERSION} ..."

cd "$SCRIPT_DIR"
CGO_ENABLED=0 go build \
  -ldflags "-s -w -X main.Version=${VERSION}" \
  -o /tmp/shellrelay ./cmd/shellrelay

mkdir -p "$INSTALL_DIR"
mv /tmp/shellrelay "$INSTALL_PATH"
chmod +x "$INSTALL_PATH"

echo "Installed: $INSTALL_PATH"
echo "Version:   $("$INSTALL_PATH" version)"

# Remove stale SHELLRELAY_URL from config (if present) to prevent old
# URLs from being read by older code paths or manual inspection confusion.
if [ -f "$CONFIG_FILE" ]; then
  if grep -q '^SHELLRELAY_URL=' "$CONFIG_FILE" 2>/dev/null; then
    sed -i '' '/^SHELLRELAY_URL=/d' "$CONFIG_FILE" 2>/dev/null || \
    sed -i '/^SHELLRELAY_URL=/d' "$CONFIG_FILE" 2>/dev/null || true
    echo "Cleaned stale SHELLRELAY_URL from config"
  fi
fi

# Restart daemon if running
if "$INSTALL_PATH" status 2>/dev/null | grep -q "is running"; then
  echo "Restarting daemon ..."
  "$INSTALL_PATH" restart
else
  echo "Daemon not running (use 'shellrelay start' to start)"
fi

echo "Done."
