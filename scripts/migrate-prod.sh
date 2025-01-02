#!/bin/bash

set -e # Exit on error
SCRIPT_DIR=$(dirname "$0")
pushd "$SCRIPT_DIR"/..

# Extract and URL-encode the PostgreSQL password
PG_HOST=$(grep POSTGRES_HOST < .env | cut -d '=' -f2)
PG_PORT=$(grep POSTGRES_PORT < .env | cut -d '=' -f2)
PG_USER=$(grep POSTGRES_USER < .env | cut -d '=' -f2)
PG_DB=$(grep POSTGRES_DB < .env | cut -d '=' -f2)
PG_PASSWORD_SECRET_NAME=$(grep POSTGRES_PASSWORD_SECRET_NAME < .env | cut -d '=' -f2 | tr -d "'")
PG_PW=$(aws secretsmanager get-secret-value --secret-id "$PG_PASSWORD_SECRET_NAME" --region eu-central-1| jq -r '.SecretString' | jq -r '.password')

echo "PG_HOST: $PG_HOST"
echo "PG_PORT: $PG_PORT"
echo "PG_USER: $PG_USER"
echo "PG_DB: $PG_DB"
echo "PG_PASSWORD_SECRET_NAME: $PG_PASSWORD_SECRET_NAME"
echo "PG_PW: $PG_PW"

# shellcheck disable=SC2162
read -p "Press Enter to continue..."

echo "TODO: reimplement the script"

popd