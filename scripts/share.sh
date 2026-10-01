#!/usr/bin/env bash
# Share a WineCraft game with a friend through a temporary Cloudflare quick tunnel.
#
#   make share            (or: scripts/share.sh)
#
# Builds the game, starts the server on 127.0.0.1 only (never the LAN), opens
# a https://….trycloudflare.com tunnel to it, keeps the Mac awake, and prints
# the link and join code to send. Ctrl-C stops the tunnel, the server and the
# keep-awake together. The link stops working the moment you stop.
set -euo pipefail
cd "$(dirname "$0")/.."

LOGS=logs
mkdir -p "$LOGS"
command -v cloudflared >/dev/null || { echo "cloudflared is not installed (brew install cloudflared)"; exit 1; }

echo "Building the game..."
make wasm >/dev/null
go build -o bin/server ./cmd/server

# A port nothing on this Mac listens on. netstat sees every user's TCP
# listeners; lsof without root would miss other users' sockets.
listening() { netstat -an -p tcp 2>/dev/null | awk '$6=="LISTEN"{print $4}' | sed 's/.*\.//' | grep -qx "$1"; }
PORT=${PORT:-8787}
while listening "$PORT"; do PORT=$((PORT + 1)); done

# Random join code like "malbec-482913", from the kernel's random source.
words=(malbec cabernet torrontes semillon bonarda syrah merlot pinot chardonnay tempranillo)
rand() { od -An -N4 -tu4 /dev/urandom | tr -d ' '; }
CODE=${CODE:-${words[$(( $(rand) % ${#words[@]} ))]}-$(printf '%06d' $(( $(rand) % 1000000 )))}

PIDS=()
cleanup() {
	trap - INT TERM EXIT
	echo
	echo "Stopping: the link no longer works."
	for pid in "${PIDS[@]}"; do kill "$pid" 2>/dev/null || true; done
	wait 2>/dev/null || true
}
trap cleanup INT TERM EXIT

./bin/server -addr "127.0.0.1:$PORT" -web web -join "$CODE" >"$LOGS/server.log" 2>&1 &
PIDS+=($!)
for _ in $(seq 50); do
	curl -fsS "http://127.0.0.1:$PORT/healthz" >/dev/null 2>&1 && break
	sleep 0.1
done
curl -fsS "http://127.0.0.1:$PORT/healthz" >/dev/null || { echo "The server did not start:"; cat "$LOGS/server.log"; exit 1; }

cloudflared tunnel --no-autoupdate --url "http://127.0.0.1:$PORT" >"$LOGS/tunnel.log" 2>&1 &
PIDS+=($!)
URL=""
for _ in $(seq 60); do
	URL=$(grep -Eo 'https://[a-z0-9-]+\.trycloudflare\.com' "$LOGS/tunnel.log" | head -1 || true)
	[ -n "$URL" ] && break
	sleep 0.5
done
[ -n "$URL" ] || { echo "The tunnel did not come up:"; tail -20 "$LOGS/tunnel.log"; exit 1; }

# Wait until the new name is in public DNS before printing it. Asking 1.1.1.1
# directly keeps this Mac's resolver from caching a "not found" answer, which
# would break the link here for minutes even after it works everywhere else.
HOST=${URL#https://}
for _ in $(seq 60); do
	[ -n "$(dig +short "$HOST" @1.1.1.1 2>/dev/null)" ] && break
	sleep 1
done

# Keep the Mac awake while the server runs (it would sleep after a minute idle).
caffeinate -i -w "${PIDS[0]}" &
PIDS+=($!)

cat <<EOF

  WineCraft is live (until you press Ctrl-C).

  Send Raymon:   $URL/?code=$CODE
  Join code:     $CODE
  You:           http://127.0.0.1:$PORT/   (or the same link)

  Server log:    $LOGS/server.log   Tunnel log: $LOGS/tunnel.log

EOF
tail -n 0 -f "$LOGS/server.log" &
PIDS+=($!)
wait "${PIDS[0]}"
