# FSTask Package

A Go package for reading and writing programming contest task directories. This package handles the filesystem-based task format used by Programme.lv, supporting various task components like tests, examples, statements, and evaluation files.

## Features

- Read and write task directories with version control
- Support for multiple statement formats (Markdown and PDF)
- Test and example management
- Solution handling with metadata
- Asset and archive file management
- Evaluation components (checker and interactor)
- Subtask and test group organization

## Directory Structure

```
task-name/
├── assets/           # Task-related assets (images, etc.)
├── archive/          # Additional task files
├── evaluation/       # Checker and interactor files
│   ├── checker.cpp
│   └── interactor.cpp
├── examples/         # Example test cases
│   ├── 001.in
│   └── 001.out
├── problem.toml      # Task metadata and configuration
├── statements/       # Task statements
│   ├── md/          # Markdown statements
│   └── pdf/         # PDF statements
├── solutions/        # Solution files with metadata
└── tests/           # Test cases
    ├── 001.in
    └── 001.ans
```

## Usage

```go
// Read a task directory
task, err := fstask.Read("path/to/task")

// Store a task to directory
err = task.Store("path/to/output")
```

## Task Components

- `Task`: Main structure containing all task information
- `Example`: Input/output examples with optional notes
- `Test`: Test cases with input and answer
- `TestGroup`: Groups of tests with scoring information
- `Subtask`: Subtask information with points and descriptions
- `Solution`: Solution files with metadata
- `MarkdownStatement`: Task statements in markdown format
- `PdfStatement`: Task statements in PDF format
- `AssetFile`: Task-related assets
- `ArchiveFile`: Additional task files

## Configuration

The `problem.toml` file contains task metadata:
- Task name and full name
- Origin and authorship information
- Resource constraints (memory, CPU time)
- Problem tags and difficulty rating
- Test group and subtask configuration
- Solution metadata

## Version Support

The package uses semantic versioning for the task format specification. Current version: v3.0.0

Version features:
- v3.0.0: Image sizing and subtask listing in problem.toml
- v2.5.0: Solutions directory and archive support
- v2.4.0: Assets directory and origin notes
- v2.3.0: Test ID system and ordering
- v2.2.0: Visible input subtasks
- v2.1.0: Test group support
- v2.0.0: Basic task format