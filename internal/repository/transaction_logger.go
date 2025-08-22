package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/phantom-sage/bankgo/internal/logging"
	"github.com/rs/zerolog"
)

// TransactionLogger wraps database transactions with comprehensive logging
type TransactionLogger struct {
	repo      *Repository
	logger    zerolog.Logger
	ctxLogger *logging.ContextLogger
	perfLogger *logging.PerformanceLogger
}

// NewTransactionLogger creates a new transaction logger
func NewTransactionLogger(repo *Repository) *TransactionLogger {
	return &TransactionLogger{
		repo:       repo,
		logger:     repo.logger.With().Str("component", "transaction").Logger(),
		ctxLogger:  logging.NewContextLoggerFromLogger(repo.logger.With().Str("component", "transaction").Logger()),
		perfLogger: logging.NewPerformanceLogger(repo.logger.With().Str("component", "transaction").Logger()),
	}
}

// TransactionContext holds transaction execution context and metrics
type TransactionContext struct {
	TransactionID   string
	StartTime       time.Time
	OperationCount  int
	Operations      []TransactionOperation
	DeadlockRetries int
	MaxRetries      int
}

// TransactionOperation represents a single operation within a transaction
type TransactionOperation struct {
	Operation   string
	Table       string
	StartTime   time.Time
	Duration    time.Duration
	RowsAffected int64
	Error       error
}

// WithTxLogged executes a function within a database transaction with comprehensive logging
func (tl *TransactionLogger) WithTxLogged(ctx context.Context, fn func(*queries.Queries, *TransactionContext) error) error {
	return tl.WithTxLoggedWithRetry(ctx, fn, 3) // Default 3 retries for deadlocks
}

// WithTxLoggedWithRetry executes a function within a database transaction with deadlock retry logic
func (tl *TransactionLogger) WithTxLoggedWithRetry(ctx context.Context, fn func(*queries.Queries, *TransactionContext) error, maxRetries int) error {
	txCtx := &TransactionContext{
		TransactionID:   tl.generateTransactionID(),
		StartTime:       time.Now(),
		OperationCount:  0,
		Operations:      make([]TransactionOperation, 0),
		DeadlockRetries: 0,
		MaxRetries:      maxRetries,
	}

	ctxLogger := logging.NewContextLogger(tl.logger, ctx).
		WithField("transaction_id", txCtx.TransactionID)

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			txCtx.DeadlockRetries = attempt
			ctxLogger.Warn().
				Int("attempt", attempt).
				Int("max_retries", maxRetries).
				Msg("Retrying transaction due to deadlock")
			
			// Exponential backoff for retries
			backoffDuration := time.Duration(attempt*attempt) * 100 * time.Millisecond
			time.Sleep(backoffDuration)
		}

		err := tl.executeTransaction(ctx, ctxLogger, txCtx, fn)
		
		if err != nil {
			// Check if this is a deadlock error that should be retried
			if tl.isDeadlockError(err) && attempt < maxRetries {
				ctxLogger.Warn().
					Err(err).
					Int("attempt", attempt).
					Msg("Transaction deadlock detected, will retry")
				continue
			}
			
			// Log final failure
			duration := time.Since(txCtx.StartTime)
			ctxLogger.Error().
				Err(err).
				Int64("total_duration_ms", duration.Milliseconds()).
				Int("operation_count", txCtx.OperationCount).
				Int("deadlock_retries", txCtx.DeadlockRetries).
				Msg("Transaction failed after all retries")
			
			// Log performance metrics for failed transaction
			tl.perfLogger.LogDatabaseTransaction(duration, txCtx.OperationCount, false)
			
			return err
		}

		// Transaction succeeded
		duration := time.Since(txCtx.StartTime)
		ctxLogger.Info().
			Int64("total_duration_ms", duration.Milliseconds()).
			Int("operation_count", txCtx.OperationCount).
			Int("deadlock_retries", txCtx.DeadlockRetries).
			Msg("Transaction completed successfully")
		
		// Log performance metrics for successful transaction
		tl.perfLogger.LogDatabaseTransaction(duration, txCtx.OperationCount, true)
		
		// Log detailed operation breakdown if debug logging is enabled
		tl.logOperationBreakdown(ctxLogger, txCtx)
		
		return nil
	}

	return fmt.Errorf("transaction failed after %d retries", maxRetries)
}

// executeTransaction executes a single transaction attempt
func (tl *TransactionLogger) executeTransaction(ctx context.Context, ctxLogger *logging.ContextLogger, txCtx *TransactionContext, fn func(*queries.Queries, *TransactionContext) error) error {
	ctxLogger.Debug().Msg("Beginning database transaction")
	
	// Check if database connection is available
	if tl.repo.db == nil || tl.repo.db.Pool == nil {
		return fmt.Errorf("failed to begin transaction: database connection not available")
	}
	
	tx, err := tl.repo.db.Pool.Begin(ctx)
	if err != nil {
		ctxLogger.Error().
			Err(err).
			Msg("Failed to begin database transaction")
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure rollback on panic or error
	defer func() {
		if p := recover(); p != nil {
			duration := time.Since(txCtx.StartTime)
			ctxLogger.Error().
				Interface("panic", p).
				Int64("duration_ms", duration.Milliseconds()).
				Int("operation_count", txCtx.OperationCount).
				Msg("Transaction panicked, rolling back")
			
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				ctxLogger.Error().
					Err(rollbackErr).
					Msg("Failed to rollback transaction after panic")
			}
			panic(p)
		}
	}()

	qtx := tl.repo.Queries.WithTx(tx)
	
	// Reset operation count for this attempt
	txCtx.OperationCount = 0
	txCtx.Operations = make([]TransactionOperation, 0)
	
	// Execute the function
	if err := fn(qtx, txCtx); err != nil {
		rollbackStart := time.Now()
		rollbackErr := tx.Rollback(ctx)
		rollbackDuration := time.Since(rollbackStart)
		
		ctxLogger.Error().
			Err(err).
			Int64("rollback_duration_ms", rollbackDuration.Milliseconds()).
			Int("operation_count", txCtx.OperationCount).
			Msg("Transaction failed, rolling back")
		
		if rollbackErr != nil {
			ctxLogger.Error().
				Err(rollbackErr).
				Msg("Failed to rollback transaction")
		}
		
		return err
	}

	// Commit the transaction
	commitStart := time.Now()
	if err := tx.Commit(ctx); err != nil {
		commitDuration := time.Since(commitStart)
		ctxLogger.Error().
			Err(err).
			Int64("commit_duration_ms", commitDuration.Milliseconds()).
			Int("operation_count", txCtx.OperationCount).
			Msg("Failed to commit database transaction")
		
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	commitDuration := time.Since(commitStart)
	ctxLogger.Debug().
		Int64("commit_duration_ms", commitDuration.Milliseconds()).
		Msg("Transaction committed successfully")
	
	return nil
}

// LogOperation logs a database operation within a transaction
func (tl *TransactionLogger) LogOperation(ctx context.Context, txCtx *TransactionContext, operation, table string, startTime time.Time, rowsAffected int64, err error) {
	duration := time.Since(startTime)
	
	// Add operation to transaction context
	txOp := TransactionOperation{
		Operation:    operation,
		Table:        table,
		StartTime:    startTime,
		Duration:     duration,
		RowsAffected: rowsAffected,
		Error:        err,
	}
	txCtx.Operations = append(txCtx.Operations, txOp)
	txCtx.OperationCount++
	
	ctxLogger := logging.NewContextLogger(tl.logger, ctx).
		WithField("transaction_id", txCtx.TransactionID).
		WithField("operation_sequence", txCtx.OperationCount)
	
	if err != nil {
		ctxLogger.Error().
			Err(err).
			Str("operation", operation).
			Str("table", table).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("Transaction operation failed")
	} else {
		ctxLogger.Debug().
			Str("operation", operation).
			Str("table", table).
			Int64("duration_ms", duration.Milliseconds()).
			Int64("rows_affected", rowsAffected).
			Msg("Transaction operation completed")
	}
}

// logOperationBreakdown logs detailed breakdown of all operations in the transaction
func (tl *TransactionLogger) logOperationBreakdown(ctxLogger *logging.ContextLogger, txCtx *TransactionContext) {
	if zerolog.GlobalLevel() > zerolog.DebugLevel {
		return // Skip if debug logging is not enabled
	}
	
	totalDuration := time.Since(txCtx.StartTime)
	
	ctxLogger.Debug().
		Int("total_operations", len(txCtx.Operations)).
		Int64("total_duration_ms", totalDuration.Milliseconds()).
		Msg("Transaction operation breakdown")
	
	for i, op := range txCtx.Operations {
		ctxLogger.Debug().
			Int("sequence", i+1).
			Str("operation", op.Operation).
			Str("table", op.Table).
			Int64("duration_ms", op.Duration.Milliseconds()).
			Int64("rows_affected", op.RowsAffected).
			Bool("success", op.Error == nil).
			Msg("Transaction operation detail")
	}
}

// isDeadlockError checks if an error is a deadlock that should be retried
func (tl *TransactionLogger) isDeadlockError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for PostgreSQL deadlock error codes
	// Note: This would need to be implemented with proper pgx error handling
	// For now, we'll check the error message as a fallback
	errMsg := err.Error()
	return strings.Contains(errMsg, "deadlock") || strings.Contains(errMsg, "40P01")
	
	return false
}

// generateTransactionID generates a unique transaction ID for logging
func (tl *TransactionLogger) generateTransactionID() string {
	return fmt.Sprintf("tx_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

// GetTransactionMetrics returns current transaction performance metrics
func (tl *TransactionLogger) GetTransactionMetrics(ctx context.Context) map[string]interface{} {
	// This could be extended to track transaction metrics over time
	return map[string]interface{}{
		"component": "transaction_logger",
		"timestamp": time.Now(),
	}
}

// HealthCheck performs a health check on the transaction logging system
func (tl *TransactionLogger) HealthCheck(ctx context.Context) error {
	// Check if database connection is available
	if tl.repo.db == nil || tl.repo.db.Pool == nil {
		return fmt.Errorf("transaction health check failed: database connection not available")
	}
	
	// Test that we can create a simple transaction
	return tl.WithTxLogged(ctx, func(qtx *queries.Queries, txCtx *TransactionContext) error {
		// Simple health check query
		startTime := time.Now()
		var result int32
		err := tl.repo.db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)
		
		// Log the operation
		tl.LogOperation(ctx, txCtx, "SELECT", "health_check", startTime, 1, err)
		
		if err != nil {
			return fmt.Errorf("transaction health check query failed: %w", err)
		}
		
		if result != 1 {
			return fmt.Errorf("unexpected health check result: got %d, expected 1", result)
		}
		
		return nil
	})
}