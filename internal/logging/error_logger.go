package logging

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/rs/zerolog"
)

// ErrorCategory represents different types of errors for classification
type ErrorCategory string

const (
	// ValidationError represents input validation failures
	ValidationError ErrorCategory = "validation_error"
	// BusinessLogicError represents business rule violations
	BusinessLogicError ErrorCategory = "business_logic_error"
	// SystemError represents infrastructure/system failures
	SystemError ErrorCategory = "system_error"
	// AuthenticationError represents authentication/authorization failures
	AuthenticationError ErrorCategory = "authentication_error"
	// ExternalServiceError represents failures from external services
	ExternalServiceError ErrorCategory = "external_service_error"
	// DatabaseError represents database operation failures
	DatabaseError ErrorCategory = "database_error"
	// NetworkError represents network-related failures
	NetworkError ErrorCategory = "network_error"
	// ConfigurationError represents configuration-related failures
	ConfigurationError ErrorCategory = "configuration_error"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	// CriticalSeverity represents critical errors that require immediate attention
	CriticalSeverity ErrorSeverity = "critical"
	// HighSeverity represents high-priority errors
	HighSeverity ErrorSeverity = "high"
	// MediumSeverity represents medium-priority errors
	MediumSeverity ErrorSeverity = "medium"
	// LowSeverity represents low-priority errors
	LowSeverity ErrorSeverity = "low"
)

// ErrorContext provides structured context for error logging
type ErrorContext struct {
	// Request context information
	RequestID   string `json:"request_id,omitempty"`
	UserID      int64  `json:"user_id,omitempty"`
	UserEmail   string `json:"user_email,omitempty"`
	
	// Operation context
	Operation   string `json:"operation,omitempty"`
	Component   string `json:"component,omitempty"`
	Method      string `json:"method,omitempty"`
	
	// Error classification
	Category    ErrorCategory `json:"category,omitempty"`
	Severity    ErrorSeverity `json:"severity,omitempty"`
	
	// Additional context
	Details     map[string]interface{} `json:"details,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	
	// Correlation and tracing
	CorrelationID string `json:"correlation_id,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
	SpanID        string `json:"span_id,omitempty"`
	
	// Error metadata
	Retryable     bool   `json:"retryable,omitempty"`
	RetryCount    int    `json:"retry_count,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
	HTTPStatus    int    `json:"http_status,omitempty"`
}

// ErrorLogger provides structured error logging capabilities
type ErrorLogger struct {
	logger  zerolog.Logger
	monitor *ErrorMonitor
}

// NewErrorLogger creates a new error logger instance
func NewErrorLogger(logger zerolog.Logger) *ErrorLogger {
	return &ErrorLogger{
		logger: logger.With().Str("log_type", "error").Logger(),
	}
}

// NewErrorLoggerWithMonitor creates a new error logger instance with monitoring
func NewErrorLoggerWithMonitor(logger zerolog.Logger, monitor *ErrorMonitor) *ErrorLogger {
	return &ErrorLogger{
		logger:  logger.With().Str("log_type", "error").Logger(),
		monitor: monitor,
	}
}

// LogError logs an error with structured context information
func (el *ErrorLogger) LogError(err error, ctx ErrorContext) {
	if err == nil {
		return
	}

	// Track error frequency if monitor is available
	if el.monitor != nil {
		el.monitor.TrackError(ctx)
	}

	// Determine log level based on severity
	var event *zerolog.Event
	switch ctx.Severity {
	case CriticalSeverity:
		event = el.logger.Fatal()
	case HighSeverity:
		event = el.logger.Error()
	case MediumSeverity:
		event = el.logger.Warn()
	case LowSeverity:
		event = el.logger.Info()
	default:
		event = el.logger.Error()
	}

	// Add the error
	event = event.Err(err)

	// Add context fields
	if ctx.RequestID != "" {
		event = event.Str("request_id", ctx.RequestID)
	}
	if ctx.UserID > 0 {
		event = event.Int64("user_id", ctx.UserID)
	}
	if ctx.UserEmail != "" {
		event = event.Str("user_email", ctx.UserEmail)
	}
	if ctx.Operation != "" {
		event = event.Str("operation", ctx.Operation)
	}
	if ctx.Component != "" {
		event = event.Str("component", ctx.Component)
	}
	if ctx.Method != "" {
		event = event.Str("method", ctx.Method)
	}
	if ctx.Category != "" {
		event = event.Str("category", string(ctx.Category))
	}
	if ctx.Severity != "" {
		event = event.Str("severity", string(ctx.Severity))
	}
	if ctx.CorrelationID != "" {
		event = event.Str("correlation_id", ctx.CorrelationID)
	}
	if ctx.TraceID != "" {
		event = event.Str("trace_id", ctx.TraceID)
	}
	if ctx.SpanID != "" {
		event = event.Str("span_id", ctx.SpanID)
	}
	if ctx.ErrorCode != "" {
		event = event.Str("error_code", ctx.ErrorCode)
	}
	if ctx.HTTPStatus > 0 {
		event = event.Int("http_status", ctx.HTTPStatus)
	}
	if ctx.RetryCount > 0 {
		event = event.Int("retry_count", ctx.RetryCount)
	}
	if ctx.Retryable {
		event = event.Bool("retryable", ctx.Retryable)
	}

	// Add details if present
	if ctx.Details != nil && len(ctx.Details) > 0 {
		event = event.Interface("details", ctx.Details)
	}

	// Add stack trace if present
	if ctx.StackTrace != "" {
		event = event.Str("stack_trace", ctx.StackTrace)
	}

	// Log the error
	message := fmt.Sprintf("Error in %s", ctx.Operation)
	if ctx.Operation == "" {
		message = "Application error occurred"
	}
	
	event.Msg(message)
}

// LogErrorWithStackTrace logs an error and automatically captures stack trace for critical errors
func (el *ErrorLogger) LogErrorWithStackTrace(err error, ctx ErrorContext) {
	if err == nil {
		return
	}

	// Capture stack trace for critical and high severity errors
	if ctx.Severity == CriticalSeverity || ctx.Severity == HighSeverity {
		if ctx.StackTrace == "" {
			ctx.StackTrace = captureStackTrace(3) // Skip this function and 2 callers
		}
	}

	el.LogError(err, ctx)
}

// LogValidationError logs a validation error with appropriate context
func (el *ErrorLogger) LogValidationError(err error, ctx ErrorContext) {
	ctx.Category = ValidationError
	if ctx.Severity == "" {
		ctx.Severity = LowSeverity
	}
	el.LogError(err, ctx)
}

// LogBusinessLogicError logs a business logic error with appropriate context
func (el *ErrorLogger) LogBusinessLogicError(err error, ctx ErrorContext) {
	ctx.Category = BusinessLogicError
	if ctx.Severity == "" {
		ctx.Severity = MediumSeverity
	}
	el.LogError(err, ctx)
}

// LogSystemError logs a system error with appropriate context and stack trace
func (el *ErrorLogger) LogSystemError(err error, ctx ErrorContext) {
	ctx.Category = SystemError
	if ctx.Severity == "" {
		ctx.Severity = HighSeverity
	}
	el.LogErrorWithStackTrace(err, ctx)
}

// LogDatabaseError logs a database error with appropriate context
func (el *ErrorLogger) LogDatabaseError(err error, ctx ErrorContext) {
	ctx.Category = DatabaseError
	if ctx.Severity == "" {
		ctx.Severity = HighSeverity
	}
	el.LogErrorWithStackTrace(err, ctx)
}

// LogAuthenticationError logs an authentication error with appropriate context
func (el *ErrorLogger) LogAuthenticationError(err error, ctx ErrorContext) {
	ctx.Category = AuthenticationError
	if ctx.Severity == "" {
		ctx.Severity = MediumSeverity
	}
	el.LogError(err, ctx)
}

// LogExternalServiceError logs an external service error with appropriate context
func (el *ErrorLogger) LogExternalServiceError(err error, ctx ErrorContext) {
	ctx.Category = ExternalServiceError
	if ctx.Severity == "" {
		ctx.Severity = MediumSeverity
	}
	ctx.Retryable = true // External service errors are typically retryable
	el.LogError(err, ctx)
}

// LogNetworkError logs a network error with appropriate context
func (el *ErrorLogger) LogNetworkError(err error, ctx ErrorContext) {
	ctx.Category = NetworkError
	if ctx.Severity == "" {
		ctx.Severity = MediumSeverity
	}
	ctx.Retryable = true // Network errors are typically retryable
	el.LogError(err, ctx)
}

// LogConfigurationError logs a configuration error with appropriate context
func (el *ErrorLogger) LogConfigurationError(err error, ctx ErrorContext) {
	ctx.Category = ConfigurationError
	if ctx.Severity == "" {
		ctx.Severity = HighSeverity
	}
	el.LogErrorWithStackTrace(err, ctx)
}

// NewErrorContextFromGinContext creates an ErrorContext from a Gin context
func NewErrorContextFromGinContext(c context.Context) ErrorContext {
	ctx := ErrorContext{}

	// Extract request ID if available
	if requestID := c.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			ctx.RequestID = id
		}
	}

	// Extract user ID if available
	if userID := c.Value("user_id"); userID != nil {
		if id, ok := userID.(int64); ok {
			ctx.UserID = id
		} else if id, ok := userID.(int); ok {
			ctx.UserID = int64(id)
		}
	}

	// Extract user email if available
	if userEmail := c.Value("user_email"); userEmail != nil {
		if email, ok := userEmail.(string); ok {
			ctx.UserEmail = email
		}
	}

	return ctx
}

// WithOperation adds operation information to the error context
func (ec ErrorContext) WithOperation(operation string) ErrorContext {
	ec.Operation = operation
	return ec
}

// WithComponent adds component information to the error context
func (ec ErrorContext) WithComponent(component string) ErrorContext {
	ec.Component = component
	return ec
}

// WithMethod adds method information to the error context
func (ec ErrorContext) WithMethod(method string) ErrorContext {
	ec.Method = method
	return ec
}

// WithSeverity adds severity information to the error context
func (ec ErrorContext) WithSeverity(severity ErrorSeverity) ErrorContext {
	ec.Severity = severity
	return ec
}

// WithCategory adds category information to the error context
func (ec ErrorContext) WithCategory(category ErrorCategory) ErrorContext {
	ec.Category = category
	return ec
}

// WithDetails adds detail information to the error context
func (ec ErrorContext) WithDetails(details map[string]interface{}) ErrorContext {
	if ec.Details == nil {
		ec.Details = make(map[string]interface{})
	}
	for key, value := range details {
		ec.Details[key] = value
	}
	return ec
}

// WithDetail adds a single detail to the error context
func (ec ErrorContext) WithDetail(key string, value interface{}) ErrorContext {
	if ec.Details == nil {
		ec.Details = make(map[string]interface{})
	}
	ec.Details[key] = value
	return ec
}

// WithHTTPStatus adds HTTP status code to the error context
func (ec ErrorContext) WithHTTPStatus(status int) ErrorContext {
	ec.HTTPStatus = status
	return ec
}

// WithErrorCode adds error code to the error context
func (ec ErrorContext) WithErrorCode(code string) ErrorContext {
	ec.ErrorCode = code
	return ec
}

// WithRetryable marks the error as retryable
func (ec ErrorContext) WithRetryable(retryable bool) ErrorContext {
	ec.Retryable = retryable
	return ec
}

// WithRetryCount adds retry count to the error context
func (ec ErrorContext) WithRetryCount(count int) ErrorContext {
	ec.RetryCount = count
	return ec
}

// captureStackTrace captures the current stack trace
func captureStackTrace(skip int) string {
	const maxStackSize = 32
	stack := make([]uintptr, maxStackSize)
	length := runtime.Callers(skip, stack)
	
	if length == 0 {
		return ""
	}

	frames := runtime.CallersFrames(stack[:length])
	var stackTrace strings.Builder
	
	for {
		frame, more := frames.Next()
		
		// Skip runtime and internal Go functions
		if strings.Contains(frame.File, "runtime/") || 
		   strings.Contains(frame.File, "internal/") {
			if !more {
				break
			}
			continue
		}
		
		stackTrace.WriteString(fmt.Sprintf("%s:%d %s\n", 
			frame.File, frame.Line, frame.Function))
		
		if !more {
			break
		}
	}
	
	return stackTrace.String()
}

// ClassifyError attempts to automatically classify an error based on its content
func ClassifyError(err error) ErrorCategory {
	if err == nil {
		return ""
	}

	errMsg := strings.ToLower(err.Error())
	
	switch {
	case strings.Contains(errMsg, "unauthorized") ||
		 strings.Contains(errMsg, "forbidden") ||
		 strings.Contains(errMsg, "access denied") ||
		 strings.Contains(errMsg, "authentication") ||
		 strings.Contains(errMsg, "token"):
		return AuthenticationError
		
	case strings.Contains(errMsg, "validation") || 
		 strings.Contains(errMsg, "invalid") ||
		 strings.Contains(errMsg, "required") ||
		 strings.Contains(errMsg, "format"):
		return ValidationError
		
	case strings.Contains(errMsg, "business") ||
		 strings.Contains(errMsg, "rule") ||
		 strings.Contains(errMsg, "insufficient") ||
		 strings.Contains(errMsg, "currency mismatch"):
		return BusinessLogicError
		
	case strings.Contains(errMsg, "database") ||
		 strings.Contains(errMsg, "sql") ||
		 strings.Contains(errMsg, "connection") ||
		 strings.Contains(errMsg, "transaction"):
		return DatabaseError
		
	case strings.Contains(errMsg, "network") ||
		 strings.Contains(errMsg, "timeout") ||
		 strings.Contains(errMsg, "connection refused") ||
		 strings.Contains(errMsg, "no route"):
		return NetworkError
		
	case strings.Contains(errMsg, "config") ||
		 strings.Contains(errMsg, "environment") ||
		 strings.Contains(errMsg, "setting"):
		return ConfigurationError
		
	default:
		return SystemError
	}
}

// DetermineSeverity attempts to determine error severity based on category and content
func DetermineSeverity(err error, category ErrorCategory) ErrorSeverity {
	if err == nil {
		return LowSeverity
	}

	errMsg := strings.ToLower(err.Error())
	
	// Check for critical keywords first
	if strings.Contains(errMsg, "panic") ||
	   strings.Contains(errMsg, "fatal") ||
	   strings.Contains(errMsg, "critical") ||
	   strings.Contains(errMsg, "corruption") {
		return CriticalSeverity
	}

	// Determine severity based on category
	switch category {
	case ValidationError:
		return LowSeverity
	case AuthenticationError:
		if strings.Contains(errMsg, "brute force") ||
		   strings.Contains(errMsg, "security") {
			return HighSeverity
		}
		return MediumSeverity
	case BusinessLogicError:
		return MediumSeverity
	case DatabaseError, SystemError, ConfigurationError:
		return HighSeverity
	case NetworkError, ExternalServiceError:
		return MediumSeverity
	default:
		return MediumSeverity
	}
}

// GetErrorMonitor returns the error monitor instance if available
func (el *ErrorLogger) GetErrorMonitor() *ErrorMonitor {
	return el.monitor
}

// SetErrorMonitor sets the error monitor for frequency tracking
func (el *ErrorLogger) SetErrorMonitor(monitor *ErrorMonitor) {
	el.monitor = monitor
}

// GetErrorFrequencies returns current error frequency data from the monitor
func (el *ErrorLogger) GetErrorFrequencies() map[string]*ErrorFrequency {
	if el.monitor == nil {
		return make(map[string]*ErrorFrequency)
	}
	return el.monitor.GetErrorFrequencies()
}

// GetErrorStats returns aggregated error statistics from the monitor
func (el *ErrorLogger) GetErrorStats() map[ErrorCategory]int64 {
	if el.monitor == nil {
		return make(map[ErrorCategory]int64)
	}
	return el.monitor.GetErrorStats()
}

// GetTopErrors returns the most frequent errors from the monitor
func (el *ErrorLogger) GetTopErrors(limit int) []*ErrorFrequency {
	if el.monitor == nil {
		return []*ErrorFrequency{}
	}
	return el.monitor.GetTopErrors(limit)
}

// AddAlertThreshold adds an alert threshold to the monitor
func (el *ErrorLogger) AddAlertThreshold(threshold AlertThreshold) {
	if el.monitor != nil {
		el.monitor.AddThreshold(threshold)
	}
}

// RemoveAlertThreshold removes an alert threshold from the monitor
func (el *ErrorLogger) RemoveAlertThreshold(category ErrorCategory, component, operation string) {
	if el.monitor != nil {
		el.monitor.RemoveThreshold(category, component, operation)
	}
}