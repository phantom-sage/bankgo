package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the admin API server
type Config struct {
	// Server configuration
	Port        int    `json:"port"`
	Environment string `json:"environment"`

	// Database configuration
	DatabaseURL string `json:"database_url"`

	// Redis configuration
	RedisURL      string `json:"redis_url"`
	RedisPassword string `json:"redis_password"`

	// Authentication configuration
	PasetoSecretKey   string        `json:"-"` // Hidden from JSON
	SessionTimeout    time.Duration `json:"session_timeout"`
	DefaultAdminUser  string        `json:"default_admin_user"`
	DefaultAdminPass  string        `json:"-"` // Hidden from JSON

	// Banking API configuration
	BankingAPIURL string `json:"banking_api_url"`

	// CORS configuration
	AllowedOrigins []string `json:"allowed_origins"`

	// WebSocket configuration
	WSReadTimeout  time.Duration `json:"ws_read_timeout"`
	WSWriteTimeout time.Duration `json:"ws_write_timeout"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		// Default values
		Port:             8081,
		Environment:      "development",
		SessionTimeout:   time.Hour,
		DefaultAdminUser: "admin",
		DefaultAdminPass: "admin",
		AllowedOrigins:   []string{"http://localhost:3000"},
		WSReadTimeout:    60 * time.Second,
		WSWriteTimeout:   10 * time.Second,
	}

	// Load from environment variables
	if port := os.Getenv("ADMIN_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Environment = env
	}

	// Required environment variables
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	cfg.RedisURL = os.Getenv("REDIS_URL")
	if cfg.RedisURL == "" {
		cfg.RedisURL = "redis://localhost:6379"
	}

	cfg.RedisPassword = os.Getenv("REDIS_PASSWORD")

	cfg.PasetoSecretKey = os.Getenv("PASETO_SECRET_KEY")
	if cfg.PasetoSecretKey == "" {
		return nil, fmt.Errorf("PASETO_SECRET_KEY environment variable is required")
	}

	cfg.BankingAPIURL = os.Getenv("BANKING_API_URL")
	if cfg.BankingAPIURL == "" {
		cfg.BankingAPIURL = "http://localhost:8080"
	}

	// Session timeout
	if timeout := os.Getenv("ADMIN_SESSION_TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			cfg.SessionTimeout = t
		}
	}

	// CORS origins
	if origins := os.Getenv("ADMIN_ALLOWED_ORIGINS"); origins != "" {
		cfg.AllowedOrigins = []string{origins}
	}

	return cfg, nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}