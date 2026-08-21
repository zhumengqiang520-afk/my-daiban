#!/bin/sh
set -eu

if [ "$#" -ne 2 ] || [ "$1" != "--confirm" ]; then
	printf '%s\n' "Usage: $0 --confirm /absolute/path/to/retail-backup.zip" >&2
	exit 2
fi

BACKUP_FILE=$2
if [ ! -f "$BACKUP_FILE" ]; then
	printf '%s\n' "Backup file does not exist: $BACKUP_FILE" >&2
	exit 2
fi

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
DEPLOY_DIR=$(dirname "$SCRIPT_DIR")
RESTORE_NAME=restore-input.zip
cp "$BACKUP_FILE" "$DEPLOY_DIR/backups/$RESTORE_NAME"

printf '%s\n' "Stopping the application and restoring $BACKUP_FILE. Current data may be overwritten."
docker compose --project-directory "$DEPLOY_DIR" --env-file "$DEPLOY_DIR/.env" -f "$DEPLOY_DIR/compose.yml" stop app
docker compose --project-directory "$DEPLOY_DIR" --env-file "$DEPLOY_DIR/.env" -f "$DEPLOY_DIR/compose.yml" run --rm app \
	restore --preserve-config "/backups/$RESTORE_NAME"
docker compose --project-directory "$DEPLOY_DIR" --env-file "$DEPLOY_DIR/.env" -f "$DEPLOY_DIR/compose.yml" up -d app caddy
printf '%s\n' "Restore command finished. Run scripts/smoke-test.sh and complete the restore checklist."

