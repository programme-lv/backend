
Execution jobs and results use Core NATS.
Test-file cache misses can also use Core NATS; see [docs/nats-execution.md](docs/nats-execution.md).

To start the server without processing tester results from NATS:
```
go run cmd/server/main.go -listen-sqs=false
```

The flag name is retained for compatibility and is misleading.

## Development

set up an .env file

```
JWT_KEY=...

NATS_URL=nats://localhost:4222

EXTERNAL_EVAL_KEY=...

POSTGRES_DB=...
POSTGRES_USER=...
POSTGRES_PORT=...
POSTGRES_SSLMODE=...
POSTGRES_HOST=...
POSTGRES_PW_SECRET_NAME='...

COOKIE_DOMAIN=...

FILE_STORAGE_ROOT=/mnt/programme-lv-storage
API_PUBLIC_BASE_URL=https://api.programme.lv
TESTFILE_DOWNLOAD_SIGNING_KEY=...
```

let's clone the database from prod

we will need docker for this. ensure you can run docker ps

the in from .scripts/local-pg run docker compose up -d

then we need to dump the prod.

i think we should place all the uni test types in a taskfile that would also
help setting up the necessary environment variables

installing dependencies:
```bash
apt install postgresql-client
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

workflow of setting up new dev environment with db

if you have access to the prod db, dump it into a .dump file.
otherwise, ask someone to give you the .dump file

run docker compose up -d in postgres/
```bash
cd postgres
docker compose up -d
sleep 5
./import.sh
```

To update postgres schema, create a migration script in ./postgres/migrate/ and then run
./postgres/migrate.sh. 
After making changes to DB, run ./postgres/print.sh to update the schema.txt in ./docs/schema.txt

A big TODO is to get rid of the ./http directory.