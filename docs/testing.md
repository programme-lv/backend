# Testing

## Unit (default)

No Postgres, NATS, or tester required.

```bash
go test ./...
```

## Integration (Postgres / HTTP)

Needs Postgres matching [`postgres/compose.yml`](../postgres/compose.yml) (`postgres` / `pw` / port `5432`), or override with `TEST_PG_USER`, `TEST_PG_PASSWORD`, `TEST_PG_HOST`, `TEST_PG_PORT`.

```bash
docker compose -f postgres/compose.yml up -d
go test -tags=integration ./...
```

Run automatically on pull requests and pushes to `main`.

## Tester (NATS + VM worker)

Live exec path against NATS and a real tester VM. Not run in CI.

```bash
go test -tags=tester ./modules/exec/...
```
