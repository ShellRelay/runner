#!/bin/bash
set -e

# ── ShellRelay Runner Entrypoint ─────────────────────────────────────────────
#
# Handles two modes:
#   1. Manual mode: SHELLRELAY_TOKEN is set → run immediately
#   2. Announce mode: SHELLRELAY_EMAIL is set → announce first, then run
#
# In announce mode, the runner self-registers with the relay server on first
# boot. The owner must claim it via the dashboard using the token printed to
# logs. On subsequent boots, the saved config is reused.

CONFIG_FILE="$HOME/.shellrelay/config"

# If token is already set (manual mode), just run
if [ -n "$SHELLRELAY_TOKEN" ] && [ -n "$SHELLRELAY_SERVER_ID" ]; then
    echo "[entrypoint] Manual mode — running with provided token"
    exec shellrelay run
fi

# If email is set, use announce mode
if [ -n "$SHELLRELAY_EMAIL" ]; then
    SERVER_ID="${SHELLRELAY_SERVER_ID:-$(hostname)}"
    SERVER_NAME="${SHELLRELAY_SERVER_NAME:-$SERVER_ID}"

    # Announce (skips if already announced and config is saved)
    shellrelay announce "$SERVER_ID" --email "$SHELLRELAY_EMAIL" --name "$SERVER_NAME"

    # Now run using saved config from announce
    exec shellrelay run
fi

echo "[entrypoint] ERROR: Set either SHELLRELAY_TOKEN + SHELLRELAY_SERVER_ID (manual mode)"
echo "             or SHELLRELAY_EMAIL (announce mode)"
exit 1
