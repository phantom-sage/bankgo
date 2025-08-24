package services

import (
	"fmt"

	"github.com/phantom-sage/bankgo/internal/admin/config"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Container holds all admin services
type Container struct {
	config *config.Config
	db     *pgxpool.Pool
	redis  *redis.Client

	// Services
	AuthService         interfaces.AdminAuthService
	UserService         interfaces.UserManagementService
	SystemService       interfaces.SystemMonitoringService
	DatabaseService     interfaces.DatabaseService
	NotificationService interfaces.NotificationService
	AlertService        interfaces.AlertService
	TransactionService  interfaces.TransactionService
	AccountService      interfaces.AccountService
}

// NewContainer creates a new service container with all dependencies
func NewContainer(cfg *config.Config) (*Container, error) {
	container := &Container{
		config: cfg,
	}

	// Initialize database connection
	if err := container.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize Redis connection
	if err := container.initRedis(); err != nil {
		return nil, fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Initialize services
	if err := container.initServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	return container, nil
}

// initDatabase initializes the database connection pool
func (c *Container) initDatabase() error {
	// TODO: Initialize pgxpool connection
	// This will be implemented in a later task
	return nil
}

// initRedis initializes the Redis client
func (c *Container) initRedis() error {
	// TODO: Initialize Redis client
	// This will be implemented in a later task
	return nil
}

// initServices initializes all admin services
func (c *Container) initServices() error {
	var err error

	// Initialize admin authentication service
	c.AuthService, err = NewAdminAuthService(
		c.config.PasetoSecretKey,
		c.config.SessionTimeout,
		c.config.DefaultAdminUser,
		c.config.DefaultAdminPass,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize auth service: %w", err)
	}

	// Initialize user management service
	c.UserService = NewUserManagementService(c.db)

	// Initialize notification service
	c.NotificationService = NewNotificationService()
	
	// Initialize alert service
	c.AlertService = NewAlertService(c.db, c.NotificationService)

	// Initialize system monitoring service (depends on alert service)
	c.SystemService = NewSystemMonitoringService(c.db, c.redis, c.config.BankingAPIURL, c.AlertService)
	
	// Initialize database service
	c.DatabaseService = NewDatabaseService(c.db)
	
	// Initialize transaction service
	c.TransactionService = NewTransactionService(c.db)
	
	// Initialize account service
	c.AccountService = NewAccountService(c.db)

	return nil
}

// Close closes all connections and cleans up resources
func (c *Container) Close() error {
	var errors []error

	if c.db != nil {
		c.db.Close()
	}

	if c.redis != nil {
		if err := c.redis.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close Redis: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errors)
	}

	return nil
}

// GetDB returns the database connection pool
func (c *Container) GetDB() *pgxpool.Pool {
	return c.db
}

// GetRedis returns the Redis client
func (c *Container) GetRedis() *redis.Client {
	return c.redis
}

// GetConfig returns the configuration
func (c *Container) GetConfig() *config.Config {
	return c.config
}