# Submission Service Package

A Go package that manages programming task submissions and their evaluations for Programme.lv. This service handles submission creation, storage, evaluation, and real-time updates of evaluation progress.

## Features

- Create and store code submissions
- Manage submission evaluations with real-time updates
- Support multiple scoring units (test, test group, subtask)
- PostgreSQL-based persistent storage
- Real-time submission updates via channels
- Integration with execution service for code testing
- Support for multiple programming languages

## Components

### Core Types

- `SubmissionSrvc`: Main service handling submission operations
- `Submission`: User submission with metadata and evaluation results
- `Evaluation`: Code evaluation details and results
- `Test`: Individual test case results
- `TestGroup`: Group of tests with scoring
- `Subtask`: Task subtask information and scoring

### Storage

- `PgSubmRepo`: PostgreSQL repository for submissions
- `PgEvalRepo`: PostgreSQL repository for evaluations
- In-memory cache for active evaluations

## Usage

```go
// Create a new submission service
taskSrvc := tasksrvc.NewTaskSrvc()
evalSrvc := execsrvc.NewDefaultExecSrvc()
submSrvc, err := submsrvc.NewSubmSrvc(taskSrvc, evalSrvc)

// Create a submission
submission, err := submSrvc.CreateSubmission(ctx, &CreateSubmissionParams{
    Submission: sourceCode,
    Username:   username,
    ProgLangID: languageID,
    TaskCodeID: taskID,
})

// Listen for evaluation updates
updates, err := submSrvc.ListenToLatestSubmEvalUpdate(ctx, submission.UUID)
for update := range updates {
    // Handle evaluation progress
}

// Get submission details
submission, err := submSrvc.GetSubm(ctx, submissionUUID)

// List submissions
submissions, err := submSrvc.ListSubms(ctx, limit, offset)
```

## Evaluation Process

1. Submission creation triggers automatic evaluation
2. Real-time updates through channels:
   - Compilation status
   - Test execution progress
   - Final results
3. Results stored in both PostgreSQL and temporary in-memory cache
4. Support for re-evaluation of submissions

## Configuration

The service requires:
- PostgreSQL connection details
- AWS S3 configuration for test storage
- Integration with task and execution services

## Error Handling

Standardized error types for common scenarios:
- Submission length exceeded
- Task not found
- User not found
- Invalid programming language
- Unauthorized access
- Submission not found
- Internal server errors

## Real-time Updates

Three types of update channels:
- Individual submission updates
- Submission list updates
- New submission notifications

Future plans include:
- Restricted evaluation updates
- Contest-specific visibility rules
- User-based access control
