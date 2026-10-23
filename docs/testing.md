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

## Development tester (full submission flow)

The devbox Compose draft (`/workspace/meta/compose.yaml.draft`) runs `tester
listen nats` as a long-lived sibling on the shared development NATS service.
After reviewing and copying that draft to the host Compose directory:

```bash
docker compose --profile tester up -d --build tester
docker compose --profile tester logs -f tester
```

The worker uses `nats://nats:4222` and the development backend's file-service
subject, so submissions never reach production. Its verified test-file cache is
persisted under `./data/tester`.

Run the smoke test from the devbox with the backend listening on `:8081`:

```bash
docs/scripts/dev-smoke
docs/scripts/dev-smoke --task lio2026dusa \
  --solution /workspace/bios/dist/lio2026dusa/solutions/ap_dp_full.cpp --expected 100
```

The script registers a disposable development user, submits a known model
solution through `POST /subm`, polls `GET /subm/{id}` until the evaluation
finishes, asserts the expected score, and deletes the temporary user so its
submissions and evaluations cascade away. It defaults to `lio2026atmini` and
`cpp17`; use `--no-wait` to only verify submission creation.

Do not point the development backend or tester at production NATS. Core NATS
subjects such as `tester.jobs` are shared by every worker connected to the same
server, so mixing environments can execute development jobs on production
infrastructure and corrupt result streams.
