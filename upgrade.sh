#!/usr/bin/env bash
#
# Upgrade the Forever sim in place: pull the latest source and the image GitHub built from it, and
# restart the stack. --build builds the image here instead.
#
# The app is stateless, so there is nothing to back up — the
# only persistent data is Caddy's certs in the caddy_data volume, which are left
# untouched.
#
# The old container keeps serving while the new image downloads; downtime is just
# the few seconds of the container swap.

set -euo pipefail

cd "$(dirname "$0")"

# Check the sim directly on its loopback port (TLS is terminated upstream by
# the host's Caddy, which we don't touch during an app upgrade).
HEALTH_URL="http://127.0.0.1:18790/version"

echo "==> Pulling latest source..."
git pull --ff-only

if [[ "${1:-}" == "--build" ]]; then
	echo "==> Building sim image..."
	docker compose build sim
else
	echo "==> Pulling sim image..."
	docker compose pull sim
fi

echo "==> Restarting stack..."
docker compose up -d

echo "==> Pruning dangling images..."
docker image prune -f

echo "==> Waiting for the sim to come back..."
for _ in $(seq 1 30); do
	if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
		echo "==> Healthy: $(curl -fsS "$HEALTH_URL")"
		exit 0
	fi
	sleep 2
done

echo "!! Sim did not respond at ${HEALTH_URL} within ~60s." >&2
echo "   Check logs:  docker compose logs --tail=50" >&2
exit 1
