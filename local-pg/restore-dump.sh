#!/bin/bash

set -ex # Exit on error, print commands

pg_restore -h localhost -p 5433 -U proglv -d proglv -v --no-owner ./prod-pg.dump
