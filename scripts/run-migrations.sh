#!/usr/bin/env sh
set -e
DB_HOST=${DB_HOST:-postgres}
DB_NAME=${DB_NAME:-postgres}
DB_USER=${DB_USER:-postgres}
for f in /work/migrations/*.sql; do
  [ -e "$f" ] || continue
  psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -f "$f"
done
