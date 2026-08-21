#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
DEPLOY_DIR=$(dirname "$SCRIPT_DIR")

set -a
# shellcheck disable=SC1091
. "$DEPLOY_DIR/.env"
set +a

mkdir -p "$DEPLOY_DIR/certbot-webroot" "$DEPLOY_DIR/letsencrypt"
docker run --rm \
	-v "$DEPLOY_DIR/certbot-webroot:/var/www/certbot" \
	-v "$DEPLOY_DIR/letsencrypt:/etc/letsencrypt" \
	"${CERTBOT_IMAGE:?set CERTBOT_IMAGE in .env}" renew --quiet

docker compose --project-directory "$DEPLOY_DIR" --env-file "$DEPLOY_DIR/.env" -f "$DEPLOY_DIR/compose.yml" exec -T caddy \
	caddy reload --config /etc/caddy/Caddyfile
printf '%s\n' "IP certificate renewal check and Caddy reload completed."
