package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// PASETOConfig holds PASETO token configuration
type PASETOConfig struct {
	SecretKey  string
	Expiration time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

// EmailConfig holds email configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port        int
	Environment string
	Host        string
	ReadTimeout time.Duration
	WriteTimeout time.Duration
	IdleTimeout time.Duration
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level               string `json:"level"`
	Format              string `json:"format"`
	Output              string `json:"output"`
	Directory           string `json:"directory"`
	MaxAge              int    `json:"max_age"`
	MaxBackups          int    `json:"max_backups"`
	MaxSize             int    `json:"max_size"`
	Compress            bool   `json:"compress"`
	LocalTime           bool   `json:"local_time"`
	CallerInfo          bool   `json:"caller_info"`
	SamplingEnabled     bool   `json:"sampling_enabled"`
	SamplingInitial     int    `json:"sampling_initial"`
	SamplingThereafter  int    `json:"sampling_thereafter"`
}

// Config holds all configuration for the application
type Config struct {
	Database DatabaseConfig
	PASETO   PASETOConfig
	Redis    RedisConfig
	Email    EmailConfig
	Server   ServerConfig
	Logging  LogConfig
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	dbConfig, err := loadDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load database config: %w", err)
	}

	pasetoConfig, err := loadPASETOConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load PASETO config: %w", err)
	}

	redisConfig, err := loadRedisConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Redis config: %w", err)
	}

	emailConfig, err := loadEmailConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load email config: %w", err)
	}

	serverConfig, err := loadServerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}

	loggingConfig, err := loadLoggingConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load logging config: %w", err)
	}

	config := &Config{
		Database: dbConfig,
		PASETO:   pasetoConfig,
		Redis:    redisConfig,
		Email:    emailConfig,
		Server:   serverConfig,
		Logging:  loggingConfig,
	}

	// Validate the complete configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// loadDatabaseConfig loads database configuration from environment variables
func loadDatabaseConfig() (DatabaseConfig, error) {
	host := getEnvOrDefault("DB_HOST", "localhost")
	portStr := getEnvOrDefault("DB_PORT", "5432")
	name := getEnvOrDefault("DB_NAME", "bankapi")
	user := getEnvOrDefault("DB_USER", "bankuser")
	password := os.Getenv("DB_PASSWORD")
	sslMode := getEnvOrDefault("DB_SSL_MODE", "disable")
	maxOpenConnsStr := getEnvOrDefault("DB_MAX_OPEN_CONNS", "25")
	maxIdleConnsStr := getEnvOrDefault("DB_MAX_IDLE_CONNS", "5")
	connMaxLifetimeStr := getEnvOrDefault("DB_CONN_MAX_LIFETIME", "5m")
	connMaxIdleTimeStr := getEnvOrDefault("DB_CONN_MAX_IDLE_TIME", "5m")

	// Validate required fields
	if password == "" {
		return DatabaseConfig{}, fmt.Errorf("DB_PASSWORD environment variable is required")
	}

	// Parse port
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return DatabaseConfig{}, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	// Parse max open connections
	maxOpenConns, err := strconv.Atoi(maxOpenConnsStr)
	if err != nil {
		return DatabaseConfig{}, fmt.Errorf("invalid DB_MAX_OPEN_CONNS: %w", err)
	}

	// Parse max idle connections
	maxIdleConns, err := strconv.Atoi(maxIdleConnsStr)
	if err != nil {
		return DatabaseConfig{}, fmt.Errorf("invalid DB_MAX_IDLE_CONNS: %w", err)
	}

	// Parse connection max lifetime
	connMaxLifetime, err := time.ParseDuration(connMaxLifetimeStr)
	if err != nil {
		return DatabaseConfig{}, fmt.Errorf("invalid DB_CONN_MAX_LIFETIME: %w", err)
	}

	// Parse connection max idle time
	connMaxIdleTime, err := time.ParseDuration(connMaxIdleTimeStr)
	if err != nil {
		return DatabaseConfig{}, fmt.Errorf("invalid DB_CONN_MAX_IDLE_TIME: %w", err)
	}

	return DatabaseConfig{
		Host:            host,
		Port:            port,
		Name:            name,
		User:            user,
		Password:        password,
		SSLMode:         sslMode,
		MaxOpenConns:    maxOpenConns,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
		ConnMaxIdleTime: connMaxIdleTime,
	}, nil
}

// loadPASETOConfig loads PASETO configuration from environment variables
func loadPASETOConfig() (PASETOConfig, error) {
	secretKey := os.Getenv("PASETO_SECRET_KEY")
	expirationStr := getEnvOrDefault("PASETO_EXPIRATION", "24h")

	// Validate required fields
	if secretKey == "" {
		return PASETOConfig{}, fmt.Errorf("PASETO_SECRET_KEY environment variable is required")
	}

	// Parse expiration duration
	expiration, err := time.ParseDuration(expirationStr)
	if err != nil {
		return PASETOConfig{}, fmt.Errorf("invalid PASETO_EXPIRATION: %w", err)
	}

	return PASETOConfig{
		SecretKey:  secretKey,
		Expiration: expiration,
	}, nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadRedisConfig loads Redis configuration from environment variables
func loadRedisConfig() (RedisConfig, error) {
	host := getEnvOrDefault("REDIS_HOST", "localhost")
	portStr := getEnvOrDefault("REDIS_PORT", "6379")
	password := os.Getenv("REDIS_PASSWORD") // Optional
	dbStr := getEnvOrDefault("REDIS_DB", "0")
	poolSizeStr := getEnvOrDefault("REDIS_POOL_SIZE", "10")
	minIdleConnsStr := getEnvOrDefault("REDIS_MIN_IDLE_CONNS", "5")

	// Parse port
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return RedisConfig{}, fmt.Errorf("invalid REDIS_PORT: %w", err)
	}

	// Parse database number
	db, err := strconv.Atoi(dbStr)
	if err != nil {
		return RedisConfig{}, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	// Parse pool size
	poolSize, err := strconv.Atoi(poolSizeStr)
	if err != nil {
		return RedisConfig{}, fmt.Errorf("invalid REDIS_POOL_SIZE: %w", err)
	}

	// Parse min idle connections
	minIdleConns, err := strconv.Atoi(minIdleConnsStr)
	if err != nil {
		return RedisConfig{}, fmt.Errorf("invalid REDIS_MIN_IDLE_CONNS: %w", err)
	}

	return RedisConfig{
		Host:         host,
		Port:         port,
		Password:     password,
		DB:           db,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
	}, nil
}

// loadEmailConfig loads email configuration from environment variables
func loadEmailConfig() (EmailConfig, error) {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPortStr := getEnvOrDefault("SMTP_PORT", "587")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	fromEmail := getEnvOrDefault("FROM_EMAIL", smtpUsername)
	fromName := getEnvOrDefault("FROM_NAME", "Bank API")

	// Validate required fields
	if smtpHost == "" {
		return EmailConfig{}, fmt.Errorf("SMTP_HOST environment variable is required")
	}
	if smtpUsername == "" {
		return EmailConfig{}, fmt.Errorf("SMTP_USERNAME environment variable is required")
	}
	if smtpPassword == "" {
		return EmailConfig{}, fmt.Errorf("SMTP_PASSWORD environment variable is required")
	}

	// Parse SMTP port
	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		return EmailConfig{}, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	return EmailConfig{
		SMTPHost:     smtpHost,
		SMTPPort:     smtpPort,
		SMTPUsername: smtpUsername,
		SMTPPassword: smtpPassword,
		FromEmail:    fromEmail,
		FromName:     fromName,
	}, nil
}

// ConnectionString returns the PostgreSQL connection string
func (db DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		db.Host, db.Port, db.User, db.Password, db.Name, db.SSLMode)
}

// Address returns the Redis address
func (r RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// loadServerConfig loads server configuration from environment variables
func loadServerConfig() (ServerConfig, error) {
	portStr := getEnvOrDefault("PORT", "8080")
	environment := getEnvOrDefault("GIN_MODE", "debug")
	host := getEnvOrDefault("HOST", "0.0.0.0")
	readTimeoutStr := getEnvOrDefault("READ_TIMEOUT", "30s")
	writeTimeoutStr := getEnvOrDefault("WRITE_TIMEOUT", "30s")
	idleTimeoutStr := getEnvOrDefault("IDLE_TIMEOUT", "120s")

	// Parse port
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("invalid PORT: %w", err)
	}

	// Validate port range
	if port < 1 || port > 65535 {
		return ServerConfig{}, fmt.Errorf("PORT must be between 1 and 65535, got %d", port)
	}

	// Parse timeouts
	readTimeout, err := time.ParseDuration(readTimeoutStr)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("invalid READ timeout: %w", err)
	}

	writeTimeout, err := time.ParseDuration(writeTimeoutStr)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("invalid write timeout: %w", err)
	}

	idleTimeout, err := time.ParseDuration(idleTimeoutStr)
	if err != nil {
		return ServerConfig{}, fmt.Errorf("invalid idle timeout: %w", err)
	}

	// Validate environment
	if environment != "debug" && environment != "release" && environment != "test" {
		return ServerConfig{}, fmt.Errorf("GIN_MODE must be one of: debug, release, test")
	}

	return ServerConfig{
		Port:         port,
		Environment:  environment,
		Host:         host,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}, nil
}

// Address returns the server address
func (s ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// loadLoggingConfig loads logging configuration from environment variables
func loadLoggingConfig() (LogConfig, error) {
	level := getEnvOrDefault("LOG_LEVEL", "info")
	format := getEnvOrDefault("LOG_FORMAT", "json")
	output := getEnvOrDefault("LOG_OUTPUT", "both")
	directory := getEnvOrDefault("LOG_DIRECTORY", "logs")
	maxAgeStr := getEnvOrDefault("LOG_MAX_AGE", "30")
	maxBackupsStr := getEnvOrDefault("LOG_MAX_BACKUPS", "10")
	maxSizeStr := getEnvOrDefault("LOG_MAX_SIZE", "100")
	compressStr := getEnvOrDefault("LOG_COMPRESS", "true")
	localTimeStr := getEnvOrDefault("LOG_LOCAL_TIME", "true")
	callerInfoStr := getEnvOrDefault("LOG_CALLER_INFO", "false")
	samplingEnabledStr := getEnvOrDefault("LOG_SAMPLING_ENABLED", "false")
	samplingInitialStr := getEnvOrDefault("LOG_SAMPLING_INITIAL", "100")
	samplingThereafterStr := getEnvOrDefault("LOG_SAMPLING_THEREAFTER", "100")

	// Parse integer values
	maxAge, err := strconv.Atoi(maxAgeStr)
	if err != nil {
		return LogConfig{}, fmt.Errorf("invalid LOG_MAX_AGE: %w", err)
	}

	maxBackups, err := strconv.Atoi(maxBackupsStr)
	if err != nil {
		return LogConfig{}, fmt.Errorf("invalid LOG_MAX_BACKUPS: %w", err)
	}

	maxSize, err := strconv.Atoi(maxSizeStr)
	if err != nil {
		return LogConfig{}, fmt.Errorf("invalid LOG_MAX_SIZE: %w", err)
	}

	samplingInitial, err := strconv.Atoi(samplingInitialStr)
	if err != nil {
		return LogConfig{}, fmt.Errorf("invalid LOG_SAMPLING_INITIAL: %w", err)
	}

	samplingThereafter, err := strconv.Atoi(samplingThereafterStr)
	if err != nil {
		return LogConfig{}, fmt.Errorf("invalid LOG_SAMPLING_THEREAFTER: %w", err)
	}

	// Parse boolean values
	compress, err := strconv.ParseBool(compressStr)
	if err != nil {
		return LogConfig{}, fmt.Errorf("invalid LOG_COMPRESS: %w", err)
	}

	localTime, err := strconv.ParseBool(localTimeStr)
	if err != nil {
		return LogConfig{}, fmt.Errorf("invalid LOG_LOCAL_TIME: %w", err)
	}

	callerInfo, err := strconv.ParseBool(callerInfoStr)
	if err != nil {
		return LogConfig{}, fmt.Errorf("invalid LOG_CALLER_INFO: %w", err)
	}

	samplingEnabled, err := strconv.ParseBool(samplingEnabledStr)
	if err != nil {
		return LogConfig{}, fmt.Errorf("invalid LOG_SAMPLING_ENABLED: %w", err)
	}

	return LogConfig{
		Level:              level,
		Format:             format,
		Output:             output,
		Directory:          directory,
		MaxAge:             maxAge,
		MaxBackups:         maxBackups,
		MaxSize:            maxSize,
		Compress:           compress,
		LocalTime:          localTime,
		CallerInfo:         callerInfo,
		SamplingEnabled:    samplingEnabled,
		SamplingInitial:    samplingInitial,
		SamplingThereafter: samplingThereafter,
	}, nil
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	// Validate database configuration
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config validation failed: %w", err)
	}

	// Validate PASETO configuration
	if err := c.PASETO.Validate(); err != nil {
		return fmt.Errorf("PASETO config validation failed: %w", err)
	}

	// Validate Redis configuration
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("Redis config validation failed: %w", err)
	}

	// Validate Email configuration
	if err := c.Email.Validate(); err != nil {
		return fmt.Errorf("email config validation failed: %w", err)
	}

	// Validate Server configuration
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config validation failed: %w", err)
	}

	// Validate Logging configuration
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config validation failed: %w", err)
	}

	return nil
}

// Validate validates database configuration
func (db DatabaseConfig) Validate() error {
	if db.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}
	if db.Port < 1 || db.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	if db.Name == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	if db.User == "" {
		return fmt.Errorf("database user cannot be empty")
	}
	if db.Password == "" {
		return fmt.Errorf("database password cannot be empty")
	}
	if db.MaxOpenConns < 1 {
		return fmt.Errorf("max open connections must be at least 1")
	}
	if db.MaxIdleConns < 0 {
		return fmt.Errorf("max idle connections cannot be negative")
	}
	if db.MaxIdleConns > db.MaxOpenConns {
		return fmt.Errorf("max idle connections cannot exceed max open connections")
	}
	return nil
}

// Validate validates PASETO configuration
func (p PASETOConfig) Validate() error {
	if p.SecretKey == "" {
		return fmt.Errorf("PASETO secret key cannot be empty")
	}
	if len(p.SecretKey) < 32 {
		return fmt.Errorf("PASETO secret key must be at least 32 characters long")
	}
	if p.Expiration <= 0 {
		return fmt.Errorf("PASETO expiration must be positive")
	}
	return nil
}

// Validate validates Redis configuration
func (r RedisConfig) Validate() error {
	if r.Host == "" {
		return fmt.Errorf("Redis host cannot be empty")
	}
	if r.Port < 1 || r.Port > 65535 {
		return fmt.Errorf("Redis port must be between 1 and 65535")
	}
	if r.DB < 0 || r.DB > 15 {
		return fmt.Errorf("Redis database must be between 0 and 15")
	}
	if r.PoolSize < 1 {
		return fmt.Errorf("Redis pool size must be at least 1")
	}
	if r.MinIdleConns < 0 {
		return fmt.Errorf("Redis min idle connections cannot be negative")
	}
	if r.MinIdleConns > r.PoolSize {
		return fmt.Errorf("Redis min idle connections cannot exceed pool size")
	}
	return nil
}

// Validate validates email configuration
func (e EmailConfig) Validate() error {
	if e.SMTPHost == "" {
		return fmt.Errorf("SMTP host cannot be empty")
	}
	if e.SMTPPort < 1 || e.SMTPPort > 65535 {
		return fmt.Errorf("SMTP port must be between 1 and 65535")
	}
	if e.SMTPUsername == "" {
		return fmt.Errorf("SMTP username cannot be empty")
	}
	if e.SMTPPassword == "" {
		return fmt.Errorf("SMTP password cannot be empty")
	}
	if e.FromEmail == "" {
		return fmt.Errorf("from email cannot be empty")
	}
	if e.FromName == "" {
		return fmt.Errorf("from name cannot be empty")
	}
	return nil
}

// Validate validates server configuration
func (s ServerConfig) Validate() error {
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}
	if s.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}
	if s.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}
	if s.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}
	if s.IdleTimeout <= 0 {
		return fmt.Errorf("idle timeout must be positive")
	}
	validModes := map[string]bool{"debug": true, "release": true, "test": true}
	if !validModes[s.Environment] {
		return fmt.Errorf("environment must be one of: debug, release, test")
	}
	return nil
}

// Validate validates logging configuration
func (l LogConfig) Validate() error {
	// Validate log level
	validLevels := map[string]bool{
		"trace": true, "debug": true, "info": true, 
		"warn": true, "error": true, "fatal": true, "panic": true,
	}
	if !validLevels[l.Level] {
		return fmt.Errorf("log level must be one of: trace, debug, info, warn, error, fatal, panic")
	}

	// Validate log format
	validFormats := map[string]bool{"json": true, "console": true}
	if !validFormats[l.Format] {
		return fmt.Errorf("log format must be one of: json, console")
	}

	// Validate log output
	validOutputs := map[string]bool{"console": true, "file": true, "both": true}
	if !validOutputs[l.Output] {
		return fmt.Errorf("log output must be one of: console, file, both")
	}

	// Validate directory
	if l.Directory == "" {
		return fmt.Errorf("log directory cannot be empty")
	}

	// Validate numeric values
	if l.MaxAge < 0 {
		return fmt.Errorf("log max age cannot be negative")
	}
	if l.MaxBackups < 0 {
		return fmt.Errorf("log max backups cannot be negative")
	}
	if l.MaxSize <= 0 {
		return fmt.Errorf("log max size must be positive")
	}
	if l.SamplingInitial <= 0 {
		return fmt.Errorf("log sampling initial must be positive")
	}
	if l.SamplingThereafter <= 0 {
		return fmt.Errorf("log sampling thereafter must be positive")
	}

	return nil
}