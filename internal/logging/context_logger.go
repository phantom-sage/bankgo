package logging

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
)

// ContextLogger provides request-scoped logging with contextual information
type ContextLogger struct {
	logger    zerolog.Logger
	requestID string
	userID    int64
	userEmail string
	fields    map[string]interface{}
}

// NewContextLogger creates a new context logger from a base logger and context
func NewContextLogger(logger zerolog.Logger, ctx context.Context) *ContextLogger {
	cl := &ContextLogger{
		logger: logger,
		fields: make(map[string]interface{}),
	}

	// Extract context values if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			cl.requestID = id
		}
	}

	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(int64); ok {
			cl.userID = id
		}
	}

	if userEmail := ctx.Value("user_email"); userEmail != nil {
		if email, ok := userEmail.(string); ok {
			cl.userEmail = email
		}
	}

	return cl
}

// NewContextLoggerFromLogger creates a new context logger from a base logger without context
func NewContextLoggerFromLogger(logger zerolog.Logger) *ContextLogger {
	return &ContextLogger{
		logger: logger,
		fields: make(map[string]interface{}),
	}
}

// WithRequestID adds or updates the request ID
func (cl *ContextLogger) WithRequestID(requestID string) *ContextLogger {
	newCL := cl.clone()
	newCL.requestID = requestID
	return newCL
}

// WithUser adds or updates user context information
func (cl *ContextLogger) WithUser(userID int64, userEmail string) *ContextLogger {
	newCL := cl.clone()
	newCL.userID = userID
	newCL.userEmail = userEmail
	return newCL
}

// WithUserID adds or updates the user ID
func (cl *ContextLogger) WithUserID(userID int64) *ContextLogger {
	newCL := cl.clone()
	newCL.userID = userID
	return newCL
}

// WithUserEmail adds or updates the user email
func (cl *ContextLogger) WithUserEmail(userEmail string) *ContextLogger {
	newCL := cl.clone()
	newCL.userEmail = userEmail
	return newCL
}

// WithFields adds custom fields to the logger context
func (cl *ContextLogger) WithFields(fields map[string]interface{}) *ContextLogger {
	newCL := cl.clone()
	for key, value := range fields {
		newCL.fields[key] = value
	}
	return newCL
}

// WithField adds a single custom field to the logger context
func (cl *ContextLogger) WithField(key string, value interface{}) *ContextLogger {
	newCL := cl.clone()
	newCL.fields[key] = value
	return newCL
}

// WithComponent adds a component field to the logger context
func (cl *ContextLogger) WithComponent(component string) *ContextLogger {
	return cl.WithField("component", component)
}

// WithOperation adds an operation field to the logger context
func (cl *ContextLogger) WithOperation(operation string) *ContextLogger {
	return cl.WithField("operation", operation)
}

// WithCorrelationID adds a correlation ID field (alias for request ID)
func (cl *ContextLogger) WithCorrelationID(correlationID string) *ContextLogger {
	return cl.WithRequestID(correlationID)
}

// clone creates a copy of the context logger
func (cl *ContextLogger) clone() *ContextLogger {
	newFields := make(map[string]interface{})
	for key, value := range cl.fields {
		newFields[key] = value
	}

	return &ContextLogger{
		logger:    cl.logger,
		requestID: cl.requestID,
		userID:    cl.userID,
		userEmail: cl.userEmail,
		fields:    newFields,
	}
}

// buildEvent creates a zerolog event with all context fields
func (cl *ContextLogger) buildEvent(event *zerolog.Event) *zerolog.Event {
	// Add request ID if present
	if cl.requestID != "" {
		event = event.Str("request_id", cl.requestID)
	}

	// Add user context if present
	if cl.userID > 0 {
		event = event.Int64("user_id", cl.userID)
	}
	if cl.userEmail != "" {
		event = event.Str("user_email", cl.userEmail)
	}

	// Add custom fields
	for key, value := range cl.fields {
		event = event.Interface(key, value)
	}

	return event
}

// Trace returns a trace level event with context
func (cl *ContextLogger) Trace() *zerolog.Event {
	return cl.buildEvent(cl.logger.Trace())
}

// Debug returns a debug level event with context
func (cl *ContextLogger) Debug() *zerolog.Event {
	return cl.buildEvent(cl.logger.Debug())
}

// Info returns an info level event with context
func (cl *ContextLogger) Info() *zerolog.Event {
	return cl.buildEvent(cl.logger.Info())
}

// Warn returns a warn level event with context
func (cl *ContextLogger) Warn() *zerolog.Event {
	return cl.buildEvent(cl.logger.Warn())
}

// Error returns an error level event with context
func (cl *ContextLogger) Error() *zerolog.Event {
	return cl.buildEvent(cl.logger.Error())
}

// Fatal returns a fatal level event with context
func (cl *ContextLogger) Fatal() *zerolog.Event {
	return cl.buildEvent(cl.logger.Fatal())
}

// Panic returns a panic level event with context
func (cl *ContextLogger) Panic() *zerolog.Event {
	return cl.buildEvent(cl.logger.Panic())
}

// Log returns an event at the specified level with context
func (cl *ContextLogger) Log(level zerolog.Level) *zerolog.Event {
	return cl.buildEvent(cl.logger.WithLevel(level))
}

// GetLogger returns the underlying zerolog logger with all context applied
func (cl *ContextLogger) GetLogger() zerolog.Logger {
	logger := cl.logger

	// Add request ID if present
	if cl.requestID != "" {
		logger = logger.With().Str("request_id", cl.requestID).Logger()
	}

	// Add user context if present
	if cl.userID > 0 {
		logger = logger.With().Int64("user_id", cl.userID).Logger()
	}
	if cl.userEmail != "" {
		logger = logger.With().Str("user_email", cl.userEmail).Logger()
	}

	// Add custom fields
	for key, value := range cl.fields {
		logger = logger.With().Interface(key, value).Logger()
	}

	return logger
}

// GetRequestID returns the current request ID
func (cl *ContextLogger) GetRequestID() string {
	return cl.requestID
}

// GetUserID returns the current user ID
func (cl *ContextLogger) GetUserID() int64 {
	return cl.userID
}

// GetUserEmail returns the current user email
func (cl *ContextLogger) GetUserEmail() string {
	return cl.userEmail
}

// GetFields returns a copy of the custom fields
func (cl *ContextLogger) GetFields() map[string]interface{} {
	fields := make(map[string]interface{})
	for key, value := range cl.fields {
		fields[key] = value
	}
	return fields
}

// HasRequestID checks if a request ID is set
func (cl *ContextLogger) HasRequestID() bool {
	return cl.requestID != ""
}

// HasUser checks if user context is set
func (cl *ContextLogger) HasUser() bool {
	return cl.userID > 0 || cl.userEmail != ""
}

// HasField checks if a specific field is set
func (cl *ContextLogger) HasField(key string) bool {
	_, exists := cl.fields[key]
	return exists
}

// LogHTTPRequest logs an HTTP request with standard fields
func (cl *ContextLogger) LogHTTPRequest(method, path string, statusCode int, duration int64, clientIP string) {
	cl.Info().
		Str("method", method).
		Str("path", path).
		Int("status", statusCode).
		Int64("duration_ms", duration).
		Str("client_ip", clientIP).
		Msg("HTTP request completed")
}

// LogDatabaseOperation logs a database operation with standard fields
func (cl *ContextLogger) LogDatabaseOperation(operation, table string, duration int64, rowsAffected int64) {
	cl.Debug().
		Str("operation", operation).
		Str("table", table).
		Int64("duration_ms", duration).
		Int64("rows_affected", rowsAffected).
		Msg("Database operation completed")
}

// LogError logs an error with context and optional additional fields
func (cl *ContextLogger) LogError(err error, message string, additionalFields map[string]interface{}) {
	event := cl.Error().Err(err)
	
	for key, value := range additionalFields {
		event = event.Interface(key, value)
	}
	
	event.Msg(message)
}

// LogBusinessEvent logs a business-level event with context
func (cl *ContextLogger) LogBusinessEvent(eventType, description string, additionalFields map[string]interface{}) {
	event := cl.Info().
		Str("event_type", eventType).
		Str("description", description)
	
	for key, value := range additionalFields {
		event = event.Interface(key, value)
	}
	
	event.Msg("Business event occurred")
}

// LogPerformanceMetric logs a performance metric with context
func (cl *ContextLogger) LogPerformanceMetric(metricName string, value float64, unit string, additionalFields map[string]interface{}) {
	event := cl.Info().
		Str("metric_name", metricName).
		Float64("value", value).
		Str("unit", unit)
	
	for key, value := range additionalFields {
		event = event.Interface(key, value)
	}
	
	event.Msg("Performance metric recorded")
}

// String returns a string representation of the context logger
func (cl *ContextLogger) String() string {
	return fmt.Sprintf("ContextLogger{requestID: %s, userID: %d, userEmail: %s, fields: %d}",
		cl.requestID, cl.userID, cl.userEmail, len(cl.fields))
}

// Ensure ContextLogger implements common interfaces
var _ fmt.Stringer = (*ContextLogger)(nil)