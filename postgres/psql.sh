#!/usr/bin/env bash

HOST="${PGHOST:-localhost}"
PORT="${PGPORT:-5432}"
USER="${PGUSER:-postgres}"
DB="${PGDATABASE:-proglv}"
export PGPASSWORD="${PGPASSWORD:-pw}"

psql -h "$HOST" -p "$PORT" -U "$USER" -d "$DB" "$@"

if [[ $? -ne 0 ]]; then
    echo "Seems like psql failed. Forgot docker compose up -d?"
    exit 1
fi

