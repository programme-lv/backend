limit to 1000 events per execution
limit each of the events to 256 kiB
limit total testing time to 5 minutes
limit timeout between consecutive events since the first event is 20 seconds
limit to handling one execution events at a time for now

the total memory usage is limited to 256 MiB per execution

for the notifier channels place the information onto disk

we have to somehow simulate the receiving of many events and monitor memory usage

# ExecSrvc - Code Execution Service

A Go package that provides a robust service for executing and testing code submissions in various programming languages.

This service is designed to handle concurrent test execution while maintaining ordered result streaming and proper resource management.

## Features

- Process code execution requests through AWS SQS queues
- Execute code in different programming languages with customizable compilation and execution commands
- Maintain sequential ordering of test results even with concurrent test execution
- Store execution results in S3 bucket
- Stream execution events for real-time progress monitoring
- Verify execution parameters like memory limits and timeouts

## Core Components

- `ExecSrvc`: Main service that handles code execution requests and result management
- `ExecResStreamOrganizer`: Manages ordered streaming of execution results
- `ExecRepo`: Interface for execution result storage (S3 or in-memory implementations)

## Usage

```go
// Create a new execution service with default configuration
srvc := execsrvc.NewDefaultExecSrvc()

// Or create with custom configuration
srvc := execsrvc.NewExecSrvc(
    logger,
    sqsClient,
    submissionQueueURL,
    execRepository,
    responseQueueURL,
    externalPartnerPassword,
)

// Enqueue code for execution
execID, err := srvc.Enqueue(
    code,
    tests,
    params,
)

// Get execution results
exec, err := srvc.Get(ctx, execID)
```

## Configuration

The service requires the following environment variables for AWS services setup:
- AWS credentials and region configuration
- SQS queue URLs for submission and response queues
- S3 bucket configuration for result storage
- External partner password for authentication

## TODO / ideas

- [ ] Add tests for S3 repository functionality
- [ ] Integrate with submission service using PostgreSQL for scoring storage
- [ ] Implement minimum memory limits per programming language
- [ ] Add support for evaluation without API key (without result persistence)

