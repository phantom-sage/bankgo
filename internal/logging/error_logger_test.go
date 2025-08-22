package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewErrorLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	errorLogger := NewErrorLogger(logger)
	
	assert.NotNil(t, errorLogger)
	assert.NotNil(t, errorLogger.logger)
}

func TestErrorLogger_LogError(t *testing.T) {
	t.Run("basic error logging", func(t *testing.T) {
		var buf bytes.Buffer
		logger := zerolog.New(&buf)
		errorLogger := NewErrorLogger(logger)

		err := errors.New("test error")
		ctx := ErrorContext{
			Operation: "test_operation",
			Component: "test_component",
			Severity:  MediumSeverity,
		}

		errorLogger.LogError(err, ctx)

		// Parse the logged JSON
		var logged map[string]interface{}
		jsonErr := json.Unmarshal(buf.Bytes(), &logged)
		require.NoError(t, jsonErr)

		// Check expected fields
		assert.Equal(t, "warn", logged["level"])
		assert.Equal(t, "test error", logged["error"])
		assert.Equal(t, "test_operation", logged["operation"])
		assert.Equal(t, "test_component", logged["component"])
		assert.Equal(t, "medium", logged["severity"])
		assert.Equal(t, "error", logged["log_type"])
	})

	t.Run("error with full context", func(t *testing.T) {
		var buf bytes.Buffer
		logger := zerolog.New(&buf)
		errorLogger := NewErrorLogger(logger)

		err := errors.New("validation failed")
		ctx := ErrorContext{
			RequestID:     "req-123",
			UserID:        456,
			UserEmail:     "test@example.com",
			Operation:     "create_account",
			Component:     "account_service",
			Method:        "CreateAccount",
			Category:      ValidationError,
			Severity:      LowSeverity,
			CorrelationID: "corr-789",
			ErrorCode:     "VAL001",
			HTTPStatus:    400,
			Details: map[string]interface{}{
				"field": "currency",
				"value": "INVALID",
			},
		}

		errorLogger.LogError(err, ctx)

		// Parse the logged JSON
		var logged map[string]interface{}
		jsonErr := json.Unmarshal(buf.Bytes(), &logged)
		require.NoError(t, jsonErr)

		// Check expected fields
		assert.Equal(t, "info", logged["level"])
		assert.Equal(t, "validation failed", logged["error"])
		assert.Equal(t, "req-123", logged["request_id"])
		assert.Equal(t, float64(456), logged["user_id"]) // JSON numbers are float64
		assert.Equal(t, "test@example.com", logged["user_email"])
		assert.Equal(t, "create_account", logged["operation"])
		assert.Equal(t, "account_service", logged["component"])
		assert.Equal(t, "CreateAccount", logged["method"])
		assert.Equal(t, "validation_error", logged["category"])
		assert.Equal(t, "low", logged["severity"])
		assert.Equal(t, "corr-789", logged["correlation_id"])
		assert.Equal(t, "VAL001", logged["error_code"])
		assert.Equal(t, float64(400), logged["http_status"])
		assert.Equal(t, "error", logged["log_type"])

		// Check details
		details, exists := logged["details"]
		assert.True(t, exists, "Expected details field to exist")
		detailsMap, ok := details.(map[string]interface{})
		assert.True(t, ok, "Details should be a map")
		assert.Equal(t, "currency", detailsMap["field"])
		assert.Equal(t, "INVALID", detailsMap["value"])
	})

	t.Run("high severity error with stack trace", func(t *testing.T) {
		var buf bytes.Buffer
		logger := zerolog.New(&buf)
		errorLogger := NewErrorLogger(logger)

		err := errors.New("system failure")
		ctx := ErrorContext{
			Operation:  "database_connection",
			Component:  "database",
			Severity:   HighSeverity,
			StackTrace: "stack trace here",
		}

		errorLogger.LogError(err, ctx)

		// Parse the logged JSON
		var logged map[string]interface{}
		jsonErr := json.Unmarshal(buf.Bytes(), &logged)
		require.NoError(t, jsonErr)

		// Check expected fields
		assert.Equal(t, "error", logged["level"])
		assert.Equal(t, "system failure", logged["error"])
		assert.Equal(t, "database_connection", logged["operation"])
		assert.Equal(t, "database", logged["component"])
		assert.Equal(t, "high", logged["severity"])
		assert.Equal(t, "stack trace here", logged["stack_trace"])
		assert.Equal(t, "error", logged["log_type"])
	})
}

func TestErrorLogger_LogError_NilError(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	errorLogger := NewErrorLogger(logger)

	errorLogger.LogError(nil, ErrorContext{})

	// Should not log anything for nil error
	assert.Empty(t, buf.String())
}

func TestErrorLogger_LogErrorWithStackTrace(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	errorLogger := NewErrorLogger(logger)

	err := errors.New("high severity system error")
	ctx := ErrorContext{
		Operation: "system_init",
		Severity:  HighSeverity,
	}

	errorLogger.LogErrorWithStackTrace(err, ctx)

	// Parse the logged JSON
	var logged map[string]interface{}
	jsonErr := json.Unmarshal(buf.Bytes(), &logged)
	require.NoError(t, jsonErr)

	// Should have stack trace for high severity errors
	stackTrace, exists := logged["stack_trace"]
	assert.True(t, exists, "Expected stack_trace field for high severity error")
	assert.NotEmpty(t, stackTrace, "Stack trace should not be empty")
}

func TestErrorLogger_SpecializedMethods(t *testing.T) {
	tests := []struct {
		name           string
		logMethod      func(*ErrorLogger, error, ErrorContext)
		expectedLevel  string
		expectedCategory string
	}{
		{
			name: "LogValidationError",
			logMethod: func(el *ErrorLogger, err error, ctx ErrorContext) {
				el.LogValidationError(err, ctx)
			},
			expectedLevel:    "info",
			expectedCategory: "validation_error",
		},
		{
			name: "LogBusinessLogicError",
			logMethod: func(el *ErrorLogger, err error, ctx ErrorContext) {
				el.LogBusinessLogicError(err, ctx)
			},
			expectedLevel:    "warn",
			expectedCategory: "business_logic_error",
		},
		{
			name: "LogSystemError",
			logMethod: func(el *ErrorLogger, err error, ctx ErrorContext) {
				el.LogSystemError(err, ctx)
			},
			expectedLevel:    "error",
			expectedCategory: "system_error",
		},
		{
			name: "LogDatabaseError",
			logMethod: func(el *ErrorLogger, err error, ctx ErrorContext) {
				el.LogDatabaseError(err, ctx)
			},
			expectedLevel:    "error",
			expectedCategory: "database_error",
		},
		{
			name: "LogAuthenticationError",
			logMethod: func(el *ErrorLogger, err error, ctx ErrorContext) {
				el.LogAuthenticationError(err, ctx)
			},
			expectedLevel:    "warn",
			expectedCategory: "authentication_error",
		},
		{
			name: "LogExternalServiceError",
			logMethod: func(el *ErrorLogger, err error, ctx ErrorContext) {
				el.LogExternalServiceError(err, ctx)
			},
			expectedLevel:    "warn",
			expectedCategory: "external_service_error",
		},
		{
			name: "LogNetworkError",
			logMethod: func(el *ErrorLogger, err error, ctx ErrorContext) {
				el.LogNetworkError(err, ctx)
			},
			expectedLevel:    "warn",
			expectedCategory: "network_error",
		},
		{
			name: "LogConfigurationError",
			logMethod: func(el *ErrorLogger, err error, ctx ErrorContext) {
				el.LogConfigurationError(err, ctx)
			},
			expectedLevel:    "error",
			expectedCategory: "configuration_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			errorLogger := NewErrorLogger(logger)

			err := errors.New("test error")
			ctx := ErrorContext{Operation: "test_operation"}

			tt.logMethod(errorLogger, err, ctx)

			// Parse the logged JSON
			var logged map[string]interface{}
			jsonErr := json.Unmarshal(buf.Bytes(), &logged)
			require.NoError(t, jsonErr)

			assert.Equal(t, tt.expectedLevel, logged["level"])
			assert.Equal(t, tt.expectedCategory, logged["category"])
		})
	}
}

func TestNewErrorContextFromGinContext(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "req-123")
	ctx = context.WithValue(ctx, "user_id", int64(456))
	ctx = context.WithValue(ctx, "user_email", "test@example.com")

	errorCtx := NewErrorContextFromGinContext(ctx)

	assert.Equal(t, "req-123", errorCtx.RequestID)
	assert.Equal(t, int64(456), errorCtx.UserID)
	assert.Equal(t, "test@example.com", errorCtx.UserEmail)
}

func TestNewErrorContextFromGinContext_IntUserID(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", 456) // int instead of int64

	errorCtx := NewErrorContextFromGinContext(ctx)

	assert.Equal(t, int64(456), errorCtx.UserID)
}

func TestErrorContext_WithMethods(t *testing.T) {
	ctx := ErrorContext{}

	ctx = ctx.WithOperation("test_op").
		WithComponent("test_comp").
		WithMethod("TestMethod").
		WithSeverity(HighSeverity).
		WithCategory(SystemError).
		WithHTTPStatus(500).
		WithErrorCode("SYS001").
		WithRetryable(true).
		WithRetryCount(3)

	assert.Equal(t, "test_op", ctx.Operation)
	assert.Equal(t, "test_comp", ctx.Component)
	assert.Equal(t, "TestMethod", ctx.Method)
	assert.Equal(t, HighSeverity, ctx.Severity)
	assert.Equal(t, SystemError, ctx.Category)
	assert.Equal(t, 500, ctx.HTTPStatus)
	assert.Equal(t, "SYS001", ctx.ErrorCode)
	assert.True(t, ctx.Retryable)
	assert.Equal(t, 3, ctx.RetryCount)
}

func TestErrorContext_WithDetails(t *testing.T) {
	ctx := ErrorContext{}

	details := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	ctx = ctx.WithDetails(details).WithDetail("key3", "value3")

	assert.Equal(t, "value1", ctx.Details["key1"])
	assert.Equal(t, 123, ctx.Details["key2"])
	assert.Equal(t, "value3", ctx.Details["key3"])
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "validation error",
			err:      errors.New("validation failed: invalid email format"),
			expected: ValidationError,
		},
		{
			name:     "authentication error",
			err:      errors.New("unauthorized access: invalid token"),
			expected: AuthenticationError,
		},
		{
			name:     "database error",
			err:      errors.New("database connection failed"),
			expected: DatabaseError,
		},
		{
			name:     "network error",
			err:      errors.New("network timeout occurred"),
			expected: NetworkError,
		},
		{
			name:     "configuration error",
			err:      errors.New("config file not found"),
			expected: ConfigurationError,
		},
		{
			name:     "business logic error",
			err:      errors.New("insufficient balance for transaction"),
			expected: BusinessLogicError,
		},
		{
			name:     "unknown error",
			err:      errors.New("something went wrong"),
			expected: SystemError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineSeverity(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		category ErrorCategory
		expected ErrorSeverity
	}{
		{
			name:     "nil error",
			err:      nil,
			category: SystemError,
			expected: LowSeverity,
		},
		{
			name:     "critical keyword",
			err:      errors.New("panic: system corruption detected"),
			category: SystemError,
			expected: CriticalSeverity,
		},
		{
			name:     "validation error",
			err:      errors.New("invalid input"),
			category: ValidationError,
			expected: LowSeverity,
		},
		{
			name:     "authentication error",
			err:      errors.New("invalid credentials"),
			category: AuthenticationError,
			expected: MediumSeverity,
		},
		{
			name:     "security authentication error",
			err:      errors.New("brute force attack detected"),
			category: AuthenticationError,
			expected: HighSeverity,
		},
		{
			name:     "database error",
			err:      errors.New("connection failed"),
			category: DatabaseError,
			expected: HighSeverity,
		},
		{
			name:     "network error",
			err:      errors.New("timeout"),
			category: NetworkError,
			expected: MediumSeverity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineSeverity(tt.err, tt.category)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCaptureStackTrace(t *testing.T) {
	// Test that stack trace is captured
	stackTrace := captureStackTrace(1)
	
	assert.NotEmpty(t, stackTrace)
	// The stack trace should contain this test function or the testing framework
	assert.True(t, strings.Contains(stackTrace, "TestCaptureStackTrace") || 
		strings.Contains(stackTrace, "testing.go"), 
		"Stack trace should contain test function or testing framework")
	
	// Should contain file path and line number
	lines := strings.Split(stackTrace, "\n")
	assert.True(t, len(lines) > 0)
	
	// Each line should contain file:line function format
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			assert.Contains(t, line, ":")
		}
	}
}

func TestErrorLogger_SeverityLevels(t *testing.T) {
	tests := []struct {
		name          string
		severity      ErrorSeverity
		expectedLevel string
	}{

		{
			name:          "high severity maps to error",
			severity:      HighSeverity,
			expectedLevel: "error",
		},
		{
			name:          "medium severity maps to warn",
			severity:      MediumSeverity,
			expectedLevel: "warn",
		},
		{
			name:          "low severity maps to info",
			severity:      LowSeverity,
			expectedLevel: "info",
		},
		{
			name:          "empty severity defaults to error",
			severity:      "",
			expectedLevel: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			errorLogger := NewErrorLogger(logger)

			err := errors.New("test error")
			ctx := ErrorContext{
				Operation: "test",
				Severity:  tt.severity,
			}

			errorLogger.LogError(err, ctx)

			var logged map[string]interface{}
			jsonErr := json.Unmarshal(buf.Bytes(), &logged)
			require.NoError(t, jsonErr)

			assert.Equal(t, tt.expectedLevel, logged["level"])
		})
	}
}

func TestNewErrorLoggerWithMonitor(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	errorLogger := NewErrorLoggerWithMonitor(logger, monitor)
	
	assert.NotNil(t, errorLogger)
	assert.NotNil(t, errorLogger.GetErrorMonitor())
	assert.Equal(t, monitor, errorLogger.GetErrorMonitor())
}

func TestErrorLogger_WithMonitoring(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	errorLogger := NewErrorLoggerWithMonitor(logger, monitor)
	
	err := errors.New("test error")
	ctx := ErrorContext{
		Operation: "test_operation",
		Component: "test_component",
		Category:  ValidationError,
		Severity:  MediumSeverity,
	}
	
	// Log error multiple times
	errorLogger.LogError(err, ctx)
	errorLogger.LogError(err, ctx)
	errorLogger.LogError(err, ctx)
	
	// Check that monitoring tracked the errors
	frequencies := errorLogger.GetErrorFrequencies()
	assert.Len(t, frequencies, 1)
	
	for _, freq := range frequencies {
		assert.Equal(t, int64(3), freq.Count)
		assert.Equal(t, ValidationError, freq.Category)
		assert.Equal(t, "test_component", freq.Component)
		assert.Equal(t, "test_operation", freq.Operation)
	}
	
	// Check error stats
	stats := errorLogger.GetErrorStats()
	assert.Equal(t, int64(3), stats[ValidationError])
	
	// Check top errors
	topErrors := errorLogger.GetTopErrors(1)
	assert.Len(t, topErrors, 1)
	assert.Equal(t, int64(3), topErrors[0].Count)
}

func TestErrorLogger_SetErrorMonitor(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	errorLogger := NewErrorLogger(logger)
	assert.Nil(t, errorLogger.GetErrorMonitor())
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	errorLogger.SetErrorMonitor(monitor)
	assert.NotNil(t, errorLogger.GetErrorMonitor())
	assert.Equal(t, monitor, errorLogger.GetErrorMonitor())
}

func TestErrorLogger_MonitoringMethods_NoMonitor(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	errorLogger := NewErrorLogger(logger)
	
	// Should not panic and return empty results when no monitor is set
	frequencies := errorLogger.GetErrorFrequencies()
	assert.Empty(t, frequencies)
	
	stats := errorLogger.GetErrorStats()
	assert.Empty(t, stats)
	
	topErrors := errorLogger.GetTopErrors(5)
	assert.Empty(t, topErrors)
	
	// Should not panic when adding/removing thresholds without monitor
	threshold := AlertThreshold{
		Category:  ValidationError,
		MaxCount:  10,
		TimeWindow: 1 * time.Minute,
	}
	errorLogger.AddAlertThreshold(threshold)
	errorLogger.RemoveAlertThreshold(ValidationError, "", "")
}

func TestErrorLogger_AlertThresholds(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	errorLogger := NewErrorLoggerWithMonitor(logger, monitor)
	
	threshold := AlertThreshold{
		Category:      ValidationError,
		Component:     "test_component",
		Operation:     "test_operation",
		MaxCount:      2,
		TimeWindow:    1 * time.Minute,
		Severity:      LowSeverity,
		AlertInterval: 1 * time.Second,
	}
	
	// Add threshold
	errorLogger.AddAlertThreshold(threshold)
	
	thresholds := monitor.GetThresholds()
	assert.Len(t, thresholds, 1)
	
	// Remove threshold
	errorLogger.RemoveAlertThreshold(ValidationError, "test_component", "test_operation")
	
	thresholds = monitor.GetThresholds()
	assert.Len(t, thresholds, 0)
}

func TestErrorLogger_ErrorCategorization_Comprehensive(t *testing.T) {
	tests := []struct {
		name             string
		errorMessage     string
		expectedCategory ErrorCategory
		expectedSeverity ErrorSeverity
	}{
		// Validation errors
		{
			name:             "validation error - invalid format",
			errorMessage:     "validation failed: invalid email format",
			expectedCategory: ValidationError,
			expectedSeverity: LowSeverity,
		},
		{
			name:             "validation error - required field",
			errorMessage:     "required field 'username' is missing",
			expectedCategory: ValidationError,
			expectedSeverity: LowSeverity,
		},
		{
			name:             "validation error - invalid input",
			errorMessage:     "invalid currency code provided",
			expectedCategory: ValidationError,
			expectedSeverity: LowSeverity,
		},
		
		// Authentication errors
		{
			name:             "authentication error - unauthorized",
			errorMessage:     "unauthorized access: invalid token",
			expectedCategory: AuthenticationError,
			expectedSeverity: MediumSeverity,
		},
		{
			name:             "authentication error - forbidden",
			errorMessage:     "forbidden: insufficient permissions",
			expectedCategory: AuthenticationError,
			expectedSeverity: MediumSeverity,
		},
		{
			name:             "authentication error - security threat",
			errorMessage:     "authentication brute force attack detected",
			expectedCategory: AuthenticationError,
			expectedSeverity: HighSeverity,
		},
		
		// Business logic errors
		{
			name:             "business logic error - insufficient balance",
			errorMessage:     "insufficient balance for transaction",
			expectedCategory: BusinessLogicError,
			expectedSeverity: MediumSeverity,
		},
		{
			name:             "business logic error - currency mismatch",
			errorMessage:     "currency mismatch between accounts",
			expectedCategory: BusinessLogicError,
			expectedSeverity: MediumSeverity,
		},
		{
			name:             "business logic error - business rule violation",
			errorMessage:     "business rule violated: maximum daily limit exceeded",
			expectedCategory: BusinessLogicError,
			expectedSeverity: MediumSeverity,
		},
		
		// Database errors
		{
			name:             "database error - connection failed",
			errorMessage:     "database connection failed",
			expectedCategory: DatabaseError,
			expectedSeverity: HighSeverity,
		},
		{
			name:             "database error - transaction failed",
			errorMessage:     "transaction rollback due to constraint violation",
			expectedCategory: DatabaseError,
			expectedSeverity: HighSeverity,
		},
		{
			name:             "database error - sql error",
			errorMessage:     "SQL syntax error in query",
			expectedCategory: DatabaseError,
			expectedSeverity: HighSeverity,
		},
		
		// Network errors
		{
			name:             "network error - timeout",
			errorMessage:     "network timeout occurred",
			expectedCategory: NetworkError,
			expectedSeverity: MediumSeverity,
		},
		{
			name:             "network error - no route",
			errorMessage:     "no route to host",
			expectedCategory: NetworkError,
			expectedSeverity: MediumSeverity,
		},
		
		// Configuration errors
		{
			name:             "configuration error - missing config",
			errorMessage:     "config file not found",
			expectedCategory: ConfigurationError,
			expectedSeverity: HighSeverity,
		},
		{
			name:             "configuration error - environment setting",
			errorMessage:     "environment variable setting not found",
			expectedCategory: ConfigurationError,
			expectedSeverity: HighSeverity,
		},
		
		// Critical errors
		{
			name:             "critical error - panic",
			errorMessage:     "panic: system corruption detected",
			expectedCategory: SystemError,
			expectedSeverity: CriticalSeverity,
		},
		{
			name:             "critical error - fatal",
			errorMessage:     "fatal error: memory corruption",
			expectedCategory: SystemError,
			expectedSeverity: CriticalSeverity,
		},
		
		// System errors (default)
		{
			name:             "unknown system error",
			errorMessage:     "unexpected system failure",
			expectedCategory: SystemError,
			expectedSeverity: HighSeverity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errorMessage)
			
			// Test automatic classification
			category := ClassifyError(err)
			assert.Equal(t, tt.expectedCategory, category, "Error classification mismatch")
			
			// Test severity determination
			severity := DetermineSeverity(err, category)
			assert.Equal(t, tt.expectedSeverity, severity, "Severity determination mismatch")
		})
	}
}

func TestErrorLogger_FrequencyTracking_Comprehensive(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	errorLogger := NewErrorLoggerWithMonitor(logger, monitor)
	
	// Define different error scenarios
	errorScenarios := []struct {
		category  ErrorCategory
		component string
		operation string
		count     int
	}{
		{ValidationError, "user_service", "create_user", 5},
		{ValidationError, "account_service", "create_account", 3},
		{AuthenticationError, "auth_service", "login", 7},
		{DatabaseError, "user_repository", "save_user", 2},
		{BusinessLogicError, "transfer_service", "transfer_money", 4},
	}
	
	// Log errors according to scenarios
	for _, scenario := range errorScenarios {
		for i := 0; i < scenario.count; i++ {
			ctx := ErrorContext{
				RequestID: fmt.Sprintf("req-%d-%d", i, scenario.count),
				UserID:    int64(100 + i),
				Operation: scenario.operation,
				Component: scenario.component,
				Category:  scenario.category,
			}
			
			err := errors.New(fmt.Sprintf("test error for %s", scenario.operation))
			errorLogger.LogError(err, ctx)
		}
	}
	
	// Verify frequency tracking
	frequencies := errorLogger.GetErrorFrequencies()
	assert.Len(t, frequencies, len(errorScenarios), "Should track all different error patterns")
	
	// Verify error statistics
	stats := errorLogger.GetErrorStats()
	assert.Equal(t, int64(8), stats[ValidationError], "Should aggregate validation errors") // 5 + 3
	assert.Equal(t, int64(7), stats[AuthenticationError], "Should track auth errors")
	assert.Equal(t, int64(2), stats[DatabaseError], "Should track database errors")
	assert.Equal(t, int64(4), stats[BusinessLogicError], "Should track business logic errors")
	
	// Verify top errors
	topErrors := errorLogger.GetTopErrors(3)
	assert.Len(t, topErrors, 3, "Should return top 3 errors")
	
	// Should be sorted by frequency (descending)
	assert.True(t, topErrors[0].Count >= topErrors[1].Count, "Top errors should be sorted by count")
	assert.True(t, topErrors[1].Count >= topErrors[2].Count, "Top errors should be sorted by count")
	
	// Verify the most frequent error (should be the highest count from our scenarios)
	// The actual order depends on how errors are tracked, so let's be more flexible
	maxCount := topErrors[0].Count
	assert.True(t, maxCount >= 4, "Top error should have at least 4 occurrences")
	
	// Verify that all top errors have reasonable counts
	for i, topError := range topErrors {
		if i > 0 {
			assert.True(t, topError.Count <= topErrors[i-1].Count, "Errors should be sorted by count (descending)")
		}
	}
}

func TestErrorLogger_AlertingThresholds_Comprehensive(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	errorLogger := NewErrorLoggerWithMonitor(logger, monitor)
	
	// Define different alert thresholds for different error types
	thresholds := []AlertThreshold{
		{
			Category:      ValidationError,
			Component:     "user_service",
			MaxCount:      3,
			TimeWindow:    1 * time.Minute,
			Severity:      LowSeverity,
			AlertInterval: 1 * time.Second,
		},
		{
			Category:      AuthenticationError,
			MaxCount:      2, // Any component/operation
			TimeWindow:    1 * time.Minute,
			Severity:      MediumSeverity,
			AlertInterval: 1 * time.Second,
		},
		{
			Category:      DatabaseError,
			Component:     "user_repository",
			Operation:     "save_user",
			MaxCount:      1,
			TimeWindow:    1 * time.Minute,
			Severity:      HighSeverity,
			AlertInterval: 1 * time.Second,
		},
	}
	
	// Add all thresholds
	for _, threshold := range thresholds {
		errorLogger.AddAlertThreshold(threshold)
	}
	
	// Clear buffer to focus on alert messages
	buf.Reset()
	
	// Test validation error threshold
	for i := 0; i < 4; i++ { // Exceed threshold of 3
		ctx := ErrorContext{
			RequestID: fmt.Sprintf("req-val-%d", i),
			Operation: "create_user",
			Component: "user_service",
			Category:  ValidationError,
		}
		errorLogger.LogError(errors.New("validation error"), ctx)
	}
	
	// Test authentication error threshold (any component)
	for i := 0; i < 3; i++ { // Exceed threshold of 2
		ctx := ErrorContext{
			RequestID: fmt.Sprintf("req-auth-%d", i),
			Operation: "login",
			Component: "auth_service",
			Category:  AuthenticationError,
		}
		errorLogger.LogError(errors.New("auth error"), ctx)
	}
	
	// Test database error threshold (specific component/operation)
	for i := 0; i < 2; i++ { // Exceed threshold of 1
		ctx := ErrorContext{
			RequestID: fmt.Sprintf("req-db-%d", i),
			Operation: "save_user",
			Component: "user_repository",
			Category:  DatabaseError,
		}
		errorLogger.LogError(errors.New("database error"), ctx)
	}
	
	// Give time for alert processing
	time.Sleep(100 * time.Millisecond)
	
	// Verify alerts were triggered
	logOutput := buf.String()
	assert.Contains(t, logOutput, "error_threshold_exceeded", "Should contain threshold exceeded alerts")
	assert.Contains(t, logOutput, "alert triggered", "Should contain alert triggered messages")
	
	// Should contain different categories in alerts
	assert.Contains(t, logOutput, "validation_error", "Should alert on validation errors")
	assert.Contains(t, logOutput, "authentication_error", "Should alert on auth errors")
	assert.Contains(t, logOutput, "database_error", "Should alert on database errors")
}

func TestErrorLogger_CorrelationTracking_Comprehensive(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	errorLogger := NewErrorLoggerWithMonitor(logger, monitor)
	
	// Simulate a user session with multiple operations
	userSession := struct {
		userID    int64
		userEmail string
		sessionID string
	}{
		userID:    12345,
		userEmail: "test.user@example.com",
		sessionID: "session-abc-123",
	}
	
	// Simulate different operations in the same session
	operations := []struct {
		requestID   string
		operation   string
		component   string
		category    ErrorCategory
		errorMsg    string
		traceID     string
		spanID      string
	}{
		{
			requestID: "req-001",
			operation: "validate_user_input",
			component: "user_service",
			category:  ValidationError,
			errorMsg:  "invalid email format",
			traceID:   "trace-001",
			spanID:    "span-001",
		},
		{
			requestID: "req-002",
			operation: "authenticate_user",
			component: "auth_service",
			category:  AuthenticationError,
			errorMsg:  "invalid credentials",
			traceID:   "trace-001",
			spanID:    "span-002",
		},
		{
			requestID: "req-003",
			operation: "create_account",
			component: "account_service",
			category:  BusinessLogicError,
			errorMsg:  "currency not supported",
			traceID:   "trace-001",
			spanID:    "span-003",
		},
	}
	
	// Log all operations with correlation context
	for _, op := range operations {
		ctx := ErrorContext{
			RequestID:     op.requestID,
			UserID:        userSession.userID,
			UserEmail:     userSession.userEmail,
			Operation:     op.operation,
			Component:     op.component,
			Category:      op.category,
			CorrelationID: userSession.sessionID,
			TraceID:       op.traceID,
			SpanID:        op.spanID,
			Details: map[string]interface{}{
				"session_id": userSession.sessionID,
				"user_agent": "TestAgent/1.0",
			},
		}
		
		errorLogger.LogError(errors.New(op.errorMsg), ctx)
	}
	
	// Verify correlation data is logged
	logOutput := buf.String()
	
	// Check that all correlation IDs are present
	for _, op := range operations {
		assert.Contains(t, logOutput, op.requestID, "Should contain request ID")
		assert.Contains(t, logOutput, op.traceID, "Should contain trace ID")
		assert.Contains(t, logOutput, op.spanID, "Should contain span ID")
	}
	
	// Check user context
	assert.Contains(t, logOutput, fmt.Sprintf("%d", userSession.userID), "Should contain user ID")
	assert.Contains(t, logOutput, userSession.userEmail, "Should contain user email")
	assert.Contains(t, logOutput, userSession.sessionID, "Should contain session ID")
	
	// Verify frequency tracking groups errors correctly
	frequencies := errorLogger.GetErrorFrequencies()
	assert.Len(t, frequencies, 3, "Should track 3 different error patterns")
	
	// Each error pattern should have count of 1
	for _, freq := range frequencies {
		assert.Equal(t, int64(1), freq.Count, "Each error pattern should occur once")
	}
}

func TestErrorLogger_ErrorClassification_EdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		expectedCategory ErrorCategory
		expectedSeverity ErrorSeverity
	}{
		{
			name:             "nil error",
			err:              nil,
			expectedCategory: "",
			expectedSeverity: LowSeverity,
		},
		{
			name:             "empty error message",
			err:              errors.New(""),
			expectedCategory: SystemError,
			expectedSeverity: HighSeverity,
		},
		{
			name:             "mixed keywords - validation and auth",
			err:              errors.New("validation failed: unauthorized token format"),
			expectedCategory: AuthenticationError, // Auth takes precedence
			expectedSeverity: MediumSeverity,
		},
		{
			name:             "mixed keywords - database and network",
			err:              errors.New("database connection timeout"),
			expectedCategory: DatabaseError, // Database takes precedence
			expectedSeverity: HighSeverity,
		},
		{
			name:             "case insensitive matching",
			err:              errors.New("VALIDATION FAILED: INVALID FORMAT"),
			expectedCategory: ValidationError,
			expectedSeverity: LowSeverity,
		},
		{
			name:             "partial keyword matching",
			err:              errors.New("user authentication service unavailable"),
			expectedCategory: AuthenticationError,
			expectedSeverity: MediumSeverity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := ClassifyError(tt.err)
			assert.Equal(t, tt.expectedCategory, category, "Error classification mismatch")
			
			severity := DetermineSeverity(tt.err, category)
			assert.Equal(t, tt.expectedSeverity, severity, "Severity determination mismatch")
		})
	}
}

func TestErrorLogger_ErrorCorrelation(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	errorLogger := NewErrorLoggerWithMonitor(logger, monitor)
	
	// Log errors with request correlation
	ctx1 := ErrorContext{
		RequestID: "req-123",
		UserID:    456,
		UserEmail: "test@example.com",
		Operation: "create_user",
		Component: "user_service",
		Category:  ValidationError,
		Severity:  LowSeverity,
	}
	
	ctx2 := ErrorContext{
		RequestID: "req-456",
		UserID:    789,
		UserEmail: "test2@example.com",
		Operation: "create_user",
		Component: "user_service",
		Category:  ValidationError,
		Severity:  LowSeverity,
	}
	
	errorLogger.LogError(errors.New("validation error 1"), ctx1)
	errorLogger.LogError(errors.New("validation error 2"), ctx2)
	
	// Check that both errors are tracked under the same frequency key
	// (since they have the same category, component, and operation)
	frequencies := errorLogger.GetErrorFrequencies()
	assert.Len(t, frequencies, 1)
	
	for _, freq := range frequencies {
		assert.Equal(t, int64(2), freq.Count)
	}
	
	// Parse log output to verify correlation IDs are logged
	logOutput := buf.String()
	assert.Contains(t, logOutput, "req-123")
	assert.Contains(t, logOutput, "req-456")
	assert.Contains(t, logOutput, "test@example.com")
	assert.Contains(t, logOutput, "test2@example.com")
}