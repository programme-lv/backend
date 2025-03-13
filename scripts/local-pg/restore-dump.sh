#!/bin/bash

set -ex # Exit on error, print commands

# Add --clean to drop existing objects and --if-exists to avoid errors if they don't exist
# Add -O to skip ownership and -x to skip privileges
pg_restore -h localhost -p 5433 -U proglv -d proglv -v --clean --if-exists -O -x ./prod-pg.dump
