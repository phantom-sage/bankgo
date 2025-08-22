#!/bin/bash

# Integration Test Runner Script for Bank REST API

set -e

echo "🚀 Starting Bank REST API Integration Tests"

# Check if required services are running
echo "📋 Checking prerequisites..."

# Check PostgreSQL
if ! pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
    echo "❌ PostgreSQL is not running on localhost:5432"
    echo "Please start PostgreSQL and ensure it's accessible"
    exit 1
fi
echo "✅ PostgreSQL is running"

# Check Redis
if ! redis-cli -h localhost -p 6379 ping >/dev/null 2>&1; then
    echo "❌ Redis is not running on localhost:6379"
    echo "Please start Redis and ensure it's accessible"
    exit 1
fi
echo "✅ Redis is running"

# Set test environment variables
echo "🔧 Setting up test environment..."

export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=bankapi_test
export DB_USER=bankuser
export DB_PASSWORD=testpassword
export DB_SSL_MODE=disable

export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_DB=1

export PASETO_SECRET_KEY=test-secret-key-32-characters-long-enough-for-security
export PASETO_EXPIRATION=1h

export SMTP_HOST=localhost
export SMTP_PORT=1025
export SMTP_USERNAME=test@example.com
export SMTP_PASSWORD=testpassword

echo "✅ Environment variables set"

# Create test database if it doesn't exist
echo "🗄️  Setting up test database..."

if ! psql -h localhost -p 5432 -U postgres -lqt | cut -d \| -f 1 | grep -qw bankapi_test; then
    echo "Creating test database..."
    createdb -h localhost -p 5432 -U postgres bankapi_test
    psql -h localhost -p 5432 -U postgres -d bankapi_test -c "CREATE USER bankuser WITH PASSWORD 'testpassword';" 2>/dev/null || true
    psql -h localhost -p 5432 -U postgres -d bankapi_test -c "GRANT ALL PRIVILEGES ON DATABASE bankapi_test TO bankuser;"
    echo "✅ Test database created"
else
    echo "✅ Test database already exists"
fi

# Clear Redis test database
echo "🧹 Clearing Redis test database..."
redis-cli -h localhost -p 6379 -n 1 FLUSHDB >/dev/null
echo "✅ Redis test database cleared"

# Run the tests
echo "🧪 Running integration tests..."

# Change to project root directory
cd "$(dirname "$0")/.."

# Run tests with verbose output and race detection
if go test -v -race -timeout 30m ./test/integration/...; then
    echo "✅ All integration tests passed!"
else
    echo "❌ Some integration tests failed"
    exit 1
fi

echo "🎉 Integration test suite completed successfully!"