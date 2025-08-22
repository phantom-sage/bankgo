# Technology Stack

## Core Technologies

- **Go 1.24.3**: Programming language with latest stable version for optimal performance and security
- **Gin**: HTTP web framework for building REST APIs
- **PostgreSQL 15+**: Primary database for transactional data with ACID compliance
- **Redis 7+**: Message broker for background job processing and caching
- **Docker**: Containerization platform for development and production deployment

## Key Libraries & Frameworks

### Database & ORM
- **SQLC**: Type-safe SQL code generation from SQL queries
- **pgx/v5**: PostgreSQL driver and toolkit
- **Database migrations**: SQL-based migration system in `internal/database/migrations/`

### Authentication & Security
- **PASETO v2**: Secure, stateless token-based authentication (preferred over JWT)
- **bcrypt**: Password hashing with appropriate cost factor
- **Gin middleware**: CORS, rate limiting, request logging, error handling

### Background Processing
- **Asyncq**: Background job processing with Redis
- **Redis client**: go-redis/v9 for Redis operations

### Validation & Utilities
- **go-playground/validator**: Request validation and binding
- **shopspring/decimal**: Precise decimal arithmetic for financial calculations
- **rs/zerolog**: High-performance structured logging with zero allocations

## Build System & Commands

### Docker Commands (Preferred)
```bash
make help          # Show all available commands
make build         # Build Docker images
make up            # Start development environment
make down          # Stop all services
make logs          # Show service logs
make clean         # Remove all containers and volumes
make test          # Run tests in containers
make health        # Check service health

# Production commands
make prod-up       # Start production environment
make prod-down     # Stop production services
make prod-logs     # Show production logs

# Database operations
make db-shell      # Connect to PostgreSQL
make redis-shell   # Connect to Redis
```

### Local Development Commands
```bash
# Dependencies
go mod download    # Install Go dependencies
go mod tidy       # Clean up dependencies

# Testing
go test ./...                    # Run all tests
go test -cover ./...            # Run tests with coverage
go test ./internal/services -v  # Run specific package tests
go test ./test/integration/...  # Run integration tests

# Building
go build -o server cmd/server/main.go  # Build binary
go run cmd/server/main.go              # Run directly

# Code generation
sqlc generate     # Generate type-safe SQL code
```

## Configuration Management

### Environment Variables
- Configuration loaded from environment variables via `internal/config/config.go`
- Required variables: `DB_PASSWORD`, `REDIS_PASSWORD`, `PASETO_SECRET_KEY`, `SMTP_PASSWORD`
- Optional variables have sensible defaults
- Validation performed on startup with detailed error messages

### Development Setup
```bash
cp .env.example .env  # Copy environment template
# Edit .env with your configuration
make dev-setup        # Initialize development environment
```

## Code Generation

### SQLC Configuration
- Configuration in `sqlc.yaml`
- Generates type-safe Go code from SQL queries
- Queries in `internal/database/queries/`
- Migrations in `internal/database/migrations/`
- Generated code in `internal/database/queries/`

## Testing Strategy

- **Unit Tests**: Business logic and individual components
- **Integration Tests**: End-to-end API workflows in `test/integration/`
- **Performance Tests**: Load testing and concurrent operations
- Target: Maintain >80% test coverage
- All tests must pass before PR submission