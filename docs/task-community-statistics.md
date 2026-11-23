# Task community statistics (backend)

The public response contract, verdict semantics and card presentation live in
the project documentation:
[`../../docs/github/task-community-statistics.md`](../../docs/github/task-community-statistics.md).
This note only records how the backend implements it.

## Implementation

- [`modules/subm/pgrepo/community.go`](../modules/subm/pgrepo/community.go)
  runs one read-only batch query for the task IDs supplied by the caller. It
  joins only `submissions.curr_eval_uuid`, so historical rejudge rows do not
  count, and keeps only `evaluations.stage = 'finished'`. Admin submissions and
  users that no longer exist are excluded. The result has one aggregate row per
  task; no per-user rows leave Postgres.
- Representative-result selection and the verdict priority are expressed in
  that SQL query. Keep its explanatory comment aligned with the shared contract
  when changing the classification.
- [`modules/subm/srvc/community.go`](../modules/subm/srvc/community.go) is a
  narrow read-only service. It short-circuits empty input, passes the batch
  through, logs an unexpected persistence error, and returns the sanitized
  `internal_server_error`.
- [`modules/task/http/community.go`](../modules/task/http/community.go) maps the
  service result to `community_stats`. The task handler accepts the service
  through the optional `WithCommunityStats` option. On a 20-second task-list
  cache miss, `GET /tasks` requests stats for the listed IDs in one call and
  caches the mapped previews. A stats failure just omits the optional field;
  the task list is still served.
- [`cmd/server/main.go`](../cmd/server/main.go) wires the existing `pgPool`
  through `NewPgCommunityStatsRepo` and `NewCommunityStatsService`.

## Tests

- [`modules/subm/pgrepo/community_test.go`](../modules/subm/pgrepo/community_test.go)
  is an integration test (`integration` build tag) against
  `testutil.MustGetMigratedTestPostgresDb`. It covers repeated attempts,
  accepted-ever, pending submissions, submission-time/UUID ordering, rejudges,
  tenths and legacy score fallback, test-flag priority, examples/ignored rows,
  zero possible points, each error category, multiple tasks, admin/deleted-user
  exclusion, ID filtering and empty input.
- [`modules/task/http/community_test.go`](../modules/task/http/community_test.go)
  covers DTO mapping, field omission on service failure, the no-reader path,
  cache reuse and dropping zero counts.
- [`modules/subm/srvc/community_test.go`](../modules/subm/srvc/community_test.go)
  covers empty input, pass-through and sanitized errors.

Run the real integration test with the sibling Postgres service:

```bash
TEST_PG_HOST=postgres go test -tags=integration ./modules/subm/pgrepo/ -run TestCommunityStats -count=1
go test ./...
go vet ./...
```

## Query cost

The batch is restricted to the listed task IDs. Its per-evaluation flag
aggregate can use the existing `eval_test_results(evaluation_uuid)` index.
Before adding an index, measure `EXPLAIN (ANALYZE, BUFFERS)` on representative
data. An index on `submissions(task_shortid)` may help a selective task list,
but a sequential scan can remain reasonable when the listed IDs cover most
tasks. A small development dataset does not establish production capacity.
