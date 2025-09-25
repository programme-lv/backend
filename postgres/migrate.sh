#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd -P)"

# Flags (order-independent)
HOST=""
PORT=""
USER=""
DB=""
PASSWORD=""
PROD="false"
HELP="false"

while [[ $# -gt 0 ]]; do
case "$1" in
	-H|--host) HOST="${2-}"; shift 2;;
	-p|--port) PORT="${2-}"; shift 2;;
	-U|--user) USER="${2-}"; shift 2;;
	-d|--db|--database) DB="${2-}"; shift 2;;
	-P|--password) PASSWORD="${2-}"; shift 2;;
	--prod) PROD="true"; shift 1;;
	-h|--help) HELP="true"; shift 1;;
	--) shift; break;;
	*) echo "Unknown option: $1" >&2; exit 1;;
esac
done

if [[ "$HELP" == "true" ]]; then
	echo "Usage: $0 [--host HOST] [--port PORT] [--user USER] [--db DB] [--password PW] [--prod]" >&2
	echo "Defaults (dev): host=localhost port=5432 user=postgres db=proglv password=pw" >&2
	exit 0
fi

# Defaults for dev (match postgres/compose.yml)
HOST="${HOST:-localhost}"
PORT="${PORT:-5432}"
USER="${USER:-postgres}"
DB="${DB:-proglv}"
PASSWORD="${PASSWORD:-}"

if [[ "$PROD" == "true" ]]; then
	ENV_FILE="$ROOT_DIR/.env.prod"
	if [[ -f "$ENV_FILE" ]]; then
		[[ -z "$HOST" || "$HOST" == "localhost" ]] && HOST="$(grep -E '^POSTGRES_HOST=' "$ENV_FILE" | cut -d '=' -f2- | tr -d "'" || true)"
		[[ -z "$USER" || "$USER" == "postgres" ]] && USER="$(grep -E '^POSTGRES_USER=' "$ENV_FILE" | cut -d '=' -f2- | tr -d "'" || true)"
		[[ -z "$DB" || "$DB" == "proglv" ]] && DB="$(grep -E '^POSTGRES_DB=' "$ENV_FILE" | cut -d '=' -f2- | tr -d "'" || true)"
		[[ -z "$PASSWORD" ]] && PASSWORD="$(grep -E '^POSTGRES_PW=' "$ENV_FILE" | cut -d '=' -f2- | tr -d "'" || true)"
	fi
fi

if [[ -z "$PASSWORD" ]]; then
	# Dev default
	PASSWORD="pw"
fi

if ! command -v migrate >/dev/null 2>&1; then
	echo "Error: migrate CLI not found. Install: https://github.com/golang-migrate/migrate" >&2
	exit 1
fi

echo "PG_HOST: $HOST"
echo "PG_PORT: $PORT"
echo "PG_USER: $USER"
echo "PG_DB:   $DB"

read -p "Press Enter to run migrations..." _

pushd "$ROOT_DIR" >/dev/null
migrate -source file://./migrate -database "postgres://$USER:$PASSWORD@$HOST:$PORT/$DB?sslmode=disable" up
popd >/dev/null

echo "Migrations applied."


