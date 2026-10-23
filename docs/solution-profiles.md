# Solution profiles

Admin calibration tool for reference solutions. It estimates how many points a
solution would earn at different CPU-time and memory limits without rerunning
the solution for every chart point.

## Flow

1. `POST /tasks/{taskId}/solutions/{fname}/profile` starts a profile. The
   request body is `{"runs": 3, "lang_id": "cpp17"}`; both fields are optional.
   `lang_id` is inferred from the filename with
   `plang.ProgrLangIDFromFilename` when omitted.
2. The service executes the solution at `10 s` CPU and `1 GiB` per test. It
   passes **no groups**, so the tester runs every test instead of skipping the
   rest of a failed group.
3. Each `FinishTest` measurement (CPU, RSS, verdict) is converted with the same
   `applyTestResult` used by real submissions.
4. After each repetition the profile row is updated, so the UI can show partial
   progress.
5. `GET /solution-profiles/{profileId}` derives both charts and caches nothing.

`GET /tasks/{taskId}/solutions` lists stored solutions with their current
expected points and latest profile. All four endpoints are admin-only.

## Storage

Migration `075_task_solution_profiles` adds `task_solution_profiles`:

- keyed by `(id)`, linked to `tasks(short_id)` with `ON DELETE CASCADE`;
  it deliberately does **not** reference `task_solutions`, because task
  updates delete and re-insert those rows and would cascade away every
  profile;
- stores `runs` as JSONB (condensed measurements, not stdout/stderr);
- stores `tests_version` and `solution_sha256` so the API can report stale
  profiles;
- stores `expected_points`, which is a snapshot of the solution's declared
  subtasks at profile time.

Do **not** insert a hidden `submissions` row. It would leak into submission
lists, max-score calculations, activity notifications and stats, and needs a
fake author. Profiles reuse the exec service, not the submission record.

## Scoring math

`domain.BuildProfileChart` treats a measured test as passing while
`cpu_ms < limit` and `mem_kib < limit` (the judge compares with `>=`). The
other resource stays at its profiling maximum, so each chart isolates one
dimension.

The score is a step function. Breakpoints are the measured values plus one
millisecond/KiB for every test that earned points at the profiling limits.
`profileScoreAt` builds a synthetic `Eval` and calls the real
`CalculateScore`, which keeps subtask, test-group and interactor partial-score
semantics intact.

For repeated runs:

- `Min`/`Median`/`Max` are the worst, middle and best scores across runs.
- `LowerBound` is the first limit where **every** run reaches
  `expected_points`.
- `UpperBound` is the first limit where **any** run exceeds it.
- `[LowerBound, UpperBound)` is therefore the interval where every run scores
  exactly the expected amount. A missing lower bound means the expected score
  is unreachable within 10 s / 1 GiB; a missing upper bound means no run
  over-scores in range.

If the solution declares no subtasks, `expected_points` is `0` and both bounds
are left unset. The UI omits the expected line.

## Assumptions and limits

- Measurements are censored at the profiling limits: a test that TLEs at 10 s
  or MLEs at 1 GiB is treated as failing for the whole chart.
- The curve assumes a test's resource use does not change when the limit
  changes. Adaptive programs, allocator/GC variance (Java, Go) and different
  tester machines can violate this. Use several repetitions.
- Repetitions run sequentially under a one-profile-at-a-time semaphore, so
  calibration cannot flood the tester queue. Each repetition has a timeout of
  `2 min + 12 s per test` (capped at 30 min); without it a missing tester would
  leave the profile `running` forever because NATS drops jobs with no
  responders.
- Profiles interrupted by a backend restart stay `running`; start a new
  profile.
- Tasks with many tests are expensive: `tests × 10 s × runs` is the worst case.

## Verification

Unit tests cover the pure math and the exec-to-profile conversion:

```bash
go test ./modules/subm/domain/ ./modules/subm/srvc/
```

For a live check, use the development tester described in
[docs/testing.md](testing.md). Do not point development at production NATS.
