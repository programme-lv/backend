#!/usr/bin/env bash

set -euo pipefail

# Print PostgreSQL schema to docs/schema.txt

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd -P)"

DB="${PGDATABASE:-proglv}"
OUT="${OUTFILE:-$ROOT_DIR/docs/schema.txt}"
USER="${PGUSER:-postgres}"
PORT="${PGPORT:-5432}"
HOST="${PGHOST:-localhost}"
export PGPASSWORD="${PGPASSWORD:-pw}"

[[ -z "$DB" ]] && { echo "PGDATABASE env var must be set" >&2; exit 1; }

# Connection args
CONN_ARGS=(-h "$HOST" -p "$PORT" -U "$USER" -d "$DB")

# Get all objects (tables, views, matviews) excluding system schemas
OBJ_LIST=$(psql -X -A -t "${CONN_ARGS[@]}" -c "
  SELECT quote_ident(nspname)||'.'||quote_ident(relname)
  FROM pg_class JOIN pg_namespace n ON n.oid = pg_class.relnamespace
  WHERE n.nspname NOT IN ('pg_catalog','information_schema')
    AND relkind IN ('r','v','m')
  ORDER BY n.nspname, relname;
")

echo "-- Schema dump of '$DB' generated $(date -u)" > "$OUT"

# Dump each object description
for obj in $OBJ_LIST; do
  {
    psql -X -q "${CONN_ARGS[@]}" --pset pager=off -c "\\d $obj"
  } >> "$OUT"
done

# Append summary of relations
{
  psql -X -q "${CONN_ARGS[@]}" --pset pager=off -c "\\d"
} >> "$OUT"

echo "Schema written to: $OUT"


