#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd -P)"

# Defaults from env if provided
HOST="${PGHOST:-}"
PORT="${PGPORT:-}"
USER="${PGUSER:-}"
DB="${PGDATABASE:-}"
PASSWORD="${PGPASSWORD:-}"

PROD="false"
HELP="false"

REMAINING_ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        -H|--host) HOST="${2-}"; shift 2;;
        -p|--port) PORT="${2-}"; shift 2;;
        -U|--user) USER="${2-}"; shift 2;;
        -d|--db|--database) DB="${2-}"; shift 2;;
        -P|--password) PASSWORD="${2-}"; shift 2;;
        --prod) PROD="true"; shift 1;;
        -h|--help)
            echo "Usage: $0 [--host HOST] [--port PORT] [--user USER] [--db DB] [--password PW] [--prod] [-- ...psql args]" >&2
            echo "Defaults (dev): host=localhost port=5432 user=postgres db=proglv password=pw" >&2
            exit 0;;
        --)
            shift
            while [[ $# -gt 0 ]]; do REMAINING_ARGS+=("$1"); shift; done
            break;;
        *)
            # Pass through unknown args to psql
            REMAINING_ARGS+=("$1"); shift 1;;
    esac
done

# Apply defaults
HOST="${HOST:-localhost}"
PORT="${PORT:-5432}"
USER="${USER:-postgres}"
DB="${DB:-proglv}"

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
    PASSWORD="pw"
fi

export PGPASSWORD="$PASSWORD"

set +e
psql -h "$HOST" -p "$PORT" -U "$USER" -d "$DB" "${REMAINING_ARGS[@]}"
status=$?
set -e

if [[ $status -ne 0 ]]; then
    echo "Seems like psql failed. Forgot docker compose up -d?"
    exit 1
fi
