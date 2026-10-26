# Statement examples in executions

Statement examples are not official scoring tests. They are executed after the
official tests as **unscored** cases so a submission can show whether it also
passes the examples that were given in the statement.

## Execution

- `domain.Eval.AddExamples(task)` appends each non-interactive example to
  `eval.Tests` with `Example = true` and inline `Input`/`Answer` text.
- `constructExecEnqueueTests` sends those examples to the tester with
  `InContent`/`AnsContent` instead of hashes and download URLs.
- The tester runs every test it receives. Examples are appended after the
  official tests and are not present in any scoring group, so group skipping
  never suppresses them.
- Interactive examples are communication traces, not input/output pairs, and
  are skipped by `AddExamples`.

## Scoring

`Eval.CalculateScore` excludes example tests from:

- the possible-score count (`test` scoring unit);
- the score bar and received score;
- the maximum CPU and memory values.

Examples therefore never change the submission score. Their per-test verdicts
are still stored and shown separately in the submission result table.

## Storage and API

Migration `079_eval_examples` adds `eval_test_results.example`. The evaluator
persists the flag and `GET` reloads it.

`SubmEval` exposes `example_test_ids`, a list of 1-based test IDs that belong to
the appended examples. The website uses it to render
"Uzdevuma formulējumā dotie piemēri" separately from the scored tests and to
omit example verdict boxes from the compact progress strip.
