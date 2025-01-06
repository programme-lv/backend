# User Service Package

A Go package that manages user authentication and account management for Programme.lv. This service handles user registration, login, and JWT-based authentication with PostgreSQL storage.

## Features

- User registration with validation
- Secure password handling with bcrypt
- JWT-based authentication
- User profile management
- PostgreSQL-based storage
- Input validation for all user fields
- Localized error messages (Latvian)

## Components

### Core Types

- `UserService`: Main service for user operations
- `User`: Core user information and profile
- `JWTClaims`: JWT token structure and validation
- `LoginParams`: Login request parameters
- `CreateUserParams`: Registration parameters

### Validation Rules

Username:
- Length: 2-32 characters
- Must be unique

Email:
- Valid email format
- Maximum length: 320 characters
- Must be unique

Password:
- Minimum length: 8 characters
- Maximum length: 1024 characters
- Securely hashed using bcrypt

Name Fields:
- Maximum length: 35 characters
- Optional fields

## Usage

```go
// Create a new user service
userSrvc := usersrvc.NewUserService()

// Register a new user
user, err := userSrvc.CreateUser(ctx, CreateUserParams{
    Username:  "username",
    Email:     "email@example.com",
    Password:  "password",
    Firstname: &firstname,
    Lastname:  &lastname,
})

// Login
user, err := userSrvc.Login(ctx, &LoginParams{
    Username: "username",
    Password: "password",
})

// Get user by username
user, err := userSrvc.GetUserByUsername(ctx, "username")

// Get user by UUID
user, err := userSrvc.GetUserByUUID(ctx, userUUID)

// Query current JWT claims
claims, err := userSrvc.QueryCurrentJWT(ctx)
```

## Error Handling

Standardized error types with localized messages:

Registration Errors:
- Username too short/long
- Username already exists
- Email too long/invalid
- Email already exists
- Password too short/long
- Name fields too long

Authentication Errors:
- User not found
- Incorrect username/password
- Invalid JWT claims

## Database Schema

Users table includes:
- UUID (primary key)
- Username (unique)
- Email (unique)
- Bcrypt password hash
- First name
- Last name
- Creation timestamp

## Security Considerations

- Passwords are never stored in plain text
- Bcrypt used for password hashing
- JWT tokens for session management
- Input validation for all user data
- Unique constraints on username and email
- Protected routes require valid JWT

## Configuration

The service requires:
- PostgreSQL connection details (via environment variables)
- JWT signing key
- Optional: custom password hashing parameters 