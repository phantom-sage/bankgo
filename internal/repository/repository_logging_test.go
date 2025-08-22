package repository

import (
	"context"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/phantom-sage/bankgo/internal/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDB is a mock implementation of the database for testing
type MockDB struct {
	mock.Mock
}

// MockQueries is a mock implementation of queries for testing
type MockQueries struct {
	mock.Mock
}

func (m *MockQueries) CreateUser(ctx context.Context, arg queries.CreateUserParams) (queries.User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.User), args.Error(1)
}

func (m *MockQueries) GetUser(ctx context.Context, id int32) (queries.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.User), args.Error(1)
}

func TestRepository_LogDatabaseOperation(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository with mock database
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	ctx := context.Background()
	startTime := time.Now().Add(-100 * time.Millisecond) // Simulate 100ms operation
	
	tests := []struct {
		name         string
		operation    string
		table        string
		rowsAffected int64
		err          error
	}{
		{
			name:         "successful insert operation",
			operation:    "INSERT",
			table:        "users",
			rowsAffected: 1,
			err:          nil,
		},
		{
			name:         "successful select operation",
			operation:    "SELECT",
			table:        "accounts",
			rowsAffected: 5,
			err:          nil,
		},
		{
			name:         "failed update operation",
			operation:    "UPDATE",
			table:        "transfers",
			rowsAffected: 0,
			err:          assert.AnError,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies that LogDatabaseOperation doesn't panic
			// and properly handles different scenarios
			assert.NotPanics(t, func() {
				repo.LogDatabaseOperation(ctx, tt.operation, tt.table, startTime, tt.rowsAffected, tt.err)
			})
		})
	}
}

func TestRepository_LogDatabaseQuery(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository with mock database
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	ctx := context.Background()
	startTime := time.Now().Add(-50 * time.Millisecond) // Simulate 50ms query
	
	tests := []struct {
		name         string
		query        string
		rowsAffected int64
		params       []interface{}
		err          error
	}{
		{
			name:         "simple select query",
			query:        "SELECT * FROM users WHERE id = $1",
			rowsAffected: 1,
			params:       []interface{}{123},
			err:          nil,
		},
		{
			name:         "query with sensitive data",
			query:        "SELECT * FROM users WHERE password = $1",
			rowsAffected: 0,
			params:       []interface{}{"secret123"},
			err:          nil,
		},
		{
			name:         "failed query",
			query:        "SELECT * FROM nonexistent_table",
			rowsAffected: 0,
			params:       []interface{}{},
			err:          assert.AnError,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies that LogDatabaseQuery doesn't panic
			// and properly handles different scenarios including sensitive data
			assert.NotPanics(t, func() {
				repo.LogDatabaseQuery(ctx, tt.query, startTime, tt.rowsAffected, tt.err, tt.params...)
			})
		})
	}
}

func TestRepository_sanitizeQuery(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "normal query",
			query:    "SELECT * FROM users WHERE id = $1",
			expected: "SELECT * FROM users WHERE id = $1",
		},
		{
			name:     "query with password",
			query:    "SELECT * FROM users WHERE password = $1",
			expected: "[QUERY CONTAINS SENSITIVE DATA - REDACTED]",
		},
		{
			name:     "query with token",
			query:    "UPDATE users SET token = $1 WHERE id = $2",
			expected: "[QUERY CONTAINS SENSITIVE DATA - REDACTED]",
		},
		{
			name:     "very long query",
			query:    "SELECT " + string(make([]byte, 600)) + " FROM users",
			expected: "SELECT " + string(make([]byte, 500-7)) + "... [TRUNCATED]",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.sanitizeQuery(tt.query)
			if tt.name == "very long query" {
				// For long query test, just check that it's truncated
				assert.Contains(t, result, "... [TRUNCATED]")
				assert.True(t, len(result) <= 520) // 500 + "... [TRUNCATED]"
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestRepository_extractQueryType(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "select query",
			query:    "SELECT * FROM users",
			expected: "select",
		},
		{
			name:     "insert query",
			query:    "INSERT INTO users (name) VALUES ($1)",
			expected: "insert",
		},
		{
			name:     "update query",
			query:    "UPDATE users SET name = $1 WHERE id = $2",
			expected: "update",
		},
		{
			name:     "delete query",
			query:    "DELETE FROM users WHERE id = $1",
			expected: "delete",
		},
		{
			name:     "begin transaction",
			query:    "BEGIN",
			expected: "transaction",
		},
		{
			name:     "commit transaction",
			query:    "COMMIT",
			expected: "commit",
		},
		{
			name:     "rollback transaction",
			query:    "ROLLBACK",
			expected: "rollback",
		},
		{
			name:     "empty query",
			query:    "",
			expected: "unknown",
		},
		{
			name:     "unknown query type",
			query:    "EXPLAIN SELECT * FROM users",
			expected: "other",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.extractQueryType(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransactionLogger_generateTransactionID(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	// Create transaction logger
	txLogger := NewTransactionLogger(repo)
	
	// Generate multiple transaction IDs
	id1 := txLogger.generateTransactionID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	id2 := txLogger.generateTransactionID()
	
	// Verify IDs are different and have expected format
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "tx_")
	assert.Contains(t, id2, "tx_")
}

func TestTransactionLogger_isDeadlockError(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	// Create transaction logger
	txLogger := NewTransactionLogger(repo)
	
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      assert.AnError,
			expected: false,
		},
		// Note: Testing actual pgx.PgError would require more complex setup
		// This test verifies the function doesn't panic with different error types
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := txLogger.isDeadlockError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransactionLogger_LogOperation(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	// Create transaction logger
	txLogger := NewTransactionLogger(repo)
	
	ctx := context.Background()
	txCtx := &TransactionContext{
		TransactionID:  "test_tx_123",
		StartTime:      time.Now(),
		OperationCount: 0,
		Operations:     make([]TransactionOperation, 0),
	}
	
	startTime := time.Now().Add(-10 * time.Millisecond)
	
	// Test successful operation
	txLogger.LogOperation(ctx, txCtx, "INSERT", "users", startTime, 1, nil)
	
	// Verify operation was recorded
	assert.Equal(t, 1, txCtx.OperationCount)
	assert.Len(t, txCtx.Operations, 1)
	assert.Equal(t, "INSERT", txCtx.Operations[0].Operation)
	assert.Equal(t, "users", txCtx.Operations[0].Table)
	assert.Equal(t, int64(1), txCtx.Operations[0].RowsAffected)
	assert.Nil(t, txCtx.Operations[0].Error)
	
	// Test failed operation
	txLogger.LogOperation(ctx, txCtx, "UPDATE", "accounts", startTime, 0, assert.AnError)
	
	// Verify second operation was recorded
	assert.Equal(t, 2, txCtx.OperationCount)
	assert.Len(t, txCtx.Operations, 2)
	assert.Equal(t, "UPDATE", txCtx.Operations[1].Operation)
	assert.Equal(t, "accounts", txCtx.Operations[1].Table)
	assert.Equal(t, int64(0), txCtx.Operations[1].RowsAffected)
	assert.NotNil(t, txCtx.Operations[1].Error)
}

func TestNewTransactionLogger(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	// Create transaction logger
	txLogger := NewTransactionLogger(repo)
	
	// Verify transaction logger was created properly
	require.NotNil(t, txLogger)
	assert.Equal(t, repo, txLogger.repo)
	assert.NotNil(t, txLogger.logger)
	assert.NotNil(t, txLogger.ctxLogger)
	assert.NotNil(t, txLogger.perfLogger)
}

// Integration test that would require actual database connection
// This is a placeholder for integration tests that would be run with a test database
func TestRepository_Integration_DatabaseLogging(t *testing.T) {
	t.Skip("Integration test - requires database connection")
	
	// This test would:
	// 1. Set up a test database connection
	// 2. Create a repository with logging
	// 3. Perform actual database operations
	// 4. Verify that logs are generated correctly
	// 5. Test transaction logging with real transactions
	// 6. Test connection pool monitoring
	// 7. Test deadlock detection and retry logic
}

// Benchmark test for logging overhead
func BenchmarkRepository_LogDatabaseOperation(b *testing.B) {
	// Create a test logger with no output to measure pure logging overhead
	logger := zerolog.New(zerolog.NewTestWriter(b)).Level(zerolog.Disabled)
	
	// Create repository
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	ctx := context.Background()
	startTime := time.Now()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.LogDatabaseOperation(ctx, "SELECT", "users", startTime, 1, nil)
	}
}

func BenchmarkRepository_LogDatabaseQuery(b *testing.B) {
	// Create a test logger with no output to measure pure logging overhead
	logger := zerolog.New(zerolog.NewTestWriter(b)).Level(zerolog.Disabled)
	
	// Create repository
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	ctx := context.Background()
	startTime := time.Now()
	query := "SELECT * FROM users WHERE id = $1"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.LogDatabaseQuery(ctx, query, startTime, 1, nil, 123)
	}
}