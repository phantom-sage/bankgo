# Integration Tests

This directory contains comprehensive integration tests for the Bank REST API.

## Test Structure

### integration_test.go
Contains the main integration test suite with end-to-end workflow tests:
- User registration and login workflow
- Account creation and management workflow  
- Money transfer operations with rollback scenarios
- Welcome email queuing and processing workflow

### performance_test.go
Contains performance and error scenario tests:
- Concurrent transfer operations
- Database transaction rollback scenarios
- Rate limiting and error handling
- Email retry logic and failure scenarios

### test_utils.go
Contains test utilities and mock implementations for testing.

## Requirements Coverage

The tests cover the following requirements:

### Requirement 1 (Account Management)
- ✅ Account creation with currency validation
- ✅ Multiple accounts per user with different currencies
- ✅ Unique currency constraint per user
- ✅ Account listing and retrieval

### Requirement 2 (Account Details)
- ✅ Account details retrieval with balance
- ✅ User can only access their own accounts
- ✅ Proper error handling for non-existent accounts

### Requirement 3 (Money Transfers)
- ✅ Transfer operations with database transactions
- ✅ Balance validation and currency matching
- ✅ Atomic operations with rollback on failure
- ✅ Concurrent transfer handling
- ✅ Transfer history and confirmation details

### Requirement 4 (CRUD Operations)
- ✅ Account creation, update, and deletion
- ✅ Zero balance validation for deletion
- ✅ Transaction history prevention for deletion

### Requirement 5 (Error Handling)
- ✅ Appropriate HTTP status codes
- ✅ Detailed error messages
- ✅ Rate limiting implementation
- ✅ Authentication and authorization errors

### Requirement 8 (Welcome Email)
- ✅ Welcome email queuing on first login
- ✅ Redis and Asyncq integration
- ✅ Email retry logic and failure handling
- ✅ Idempotency (no duplicate emails)

## Running Tests

### Prerequisites
- PostgreSQL database running on localhost:5432
- Redis server running on localhost:6379
- Test database: `bankapi_test`
- Test user: `bankuser` with password `testpassword`

### Environment Setup
```bash
# Set up test database
createdb bankapi_test
psql -d bankapi_test -c "CREATE USER bankuser WITH PASSWORD 'testpassword';"
psql -d bankapi_test -c "GRANT ALL PRIVILEGES ON DATABASE bankapi_test TO bankuser;"

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=bankapi_test
export DB_USER=bankuser
export DB_PASSWORD=testpassword
export DB_SSL_MODE=disable

export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_DB=1

export PASETO_SECRET_KEY=test-secret-key-32-characters-long
export PASETO_EXPIRATION=1h

export SMTP_HOST=localhost
export SMTP_PORT=1025
export SMTP_USERNAME=test@example.com
export SMTP_PASSWORD=testpassword
```

### Run Tests
```bash
# Run all integration tests
go test -v ./test/integration/...

# Run specific test suite
go test -v ./test/integration/ -run TestIntegrationSuite

# Run with race detection
go test -v -race ./test/integration/...

# Run with coverage
go test -v -cover ./test/integration/...
```

## Test Features

### Database Transactions
- All transfer operations are tested within database transactions
- Rollback scenarios are verified to ensure data consistency
- Concurrent operations are tested for proper isolation

### Error Scenarios
- Invalid input validation
- Authentication and authorization failures
- Business logic violations (insufficient balance, currency mismatch)
- Database connection failures
- Rate limiting enforcement

### Performance Testing
- Concurrent transfer operations
- Bulk email processing
- Rate limiting under load
- Queue persistence across restarts

### Email Processing
- Welcome email queuing and processing
- Retry logic for failed emails
- Idempotency checks
- Queue failure handling

## Mock Implementations

The test suite includes mock implementations for:
- Email processing (TestEmailProcessor)
- Queue operations (QueueManagerTestExtensions)
- Database operations (using real database with cleanup)

## Test Data Management

- Each test starts with a clean database state
- Test data is automatically cleaned up after each test
- Redis queues are cleared between tests
- Proper isolation between test cases