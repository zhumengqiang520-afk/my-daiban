#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
DEPLOY_DIR=$(dirname "$SCRIPT_DIR")
STAMP=$(date '+%Y-%m-%d_%H-%M-%S')

mkdir -p "$DEPLOY_DIR/backups"
docker compose --project-directory "$DEPLOY_DIR" --env-file "$DEPLOY_DIR/.env" -f "$DEPLOY_DIR/compose.yml" exec -T app \
	/app/vikunja/vikunja dump --path /backups --filename "retail-$STAMP.zip"

test -s "$DEPLOY_DIR/backups/retail-$STAMP.zip"
printf '%s\n' "Backup verified: $DEPLOY_DIR/backups/retail-$STAMP.zip"

