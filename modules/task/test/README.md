# Task Integration Tests

These are **integration tests** that verify the task module's complete functionality across all layers:
- HTTP handlers and routing
- Service layer business logic  
- Repository layer database operations
- External dependencies (PostgreSQL, S3)

## What's Tested

### `statement_test.go`
- Task statement PATCH operations via HTTP
- Authentication/authorization flow
- Statement persistence and retrieval
- Full request/response cycle

### `image_test.go`  
- Image upload via multipart form HTTP requests
- Image deletion via HTTP DELETE
- S3 integration for image storage
- Authentication/authorization flow
- Image metadata extraction and validation

## Test Infrastructure

- **Database**: Uses `pgtestdb` for isolated PostgreSQL test databases
- **S3**: Uses real S3 bucket `proglv-testing` for image operations
- **Authentication**: Generates JWT tokens for testing protected endpoints
- **HTTP**: Full HTTP request/response testing with `httptest`

## Requirements

Before running these tests, ensure:

1. **PostgreSQL** is running on `localhost:5433`
   - User: `proglv`
   - Password: `proglv`
   - Database: Created automatically by pgtestdb

2. **S3 Access** is configured for bucket `proglv-testing`
   - AWS credentials must be available via environment or IAM roles

3. **Migrations** are available at `../../../migrate/`

## Running Tests

```bash
# From backend root directory
go test ./task/tests/integration/

# With verbose output
go test -v ./task/tests/integration/

# Run specific test
go test -v ./task/tests/integration/ -run TestPostStatementImageHttpRequest
```

## Test Data

Test images and fixtures are located in `../../testdata/` relative to this directory. 