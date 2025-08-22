package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/phantom-sage/bankgo/internal/database"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/phantom-sage/bankgo/internal/logging"
	"github.com/rs/zerolog"
)

// Repository provides access to all database operations
type Repository struct {
	*queries.Queries
	db              *database.DB
	logger          zerolog.Logger
	contextLogger   *logging.ContextLogger
	perfLogger      *logging.PerformanceLogger
}

// New creates a new repository instance
func New(db *database.DB, logger zerolog.Logger) *Repository {
	repo := &Repository{
		db:            db,
		logger:        logger.With().Str("component", "repository").Logger(),
		contextLogger: logging.NewContextLoggerFromLogger(logger.With().Str("component", "repository").Logger()),
		perfLogger:    logging.NewPerformanceLogger(logger.With().Str("component", "repository").Logger()),
	}
	
	if db != nil && db.Pool != nil {
		repo.Queries = queries.New(db.Pool)
	}
	
	return repo
}

// WithTx executes a function within a database transaction with logging
func (r *Repository) WithTx(ctx context.Context, fn func(*queries.Queries) error) error {
	startTime := time.Now()
	operationCount := 0
	
	// Get context logger with request context
	ctxLogger := logging.NewContextLogger(r.logger, ctx)
	
	ctxLogger.Debug().Msg("Beginning database transaction")
	
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		duration := time.Since(startTime)
		ctxLogger.Error().
			Err(err).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("Failed to begin database transaction")
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	// Ensure rollback on panic or error
	defer func() {
		if p := recover(); p != nil {
			duration := time.Since(startTime)
			ctxLogger.Error().
				Interface("panic", p).
				Int64("duration_ms", duration.Milliseconds()).
				Int("operation_count", operationCount).
				Msg("Database transaction panicked, rolling back")
			tx.Rollback(ctx)
			panic(p)
		}
	}()

	qtx := r.Queries.WithTx(tx)
	
	// Execute the function with operation counting
	if err := fn(qtx); err != nil {
		duration := time.Since(startTime)
		rollbackErr := tx.Rollback(ctx)
		
		ctxLogger.Error().
			Err(err).
			Int64("duration_ms", duration.Milliseconds()).
			Int("operation_count", operationCount).
			Msg("Database transaction failed, rolling back")
		
		if rollbackErr != nil {
			ctxLogger.Error().
				Err(rollbackErr).
				Msg("Failed to rollback transaction")
		}
		
		// Log performance metrics for failed transaction
		r.perfLogger.LogDatabaseTransaction(duration, operationCount, false)
		
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		duration := time.Since(startTime)
		ctxLogger.Error().
			Err(err).
			Int64("duration_ms", duration.Milliseconds()).
			Int("operation_count", operationCount).
			Msg("Failed to commit database transaction")
		
		// Log performance metrics for failed commit
		r.perfLogger.LogDatabaseTransaction(duration, operationCount, false)
		
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	duration := time.Since(startTime)
	ctxLogger.Debug().
		Int64("duration_ms", duration.Milliseconds()).
		Int("operation_count", operationCount).
		Msg("Database transaction completed successfully")
	
	// Log performance metrics for successful transaction
	r.perfLogger.LogDatabaseTransaction(duration, operationCount, true)
	
	return nil
}

// GetDB returns the underlying database connection
func (r *Repository) GetDB() *database.DB {
	return r.db
}

// GetLogger returns the repository logger
func (r *Repository) GetLogger() zerolog.Logger {
	return r.logger
}

// GetContextLogger returns the context logger
func (r *Repository) GetContextLogger() *logging.ContextLogger {
	return r.contextLogger
}

// GetPerformanceLogger returns the performance logger
func (r *Repository) GetPerformanceLogger() *logging.PerformanceLogger {
	return r.perfLogger
}

// LogDatabaseOperation logs a database operation with performance metrics
func (r *Repository) LogDatabaseOperation(ctx context.Context, operation, table string, startTime time.Time, rowsAffected int64, err error) {
	duration := time.Since(startTime)
	ctxLogger := logging.NewContextLogger(r.logger, ctx)
	
	if err != nil {
		ctxLogger.Error().
			Err(err).
			Str("operation", operation).
			Str("table", table).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("Database operation failed")
	} else {
		ctxLogger.Debug().
			Str("operation", operation).
			Str("table", table).
			Int64("duration_ms", duration.Milliseconds()).
			Int64("rows_affected", rowsAffected).
			Msg("Database operation completed")
	}
	
	// Log performance metrics
	r.perfLogger.LogDatabaseQuery(fmt.Sprintf("%s %s", operation, table), duration, rowsAffected)
}

// LogDatabaseQuery logs a database query with sanitized parameters
func (r *Repository) LogDatabaseQuery(ctx context.Context, query string, startTime time.Time, rowsAffected int64, err error, params ...interface{}) {
	duration := time.Since(startTime)
	ctxLogger := logging.NewContextLogger(r.logger, ctx)
	
	// Sanitize query for logging (remove sensitive data)
	sanitizedQuery := r.sanitizeQuery(query)
	queryType := r.extractQueryType(query)
	
	if err != nil {
		ctxLogger.Error().
			Err(err).
			Str("query_type", queryType).
			Str("query", sanitizedQuery).
			Int64("duration_ms", duration.Milliseconds()).
			Int("param_count", len(params)).
			Msg("Database query failed")
	} else {
		ctxLogger.Debug().
			Str("query_type", queryType).
			Str("query", sanitizedQuery).
			Int64("duration_ms", duration.Milliseconds()).
			Int64("rows_affected", rowsAffected).
			Int("param_count", len(params)).
			Msg("Database query completed")
	}
	
	// Log performance metrics
	r.perfLogger.LogDatabaseQuery(sanitizedQuery, duration, rowsAffected)
}

// LogConnectionPoolMetrics logs connection pool health and metrics
func (r *Repository) LogConnectionPoolMetrics(ctx context.Context) {
	if r.db == nil || r.db.Pool == nil {
		return
	}
	
	stats := r.db.Stats()
	ctxLogger := logging.NewContextLogger(r.logger, ctx)
	
	ctxLogger.Debug().
		Int32("acquired_conns", stats.AcquiredConns()).
		Int32("constructing_conns", stats.ConstructingConns()).
		Int32("idle_conns", stats.IdleConns()).
		Int32("max_conns", stats.MaxConns()).
		Int32("total_conns", stats.TotalConns()).
		Int64("acquire_count", stats.AcquireCount()).
		Int64("acquire_duration_ns", stats.AcquireDuration().Nanoseconds()).
		Int64("canceled_acquire_count", stats.CanceledAcquireCount()).
		Int64("empty_acquire_count", stats.EmptyAcquireCount()).
		Msg("Database connection pool metrics")
	
	// Log performance metrics for connection pool
	r.perfLogger.LogConnectionPoolMetrics(
		int(stats.AcquiredConns()),
		int(stats.IdleConns()),
		int(stats.MaxConns()),
		stats.AcquireDuration(),
	)
}

// HealthCheck performs a health check on the database connection with logging
func (r *Repository) HealthCheck(ctx context.Context) error {
	startTime := time.Now()
	ctxLogger := logging.NewContextLogger(r.logger, ctx)
	
	ctxLogger.Debug().Msg("Starting database health check")
	
	err := r.db.HealthCheck(ctx)
	duration := time.Since(startTime)
	
	if err != nil {
		ctxLogger.Error().
			Err(err).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("Database health check failed")
		return err
	}
	
	ctxLogger.Debug().
		Int64("duration_ms", duration.Milliseconds()).
		Msg("Database health check completed successfully")
	
	// Log connection pool metrics as part of health check
	r.LogConnectionPoolMetrics(ctx)
	
	return nil
}

// sanitizeQuery removes sensitive information from SQL queries for logging
func (r *Repository) sanitizeQuery(query string) string {
	// Remove potential sensitive data patterns
	sensitivePatterns := []string{
		"password",
		"token",
		"secret",
		"key",
		"hash",
	}
	
	lowerQuery := strings.ToLower(query)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerQuery, pattern) {
			// Replace the query with a generic message if it contains sensitive data
			return "[QUERY CONTAINS SENSITIVE DATA - REDACTED]"
		}
	}
	
	// Limit query length for logging
	if len(query) > 500 {
		return query[:500] + "... [TRUNCATED]"
	}
	
	return query
}

// extractQueryType extracts the operation type from a SQL query
func (r *Repository) extractQueryType(query string) string {
	if len(query) == 0 {
		return "unknown"
	}
	
	query = strings.ToUpper(strings.TrimSpace(query))
	words := strings.Fields(query)
	if len(words) == 0 {
		return "unknown"
	}
	
	switch words[0] {
	case "SELECT":
		return "select"
	case "INSERT":
		return "insert"
	case "UPDATE":
		return "update"
	case "DELETE":
		return "delete"
	case "CREATE":
		return "create"
	case "DROP":
		return "drop"
	case "ALTER":
		return "alter"
	case "BEGIN", "START":
		return "transaction"
	case "COMMIT":
		return "commit"
	case "ROLLBACK":
		return "rollback"
	default:
		return "other"
	}
}

// Repositories holds all repository instances
type Repositories struct {
	AccountRepo  AccountRepository
	TransferRepo TransferRepository
	UserRepo     UserRepository
}

// NewRepositories creates a new repositories instance with all repository implementations
func NewRepositories(repo *Repository) *Repositories {
	return &Repositories{
		AccountRepo:  NewAccountRepository(repo),
		TransferRepo: NewTransferRepository(repo),
		UserRepo:     NewUserRepository(repo),
	}
}

// WithContext returns a new repository instance with context-aware logging
func (r *Repository) WithContext(ctx context.Context) *Repository {
	return &Repository{
		Queries:       r.Queries,
		db:            r.db,
		logger:        r.logger,
		contextLogger: logging.NewContextLogger(r.logger, ctx),
		perfLogger:    r.perfLogger,
	}
}