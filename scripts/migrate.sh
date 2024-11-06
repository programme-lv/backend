#!/bin/bash

set -e # Exit on error
SCRIPT_DIR=$(dirname "$0")
pushd "$SCRIPT_DIR"/..

# Extract and URL-encode the PostgreSQL password
ENCODED_PASSWORD=$(grep POSTGRES_PASSWORD < .env | cut -d '=' -f2| python3 -c "import urllib.parse, sys; print(urllib.parse.quote(sys.stdin.read().replace('\'','').strip()))")

# Run the migrate command with the encoded password
migrate -source file://./migrate -database "postgres://postgres:${ENCODED_PASSWORD}@database-2.c9uc4usgm7ng.eu-central-1.rds.amazonaws.com:5432/postgres" up 

popd