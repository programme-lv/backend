# NATS execution transport

The backend and tester use Core NATS.
They do not depend on JetStream, streams, consumers, or Object Store.
Deleting NATS loses in-flight jobs, results, and file transfers.
The backend database and file store remain the durable sources of truth.

## Subjects and inboxes

The backend publishes execution jobs to `tester.jobs`.
Testers use the `workers` queue group so one tester receives each job.

Each backend process owns two random inbox subjects:

- the result inbox receives all execution events for that backend process;
- the file-service subject receives test-file requests for jobs created by that process.

The backend puts the result inbox in the job's NATS reply field and the file-service subject in the `Proglv-File-Subject` header.
Execution UUIDs correlate messages that share a process inbox.

The job JSON is `tester/api.ExecReq`. Optional `groups` is a list of scoring units (each a list of 1-based test IDs).
When present, the tester skips later tests in a unit after a failure and emits `test_ignore`.
Omitted `groups` runs every test. Old testers ignore the unknown field.

For every missing file, the tester creates a short-lived reply inbox and sends a request containing `eval_uuid` and `sha256` to the file-service subject.
The backend only serves hashes registered for that execution.

## File stream

Test files remain in the backend file store as `{sha256}.zst`.
The backend publishes the compressed bytes in ordered Core NATS messages.

Each response has a `Proglv-File-Status` header:

- `chunk` includes raw compressed bytes and a zero-based `Proglv-File-Sequence`;
- `done` terminates a successful stream;
- `error` terminates a failed stream and includes `Proglv-File-Error`.

Chunks are no larger than 512 KiB and must also fit the server's advertised maximum payload.
Core NATS preserves publication order for one publisher and subscriber, but does not replay missed chunks.
A failed transfer restarts from the beginning.

## Lifecycle and verification

The file-service subscription lasts for the backend process.
Allowed file hashes last for one execution and are removed with its in-memory execution state.
Disk reads and chunk publication happen without holding the execution-state mutex.

The tester bounds transfer time and size, decompresses into a temporary file, verifies the SHA-256 hash of decompressed contents, and atomically moves valid files into its cache.
For a NATS job, lookup order is local cache, NATS transfer, then the signed HTTPS URL.
Jobs without `Proglv-File-Subject`, including the SQS transport, continue to use HTTPS.

To verify the fallback, run a backend and `tester listen nats` against the same NATS server, clear the tester file cache, set `API_PUBLIC_BASE_URL` to an address the tester cannot reach, and submit an execution.
The execution must complete after the tester retrieves each missing file over NATS.
