package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Set required environment variables
	setRequiredEnvVars(t)
	defer cleanupEnvVars()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected config to be loaded, got nil")
	}

	// Test default values
	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected default host 'localhost', got %s", cfg.Database.Host)
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("Expected default port 5432, got %d", cfg.Database.Port)
	}

	if cfg.Database.Password != "test_db_password_32_chars_long" {
		t.Errorf("Expected password 'test_db_password_32_chars_long', got %s", cfg.Database.Password)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default server port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Server.Environment != "debug" {
		t.Errorf("Expected default environment 'debug', got %s", cfg.Server.Environment)
	}
}

func TestLoadConfigMissingRequiredVars(t *testing.T) {
	tests := []struct {
		name    string
		setup   func()
		cleanup func()
	}{
		{
			name: "missing DB_PASSWORD",
			setup: func() {
				setRequiredEnvVars(t)
				os.Unsetenv("DB_PASSWORD")
			},
			cleanup: cleanupEnvVars,
		},
		{
			name: "missing PASETO_SECRET_KEY",
			setup: func() {
				setRequiredEnvVars(t)
				os.Unsetenv("PASETO_SECRET_KEY")
			},
			cleanup: cleanupEnvVars,
		},
		{
			name: "missing SMTP_HOST",
			setup: func() {
				setRequiredEnvVars(t)
				os.Unsetenv("SMTP_HOST")
			},
			cleanup: cleanupEnvVars,
		},
		{
			name: "missing SMTP_USERNAME",
			setup: func() {
				setRequiredEnvVars(t)
				os.Unsetenv("SMTP_USERNAME")
			},
			cleanup: cleanupEnvVars,
		},
		{
			name: "missing SMTP_PASSWORD",
			setup: func() {
				setRequiredEnvVars(t)
				os.Unsetenv("SMTP_PASSWORD")
			},
			cleanup: cleanupEnvVars,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			_, err := LoadConfig()
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestConnectionString(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "testhost",
		Port:     5433,
		Name:     "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "require",
	}

	expected := "host=testhost port=5433 user=testuser password=testpass dbname=testdb sslmode=require"
	actual := cfg.ConnectionString()

	if actual != expected {
		t.Errorf("Expected connection string %q, got %q", expected, actual)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	// Test with existing environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result := getEnvOrDefault("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got %s", result)
	}

	// Test with non-existing environment variable
	result = getEnvOrDefault("NON_EXISTING_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got %s", result)
	}
}

func TestDatabaseConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  DatabaseConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: DatabaseConfig{
				Host:            "localhost",
				Port:            5432,
				Name:            "testdb",
				User:            "testuser",
				Password:        "testpass",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
				ConnMaxIdleTime: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "empty host",
			config: DatabaseConfig{
				Host:         "",
				Port:         5432,
				Name:         "testdb",
				User:         "testuser",
				Password:     "testpass",
				MaxOpenConns: 10,
				MaxIdleConns: 5,
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: DatabaseConfig{
				Host:         "localhost",
				Port:         0,
				Name:         "testdb",
				User:         "testuser",
				Password:     "testpass",
				MaxOpenConns: 10,
				MaxIdleConns: 5,
			},
			wantErr: true,
		},
		{
			name: "max idle conns exceeds max open conns",
			config: DatabaseConfig{
				Host:         "localhost",
				Port:         5432,
				Name:         "testdb",
				User:         "testuser",
				Password:     "testpass",
				MaxOpenConns: 5,
				MaxIdleConns: 10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DatabaseConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPASETOConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  PASETOConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: PASETOConfig{
				SecretKey:  "this_is_a_32_character_secret_key",
				Expiration: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "empty secret key",
			config: PASETOConfig{
				SecretKey:  "",
				Expiration: 24 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "short secret key",
			config: PASETOConfig{
				SecretKey:  "short",
				Expiration: 24 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "zero expiration",
			config: PASETOConfig{
				SecretKey:  "this_is_a_32_character_secret_key",
				Expiration: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PASETOConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedisConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  RedisConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: RedisConfig{
				Host:         "localhost",
				Port:         6379,
				DB:           0,
				PoolSize:     10,
				MinIdleConns: 5,
			},
			wantErr: false,
		},
		{
			name: "empty host",
			config: RedisConfig{
				Host:         "",
				Port:         6379,
				DB:           0,
				PoolSize:     10,
				MinIdleConns: 5,
			},
			wantErr: true,
		},
		{
			name: "invalid database number",
			config: RedisConfig{
				Host:         "localhost",
				Port:         6379,
				DB:           16,
				PoolSize:     10,
				MinIdleConns: 5,
			},
			wantErr: true,
		},
		{
			name: "min idle conns exceeds pool size",
			config: RedisConfig{
				Host:         "localhost",
				Port:         6379,
				DB:           0,
				PoolSize:     5,
				MinIdleConns: 10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  ServerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ServerConfig{
				Port:         8080,
				Environment:  "debug",
				Host:         "0.0.0.0",
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			config: ServerConfig{
				Port:         0,
				Environment:  "debug",
				Host:         "0.0.0.0",
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid environment",
			config: ServerConfig{
				Port:         8080,
				Environment:  "invalid",
				Host:         "0.0.0.0",
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: ServerConfig{
				Port:         8080,
				Environment:  "debug",
				Host:         "0.0.0.0",
				ReadTimeout:  0,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ServerConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmailConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  EmailConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "test@example.com",
				SMTPPassword: "password",
				FromEmail:    "test@example.com",
				FromName:     "Test Bank",
			},
			wantErr: false,
		},
		{
			name: "empty SMTP host",
			config: EmailConfig{
				SMTPHost:     "",
				SMTPPort:     587,
				SMTPUsername: "test@example.com",
				SMTPPassword: "password",
				FromEmail:    "test@example.com",
				FromName:     "Test Bank",
			},
			wantErr: true,
		},
		{
			name: "invalid SMTP port",
			config: EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     0,
				SMTPUsername: "test@example.com",
				SMTPPassword: "password",
				FromEmail:    "test@example.com",
				FromName:     "Test Bank",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EmailConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLogConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  LogConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: LogConfig{
				Level:              "info",
				Format:             "json",
				Output:             "both",
				Directory:          "logs",
				MaxAge:             30,
				MaxBackups:         10,
				MaxSize:            100,
				Compress:           true,
				LocalTime:          true,
				CallerInfo:         false,
				SamplingEnabled:    false,
				SamplingInitial:    100,
				SamplingThereafter: 100,
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			config: LogConfig{
				Level:              "invalid",
				Format:             "json",
				Output:             "both",
				Directory:          "logs",
				MaxAge:             30,
				MaxBackups:         10,
				MaxSize:            100,
				SamplingInitial:    100,
				SamplingThereafter: 100,
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			config: LogConfig{
				Level:              "info",
				Format:             "invalid",
				Output:             "both",
				Directory:          "logs",
				MaxAge:             30,
				MaxBackups:         10,
				MaxSize:            100,
				SamplingInitial:    100,
				SamplingThereafter: 100,
			},
			wantErr: true,
		},
		{
			name: "invalid log output",
			config: LogConfig{
				Level:              "info",
				Format:             "json",
				Output:             "invalid",
				Directory:          "logs",
				MaxAge:             30,
				MaxBackups:         10,
				MaxSize:            100,
				SamplingInitial:    100,
				SamplingThereafter: 100,
			},
			wantErr: true,
		},
		{
			name: "empty directory",
			config: LogConfig{
				Level:              "info",
				Format:             "json",
				Output:             "both",
				Directory:          "",
				MaxAge:             30,
				MaxBackups:         10,
				MaxSize:            100,
				SamplingInitial:    100,
				SamplingThereafter: 100,
			},
			wantErr: true,
		},
		{
			name: "negative max age",
			config: LogConfig{
				Level:              "info",
				Format:             "json",
				Output:             "both",
				Directory:          "logs",
				MaxAge:             -1,
				MaxBackups:         10,
				MaxSize:            100,
				SamplingInitial:    100,
				SamplingThereafter: 100,
			},
			wantErr: true,
		},
		{
			name: "zero max size",
			config: LogConfig{
				Level:              "info",
				Format:             "json",
				Output:             "both",
				Directory:          "logs",
				MaxAge:             30,
				MaxBackups:         10,
				MaxSize:            0,
				SamplingInitial:    100,
				SamplingThereafter: 100,
			},
			wantErr: true,
		},
		{
			name: "zero sampling initial",
			config: LogConfig{
				Level:              "info",
				Format:             "json",
				Output:             "both",
				Directory:          "logs",
				MaxAge:             30,
				MaxBackups:         10,
				MaxSize:            100,
				SamplingInitial:    0,
				SamplingThereafter: 100,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("LogConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadLoggingConfig(t *testing.T) {
	// Test with default values
	cfg, err := loadLoggingConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check default values
	if cfg.Level != "info" {
		t.Errorf("Expected default level 'info', got %s", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("Expected default format 'json', got %s", cfg.Format)
	}
	if cfg.Output != "both" {
		t.Errorf("Expected default output 'both', got %s", cfg.Output)
	}
	if cfg.Directory != "logs" {
		t.Errorf("Expected default directory 'logs', got %s", cfg.Directory)
	}
	if cfg.MaxAge != 30 {
		t.Errorf("Expected default max age 30, got %d", cfg.MaxAge)
	}
	if cfg.MaxBackups != 10 {
		t.Errorf("Expected default max backups 10, got %d", cfg.MaxBackups)
	}
	if cfg.MaxSize != 100 {
		t.Errorf("Expected default max size 100, got %d", cfg.MaxSize)
	}
	if !cfg.Compress {
		t.Errorf("Expected default compress true, got %v", cfg.Compress)
	}
	if !cfg.LocalTime {
		t.Errorf("Expected default local time true, got %v", cfg.LocalTime)
	}
	if cfg.CallerInfo {
		t.Errorf("Expected default caller info false, got %v", cfg.CallerInfo)
	}
	if cfg.SamplingEnabled {
		t.Errorf("Expected default sampling enabled false, got %v", cfg.SamplingEnabled)
	}
	if cfg.SamplingInitial != 100 {
		t.Errorf("Expected default sampling initial 100, got %d", cfg.SamplingInitial)
	}
	if cfg.SamplingThereafter != 100 {
		t.Errorf("Expected default sampling thereafter 100, got %d", cfg.SamplingThereafter)
	}
}

func TestLoadLoggingConfigWithEnvVars(t *testing.T) {
	// Set custom environment variables
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "console")
	os.Setenv("LOG_OUTPUT", "file")
	os.Setenv("LOG_DIRECTORY", "custom_logs")
	os.Setenv("LOG_MAX_AGE", "7")
	os.Setenv("LOG_MAX_BACKUPS", "5")
	os.Setenv("LOG_MAX_SIZE", "50")
	os.Setenv("LOG_COMPRESS", "false")
	os.Setenv("LOG_LOCAL_TIME", "false")
	os.Setenv("LOG_CALLER_INFO", "true")
	os.Setenv("LOG_SAMPLING_ENABLED", "true")
	os.Setenv("LOG_SAMPLING_INITIAL", "50")
	os.Setenv("LOG_SAMPLING_THEREAFTER", "25")

	defer func() {
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("LOG_FORMAT")
		os.Unsetenv("LOG_OUTPUT")
		os.Unsetenv("LOG_DIRECTORY")
		os.Unsetenv("LOG_MAX_AGE")
		os.Unsetenv("LOG_MAX_BACKUPS")
		os.Unsetenv("LOG_MAX_SIZE")
		os.Unsetenv("LOG_COMPRESS")
		os.Unsetenv("LOG_LOCAL_TIME")
		os.Unsetenv("LOG_CALLER_INFO")
		os.Unsetenv("LOG_SAMPLING_ENABLED")
		os.Unsetenv("LOG_SAMPLING_INITIAL")
		os.Unsetenv("LOG_SAMPLING_THEREAFTER")
	}()

	cfg, err := loadLoggingConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check custom values
	if cfg.Level != "debug" {
		t.Errorf("Expected level 'debug', got %s", cfg.Level)
	}
	if cfg.Format != "console" {
		t.Errorf("Expected format 'console', got %s", cfg.Format)
	}
	if cfg.Output != "file" {
		t.Errorf("Expected output 'file', got %s", cfg.Output)
	}
	if cfg.Directory != "custom_logs" {
		t.Errorf("Expected directory 'custom_logs', got %s", cfg.Directory)
	}
	if cfg.MaxAge != 7 {
		t.Errorf("Expected max age 7, got %d", cfg.MaxAge)
	}
	if cfg.MaxBackups != 5 {
		t.Errorf("Expected max backups 5, got %d", cfg.MaxBackups)
	}
	if cfg.MaxSize != 50 {
		t.Errorf("Expected max size 50, got %d", cfg.MaxSize)
	}
	if cfg.Compress {
		t.Errorf("Expected compress false, got %v", cfg.Compress)
	}
	if cfg.LocalTime {
		t.Errorf("Expected local time false, got %v", cfg.LocalTime)
	}
	if !cfg.CallerInfo {
		t.Errorf("Expected caller info true, got %v", cfg.CallerInfo)
	}
	if !cfg.SamplingEnabled {
		t.Errorf("Expected sampling enabled true, got %v", cfg.SamplingEnabled)
	}
	if cfg.SamplingInitial != 50 {
		t.Errorf("Expected sampling initial 50, got %d", cfg.SamplingInitial)
	}
	if cfg.SamplingThereafter != 25 {
		t.Errorf("Expected sampling thereafter 25, got %d", cfg.SamplingThereafter)
	}
}

func TestConfigValidation(t *testing.T) {
	setRequiredEnvVars(t)
	defer cleanupEnvVars()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test that validation passes for a complete config
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}

	// Test that logging config is loaded
	if cfg.Logging.Level == "" {
		t.Error("Expected logging config to be loaded")
	}
}

func TestInvalidEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
	}{
		{"invalid port", "PORT", "invalid"},
		{"invalid DB port", "DB_PORT", "invalid"},
		{"invalid Redis port", "REDIS_PORT", "invalid"},
		{"invalid SMTP port", "SMTP_PORT", "invalid"},
		{"invalid timeout", "READ_TIMEOUT", "invalid"},
		{"invalid environment", "GIN_MODE", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnvVars(t)
			os.Setenv(tt.envVar, tt.envValue)
			defer cleanupEnvVars()

			_, err := LoadConfig()
			if err == nil {
				t.Errorf("Expected error for invalid %s, got nil", tt.envVar)
			}
		})
	}
}

func TestAddressMethods(t *testing.T) {
	redisConfig := RedisConfig{Host: "localhost", Port: 6379}
	expected := "localhost:6379"
	if addr := redisConfig.Address(); addr != expected {
		t.Errorf("Expected Redis address %s, got %s", expected, addr)
	}

	serverConfig := ServerConfig{Host: "0.0.0.0", Port: 8080}
	expected = "0.0.0.0:8080"
	if addr := serverConfig.Address(); addr != expected {
		t.Errorf("Expected server address %s, got %s", expected, addr)
	}
}

// Helper functions for test setup and cleanup
func setRequiredEnvVars(t *testing.T) {
	t.Helper()
	os.Setenv("DB_PASSWORD", "test_db_password_32_chars_long")
	os.Setenv("PASETO_SECRET_KEY", "test_paseto_secret_key_32_chars_long")
	os.Setenv("SMTP_HOST", "smtp.test.com")
	os.Setenv("SMTP_USERNAME", "test@example.com")
	os.Setenv("SMTP_PASSWORD", "test_smtp_password")
}

func cleanupEnvVars() {
	envVars := []string{
		"DB_PASSWORD", "PASETO_SECRET_KEY", "SMTP_HOST", "SMTP_USERNAME", "SMTP_PASSWORD",
		"PORT", "DB_PORT", "REDIS_PORT", "SMTP_PORT", "READ_TIMEOUT", "GIN_MODE",
		"DB_HOST", "DB_NAME", "DB_USER", "REDIS_HOST", "HOST",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}