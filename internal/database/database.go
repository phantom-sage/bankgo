package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/phantom-sage/bankgo/internal/config"
)

// DB wraps the database connection pool
type DB struct {
	Pool   *pgxpool.Pool
	SqlDB  *sql.DB
	config config.DatabaseConfig
}

// New creates a new database connection with connection pooling
func New(cfg config.DatabaseConfig) (*DB, error) {
	// Create pgxpool config
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool settings
	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Create sql.DB for compatibility with other libraries
	sqlDB := stdlib.OpenDBFromPool(pool)

	// Configure sql.DB connection pool settings
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	db := &DB{
		Pool:   pool,
		SqlDB:  sqlDB,
		config: cfg,
	}

	// Test the connection
	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Ping tests the database connection
func (db *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// HealthCheck performs a comprehensive health check of the database
func (db *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Test basic connectivity
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("connectivity check failed: %w", err)
	}

	// Test query execution
	var result int
	err := db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("query execution check failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected query result: got %d, expected 1", result)
	}

	return nil
}

// Stats returns database connection pool statistics
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.SqlDB != nil {
		db.SqlDB.Close()
	}
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// BeginTx starts a new database transaction
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (pgx.Tx, error) {
	txOpts := pgx.TxOptions{}
	if opts != nil {
		// Convert sql.IsolationLevel to pgx.TxIsoLevel
		switch opts.Isolation {
		case sql.LevelDefault:
			txOpts.IsoLevel = pgx.TxIsoLevel("")
		case sql.LevelReadUncommitted:
			txOpts.IsoLevel = pgx.ReadUncommitted
		case sql.LevelReadCommitted:
			txOpts.IsoLevel = pgx.ReadCommitted
		case sql.LevelRepeatableRead:
			txOpts.IsoLevel = pgx.RepeatableRead
		case sql.LevelSerializable:
			txOpts.IsoLevel = pgx.Serializable
		default:
			txOpts.IsoLevel = pgx.ReadCommitted
		}
		
		if opts.ReadOnly {
			txOpts.AccessMode = pgx.ReadOnly
		} else {
			txOpts.AccessMode = pgx.ReadWrite
		}
	}
	return db.Pool.BeginTx(ctx, txOpts)
}

// WithTx executes a function within a database transaction
func (db *DB) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}