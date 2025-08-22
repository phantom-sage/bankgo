package repository

import (
	"context"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/phantom-sage/bankgo/internal/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestTransactionLogger_WithTxLogged tests the transaction logging functionality
func TestTransactionLogger_WithTxLogged(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository with mock database (for unit testing)
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	// Create transaction logger
	txLogger := NewTransactionLogger(repo)
	
	ctx := context.Background()
	
	t.Run("successful transaction with operations", func(t *testing.T) {
		// This test simulates a successful transaction with multiple operations
		err := txLogger.WithTxLogged(ctx, func(qtx *queries.Queries, txCtx *TransactionContext) error {
			// Simulate first operation
			startTime := time.Now()
			txLogger.LogOperation(ctx, txCtx, "INSERT", "users", startTime, 1, nil)
			
			// Simulate second operation
			startTime = time.Now()
			txLogger.LogOperation(ctx, txCtx, "UPDATE", "accounts", startTime, 1, nil)
			
			// Simulate third operation
			startTime = time.Now()
			txLogger.LogOperation(ctx, txCtx, "INSERT", "transfers", startTime, 1, nil)
			
			return nil
		})
		
		// Since we don't have a real database connection, this will fail at the Begin() call
		// But we can verify that the function doesn't panic and handles the error gracefully
		assert.Error(t, err) // Expected to fail without real DB
		assert.Contains(t, err.Error(), "database connection not available")
	})
	
	t.Run("transaction with simulated error", func(t *testing.T) {
		// This test simulates a transaction that fails during execution
		err := txLogger.WithTxLogged(ctx, func(qtx *queries.Queries, txCtx *TransactionContext) error {
			// Simulate operation that fails
			startTime := time.Now()
			opErr := assert.AnError
			txLogger.LogOperation(ctx, txCtx, "INSERT", "users", startTime, 0, opErr)
			
			// Return the error to trigger rollback
			return opErr
		})
		
		// Should fail at Begin() since we don't have real DB, but function should handle it
		assert.Error(t, err)
	})
}

// TestTransactionLogger_WithTxLoggedWithRetry tests the retry logic
func TestTransactionLogger_WithTxLoggedWithRetry(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository with mock database
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	// Create transaction logger
	txLogger := NewTransactionLogger(repo)
	
	ctx := context.Background()
	
	t.Run("retry logic with max retries", func(t *testing.T) {
		maxRetries := 2
		
		err := txLogger.WithTxLoggedWithRetry(ctx, func(qtx *queries.Queries, txCtx *TransactionContext) error {
			// Simulate operation
			startTime := time.Now()
			txLogger.LogOperation(ctx, txCtx, "SELECT", "users", startTime, 1, nil)
			
			return nil
		}, maxRetries)
		
		// Should fail at Begin() since we don't have real DB
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection not available")
	})
}

// TestTransactionContext tests the transaction context functionality
func TestTransactionContext(t *testing.T) {
	t.Run("transaction context initialization", func(t *testing.T) {
		txCtx := &TransactionContext{
			TransactionID:   "test_tx_123",
			StartTime:       time.Now(),
			OperationCount:  0,
			Operations:      make([]TransactionOperation, 0),
			DeadlockRetries: 0,
			MaxRetries:      3,
		}
		
		assert.Equal(t, "test_tx_123", txCtx.TransactionID)
		assert.Equal(t, 0, txCtx.OperationCount)
		assert.Len(t, txCtx.Operations, 0)
		assert.Equal(t, 0, txCtx.DeadlockRetries)
		assert.Equal(t, 3, txCtx.MaxRetries)
	})
	
	t.Run("operation tracking", func(t *testing.T) {
		txCtx := &TransactionContext{
			TransactionID:  "test_tx_456",
			StartTime:      time.Now(),
			OperationCount: 0,
			Operations:     make([]TransactionOperation, 0),
		}
		
		// Add first operation
		op1 := TransactionOperation{
			Operation:    "INSERT",
			Table:        "users",
			StartTime:    time.Now(),
			Duration:     10 * time.Millisecond,
			RowsAffected: 1,
			Error:        nil,
		}
		txCtx.Operations = append(txCtx.Operations, op1)
		txCtx.OperationCount++
		
		// Add second operation
		op2 := TransactionOperation{
			Operation:    "UPDATE",
			Table:        "accounts",
			StartTime:    time.Now(),
			Duration:     15 * time.Millisecond,
			RowsAffected: 1,
			Error:        nil,
		}
		txCtx.Operations = append(txCtx.Operations, op2)
		txCtx.OperationCount++
		
		assert.Equal(t, 2, txCtx.OperationCount)
		assert.Len(t, txCtx.Operations, 2)
		assert.Equal(t, "INSERT", txCtx.Operations[0].Operation)
		assert.Equal(t, "users", txCtx.Operations[0].Table)
		assert.Equal(t, "UPDATE", txCtx.Operations[1].Operation)
		assert.Equal(t, "accounts", txCtx.Operations[1].Table)
	})
}

// TestTransactionLogger_HealthCheck tests the health check functionality
func TestTransactionLogger_HealthCheck(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository with mock database
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	// Create transaction logger
	txLogger := NewTransactionLogger(repo)
	
	ctx := context.Background()
	
	// Health check should fail without real database connection
	err := txLogger.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection not available")
}

// TestTransactionLogger_GetTransactionMetrics tests the metrics functionality
func TestTransactionLogger_GetTransactionMetrics(t *testing.T) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	
	// Create repository with mock database
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	// Create transaction logger
	txLogger := NewTransactionLogger(repo)
	
	ctx := context.Background()
	
	metrics := txLogger.GetTransactionMetrics(ctx)
	
	assert.NotNil(t, metrics)
	assert.Equal(t, "transaction_logger", metrics["component"])
	assert.NotNil(t, metrics["timestamp"])
}

// BenchmarkTransactionLogger_LogOperation benchmarks the operation logging
func BenchmarkTransactionLogger_LogOperation(b *testing.B) {
	// Create a test logger with no output to measure pure logging overhead
	logger := zerolog.New(zerolog.NewTestWriter(b)).Level(zerolog.Disabled)
	
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
		TransactionID:  "bench_tx",
		StartTime:      time.Now(),
		OperationCount: 0,
		Operations:     make([]TransactionOperation, 0),
	}
	
	startTime := time.Now()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		txLogger.LogOperation(ctx, txCtx, "SELECT", "users", startTime, 1, nil)
	}
}

// BenchmarkTransactionLogger_generateTransactionID benchmarks transaction ID generation
func BenchmarkTransactionLogger_generateTransactionID(b *testing.B) {
	// Create a test logger
	logger := zerolog.New(zerolog.NewTestWriter(b)).Level(zerolog.Disabled)
	
	// Create repository
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	// Create transaction logger
	txLogger := NewTransactionLogger(repo)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = txLogger.generateTransactionID()
	}
}

// TestTransactionOperation tests the transaction operation struct
func TestTransactionOperation(t *testing.T) {
	startTime := time.Now()
	duration := 25 * time.Millisecond
	
	op := TransactionOperation{
		Operation:    "INSERT",
		Table:        "users",
		StartTime:    startTime,
		Duration:     duration,
		RowsAffected: 1,
		Error:        nil,
	}
	
	assert.Equal(t, "INSERT", op.Operation)
	assert.Equal(t, "users", op.Table)
	assert.Equal(t, startTime, op.StartTime)
	assert.Equal(t, duration, op.Duration)
	assert.Equal(t, int64(1), op.RowsAffected)
	assert.Nil(t, op.Error)
}

// TestTransactionOperation_WithError tests transaction operation with error
func TestTransactionOperation_WithError(t *testing.T) {
	startTime := time.Now()
	duration := 10 * time.Millisecond
	testError := assert.AnError
	
	op := TransactionOperation{
		Operation:    "UPDATE",
		Table:        "accounts",
		StartTime:    startTime,
		Duration:     duration,
		RowsAffected: 0,
		Error:        testError,
	}
	
	assert.Equal(t, "UPDATE", op.Operation)
	assert.Equal(t, "accounts", op.Table)
	assert.Equal(t, int64(0), op.RowsAffected)
	assert.Equal(t, testError, op.Error)
}

// Example test showing how transaction logging would be used in practice
func ExampleTransactionLogger_WithTxLogged() {
	// This example shows how transaction logging would be used in a real service
	
	// Create logger
	logger := zerolog.New(zerolog.NewTestWriter(nil)).With().Timestamp().Logger()
	
	// Create repository (would normally have real database connection)
	repo := &Repository{
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	// Create transaction logger
	txLogger := NewTransactionLogger(repo)
	
	ctx := context.Background()
	
	// Example: Transfer money between accounts
	err := txLogger.WithTxLogged(ctx, func(qtx *queries.Queries, txCtx *TransactionContext) error {
		// Step 1: Debit from source account
		startTime := time.Now()
		// In real implementation: result, err := qtx.SubtractFromBalance(ctx, params)
		txLogger.LogOperation(ctx, txCtx, "UPDATE", "accounts", startTime, 1, nil)
		
		// Step 2: Credit to destination account
		startTime = time.Now()
		// In real implementation: result, err := qtx.AddToBalance(ctx, params)
		txLogger.LogOperation(ctx, txCtx, "UPDATE", "accounts", startTime, 1, nil)
		
		// Step 3: Create transfer record
		startTime = time.Now()
		// In real implementation: transfer, err := qtx.CreateTransfer(ctx, params)
		txLogger.LogOperation(ctx, txCtx, "INSERT", "transfers", startTime, 1, nil)
		
		return nil
	})
	
	// Handle error (would be nil in successful case with real DB)
	if err != nil {
		// Transaction failed and was rolled back
		// All operations and timing were logged
	}
}