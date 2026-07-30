limit to 1000 events per execution
limit each of the events to 256 kiB
limit total testing time to 5 minutes
limit timeout between consecutive events since the first event is 20 seconds
limit to handling one execution events at a time for now

the total memory usage is limited to 256 MiB per execution

for the notifier channels place the information onto disk

we have to somehow simulate the receiving of many events and monitor memory usage

get rid of the handlers channel to avoid writing to closed channel and goroutine circus.

# ExecSrvc - code execution service

The current execution transport is Core NATS.
See [NATS execution transport](../../docs/nats-execution.md) for subject ownership, result streaming, and test-file retrieval.
The SQS notes below are obsolete and have been removed.

A Go package that provides a robust service for executing and testing code submissions in various programming languages.

This service is designed to handle concurrent test execution while maintaining ordered result streaming and proper resource management.

## Features

- Publish code execution requests through Core NATS
- Execute code in different programming languages with customizable compilation and execution commands
- Maintain sequential ordering of test results even with concurrent test execution
- Store completed execution results in the configured file store
- Stream execution events for real-time progress monitoring
- Verify execution parameters like memory limits and timeouts

## Core Components

- `ExecSrvc`: Main service that handles code execution requests and result management
- `ExecResStreamOrganizer`: Manages ordered streaming of execution results
- `ExecRepo`: Interface for execution result storage

## Usage

```go
srvc := execsrvc.NewExecSrvc(
    ctx,
    execRepository,
    natsConnection,
    testfileStore,
)

err := srvc.Enqueue(ctx, execUUID, sourceCode, languageID, tests, params)

// Get execution results
exec, err := srvc.Get(ctx, execUUID)
```

## Configuration

The service uses `NATS_URL` and `FILE_STORAGE_ROOT`.
The HTTP fallback for test files additionally depends on `API_PUBLIC_BASE_URL` and `TESTFILE_DOWNLOAD_SIGNING_KEY`.

## TODO / ideas

- [ ] Integrate with submission service using PostgreSQL for scoring storage
- [ ] Implement minimum memory limits per programming language
- [ ] Add support for evaluation without API key (without result persistence)

