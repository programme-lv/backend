# Task Module

The task module handles programming task management:
- creation
- retrieval
- statement updates
- image uploads

Following the modular monolith architecture with these layers:
- `http/`: HTTP handlers for REST API endpoints
- `srvc/`: Service layer for business logic orchestration  
- `repo/`: Repository layer for database persistence

Additionally, there are integration tests in the `test/` directory.

Integration tests require:
- PostgreSQL database (configured in `test/helpers_test.go`)
- S3 bucket access for image storage
- Proper authentication tokens for protected endpoints 

```bash
go test ./task/...
```