
Execution jobs and results use Core NATS.
Test-file cache misses can also use Core NATS; see [docs/nats-execution.md](docs/nats-execution.md).
How to run unit / integration / tester suites: [docs/testing.md](docs/testing.md).

To start the server without processing tester results from NATS:
```
go run cmd/server/main.go -listen-sqs=false
```

The flag name is retained for compatibility and is misleading.

## Development

set up an .env file

```
JWT_KEY=...
ADMIN_API_KEY=...

NATS_URL=nats://localhost:4222

POSTGRES_DB=...
POSTGRES_USER=...
POSTGRES_PORT=...
POSTGRES_SSLMODE=...
POSTGRES_HOST=...
POSTGRES_PW=...

COOKIE_DOMAIN=...
COOKIE_SECURE=false

FILE_STORAGE_ROOT=/mnt/programme-lv-storage
API_PUBLIC_BASE_URL=https://api.programme.lv
TESTFILE_DOWNLOAD_SIGNING_KEY=...

# Transactional email (SES SMTP). Default SMTP_ENABLED=false skips sending.
SMTP_ENABLED=false
SMTP_HOST=email-smtp.eu-central-1.amazonaws.com
SMTP_PORT=587
SMTP_USERNAME=...
SMTP_PASSWORD=...
SMTP_FROM=noreply@programme.lv
SMTP_FROM_NAME=programme.lv
WEBSITE_PUBLIC_BASE_URL=https://programme.lv
# Optional overrides (Go durations / int):
# EMAIL_RESET_TOKEN_TTL=1h
# EMAIL_VERIFY_TOKEN_TTL=24h
# EMAIL_PER_USER_COOLDOWN=5m
# EMAIL_GLOBAL_HOURLY_LIMIT=60
```

Local email preview: point SMTP at [Mailpit](https://github.com/axllent/mailpit) (`SMTP_HOST=localhost`, `SMTP_PORT=1025`, empty username/password, `SMTP_ENABLED=true`) and open its UI.

Admin routes accept either an admin user's `auth_token` cookie or the
server-to-server API key:

```http
Authorization: Bearer <ADMIN_API_KEY>
```

Do not expose `ADMIN_API_KEY` to browser-side code.

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