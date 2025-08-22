package logging

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestNewContextLogger(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

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
			ctx:  context.WithValue(context.Background(), "request_id", "test-request-123"),
		},
		{
			name: "context with user ID",
			ctx:  context.WithValue(context.Background(), "user_id", int64(123)),
		},
		{
			name: "context with user email",
			ctx:  context.WithValue(context.Background(), "user_email", "test@example.com"),
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
			cl := NewContextLogger(logger, tt.ctx)
			assert.NotNil(t, cl)
			assert.NotNil(t, cl.fields)

			// Check that context values were extracted correctly
			if requestID := tt.ctx.Value("request_id"); requestID != nil {
				assert.Equal(t, requestID.(string), cl.GetRequestID())
			}
			if userID := tt.ctx.Value("user_id"); userID != nil {
				assert.Equal(t, userID.(int64), cl.GetUserID())
			}
			if userEmail := tt.ctx.Value("user_email"); userEmail != nil {
				assert.Equal(t, userEmail.(string), cl.GetUserEmail())
			}
		})
	}
}

func TestNewContextLoggerFromLogger(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	cl := NewContextLoggerFromLogger(logger)
	assert.NotNil(t, cl)
	assert.NotNil(t, cl.fields)
	assert.Empty(t, cl.GetRequestID())
	assert.Equal(t, int64(0), cl.GetUserID())
	assert.Empty(t, cl.GetUserEmail())
}

func TestContextLogger_WithRequestID(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)

	requestID := "test-request-123"
	newCL := cl.WithRequestID(requestID)

	// Original should be unchanged
	assert.Empty(t, cl.GetRequestID())
	// New logger should have the request ID
	assert.Equal(t, requestID, newCL.GetRequestID())
	assert.True(t, newCL.HasRequestID())
}

func TestContextLogger_WithUser(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)

	userID := int64(123)
	userEmail := "test@example.com"
	newCL := cl.WithUser(userID, userEmail)

	// Original should be unchanged
	assert.Equal(t, int64(0), cl.GetUserID())
	assert.Empty(t, cl.GetUserEmail())
	// New logger should have user context
	assert.Equal(t, userID, newCL.GetUserID())
	assert.Equal(t, userEmail, newCL.GetUserEmail())
	assert.True(t, newCL.HasUser())
}

func TestContextLogger_WithUserID(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)

	userID := int64(123)
	newCL := cl.WithUserID(userID)

	assert.Equal(t, int64(0), cl.GetUserID())
	assert.Equal(t, userID, newCL.GetUserID())
}

func TestContextLogger_WithUserEmail(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)

	userEmail := "test@example.com"
	newCL := cl.WithUserEmail(userEmail)

	assert.Empty(t, cl.GetUserEmail())
	assert.Equal(t, userEmail, newCL.GetUserEmail())
}

func TestContextLogger_WithFields(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)

	fields := map[string]interface{}{
		"string_field": "test_value",
		"int_field":    123,
		"bool_field":   true,
		"float_field":  3.14,
	}

	newCL := cl.WithFields(fields)

	// Original should be unchanged
	assert.Empty(t, cl.GetFields())
	// New logger should have the fields
	newFields := newCL.GetFields()
	assert.Len(t, newFields, 4)
	assert.Equal(t, "test_value", newFields["string_field"])
	assert.Equal(t, 123, newFields["int_field"])
	assert.Equal(t, true, newFields["bool_field"])
	assert.Equal(t, 3.14, newFields["float_field"])
}

func TestContextLogger_WithField(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)

	key := "test_key"
	value := "test_value"
	newCL := cl.WithField(key, value)

	assert.Empty(t, cl.GetFields())
	assert.True(t, newCL.HasField(key))
	assert.Equal(t, value, newCL.GetFields()[key])
}

func TestContextLogger_WithComponent(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)

	component := "test-component"
	newCL := cl.WithComponent(component)

	assert.False(t, cl.HasField("component"))
	assert.True(t, newCL.HasField("component"))
	assert.Equal(t, component, newCL.GetFields()["component"])
}

func TestContextLogger_WithOperation(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)

	operation := "test-operation"
	newCL := cl.WithOperation(operation)

	assert.False(t, cl.HasField("operation"))
	assert.True(t, newCL.HasField("operation"))
	assert.Equal(t, operation, newCL.GetFields()["operation"])
}

func TestContextLogger_WithCorrelationID(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)

	correlationID := "correlation-123"
	newCL := cl.WithCorrelationID(correlationID)

	assert.Empty(t, cl.GetRequestID())
	assert.Equal(t, correlationID, newCL.GetRequestID())
}

func TestContextLogger_LoggingMethods(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger).
		WithRequestID("test-request-123").
		WithUser(123, "test@example.com").
		WithComponent("test-component")

	// Test all logging levels
	cl.Trace().Msg("trace message")
	cl.Debug().Msg("debug message")
	cl.Info().Msg("info message")
	cl.Warn().Msg("warn message")
	cl.Error().Msg("error message")

	// Test Log method with specific level
	cl.Log(zerolog.InfoLevel).Msg("log with specific level")
}

func TestContextLogger_GetLogger(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger).
		WithRequestID("test-request-123").
		WithUser(123, "test@example.com").
		WithField("custom_field", "custom_value")

	contextLogger := cl.GetLogger()
	assert.NotNil(t, contextLogger)

	// Test that we can log with the context logger
	contextLogger.Info().Msg("test message with full context")
}

func TestContextLogger_HasMethods(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)

	// Initially should have no context
	assert.False(t, cl.HasRequestID())
	assert.False(t, cl.HasUser())
	assert.False(t, cl.HasField("test_field"))

	// Add context and test
	cl = cl.WithRequestID("test-request").
		WithUser(123, "test@example.com").
		WithField("test_field", "test_value")

	assert.True(t, cl.HasRequestID())
	assert.True(t, cl.HasUser())
	assert.True(t, cl.HasField("test_field"))
	assert.False(t, cl.HasField("nonexistent_field"))
}

func TestContextLogger_LogHTTPRequest(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger).
		WithRequestID("test-request-123")

	cl.LogHTTPRequest("GET", "/api/v1/users", 200, 150, "192.168.1.1")
}

func TestContextLogger_LogDatabaseOperation(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger).
		WithRequestID("test-request-123")

	cl.LogDatabaseOperation("SELECT", "users", 25, 5)
}

func TestContextLogger_LogError(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger).
		WithRequestID("test-request-123").
		WithUser(123, "test@example.com")

	err := errors.New("test error")
	additionalFields := map[string]interface{}{
		"error_code": "E001",
		"component":  "user-service",
	}

	cl.LogError(err, "Failed to process user request", additionalFields)
}

func TestContextLogger_LogBusinessEvent(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger).
		WithRequestID("test-request-123").
		WithUser(123, "test@example.com")

	additionalFields := map[string]interface{}{
		"account_id": 456,
		"amount":     100.50,
	}

	cl.LogBusinessEvent("user_registration", "New user registered successfully", additionalFields)
}

func TestContextLogger_LogPerformanceMetric(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger).
		WithRequestID("test-request-123")

	additionalFields := map[string]interface{}{
		"endpoint": "/api/v1/users",
		"method":   "GET",
	}

	cl.LogPerformanceMetric("response_time", 150.5, "ms", additionalFields)
}

func TestContextLogger_String(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger).
		WithRequestID("test-request-123").
		WithUser(123, "test@example.com").
		WithField("field1", "value1").
		WithField("field2", "value2")

	str := cl.String()
	assert.Contains(t, str, "test-request-123")
	assert.Contains(t, str, "123")
	assert.Contains(t, str, "test@example.com")
	assert.Contains(t, str, "fields: 2")
}

func TestContextLogger_Clone(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	original := NewContextLoggerFromLogger(logger).
		WithRequestID("test-request-123").
		WithUser(123, "test@example.com").
		WithField("field1", "value1")

	// Create a new logger with additional field
	cloned := original.WithField("field2", "value2")

	// Original should be unchanged
	assert.Len(t, original.GetFields(), 1)
	assert.True(t, original.HasField("field1"))
	assert.False(t, original.HasField("field2"))

	// Cloned should have both fields
	assert.Len(t, cloned.GetFields(), 2)
	assert.True(t, cloned.HasField("field1"))
	assert.True(t, cloned.HasField("field2"))

	// Both should have the same request ID and user context
	assert.Equal(t, original.GetRequestID(), cloned.GetRequestID())
	assert.Equal(t, original.GetUserID(), cloned.GetUserID())
	assert.Equal(t, original.GetUserEmail(), cloned.GetUserEmail())
}

func TestContextLogger_ChainedOperations(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	
	cl := NewContextLoggerFromLogger(logger).
		WithRequestID("test-request-123").
		WithUser(123, "test@example.com").
		WithComponent("user-service").
		WithOperation("create_user").
		WithField("custom_field", "custom_value")

	assert.Equal(t, "test-request-123", cl.GetRequestID())
	assert.Equal(t, int64(123), cl.GetUserID())
	assert.Equal(t, "test@example.com", cl.GetUserEmail())
	assert.True(t, cl.HasField("component"))
	assert.True(t, cl.HasField("operation"))
	assert.True(t, cl.HasField("custom_field"))

	// Test that we can log with all context
	cl.Info().Msg("chained operations test")
}

func TestContextLogger_ContextExtraction(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	tests := []struct {
		name     string
		ctx      context.Context
		expected struct {
			requestID string
			userID    int64
			userEmail string
		}
	}{
		{
			name: "invalid types in context",
			ctx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "request_id", 123) // wrong type
				ctx = context.WithValue(ctx, "user_id", "abc")  // wrong type
				ctx = context.WithValue(ctx, "user_email", 456) // wrong type
				return ctx
			}(),
			expected: struct {
				requestID string
				userID    int64
				userEmail string
			}{"", 0, ""},
		},
		{
			name: "nil values in context",
			ctx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, "request_id", nil)
				ctx = context.WithValue(ctx, "user_id", nil)
				ctx = context.WithValue(ctx, "user_email", nil)
				return ctx
			}(),
			expected: struct {
				requestID string
				userID    int64
				userEmail string
			}{"", 0, ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := NewContextLogger(logger, tt.ctx)
			assert.Equal(t, tt.expected.requestID, cl.GetRequestID())
			assert.Equal(t, tt.expected.userID, cl.GetUserID())
			assert.Equal(t, tt.expected.userEmail, cl.GetUserEmail())
		})
	}
}

// Benchmark tests
func BenchmarkContextLogger_WithField(b *testing.B) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cl.WithField("field", "value")
	}
}

func BenchmarkContextLogger_WithFields(b *testing.B) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger)
	fields := map[string]interface{}{
		"field1": "value1",
		"field2": 123,
		"field3": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cl.WithFields(fields)
	}
}

func BenchmarkContextLogger_Info(b *testing.B) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cl := NewContextLoggerFromLogger(logger).
		WithRequestID("test-request-123").
		WithUser(123, "test@example.com").
		WithField("component", "benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cl.Info().Int("iteration", i).Msg("benchmark message")
	}
}