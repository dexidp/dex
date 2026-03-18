#!/usr/bin/env bash
#
# OIDC Conformance Test Runner
#
# Starts Dex with a test configuration and exposes it via a public tunnel
# for use with https://www.certification.openid.net/
#
# Usage:
#   ./run.sh                         # uses cloudflared (default)
#   ./run.sh --tunnel ngrok          # uses ngrok
#   ./run.sh --url https://my.url    # uses a pre-existing public URL (no tunnel)
#   ./run.sh --alias my-dex          # custom alias for the test plan (default: dex)
#
# Prerequisites:
#   - Dex binary in PATH or ../../bin/dex
#   - ngrok or cloudflared installed (unless --url is provided)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
DEX_PORT=5556
TUNNEL_TYPE="cloudflared"
PUBLIC_URL=""
ALIAS="dex"

while [[ $# -gt 0 ]]; do
    case $1 in
        --tunnel)  TUNNEL_TYPE="$2"; shift 2 ;;
        --url)     PUBLIC_URL="$2"; shift 2 ;;
        --alias)   ALIAS="$2"; shift 2 ;;
        --port)    DEX_PORT="$2"; shift 2 ;;
        -h|--help)
            sed -n '2,/^$/p' "$0" | sed 's/^# \?//'
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Find dex binary.
DEX_BIN=""
for candidate in "dex" "$ROOT_DIR/bin/dex"; do
    if command -v "$candidate" &>/dev/null || [[ -x "$candidate" ]]; then
        DEX_BIN="$candidate"
        break
    fi
done
if [[ -z "$DEX_BIN" ]]; then
    echo "Error: dex binary not found. Run 'make build' first or install dex."
    exit 1
fi

cleanup() {
    echo ""
    echo "Shutting down..."
    kill "${TUNNEL_PID:-}" "${DEX_PID:-}" 2>/dev/null || true
    rm -f "$CONFIG_FILE"
    wait 2>/dev/null
}
trap cleanup EXIT

# Start tunnel if no URL provided.
TUNNEL_PID=""
if [[ -z "$PUBLIC_URL" ]]; then
    case "$TUNNEL_TYPE" in
        ngrok)
            if ! command -v ngrok &>/dev/null; then
                echo "Error: ngrok not found. Install it from https://ngrok.com/ or use --url."
                exit 1
            fi
            ngrok http "$DEX_PORT" --log=stdout --log-level=warn &>/dev/null &
            TUNNEL_PID=$!
            echo "Waiting for ngrok tunnel..."
            sleep 3
            PUBLIC_URL=$(curl -s http://localhost:4040/api/tunnels | grep -o '"public_url":"https://[^"]*' | head -1 | cut -d'"' -f4)
            if [[ -z "$PUBLIC_URL" ]]; then
                echo "Error: failed to get ngrok public URL. Is ngrok running?"
                exit 1
            fi
            ;;
        cloudflared)
            if ! command -v cloudflared &>/dev/null; then
                echo "Error: cloudflared not found. Install it from https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/"
                exit 1
            fi
            CLOUDFLARED_LOG=$(mktemp)
            cloudflared tunnel --url "http://localhost:$DEX_PORT" --no-autoupdate 2>"$CLOUDFLARED_LOG" &
            TUNNEL_PID=$!
            echo "Waiting for cloudflared tunnel..."
            for _ in $(seq 1 30); do
                PUBLIC_URL=$(grep -o 'https://[^ ]*\.trycloudflare\.com' "$CLOUDFLARED_LOG" | head -1) && break
                sleep 1
            done
            rm -f "$CLOUDFLARED_LOG"
            if [[ -z "$PUBLIC_URL" ]]; then
                echo "Error: failed to get cloudflared URL."
                exit 1
            fi
            ;;
        *)
            echo "Error: unknown tunnel type '$TUNNEL_TYPE'. Use 'ngrok' or 'cloudflared'."
            exit 1
            ;;
    esac
fi

echo "Public URL: $PUBLIC_URL"

# Generate config from template.
CONFIG_FILE=$(mktemp)
sed -e "s|ISSUER_URL|$PUBLIC_URL|g" -e "s|ALIAS|$ALIAS|g" "$SCRIPT_DIR/config.yaml.tmpl" > "$CONFIG_FILE"

echo "Starting Dex on port $DEX_PORT..."
"$DEX_BIN" serve "$CONFIG_FILE" &
DEX_PID=$!
sleep 2

DISCOVERY_URL="$PUBLIC_URL/dex/.well-known/openid-configuration"

echo ""
echo "============================================================"
echo "  OIDC Conformance Test Setup Ready"
echo "============================================================"
echo ""
echo "  Discovery URL: $DISCOVERY_URL"
echo "  Alias:         $ALIAS"
echo ""
echo "  Client 1: id=first_client  secret=89d6205220381728e85c4cf5"
echo "  Client 2: id=second_client secret=51c612288018fd384b05d6ad"
echo ""
echo "  Steps:"
echo "  1. Open https://www.certification.openid.net/"
echo "  2. Log in with Google or GitLab"
echo "  3. Create a new test plan:"
echo "     - Plan: OpenID Connect Core: Basic Certification Profile"
echo "     - Server metadata: discovery"
echo "     - Client registration: static_client"
echo "     - Alias: $ALIAS"
echo "     - Discovery URL: $DISCOVERY_URL"
echo "     - Enter both client credentials above"
echo "  4. Run tests and follow instructions"
echo ""
echo "  Press Ctrl+C to stop."
echo "============================================================"

wait "$DEX_PID"
