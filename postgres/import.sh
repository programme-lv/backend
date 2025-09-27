#!/usr/bin/env bash

set -euo pipefail # exit on error, print commands, fail on errors in pipes

# Parse flags (order-independent)
HOST=""
USER=""
PASSWORD=""
DB=""
PORT=""
DUMP_FILE=""
HELP="false"

while [[ $# -gt 0 ]]; do
case "$1" in
	-H|--host) HOST="${2-}"; shift 2;;
	-U|--user) USER="${2-}"; shift 2;;
	-P|--password) PASSWORD="${2-}"; shift 2;;
	-d|--db|--database) DB="${2-}"; shift 2;;
	-p|--port) PORT="${2-}"; shift 2;;
	-f|--file|--dump) DUMP_FILE="${2-}"; shift 2;;
	-h|--help) HELP="true"; shift 1;;
	--) shift; break;;
	*) echo "Unknown option: $1" >&2; exit 1;;
esac
done

if [[ "$HELP" == "true" ]]; then
	echo "Usage: $0 [--host HOST] [--user USER] [--password PW] [--db DB] [--port PORT] [--file DUMP_FILE]" >&2
	echo "Defaults: host=localhost; user=postgres; db=proglv; port=5432; password=pw. If --file not provided, picks newest *.dump in CWD." >&2
	exit 0
fi

# Defaults
USER="${USER:-postgres}"
DB="${DB:-proglv}"
PORT="${PORT:-5432}"
PASSWORD="${PASSWORD:-pw}"
HOST="${HOST:-localhost}"

# Determine dump file if not provided: pick newest *.dump in CWD
if [[ -z "${DUMP_FILE}" ]]; then
	DUMP_FILE="$(ls -1t -- *.dump 2>/dev/null | head -n 1 || true)"
	if [[ -z "${DUMP_FILE}" ]]; then
		echo "Error: no dump file provided and no *.dump files found in current directory." >&2
		exit 1
	fi
fi

if [[ ! -f "$DUMP_FILE" ]]; then
	echo "Error: dump file '$DUMP_FILE' not found." >&2
	exit 1
fi

# Check tools
if ! command -v pg_restore >/dev/null 2>&1; then
	echo "Error: pg_restore not found in PATH. Maybe sudo apt install postgresql-client..." >&2
	exit 1
fi

pg_restore --version

echo "Importing '$DUMP_FILE' into ${DB} on ${HOST}:${PORT} as ${USER}..."

read -p "Continue (y/n)? " choice
case "$choice" in
	y|Y ) echo "Proceeding...";;
	n|N ) echo "Aborted."; exit 1;;
	* ) echo "Aborted."; exit 1;;
esac

# Add --clean to drop existing objects and --if-exists to avoid errors if they don't exist
# Add -O to skip ownership and -x to skip privileges
PGPASSWORD="$PASSWORD" pg_restore \
	-h "$HOST" \
	-p "$PORT" \
	-U "$USER" \
	-d "$DB" \
	-v \
	--clean \
	--if-exists \
	-O \
	-x \
	"$DUMP_FILE"

echo "Done. If the dump is no longer necessary, consider deleting: $DUMP_FILE"
