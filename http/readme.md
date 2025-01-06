# HTTP Package

A Go package that provides HTTP server implementation for the Programme.lv backend services. This package handles all REST API endpoints, authentication, and real-time updates through server-sent events (SSE).

## Core Components

- `HttpServer`: Main server implementation that integrates various services
- `JsonResponse`: Standardized JSON response handling
- JWT authentication middleware
- CORS configuration for cross-origin requests
- Request statistics logging

## API Endpoints

### Authentication
- `POST /auth/login`: User login
- `POST /users`: User registration

### Tasks
- `GET /tasks`: List available tasks
- `GET /tasks/{taskId}`: Get specific task details
- `GET /programming-languages`: List supported programming languages
- `GET /langs`: Alias for programming languages list

### Submissions
- `POST /submissions`: Create new submission
- `GET /submissions`: List submissions
- `GET /submissions/{submUuid}`: Get specific submission
- `POST /reevaluate`: Reevaluate selected submissions
- `GET /subm-updates`: SSE endpoint for real-time submission updates

### Execution
- `POST /tester/run`: Start code execution
- `GET /tester/run/{evalUuid}`: SSE endpoint for execution progress
- `GET /exec/{execUuid}`: Get execution results

## Features

- JWT-based authentication
- Real-time updates using Server-Sent Events
- CORS support for frontend integration
- Standardized error handling
- Request statistics monitoring
- Integration with multiple backend services:
  - User Service
  - Task Service
  - Submission Service
  - Execution Service

## Configuration

The server requires:
- JWT signing key
- CORS allowed origins configuration
- Service dependencies (user, task, submission, execution services)

## Usage

```go
// Create a new HTTP server
server := http.NewHttpServer(
    submissionService,
    userService,
    taskService,
    executionService,
    jwtKey,
)

// Start the server
err := server.Start(":8080")
```

