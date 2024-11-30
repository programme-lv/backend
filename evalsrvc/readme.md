# EvalSrvc - evaluation service / module

*Evaluation* is the result of executing tests against a user's solution.

Evaluation service:
1. picks **UUID** v7, stores an empty evaluation in-memory
2. **enqueues** evaluation request into SQS *submission queue*
3. **receives** events from the *tester* via SQS *response queue*
4. **constructs** the full evaluation from events in-memory
    - test full stdout / stderr are stored immediately to S3
5. **sends** each evaluation event to a listener at most once
    - all evaluation related events are deleted after 5 minutes
    - evaluation events are deleted after being sent to listener
6. once evaluation is constructed, it is **persisted** to S3
7. **deletes** evaluation from in-memory storage

For in-depth details, refer to `godoc` documentation.

## To-Do

- migrate `Enqueue` and `EnqueueExternal` functions to `NewEvaluation`

We will continue to construct evaluation in submission service for a little while.

The current objective for refactoring is so that evaluation is constructed in parallel
to submission service and when evaluation is complete, it is persisted to S3.


