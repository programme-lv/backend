# Task Service Package

A Go package that manages programming contest tasks for Programme.lv. This service handles task storage, retrieval, and management of task-related assets including test files, statements, and illustrations.

## Features

- Task creation and storage in AWS S3
- Support for multiple statement formats (Markdown and PDF)
- Test file management with Zstandard compression
- Image handling for task illustrations and markdown content
- Multi-language support for task statements
- Subtask and test group organization
- Task metadata management

## Components

### Core Types

- `TaskService`: Main service for task operations
- `Task`: Complete task information including metadata and evaluation details
- `MarkdownStatement`: Task statements in markdown format with image support
- `PdfStatement`: Task statements in PDF format
- `TestGroup`: Groups of tests with scoring information
- `Subtask`: Task subtask information with descriptions
- `Test`: Test case information with input/output checksums

### Storage Organization

Tasks and related files are stored in three S3 buckets:
- `proglv-tasks`: Task definitions in JSON format
- `proglv-public`: Public assets (PDF statements, illustrations, markdown images)
- `proglv-tests`: Compressed test files

## Usage

```go
// Create a new task service
taskSrvc, err := tasksrvc.NewTaskSrvc()

// Get a specific task
task, err := taskSrvc.GetTask(ctx, "task-id")

// List all tasks
tasks, err := taskSrvc.ListTasks(ctx)

// Create or update a task
err = taskSrvc.PutTask(&Task{
    ShortId:  "task-id",
    FullName: "Task Name",
    // ... other task properties
})

// Upload task assets
pdfUrl, err := taskSrvc.UploadStatementPdf(pdfData)
imgUrl, err := taskSrvc.UploadIllustrationImg(mimeType, imgData)
err = taskSrvc.UploadTestFile(testData)
```

## Task Structure

A task includes:
- Basic information (ID, name, constraints)
- Origin and metadata
- Statements in multiple formats and languages
- Examples and visible input subtasks
- Test cases and evaluation details
- Scoring information (subtasks and test groups)

## File Management

### Test Files
- Automatically compressed using Zstandard
- Stored with SHA256-based naming
- Cached for performance

### Public Assets
- PDF statements: `task-pdf-statements/<sha256>.pdf`
- Illustrations: `task-illustrations/<sha256>.<ext>`
- Markdown images: `task-md-images/<sha256>.<ext>`

### Task Definitions
- Stored as JSON files
- Named by task ID: `<task-id>.json`
- Includes all task metadata and configuration

## Error Handling

Standardized error types for common scenarios:
- Task not found
- File upload failures
- Invalid file formats
- S3 storage errors

## Performance Considerations

- Test files are cached in memory
- Task definitions are loaded at service startup
- File deduplication using content-based hashing
- Efficient compression for test files 