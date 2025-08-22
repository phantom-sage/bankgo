package logging

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoggerManager(t *testing.T) {
	tests := []struct {
		name    string
		config  LogConfig
		wantErr bool
	}{
		{
			name: "valid console config",
			config: LogConfig{
				Level:  "info",
				Format: "console",
				Output: "console",
			},
			wantErr: false,
		},
		{
			name: "valid file config",
			config: LogConfig{
				Level:     "debug",
				Format:    "json",
				Output:    "file",
				Directory: t.TempDir(),
			},
			wantErr: false,
		},
		{
			name: "valid both config",
			config: LogConfig{
				Level:     "warn",
				Format:    "json",
				Output:    "both",
				Directory: t.TempDir(),
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			config: LogConfig{
				Level:  "invalid",
				Format: "json",
				Output: "console",
			},
			wantErr: true,
		},
		{
			name: "invalid output",
			config: LogConfig{
				Level:  "info",
				Format: "json",
				Output: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm, err := NewLoggerManager(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, lm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, lm)
				
				if lm != nil {
					defer lm.Close()
					
					// Test that we can get a logger
					logger := lm.GetLogger()
					assert.NotNil(t, logger)
					
					// Test that the config is stored correctly
					config := lm.GetConfig()
					assert.Equal(t, tt.config.Level, config.Level)
					assert.Equal(t, tt.config.Format, config.Format)
					assert.Equal(t, tt.config.Output, config.Output)
				}
			}
		})
	}
}

func TestLoggerManager_GetLogger(t *testing.T) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	logger := lm.GetLogger()
	assert.NotNil(t, logger)

	// Test that we can log with the logger
	logger.Info().Msg("test message")
}

func TestLoggerManager_GetContextLogger(t *testing.T) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "empty context",
			ctx:  context.Background(),
		},
		{
			name: "context with request ID",
			ctx: context.WithValue(context.Background(), "request_id", "test-request-123"),
		},
		{
			name: "context with user ID",
			ctx: context.WithValue(context.Background(), "user_id", int64(123)),
		},
		{
			name: "context with user email",
			ctx: context.WithValue(context.Background(), "user_email", "test@example.com"),
		},
		{
			name: "context with all fields",
			ctx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "request_id", "test-request-123")
				ctx = context.WithValue(ctx, "user_id", int64(123))
				ctx = context.WithValue(ctx, "user_email", "test@example.com")
				return ctx
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := lm.GetContextLogger(tt.ctx)
			assert.NotNil(t, logger)

			// Test that we can log with the context logger
			logger.Info().Msg("test context message")
		})
	}
}

func TestLoggerManager_WithFields(t *testing.T) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	fields := map[string]interface{}{
		"string_field": "test_value",
		"int_field":    123,
		"bool_field":   true,
		"float_field":  3.14,
	}

	logger := lm.WithFields(fields)
	assert.NotNil(t, logger)

	// Test that we can log with the fields
	logger.Info().Msg("test message with fields")
}

func TestLoggerManager_WithComponent(t *testing.T) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	logger := lm.WithComponent("test-component")
	assert.NotNil(t, logger)

	// Test that we can log with the component
	logger.Info().Msg("test message with component")
}

func TestLoggerManager_WithRequestID(t *testing.T) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	requestID := "test-request-123"
	logger := lm.WithRequestID(requestID)
	assert.NotNil(t, logger)

	// Test that we can log with the request ID
	logger.Info().Msg("test message with request ID")
}

func TestLoggerManager_WithUser(t *testing.T) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	userID := int64(123)
	userEmail := "test@example.com"
	logger := lm.WithUser(userID, userEmail)
	assert.NotNil(t, logger)

	// Test that we can log with user context
	logger.Info().Msg("test message with user context")
}

func TestLoggerManager_UpdateLogLevel(t *testing.T) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	// Test valid level update
	err = lm.UpdateLogLevel("debug")
	assert.NoError(t, err)
	assert.Equal(t, "debug", lm.GetConfig().Level)

	// Test invalid level update
	err = lm.UpdateLogLevel("invalid")
	assert.Error(t, err)
}

func TestLoggerManager_IsLevelEnabled(t *testing.T) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	// Test level checking (info level is set, so info and higher levels should be enabled)
	// In zerolog, lower numeric values are more verbose, higher values are less verbose
	assert.True(t, lm.IsLevelEnabled(zerolog.InfoLevel))   // 1 - enabled
	assert.True(t, lm.IsLevelEnabled(zerolog.WarnLevel))   // 2 - enabled
	assert.True(t, lm.IsLevelEnabled(zerolog.ErrorLevel))  // 3 - enabled
	assert.True(t, lm.IsLevelEnabled(zerolog.FatalLevel))  // 4 - enabled
	assert.False(t, lm.IsLevelEnabled(zerolog.DebugLevel)) // 0 - disabled (more verbose than info)
	assert.False(t, lm.IsLevelEnabled(zerolog.TraceLevel)) // -1 - disabled (more verbose than info)
}

func TestLoggerManager_FileOperations(t *testing.T) {
	tempDir := t.TempDir()
	
	config := LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "file",
		Directory:  tempDir,
		MaxAge:     7,
		MaxBackups: 5,
		Compress:   false, // Disable compression to avoid cleanup issues in tests
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	// Write some log entries to ensure file exists
	logger := lm.GetLogger()
	logger.Info().Msg("test log entry")

	// Test getting current log file
	logFile := lm.GetCurrentLogFile()
	assert.NotEmpty(t, logFile)
	assert.True(t, strings.Contains(logFile, tempDir))

	// Test that log file exists
	_, err = os.Stat(logFile)
	assert.NoError(t, err)

	// Test getting log files
	files, err := lm.GetLogFiles()
	assert.NoError(t, err)
	assert.Len(t, files, 1)

	// Test manual rotation
	err = lm.RotateLogFile()
	assert.NoError(t, err)

	// Test cleanup
	err = lm.CleanupLogFiles()
	assert.NoError(t, err)
}

func TestLoggerManager_FileOperationsWithoutFileLogging(t *testing.T) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	// Test operations that should fail without file logging
	logFile := lm.GetCurrentLogFile()
	assert.Empty(t, logFile)

	err = lm.RotateLogFile()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file logging not enabled")

	err = lm.CleanupLogFiles()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file logging not enabled")

	files, err := lm.GetLogFiles()
	assert.Error(t, err)
	assert.Nil(t, files)
	assert.Contains(t, err.Error(), "file logging not enabled")
}

func TestLoggerManager_HealthCheck(t *testing.T) {
	tests := []struct {
		name   string
		config LogConfig
	}{
		{
			name: "console output",
			config: LogConfig{
				Level:  "info",
				Format: "json",
				Output: "console",
			},
		},
		{
			name: "file output",
			config: LogConfig{
				Level:     "info",
				Format:    "json",
				Output:    "file",
				Directory: t.TempDir(),
			},
		},
		{
			name: "both outputs",
			config: LogConfig{
				Level:     "info",
				Format:    "json",
				Output:    "both",
				Directory: t.TempDir(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm, err := NewLoggerManager(tt.config)
			require.NoError(t, err)
			defer lm.Close()

			err = lm.HealthCheck()
			assert.NoError(t, err)
		})
	}
}

func TestLoggerManager_Sync(t *testing.T) {
	tempDir := t.TempDir()
	
	config := LogConfig{
		Level:     "info",
		Format:    "json",
		Output:    "file",
		Directory: tempDir,
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	// Write some log entries
	logger := lm.GetLogger()
	logger.Info().Msg("test message 1")
	logger.Info().Msg("test message 2")

	// Test sync
	err = lm.Sync()
	assert.NoError(t, err)
}

func TestLoggerManager_SamplingConfiguration(t *testing.T) {
	config := LogConfig{
		Level:              "info",
		Format:             "json",
		Output:             "console",
		SamplingEnabled:    true,
		SamplingInitial:    10,
		SamplingThereafter: 100,
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	logger := lm.GetLogger()
	assert.NotNil(t, logger)

	// Test that we can log with sampling enabled
	for i := 0; i < 20; i++ {
		logger.Info().Int("iteration", i).Msg("sampled log message")
	}
}

func TestLoggerManager_CallerInfo(t *testing.T) {
	config := LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "console",
		CallerInfo: true,
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()

	logger := lm.GetLogger()
	assert.NotNil(t, logger)

	// Test that we can log with caller info enabled
	logger.Info().Msg("test message with caller info")
}

func TestLoggerManager_DifferentFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
		output string
	}{
		{
			name:   "json console",
			format: "json",
			output: "console",
		},
		{
			name:   "console format",
			format: "console",
			output: "console",
		},
		{
			name:   "json file",
			format: "json",
			output: "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := LogConfig{
				Level:  "info",
				Format: tt.format,
				Output: tt.output,
			}

			if tt.output == "file" || tt.output == "both" {
				config.Directory = t.TempDir()
			}

			lm, err := NewLoggerManager(config)
			require.NoError(t, err)
			defer lm.Close()

			logger := lm.GetLogger()
			logger.Info().Str("format", tt.format).Msg("test message")
		})
	}
}

func TestLoggerManager_Close(t *testing.T) {
	tempDir := t.TempDir()
	
	config := LogConfig{
		Level:     "info",
		Format:    "json",
		Output:    "file",
		Directory: tempDir,
	}

	lm, err := NewLoggerManager(config)
	require.NoError(t, err)

	// Write a log entry
	logger := lm.GetLogger()
	logger.Info().Msg("test message before close")

	// Close should not error
	err = lm.Close()
	assert.NoError(t, err)

	// Second close should not error
	err = lm.Close()
	assert.NoError(t, err)
}

// Benchmark tests
func BenchmarkLoggerManager_GetLogger(b *testing.B) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(b, err)
	defer lm.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lm.GetLogger()
	}
}

func BenchmarkLoggerManager_WithFields(b *testing.B) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(b, err)
	defer lm.Close()

	fields := map[string]interface{}{
		"field1": "value1",
		"field2": 123,
		"field3": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lm.WithFields(fields)
	}
}

func BenchmarkLoggerManager_Logging(b *testing.B) {
	config := LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	}

	lm, err := NewLoggerManager(config)
	require.NoError(b, err)
	defer lm.Close()

	logger := lm.GetLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info().
			Int("iteration", i).
			Str("component", "benchmark").
			Msg("benchmark log message")
	}
}