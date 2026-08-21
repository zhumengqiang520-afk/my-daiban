#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
DEPLOY_DIR=$(dirname "$SCRIPT_DIR")

set -a
# shellcheck disable=SC1091
. "$DEPLOY_DIR/.env"
set +a

case "$SITE_ADDRESS" in
	http://*) IP_ADDRESS=${SITE_ADDRESS#http://} ;;
	https://*) IP_ADDRESS=${SITE_ADDRESS#https://} ;;
	*) IP_ADDRESS=$SITE_ADDRESS ;;
esac

case "$IP_ADDRESS" in
	*[!0-9.]*|'')
		printf '%s\n' "SITE_ADDRESS must be a public IPv4 address for IP certificate issuance." >&2
		exit 2
		;;
esac

mkdir -p "$DEPLOY_DIR/certbot-webroot" "$DEPLOY_DIR/letsencrypt"
docker run --rm \
	-v "$DEPLOY_DIR/certbot-webroot:/var/www/certbot" \
	-v "$DEPLOY_DIR/letsencrypt:/etc/letsencrypt" \
	"${CERTBOT_IMAGE:?set CERTBOT_IMAGE in .env}" certonly \
	--webroot --webroot-path /var/www/certbot \
	--preferred-profile shortlived \
	--ip-address "$IP_ADDRESS" \
	--non-interactive --agree-tos --register-unsafely-without-email

test -s "$DEPLOY_DIR/letsencrypt/live/$IP_ADDRESS/fullchain.pem"
test -s "$DEPLOY_DIR/letsencrypt/live/$IP_ADDRESS/privkey.pem"
printf '%s\n' "IP certificate issued for $IP_ADDRESS."
