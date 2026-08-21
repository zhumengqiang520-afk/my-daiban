#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
DEPLOY_DIR=$(dirname "$SCRIPT_DIR")

if [ ! -f "$DEPLOY_DIR/.env" ]; then
	cp "$DEPLOY_DIR/.env.example" "$DEPLOY_DIR/.env"
	printf '%s\n' "Created $DEPLOY_DIR/.env. Replace every CHANGE_ME value before deployment."
	exit 1
fi

if grep -q 'CHANGE_ME' "$DEPLOY_DIR/.env"; then
	printf '%s\n' "Refusing to deploy while .env contains CHANGE_ME values." >&2
	exit 1
fi

mkdir -p "$DEPLOY_DIR/data/files" "$DEPLOY_DIR/backups"
chmod 700 "$DEPLOY_DIR/backups"

docker compose --project-directory "$DEPLOY_DIR" --env-file "$DEPLOY_DIR/.env" -f "$DEPLOY_DIR/compose.yml" config >/dev/null
printf '%s\n' "Deployment directories and Compose configuration are ready."

