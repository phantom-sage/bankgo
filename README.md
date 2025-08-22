# Bank REST API

A production-ready banking REST API service built with Go, Gin, PostgreSQL, and Redis. This service provides comprehensive banking functionality including multi-currency account management, secure money transfers, user authentication, and background email processing.

## 🚀 Features

- **Multi-currency account management**: Create and manage accounts in different currencies with unique constraints
- **Secure money transfers**: Atomic database transactions with automatic rollback capability
- **PASETO authentication**: Secure, stateless token-based authentication system
- **Background email processing**: Asynchronous welcome emails using Redis and Asyncq
- **Comprehensive error handling**: Detailed validation and business logic error responses
- **Production-ready security**: Rate limiting, CORS, request logging, and security headers
- **Docker deployment**: Complete containerization with development and production configurations
- **Health monitoring**: Built-in health checks and service monitoring endpoints
- **Advanced logging**: High-performance structured logging with zerolog, daily file rotation, audit trails, and performance monitoring
- **Comprehensive testing**: Unit, integration, and performance tests

## 📋 Prerequisites

### Development
- **Docker 20.10+** and **Docker Compose 2.0+** (recommended)
- **Go 1.24.3**: Latest stable version for optimal performance and security
- **PostgreSQL 15+**: Primary database for transactional data
- **Redis 7+**: Message broker for background email jobs

### Production
- **8GB RAM minimum** (4GB for development)
- **50GB disk space** (10GB for development)
- **SSL certificate** (recommended)
- **SMTP server**: For sending welcome emails (Gmail, SendGrid, etc.)

## 🚀 Quick Start

### Option 1: Docker (Recommended)

The fastest way to get started is using Docker:

```bash
# Clone the repository
git clone <repository-url>
cd bank-rest-api

# Set up development environment
make dev-setup

# Start all services
make up

# Verify deployment
make health
```

The API will be available at `http://localhost:8080`

### Option 2: Manual Setup

If you prefer to run services manually:

#### 1. Clone and Setup

```bash
git clone <repository-url>
cd bank-rest-api
```

#### 2. Environment Configuration

```bash
cp .env.example .env
# Edit .env with your configuration
```

**Required environment variables:**
```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=bankapi
DB_USER=bankuser
DB_PASSWORD=your_secure_password
DB_SSL_MODE=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
DB_CONN_MAX_IDLE_TIME=5m

# Redis Configuration  
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=5

# PASETO Authentication (must be at least 32 characters)
PASETO_SECRET_KEY=your_32_character_secret_key_here
PASETO_EXPIRATION=24h

# Email Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your_email@gmail.com
SMTP_PASSWORD=your_app_password
FROM_EMAIL=noreply@bankapi.com
FROM_NAME=Bank API

# Server Configuration
PORT=8080
HOST=0.0.0.0
GIN_MODE=release
READ_TIMEOUT=30s
WRITE_TIMEOUT=30s
IDLE_TIMEOUT=120s

# Logging Configuration (Zerolog)
LOG_LEVEL=info                    # debug, info, warn, error, fatal
LOG_FORMAT=json                   # json, console
LOG_OUTPUT=both                   # console, file, both
LOG_DIRECTORY=logs               # Directory for log files
LOG_MAX_AGE=30                   # Days to keep log files
LOG_MAX_BACKUPS=10               # Number of backup files to keep
LOG_MAX_SIZE=100                 # Maximum size in MB before rotation
LOG_COMPRESS=true                # Compress rotated files
LOG_LOCAL_TIME=true              # Use local time for file names
LOG_CALLER_INFO=false            # Include caller information
LOG_SAMPLING_ENABLED=false       # Enable log sampling for high volume
LOG_SAMPLING_INITIAL=100         # Initial sampling rate
LOG_SAMPLING_THEREAFTER=100      # Subsequent sampling rate
```

### 3. Database Setup

#### Install PostgreSQL

**macOS (using Homebrew):**
```bash
brew install postgresql@15
brew services start postgresql@15
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install postgresql-15 postgresql-contrib-15
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

#### Create Database and User

```bash
# Connect to PostgreSQL as superuser
sudo -u postgres psql

# Create database and user
CREATE DATABASE bankapi;
CREATE USER bankuser WITH ENCRYPTED PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE bankapi TO bankuser;
ALTER USER bankuser CREATEDB;
\q
```

#### Run Database Migrations

```bash
# Install dependencies
go mod download

# Run migrations
go run cmd/server/main.go migrate
```

### 4. Redis Setup

#### Install Redis

**macOS (using Homebrew):**
```bash
brew install redis
brew services start redis
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

#### Configure Redis (Optional)

If you need password authentication, edit `/etc/redis/redis.conf`:
```bash
requirepass your_redis_password
```

Then restart Redis:
```bash
sudo systemctl restart redis-server
```

### 5. Install Dependencies and Run

```bash
# Install Go dependencies
go mod download

# Run the application
go run cmd/server/main.go

# Or build and run
go build -o server cmd/server/main.go
./server
```

The server will start on `http://localhost:8080` (or your configured port).

## 🐳 Docker Commands

The project includes a comprehensive Makefile for easy Docker management:

```bash
make help          # Show all available commands
make build         # Build Docker images
make up            # Start development environment
make down          # Stop all services
make logs          # Show service logs
make restart       # Restart all services
make clean         # Remove all containers and volumes
make clean-volumes # Remove volumes only
make test          # Run tests in containers
make test-integration # Run integration tests
make health        # Check service health
make db-shell      # Connect to PostgreSQL
make redis-shell   # Connect to Redis
make db-migrate    # Run database migrations

# Production commands
make prod-up       # Start production environment
make prod-down     # Stop production services
make prod-logs     # Show production logs

# Development helpers
make dev-setup     # Set up development environment
make dev-reset     # Reset development environment
```

## 📚 Documentation

- **[API Documentation](docs/API.md)**: Complete API reference with examples
- **[Deployment Guide](docs/DEPLOYMENT.md)**: Production deployment instructions
- **[Logging Guide](docs/LOGGING.md)**: Comprehensive logging best practices and configuration
- **[Troubleshooting Guide](docs/TROUBLESHOOTING.md)**: Common issues and solutions

## 🏗️ Architecture

The application follows a clean architecture pattern with clear separation of concerns:

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

## 📁 Project Structure

```
bank-rest-api/
├── cmd/
│   └── server/              # Application entry point
├── internal/
│   ├── config/              # Configuration management
│   ├── database/            # Database layer
│   │   ├── migrations/      # SQL migration files
│   │   └── queries/         # SQLC generated code
│   ├── handlers/            # HTTP request handlers
│   ├── logging/             # Comprehensive logging system
│   ├── middleware/          # HTTP middleware (auth, CORS, logging)
│   ├── models/              # Data models and validation
│   ├── queue/               # Background job processing
│   ├── repository/          # Data access layer
│   ├── router/              # Route definitions
│   ├── services/            # Business logic layer
│   └── utils/               # Utility functions
├── pkg/
│   ├── auth/                # Authentication utilities
│   └── email/               # Email service
├── docs/                    # Documentation
│   ├── API.md              # API documentation
│   ├── DEPLOYMENT.md       # Deployment guide
│   └── TROUBLESHOOTING.md  # Troubleshooting guide
├── test/
│   └── integration/         # Integration tests
├── scripts/                 # Build and utility scripts
├── .kiro/
│   └── specs/              # Feature specifications
├── docker-compose.yml      # Development environment
├── docker-compose.prod.yml # Production environment
├── Dockerfile              # Container definition
├── Makefile               # Development commands
└── README.md              # This file
```

## 🔌 API Overview

### Base URL
```
http://localhost:8080/api/v1
```

### Quick API Examples

**Health Check:**
```bash
curl http://localhost:8080/api/v1/health
```

**Register User:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123!","first_name":"John","last_name":"Doe"}'
```

**Login:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123!"}'
```

**Create Account:**
```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"currency":"USD"}'
```

**Transfer Money:**
```bash
curl -X POST http://localhost:8080/api/v1/transfers \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"from_account_id":1,"to_account_id":2,"amount":"100.00","description":"Payment"}'
```

For complete API documentation with all endpoints, request/response examples, and error codes, see **[API Documentation](docs/API.md)**.

## 🔒 Security Features

- **PASETO Authentication**: Secure, stateless tokens with automatic expiration
- **Password Security**: bcrypt hashing with appropriate cost factor
- **Rate Limiting**: Per-IP and per-user request limits
- **CORS Protection**: Configurable cross-origin resource sharing
- **Input Validation**: Comprehensive request validation and sanitization
- **SQL Injection Prevention**: SQLC-generated parameterized queries
- **Environment Security**: Sensitive data in environment variables only
- **HTTPS Support**: TLS termination at load balancer level
- **Comprehensive Logging**: High-performance structured logging with zerolog, audit trails, and performance monitoring

## 📊 Logging System

The application uses **zerolog** for high-performance structured logging with zero allocations. The logging system provides comprehensive monitoring, audit trails, and debugging capabilities.

### Logging Features

- **Daily File Rotation**: Automatic daily log file creation with timestamped names (`app-YYYY-MM-DD.log`)
- **Multiple Output Modes**: Console, file, or both simultaneously
- **Structured JSON Logs**: Machine-readable logs for production environments
- **Human-Readable Console**: Developer-friendly output for local development
- **Audit Logging**: Security and compliance logging for sensitive operations
- **Performance Monitoring**: Request timing, database query performance, and resource usage
- **Contextual Logging**: Request ID correlation and user context throughout request lifecycle
- **Log File Management**: Automatic cleanup, compression, and retention policies

### Log Configuration

Configure logging behavior through environment variables:

```bash
# Basic Configuration
LOG_LEVEL=info                    # Minimum log level (debug, info, warn, error, fatal)
LOG_FORMAT=json                   # Output format (json for production, console for development)
LOG_OUTPUT=both                   # Output destination (console, file, both)

# File Management
LOG_DIRECTORY=logs               # Directory for log files
LOG_MAX_AGE=30                   # Days to keep log files (0 = keep all)
LOG_MAX_BACKUPS=10               # Number of backup files to keep (0 = keep all)
LOG_MAX_SIZE=100                 # Maximum file size in MB before rotation
LOG_COMPRESS=true                # Compress rotated files to save space
LOG_LOCAL_TIME=true              # Use local time for file names

# Advanced Options
LOG_CALLER_INFO=false            # Include file and line number in logs
LOG_SAMPLING_ENABLED=false       # Enable sampling for high-volume scenarios
LOG_SAMPLING_INITIAL=100         # Initial sampling rate
LOG_SAMPLING_THEREAFTER=100      # Subsequent sampling rate
```

### Log Levels

| Level | Description | Use Case |
|-------|-------------|----------|
| `debug` | Detailed debugging information | Development and troubleshooting |
| `info` | General application information | Normal operation events |
| `warn` | Warning conditions | Potential issues that don't stop operation |
| `error` | Error conditions | Errors that affect specific operations |
| `fatal` | Critical errors | Errors that cause application shutdown |

### Log Formats

**Production (JSON):**
```json
{
  "timestamp": "2024-01-15T10:30:45.123Z",
  "level": "info",
  "message": "Request completed",
  "request_id": "20240115103045-abc12345",
  "user_id": 123,
  "method": "POST",
  "path": "/api/v1/transfers",
  "status": 200,
  "duration_ms": 45,
  "client_ip": "192.168.1.100"
}
```

**Development (Console):**
```
2024-01-15T10:30:45Z INF Request completed request_id=20240115103045-abc12345 user_id=123 method=POST path=/api/v1/transfers status=200 duration_ms=45
```

### Audit Logging

Security-sensitive operations are automatically logged with detailed context:

- **Authentication Events**: Login attempts, token generation, failures
- **Account Operations**: Account creation, updates, deletions
- **Money Transfers**: All transfer operations with amounts and participants
- **Administrative Actions**: Admin operations with user identification
- **Security Events**: Rate limiting, failed authentication, suspicious activity

**Audit Log Example:**
```json
{
  "timestamp": "2024-01-15T10:30:45.123Z",
  "level": "info",
  "message": "Money transfer completed",
  "log_type": "audit",
  "event_type": "transfer",
  "user_id": 123,
  "from_account_id": 1,
  "to_account_id": 2,
  "amount": "100.00",
  "currency": "USD",
  "result": "success"
}
```

### Performance Monitoring

The logging system automatically tracks performance metrics:

- **HTTP Requests**: Response times, status codes, request sizes
- **Database Operations**: Query execution times, affected rows
- **Background Jobs**: Job execution duration, success rates
- **Resource Usage**: Memory and CPU utilization
- **External Services**: Third-party API response times

### Log File Management

**File Naming Convention:**
- Current log: `app-YYYY-MM-DD.log`
- Rotated logs: `app-YYYY-MM-DD.log.gz` (compressed)

**Automatic Cleanup:**
- Files older than `LOG_MAX_AGE` days are removed
- Only `LOG_MAX_BACKUPS` most recent files are kept
- Rotated files are compressed to save disk space

**Viewing Logs:**
```bash
# View current log file
tail -f logs/app-$(date +%Y-%m-%d).log

# View logs with Docker
make logs

# Search logs for specific patterns
grep "error" logs/app-*.log
grep "user_id=123" logs/app-*.log | jq .
```

### Troubleshooting Logging Issues

**Common Issues:**

1. **Log files not created:**
   - Check `LOG_DIRECTORY` permissions
   - Verify disk space availability
   - Check `LOG_OUTPUT` configuration

2. **Performance impact:**
   - Enable sampling with `LOG_SAMPLING_ENABLED=true`
   - Reduce log level to `warn` or `error`
   - Use `LOG_OUTPUT=file` to disable console output

3. **Disk space issues:**
   - Reduce `LOG_MAX_AGE` and `LOG_MAX_BACKUPS`
   - Enable compression with `LOG_COMPRESS=true`
   - Monitor disk usage regularly

**Health Check:**
```bash
# Check logging system health
curl http://localhost:8080/api/v1/health

# Verify log file creation
ls -la logs/

# Test log rotation
# (Logs rotate automatically at midnight)
```

## 🏦 Business Rules

### Account Management
- Users can create multiple accounts with different currencies
- Only one account per currency per user is allowed
- Account deletion requires zero balance and no transaction history
- Users can only access their own accounts

### Money Transfers
- Both accounts must have the same currency
- Source account must have sufficient balance
- All transfer operations are atomic (database transactions)
- Failed transfers are automatically rolled back
- Transfer history is maintained for all accounts

### Authentication & Email
- PASETO tokens expire after 24 hours (configurable)
- Welcome emails are sent on first login
- Email processing is handled asynchronously with retry logic
- Failed email deliveries are retried automatically

## 🧪 Testing

### Running Tests

```bash
# Using Docker (recommended)
make test

# Run all tests locally
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests
make test-integration

# Run specific package tests
go test ./internal/services -v
```

### Test Coverage

The project maintains high test coverage across all layers:
- **Unit Tests**: Business logic and individual components
- **Integration Tests**: End-to-end API workflows
- **Performance Tests**: Load testing and concurrent operations

### Development Mode

```bash
# Enable debug mode for detailed logging
GIN_MODE=debug

# Enable development features
make up
make logs  # Watch logs in real-time
```

## 🚀 Deployment

### Development Deployment

```bash
# Quick start with Docker
make up

# Manual setup
cp .env.example .env
# Edit .env with your configuration
go run cmd/server/main.go
```

### Production Deployment

```bash
# Using Docker Compose
make prod-up

# Or follow the complete production guide
```

For detailed deployment instructions including:
- Production environment setup
- SSL/TLS configuration
- Database optimization
- Monitoring and logging
- Backup and recovery

See **[Deployment Guide](docs/DEPLOYMENT.md)**.

## 🔧 Development Methodology

This project follows **Spec-Driven Development** methodology. All features are developed through a structured process:

1. **Requirements Gathering**: Define clear, testable requirements in EARS format
2. **Design Documentation**: Create comprehensive system design
3. **Implementation Planning**: Break down into actionable coding tasks
4. **Iterative Development**: Implement features incrementally with testing

See the `.kiro/specs/bank-rest-api/` directory for:
- **[Requirements](/.kiro/specs/bank-rest-api/requirements.md)**: Feature requirements in EARS format
- **[Design](/.kiro/specs/bank-rest-api/design.md)**: System architecture and component design  
- **[Tasks](/.kiro/specs/bank-rest-api/tasks.md)**: Implementation plan and task breakdown

## 🔧 Troubleshooting

### Quick Fixes

**Service Health Check:**
```bash
make health
# or
curl http://localhost:8080/api/v1/health
```

**View Logs:**
```bash
make logs
# or
docker-compose logs -f
```

**Restart Services:**
```bash
make down && make up
```

**Reset Development Environment:**
```bash
make clean
make dev-setup
make up
```

### Common Issues

| Issue | Quick Fix | Documentation |
|-------|-----------|---------------|
| Database connection failed | `make down && make up` | [Troubleshooting Guide](docs/TROUBLESHOOTING.md#database-issues) |
| Redis connection failed | Check Redis container status | [Troubleshooting Guide](docs/TROUBLESHOOTING.md#redis-issues) |
| Email not sending | Verify SMTP credentials | [Troubleshooting Guide](docs/TROUBLESHOOTING.md#email-issues) |
| Authentication failed | Check PASETO secret key | [Troubleshooting Guide](docs/TROUBLESHOOTING.md#authentication-issues) |
| Performance issues | Check resource limits | [Troubleshooting Guide](docs/TROUBLESHOOTING.md#performance-issues) |

For comprehensive troubleshooting including diagnostics, solutions, and debugging tools, see **[Troubleshooting Guide](docs/TROUBLESHOOTING.md)**.

## 📊 Environment Variables

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DB_PASSWORD` | Database password | `secure_db_password` |
| `REDIS_PASSWORD` | Redis password | `secure_redis_password` |
| `PASETO_SECRET_KEY` | Token signing key (32+ chars) | `your_32_character_secret_key_here` |
| `SMTP_PASSWORD` | Email service password | `your_email_app_password` |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_MAX_OPEN_CONNS` | `25` | Maximum open database connections |
| `DB_MAX_IDLE_CONNS` | `5` | Maximum idle database connections |
| `DB_CONN_MAX_LIFETIME` | `5m` | Maximum connection lifetime |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_DB` | `0` | Redis database number |
| `REDIS_POOL_SIZE` | `10` | Redis connection pool size |
| `PORT` | `8080` | Server port |
| `HOST` | `0.0.0.0` | Server host |
| `GIN_MODE` | `debug` | Application mode |
| `READ_TIMEOUT` | `30s` | HTTP read timeout |
| `WRITE_TIMEOUT` | `30s` | HTTP write timeout |
| `IDLE_TIMEOUT` | `120s` | HTTP idle timeout |
| `LOG_LEVEL` | `info` | Logging level (debug/info/warn/error/fatal) |
| `LOG_FORMAT` | `json` | Log format (json/console) |
| `LOG_OUTPUT` | `both` | Log output (console/file/both) |
| `LOG_DIRECTORY` | `logs` | Directory for log files |
| `LOG_MAX_AGE` | `30` | Days to keep log files |
| `LOG_MAX_BACKUPS` | `10` | Number of backup files to keep |
| `LOG_MAX_SIZE` | `100` | Maximum file size in MB |
| `LOG_COMPRESS` | `true` | Compress rotated files |
| `LOG_LOCAL_TIME` | `true` | Use local time for file names |
| `LOG_CALLER_INFO` | `false` | Include caller information |
| `LOG_SAMPLING_ENABLED` | `false` | Enable log sampling |

### Security Best Practices

```bash
# Generate secure secrets
PASETO_SECRET_KEY=$(openssl rand -base64 32)
DB_PASSWORD=$(openssl rand -base64 24)
REDIS_PASSWORD=$(openssl rand -base64 24)
```

## 🤝 Contributing

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Follow the spec-driven development process**:
   - Update requirements if needed
   - Update design documentation
   - Implement with tests
4. **Commit changes**: `git commit -m 'Add amazing feature'`
5. **Push to branch**: `git push origin feature/amazing-feature`
6. **Open a Pull Request**

### Development Guidelines

- Follow Go best practices and conventions
- Maintain test coverage above 80%
- Update documentation for new features
- Use conventional commit messages
- Ensure all tests pass before submitting PR

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Documentation**: Check the `docs/` directory for comprehensive guides
- **PDF Documentation**: Generate PDF version with `./scripts/generate-docs-pdf.sh`
- **Issues**: Create a GitHub issue with detailed information
- **Troubleshooting**: See [Troubleshooting Guide](docs/TROUBLESHOOTING.md)
- **API Reference**: See [API Documentation](docs/API.md)

## 🏗️ Built With

- **[Go 1.24.3](https://golang.org/)** - Programming language
- **[Gin](https://gin-gonic.com/)** - HTTP web framework  
- **[PostgreSQL 15+](https://www.postgresql.org/)** - Primary database with ACID compliance
- **[Redis 7+](https://redis.io/)** - Caching and message broker for background jobs
- **[SQLC](https://sqlc.dev/)** - Type-safe SQL code generation from SQL queries
- **[Asynq](https://github.com/hibiken/asynq)** - Background job processing with Redis
- **[PASETO v2](https://paseto.io/)** - Secure, stateless authentication tokens
- **[pgx/v5](https://github.com/jackc/pgx)** - PostgreSQL driver and toolkit
- **[Zerolog](https://github.com/rs/zerolog)** - Structured logging with performance focus
- **[Docker](https://www.docker.com/)** - Containerization platform

---

**Ready to get started?** Run `make dev-setup && make up` and visit `http://localhost:8080/api/v1/health` to verify your deployment! 🚀