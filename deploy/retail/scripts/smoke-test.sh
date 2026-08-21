#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
DEPLOY_DIR=$(dirname "$SCRIPT_DIR")

set -a
# shellcheck disable=SC1091
. "$DEPLOY_DIR/.env"
set +a

BASE_URL=${PUBLIC_URL%/}
curl --fail --silent --show-error "$BASE_URL/api/v2/health" | grep -q '"status":"OK"'
curl --fail --silent --show-error "$BASE_URL/api/v1/info" | grep -q '"retail_enabled":true'
curl --fail --silent --show-error --head "$BASE_URL/" >/dev/null
printf '%s\n' "Smoke test passed: health, retail feature flag, and frontend are reachable at $BASE_URL"
