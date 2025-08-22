package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogConfig holds configuration for the logging system
type LogConfig struct {
	Level               string `mapstructure:"level" default:"info"`
	Format              string `mapstructure:"format" default:"json"`
	Output              string `mapstructure:"output" default:"both"`
	Directory           string `mapstructure:"directory" default:"logs"`
	MaxAge              int    `mapstructure:"max_age" default:"30"`
	MaxBackups          int    `mapstructure:"max_backups" default:"10"`
	MaxSize             int    `mapstructure:"max_size" default:"100"`
	Compress            bool   `mapstructure:"compress" default:"true"`
	LocalTime           bool   `mapstructure:"local_time" default:"true"`
	CallerInfo          bool   `mapstructure:"caller_info" default:"false"`
	SamplingEnabled     bool   `mapstructure:"sampling_enabled" default:"false"`
	SamplingInitial     int    `mapstructure:"sampling_initial" default:"100"`
	SamplingThereafter  int    `mapstructure:"sampling_thereafter" default:"100"`
}

// LoggerManager manages zerolog configuration and provides logger instances
type LoggerManager struct {
	logger      zerolog.Logger
	config      LogConfig
	fileWriter  io.Writer
	multiWriter io.Writer
}

// NewLoggerManager creates a new logger manager with the given configuration
func NewLoggerManager(config LogConfig) (*LoggerManager, error) {
	lm := &LoggerManager{
		config: config,
	}

	// Set up writers based on output configuration
	if err := lm.setupWriters(); err != nil {
		return nil, fmt.Errorf("failed to setup writers: %w", err)
	}

	// Configure zerolog
	if err := lm.configureLogger(); err != nil {
		return nil, fmt.Errorf("failed to configure logger: %w", err)
	}

	return lm, nil
}

// setupWriters configures the output writers based on configuration
func (lm *LoggerManager) setupWriters() error {
	var writers []io.Writer

	// Configure console writer if needed
	if lm.config.Output == "console" || lm.config.Output == "both" {
		var consoleWriter io.Writer
		
		if lm.config.Format == "console" || (lm.config.Format == "json" && lm.config.Output == "console") {
			// Use pretty console writer for development
			consoleWriter = zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC3339,
				NoColor:    false,
			}
		} else {
			// Use stdout for JSON format
			consoleWriter = os.Stdout
		}
		
		writers = append(writers, consoleWriter)
	}

	// Configure file writer if needed
	if lm.config.Output == "file" || lm.config.Output == "both" {
		fileConfig := DailyFileConfig{
			Directory:  lm.config.Directory,
			MaxAge:     lm.config.MaxAge,
			MaxBackups: lm.config.MaxBackups,
			Compress:   lm.config.Compress,
			LocalTime:  lm.config.LocalTime,
		}

		fileWriter, err := NewDailyFileWriter(fileConfig)
		if err != nil {
			return fmt.Errorf("failed to create file writer: %w", err)
		}

		lm.fileWriter = fileWriter
		writers = append(writers, fileWriter)
	}

	// Create multi-writer if we have multiple outputs
	if len(writers) == 0 {
		return fmt.Errorf("no output writers configured")
	} else if len(writers) == 1 {
		lm.multiWriter = writers[0]
	} else {
		lm.multiWriter = io.MultiWriter(writers...)
	}

	return nil
}

// configureLogger sets up the zerolog logger with the configured options
func (lm *LoggerManager) configureLogger() error {
	// Parse log level
	level, err := zerolog.ParseLevel(strings.ToLower(lm.config.Level))
	if err != nil {
		return fmt.Errorf("invalid log level %s: %w", lm.config.Level, err)
	}

	// Set global log level
	zerolog.SetGlobalLevel(level)

	// Configure time format
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Create logger with multi-writer
	logger := zerolog.New(lm.multiWriter).With().Timestamp()

	// Add caller info if configured
	if lm.config.CallerInfo {
		logger = logger.Caller()
	}

	// Create the logger
	lm.logger = logger.Logger()

	// Configure sampling if enabled
	if lm.config.SamplingEnabled {
		if lm.config.SamplingThereafter > 0 {
			sampler := &zerolog.BurstSampler{
				Burst:       uint32(lm.config.SamplingInitial),
				Period:      1 * time.Second,
				NextSampler: &zerolog.BasicSampler{N: uint32(lm.config.SamplingThereafter)},
			}
			lm.logger = lm.logger.Sample(sampler)
		} else {
			sampler := &zerolog.BasicSampler{N: uint32(lm.config.SamplingInitial)}
			lm.logger = lm.logger.Sample(sampler)
		}
	}

	// Set as global logger
	log.Logger = lm.logger

	return nil
}

// GetLogger returns the configured zerolog logger
func (lm *LoggerManager) GetLogger() zerolog.Logger {
	return lm.logger
}

// GetContextLogger returns a logger with context information
func (lm *LoggerManager) GetContextLogger(ctx context.Context) zerolog.Logger {
	logger := lm.logger

	// Add request ID if available in context
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			logger = logger.With().Str("request_id", id).Logger()
		}
	}

	// Add user ID if available in context
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(int64); ok {
			logger = logger.With().Int64("user_id", id).Logger()
		}
	}

	// Add user email if available in context
	if userEmail := ctx.Value("user_email"); userEmail != nil {
		if email, ok := userEmail.(string); ok {
			logger = logger.With().Str("user_email", email).Logger()
		}
	}

	return logger
}

// WithFields returns a logger with additional fields
func (lm *LoggerManager) WithFields(fields map[string]interface{}) zerolog.Logger {
	logger := lm.logger
	
	for key, value := range fields {
		logger = logger.With().Interface(key, value).Logger()
	}
	
	return logger
}

// WithComponent returns a logger with component field
func (lm *LoggerManager) WithComponent(component string) zerolog.Logger {
	return lm.logger.With().Str("component", component).Logger()
}

// WithRequestID returns a logger with request ID field
func (lm *LoggerManager) WithRequestID(requestID string) zerolog.Logger {
	return lm.logger.With().Str("request_id", requestID).Logger()
}

// WithUser returns a logger with user context fields
func (lm *LoggerManager) WithUser(userID int64, userEmail string) zerolog.Logger {
	return lm.logger.With().
		Int64("user_id", userID).
		Str("user_email", userEmail).
		Logger()
}

// Close closes the logger manager and any associated resources
func (lm *LoggerManager) Close() error {
	if closer, ok := lm.fileWriter.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Sync flushes any buffered log entries
func (lm *LoggerManager) Sync() error {
	if syncer, ok := lm.fileWriter.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	return nil
}

// HealthCheck verifies that the logging system is working correctly
func (lm *LoggerManager) HealthCheck() error {
	// Test that we can write a log entry
	testLogger := lm.logger.With().Str("health_check", "test").Logger()
	testLogger.Debug().Msg("Health check test log entry")

	// If we have a file writer, check that it's working
	if lm.fileWriter != nil {
		if _, err := lm.fileWriter.Write([]byte("")); err != nil {
			return fmt.Errorf("file writer health check failed: %w", err)
		}
	}

	return nil
}

// GetConfig returns the current logger configuration
func (lm *LoggerManager) GetConfig() LogConfig {
	return lm.config
}

// UpdateLogLevel dynamically updates the log level
func (lm *LoggerManager) UpdateLogLevel(level string) error {
	parsedLevel, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		return fmt.Errorf("invalid log level %s: %w", level, err)
	}

	zerolog.SetGlobalLevel(parsedLevel)
	lm.config.Level = level
	
	lm.logger.Info().
		Str("old_level", lm.config.Level).
		Str("new_level", level).
		Msg("Log level updated")

	return nil
}

// IsLevelEnabled checks if a specific log level is enabled
func (lm *LoggerManager) IsLevelEnabled(level zerolog.Level) bool {
	// Use the global level since that's what controls logging
	return zerolog.GlobalLevel() <= level
}

// GetCurrentLogFile returns the current log file path if file logging is enabled
func (lm *LoggerManager) GetCurrentLogFile() string {
	if dfw, ok := lm.fileWriter.(*DailyFileWriter); ok {
		return dfw.GetCurrentFileName()
	}
	return ""
}

// RotateLogFile manually triggers log file rotation if file logging is enabled
func (lm *LoggerManager) RotateLogFile() error {
	if dfw, ok := lm.fileWriter.(*DailyFileWriter); ok {
		return dfw.Rotate()
	}
	return fmt.Errorf("file logging not enabled")
}

// CleanupLogFiles manually triggers cleanup of old log files
func (lm *LoggerManager) CleanupLogFiles() error {
	if dfw, ok := lm.fileWriter.(*DailyFileWriter); ok {
		return dfw.Cleanup()
	}
	return fmt.Errorf("file logging not enabled")
}

// GetLogFiles returns a list of all log files if file logging is enabled
func (lm *LoggerManager) GetLogFiles() ([]string, error) {
	if dfw, ok := lm.fileWriter.(*DailyFileWriter); ok {
		return dfw.GetLogFiles()
	}
	return nil, fmt.Errorf("file logging not enabled")
}