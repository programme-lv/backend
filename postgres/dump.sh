#!/usr/bin/env bash

set -euo pipefail # exit on error, print commands, fail on errors in pipes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd -P)"

# Parse flags (order-independent)
HOST=""
USER=""
PASSWORD=""
DB=""
PORT=""
OUTFILE=""
PG_TAG=""
PROD="false"
HELP="false"

while [[ $# -gt 0 ]]; do
	case "$1" in
		-H|--host) HOST="${2-}"; shift 2;;
		-U|--user) USER="${2-}"; shift 2;;
		-P|--password) PASSWORD="${2-}"; shift 2;;
		-d|--db|--database) DB="${2-}"; shift 2;;
		-p|--port) PORT="${2-}"; shift 2;;
		-o|--outfile) OUTFILE="${2-}"; shift 2;;
		-t|--tag) PG_TAG="${2-}"; shift 2;;
		--prod) PROD="true"; shift 1;;
		-h|--help) HELP="true"; shift 1;;
		--) shift; break;;
		*) echo "Unknown option: $1" >&2; exit 1;;
	esac
done

if [[ "$HELP" == "true" ]]; then
	echo "Usage: $0 [--host HOST] [--user USER] [--password PW] [--db DB] [--port PORT] [--outfile FILE] [--tag PG_TAG] [--prod]" >&2
	echo "Defaults: host from --prod/.env.prod or localhost; user=postgres; db=proglv; port=5432; tag=17; outfile=./DB-YYYY-MM-DD_HHMMSS.dump; password=pw (unless --prod)" >&2
	exit 0
fi

# Defaults
USER="${USER:-postgres}"
DB="${DB:-proglv}"
PORT="${PORT:-5432}"
PG_TAG="${PG_TAG:-17}"

TIMESTAMP="$(date +%F_%H%M%S)"
OUTFILE="${OUTFILE:-./${DB}-${TIMESTAMP}.dump}"

# Resolve password from .env.prod if --prod is set, otherwise use "password123"
if [[ -z "${PASSWORD}" ]]; then
	if [[ "$PROD" == "true" ]]; then
		ENV_FILE="$ROOT_DIR/.env.prod"
		if [[ -f "$ENV_FILE" ]]; then
			PASSWORD="$(grep -E "^POSTGRES_PW=" "$ENV_FILE" | cut -d '=' -f2- | tr -d "'" || true)"
		fi
		if [[ -z "${PASSWORD}" ]]; then
			echo "Error: --prod set but password not provided and POSTGRES_PW not found in .env.prod." >&2
			exit 1
		fi
	else
		PASSWORD="pw"
	fi
fi

# Resolve host from .env.prod if --prod is set, otherwise use "localhost"
if [[ -z "${HOST}" ]]; then
	if [[ "$PROD" == "true" ]]; then
		ENV_FILE="$ROOT_DIR/.env.prod"
		if [[ -f "$ENV_FILE" ]]; then
			HOST="$(grep -E "^POSTGRES_HOST=" "$ENV_FILE" | cut -d '=' -f2- | tr -d "'" || true)"
		fi
		if [[ -z "${HOST}" ]]; then
			echo "Error: --prod set but host not provided and POSTGRES_HOST not found in .env.prod." >&2
			exit 1
		fi
	else
		HOST="localhost"
	fi
fi

if ! command -v pg_dump >/dev/null 2>&1; then
	echo "Error: pg_dump not found in PATH. Maybe sudo apt install postgresql-client..." >&2
	exit 1
fi

pg_dump --version

echo "Dumping ${DB} from ${HOST}:${PORT} to ${OUTFILE}..."

read -p "Continue (y/n)? " choice
case "$choice" in
	y|Y ) echo "Proceeding...";;
	n|N ) echo "Aborted."; exit 1;;
	* ) echo "Aborted."; exit 1;;
esac

mkdir -p "$(dirname "$OUTFILE")"

# -b includes large objects (BLOBs)
# -F c uses custom format for pg_restore
PGPASSWORD="$PASSWORD" pg_dump \
	-h "$HOST" \
	-p "$PORT" \
	-U "$USER" \
	-b \
	-v \
	-F c \
	-f "$OUTFILE" \
	"$DB"

echo "Done."


