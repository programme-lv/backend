# Programme.lv Backend

A Go-based backend system for Programme.lv, a programming contest and learning platform. This system provides services for task management, code execution, user authentication, and submission handling.

## Core Services

### Task Service (`tasksrvc`)
- Programming task management
- Multiple statement formats (Markdown, PDF)
- Test case handling with compression
- Asset management (illustrations, images)
- Subtask and test group organization

### Execution Service (`execsrvc`)
- Code execution and testing
- Real-time execution updates
- Multiple programming language support
- Resource limit enforcement
- Test result streaming

### User Service (`usersrvc`)
- User authentication and registration
- JWT-based session management
- Profile management
- Secure password handling
- Input validation

### Submission Service (`submsrvc`)
- Code submission handling
- Real-time evaluation updates
- Multiple scoring methods
- Submission history
- Result persistence

### HTTP Service (`http`)
- RESTful API endpoints
- Real-time updates via SSE
- Authentication middleware
- CORS support
- Request statistics

## Getting Started

### Prerequisites
- Go 1.21 or later
- PostgreSQL 15 or later
- AWS S3 access
- Docker (optional)

### Environment Variables
```bash
# PostgreSQL
POSTGRES_USER=your_user
POSTGRES_PASSWORD=your_password
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=proglv
POSTGRES_SSLMODE=disable

# JWT
JWT_SECRET_KEY=your_secret

# Server
PORT=8080
```

### Installation
```bash
# Clone the repository
git clone https://github.com/programme-lv/backend.git
cd backend

# Install dependencies
go mod download

# Run migrations
go run ./migrate

# Start the server
go run main.go
```

## Architecture

### Storage
- PostgreSQL: User data, submissions, evaluations
- S3 Buckets:
  - `proglv-tasks`: Task definitions
  - `proglv-public`: Public assets
  - `proglv-tests`: Test files

### Communication
- HTTP REST API
- Server-Sent Events for real-time updates
- JWT for authentication
- AWS S3 for file storage

### Security
- Bcrypt password hashing
- JWT token validation
- Input validation
- Resource limits
- CORS configuration

## API Documentation

### Authentication
- `POST /auth/login`: User login
- `POST /users`: User registration

### Tasks
- `GET /tasks`: List tasks
- `GET /tasks/{taskId}`: Get task details
- `GET /programming-languages`: List supported languages

### Submissions
- `POST /submissions`: Create submission
- `GET /submissions`: List submissions
- `GET /submissions/{submUuid}`: Get submission details
- `GET /subm-updates`: SSE endpoint for updates

### Execution
- `POST /tester/run`: Execute code
- `GET /tester/run/{evalUuid}`: Get execution progress
- `GET /exec/{execUuid}`: Get execution results

## Development

### Project Structure
```
backend/
├── execsrvc/    # Execution service
├── fstask/      # Task filesystem handling
├── http/        # HTTP server and endpoints
├── migrate/     # Database migrations
├── submsrvc/    # Submission service
├── tasksrvc/    # Task service
├── usersrvc/    # User service
└── main.go      # Application entry point
```
