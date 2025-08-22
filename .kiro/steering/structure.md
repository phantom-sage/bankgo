# Project Structure & Architecture

## Clean Architecture Pattern

The project follows clean architecture with clear separation of concerns across layers:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Client   │    │   Email Queue   │    │   Admin Tools   │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          ▼                      ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Gin HTTP Server                         │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │ Middleware  │ │  Handlers   │ │    Auth     │ │   CORS    │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Business Layer                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │   Account   │ │  Transfer   │ │    User     │ │   Email   │ │
│  │  Service    │ │  Service    │ │  Service    │ │  Service  │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Data Layer                               │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │ Repository  │ │    SQLC     │ │    Queue    │ │   Cache   │ │
│  │  Pattern    │ │  Generated  │ │  Manager    │ │  Manager  │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Infrastructure                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │ PostgreSQL  │ │    Redis    │ │    SMTP     │ │  Docker   │ │
│  │  Database   │ │    Queue    │ │   Server    │ │ Container │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Directory Structure

### Application Entry Point
- `cmd/server/`: Application entry point and main function
  - `main.go`: Server initialization, configuration loading, graceful shutdown

### Internal Packages (Business Logic)
- `internal/config/`: Configuration management with environment variable loading
- `internal/database/`: Database layer with migrations and SQLC generated code
  - `migrations/`: SQL migration files (up/down)
  - `queries/`: SQLC generated type-safe database code
- `internal/handlers/`: HTTP request handlers (presentation layer)
- `internal/middleware/`: HTTP middleware (auth, CORS, logging, rate limiting, error handling)
- `internal/models/`: Data models and validation logic
- `internal/queue/`: Background job processing with Redis and Asyncq
- `internal/repository/`: Data access layer implementing repository pattern
- `internal/router/`: Route definitions and HTTP server setup
- `internal/services/`: Business logic layer (core application logic)
- `internal/utils/`: Utility functions and helpers
- `internal/logging/`: Comprehensive logging system with audit, performance, and context logging

### Public Packages (Reusable Components)
- `pkg/auth/`: Authentication utilities (PASETO token management)
- `pkg/email/`: Email service for SMTP operations

### External Resources
- `docs/`: Documentation (API.md, DEPLOYMENT.md, TROUBLESHOOTING.md)
- `test/integration/`: Integration tests and test utilities
- `scripts/`: Build and utility scripts
- `.kiro/specs/`: Feature specifications for spec-driven development

## Naming Conventions

### Files & Directories
- Use snake_case for file names: `user_service.go`, `auth_middleware.go`
- Use lowercase for directory names: `internal/handlers/`, `pkg/auth/`
- Test files end with `_test.go`: `user_service_test.go`
- Integration test files: `*_integration_test.go`

### Go Code Conventions
- Use PascalCase for exported functions/types: `CreateUser()`, `UserService`
- Use camelCase for unexported functions/variables: `validateEmail()`, `userRepo`
- Interface names end with 'er' when possible: `Querier`, `Logger`
- Error variables start with 'Err': `ErrUserNotFound`, `ErrInvalidCredentials`

### Database Conventions
- Table names are plural: `users`, `accounts`, `transfers`
- Column names use snake_case: `user_id`, `created_at`, `welcome_email_sent`
- Foreign key format: `{table}_id` (e.g., `user_id`, `account_id`)

## Layer Responsibilities

### Handlers Layer (`internal/handlers/`)
- HTTP request/response handling
- Request validation and binding
- Authentication middleware integration
- Error response formatting
- No business logic - delegate to services

### Services Layer (`internal/services/`)
- Core business logic implementation
- Transaction management
- Business rule validation
- Cross-cutting concerns (logging, metrics)
- Coordinate between repositories

### Repository Layer (`internal/repository/`)
- Data access abstraction
- SQLC integration
- Database transaction management
- Query optimization
- No business logic

### Models Layer (`internal/models/`)
- Data structure definitions
- Validation rules
- Business entity representations
- Shared constants and enums

## Configuration Structure

### Environment-based Configuration
- Development: `.env` file with `docker-compose.yml`
- Production: Environment variables with `docker-compose.prod.yml`
- Configuration validation on startup
- Sensible defaults for optional settings

### Database Configuration
- Connection pooling settings
- Migration management
- SSL/TLS configuration
- Timeout and retry settings

## Testing Structure

### Unit Tests
- Co-located with source files: `*_test.go`
- Mock external dependencies
- Focus on business logic validation

### Integration Tests
- Located in `test/integration/`
- Test complete workflows
- Use test database and Redis instances
- Performance and load testing

## Error Handling Patterns

### Structured Error Responses
```go
type ErrorResponse struct {
    Error   string            `json:"error"`
    Message string            `json:"message"`
    Code    int               `json:"code"`
    Details map[string]string `json:"details,omitempty"`
}
```

### Error Categories
- `validation_error`: Input validation failures
- `authentication_failed`: Auth/authorization issues
- `business_rule_violation`: Business logic constraints
- `internal_error`: System/infrastructure errors