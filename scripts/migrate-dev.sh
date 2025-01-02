#!/bin/bash

set -e # Exit on error
SCRIPT_DIR=$(dirname "$0")
pushd "$SCRIPT_DIR"/..

# Extract and URL-encode the PostgreSQL password
PG_HOST=localhost
PG_PORT=5433
PG_USER=proglv
PG_DB=proglv
PG_PW=proglv

echo "PG_HOST: $PG_HOST"
echo "PG_PORT: $PG_PORT"
echo "PG_USER: $PG_USER"
echo "PG_DB: $PG_DB"
echo "PG_PW: $PG_PW"

# shellcheck disable=SC2162
read -p "Press Enter to continue..."

# Run the migrate command with the encoded password
migrate -source file://./migrate -database "postgres://proglv:proglv@localhost:5433/proglv?sslmode=disable" up 

popd