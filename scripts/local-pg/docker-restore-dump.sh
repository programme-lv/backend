#!/bin/bash

set -ex # Exit on error, print commands

# Get absolute path to dump file
DUMP_FILE=$(realpath ./prod-pg.dump)
DUMP_DIR=$(dirname "$DUMP_FILE")
DUMP_FILENAME=$(basename "$DUMP_FILE")

# Run pg_restore inside Docker container
# Mount the dump file directory and use PostgreSQL 16 image
docker run --rm \
  --network=host \
  -v "$DUMP_DIR:/dump" \
  -e PGPASSWORD=proglv \
  postgres:16 \
  pg_restore \
  -h localhost \
  -p 5433 \
  -U proglv \
  -d proglv \
  -v \
  --clean \
  --if-exists \
  -O \
  -x \
  "/dump/$DUMP_FILENAME" 