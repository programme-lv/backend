#!/usr/bin/env bash
#
# Print PostgreSQL schema to docs/schema.txt
#
set -euo pipefail

DB="${PGDATABASE:-proglv}"
OUT="./docs/schema.txt"
USER="${PGUSER:-proglv}"
PORT="${PGPORT:-5433}"
export PGPASSWORD="${PGPASSWORD:-proglv}"

[[ -z "$DB" ]] && { echo "PGDATABASE env var must be set" >&2; exit 1; }

# Connection args
CONN_ARGS=(-h "${PGHOST:-localhost}" -p "$PORT" -U "$USER" -d "$DB")

# Get all objects
OBJ_LIST=$(psql -X -A -t "${CONN_ARGS[@]}" -c "
  SELECT quote_ident(nspname)||'.'||quote_ident(relname)
  FROM pg_class JOIN pg_namespace n ON n.oid = pg_class.relnamespace
  WHERE n.nspname NOT IN ('pg_catalog','information_schema')
    AND relkind IN ('r','v','m')
  ORDER BY n.nspname, relname;
")

echo "-- Schema dump of '$DB' generated $(date -u)" > "$OUT"

# Dump each object
for obj in $OBJ_LIST; do
  {
    psql -X -q "${CONN_ARGS[@]}" --pset pager=off -c "\\d $obj"
  } >> "$OUT"
done

{
  psql -X -q "${CONN_ARGS[@]}" --pset pager=off -c "\\d"
} >> "$OUT"

echo "Schema written to: $OUT"
