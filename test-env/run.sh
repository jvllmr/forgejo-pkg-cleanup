#!/usr/bin/env bash
# Start the test Forgejo and run the e2e Go tests against it.
#
# usage:
#   ./run.sh        # start forgejo, ensure test user, run go tests
#   ./run.sh down   # stop forgejo and delete its data
set -euo pipefail

cd "$(dirname "$0")"

# Never use host docker credentials or any production data:
# isolated docker config, deleted on exit.
export DOCKER_CONFIG="$(mktemp -d)"
trap 'rm -rf "$DOCKER_CONFIG"' EXIT
echo '{}' > "$DOCKER_CONFIG/config.json"

# Compose implementation, e.g. "docker compose" (default) or "podman-compose"
COMPOSE="${COMPOSE:-docker compose}"
compose() { $COMPOSE "$@"; }

FORGEJO_URL=http://127.0.0.1:3000
FORGEJO_USER=testuser
FORGEJO_PASSWORD=testpassword

if [[ "${1:-}" == "down" ]]; then
	compose down -v --remove-orphans
	exit 0
fi

echo "== start forgejo =="
compose up -d forgejo

echo "== wait for forgejo =="
ok=0
for _ in $(seq 1 60); do
	if curl -fsS "$FORGEJO_URL/api/healthz" >/dev/null 2>&1; then
		ok=1
		break
	fi
	sleep 2
done
[[ $ok == 1 ]] || { echo "forgejo did not become ready" >&2; exit 1; }

echo "== create test user (idempotent) =="
for _ in $(seq 1 10); do
	if compose exec -T -u git forgejo forgejo admin user create \
		--username "$FORGEJO_USER" \
		--password "$FORGEJO_PASSWORD" \
		--email testuser@example.com \
		--must-change-password=false; then
		break
	fi
	# "user already exists" is fine, everything else: retry then fail
	if curl -fsS -u "$FORGEJO_USER:$FORGEJO_PASSWORD" \
		"$FORGEJO_URL/api/v1/packages/$FORGEJO_USER" >/dev/null 2>&1; then
		echo "user exists"
		break
	fi
	sleep 2
done
curl -fsS -u "$FORGEJO_USER:$FORGEJO_PASSWORD" \
	"$FORGEJO_URL/api/v1/packages/$FORGEJO_USER" >/dev/null

echo "== run tests =="
cd ..
# CGO_ENABLED=0 matches the project Dockerfile and avoids host toolchain issues
CGO_ENABLED=0 go test -tags e2e -count=1 -v ./...

echo
echo "forgejo still running at $FORGEJO_URL (test-env/run.sh down to clean up)"
