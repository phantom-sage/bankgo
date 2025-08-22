package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/logging"
)

// LoggerConfig represents logging configuration
type LoggerConfig struct {
	LoggerManager    *logging.LoggerManager
	SkipPaths        []string
	SensitiveHeaders []string
	SensitiveFields  []string
	EnablePerformanceMetrics bool
	SlowRequestThreshold     time.Duration
}

// DefaultLoggerConfig returns a default logger configuration
func DefaultLoggerConfig(loggerManager *logging.LoggerManager) LoggerConfig {
	return LoggerConfig{
		LoggerManager: loggerManager,
		SkipPaths: []string{
			"/health",
			"/metrics",
		},
		SensitiveHeaders: []string{
			"authorization",
			"cookie",
			"set-cookie",
			"x-api-key",
		},
		SensitiveFields: []string{
			"password",
			"password_hash",
			"token",
			"secret",
			"key",
			"authorization",
		},
		EnablePerformanceMetrics: true,
		SlowRequestThreshold:     500 * time.Millisecond, // 500ms threshold for slow requests
	}
}

// RequestLogger returns a request logging middleware
func RequestLogger(config LoggerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain paths
		for _, skipPath := range config.SkipPaths {
			if c.Request.URL.Path == skipPath {
				c.Next()
				return
			}
		}

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Get or generate request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			if id, exists := c.Get("request_id"); exists {
				if idStr, ok := id.(string); ok {
					requestID = idStr
				}
			}
		}

		// Read and restore request body for logging
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Process the request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Get base logger and create context logger
		baseLogger := config.LoggerManager.GetLogger()
		contextLogger := logging.NewContextLoggerFromLogger(baseLogger)

		// Add request ID correlation
		if requestID != "" {
			contextLogger = contextLogger.WithRequestID(requestID)
		}

		// Add user context if available
		if userID, exists := c.Get("user_id"); exists {
			if id, ok := userID.(int64); ok {
				contextLogger = contextLogger.WithUserID(id)
			}
		}
		if userEmail, exists := c.Get("user_email"); exists {
			if email, ok := userEmail.(string); ok {
				contextLogger = contextLogger.WithUserEmail(email)
			}
		}

		// Build log event with structured fields
		event := contextLogger.Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", c.Writer.Status()).
			Int64("duration_ms", duration.Milliseconds()).
			Str("client_ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Int64("response_size", int64(c.Writer.Size()))

		// Add query parameters if present
		if raw != "" {
			event = event.Str("query", raw)
		}

		// Add request headers (excluding sensitive ones)
		headers := make(map[string]string)
		for name, values := range c.Request.Header {
			if !isSensitiveHeader(name, config.SensitiveHeaders) && len(values) > 0 {
				headers[strings.ToLower(name)] = values[0]
			}
		}
		if len(headers) > 0 {
			event = event.Interface("headers", headers)
		}

		// Add request body if it's JSON and not too large (excluding sensitive fields)
		if len(requestBody) > 0 && len(requestBody) < 1024 && isJSONContent(c.Request.Header.Get("Content-Type")) {
			if sanitizedBody := sanitizeRequestBody(requestBody, config.SensitiveFields); sanitizedBody != nil {
				event = event.Interface("request_body", sanitizedBody)
			}
		}

		// Add error information if present
		if len(c.Errors) > 0 {
			event = event.Str("error", c.Errors.String())
		}

		// Determine log level and message based on status code
		var message string
		switch {
		case c.Writer.Status() >= 500:
			message = "Server error"
			// Use error level for server errors
			contextLogger.Error().
				Str("method", c.Request.Method).
				Str("path", path).
				Int("status", c.Writer.Status()).
				Int64("duration_ms", duration.Milliseconds()).
				Str("client_ip", c.ClientIP()).
				Str("user_agent", c.Request.UserAgent()).
				Int64("response_size", int64(c.Writer.Size())).
				Interface("headers", headers).
				Str("error", c.Errors.String()).
				Msg(message)
			return
		case c.Writer.Status() >= 400:
			message = "Client error"
			// Use warn level for client errors
			contextLogger.Warn().
				Str("method", c.Request.Method).
				Str("path", path).
				Int("status", c.Writer.Status()).
				Int64("duration_ms", duration.Milliseconds()).
				Str("client_ip", c.ClientIP()).
				Str("user_agent", c.Request.UserAgent()).
				Int64("response_size", int64(c.Writer.Size())).
				Interface("headers", headers).
				Msg(message)
			return
		case c.Writer.Status() >= 300:
			message = "Redirect"
		default:
			message = "Request completed"
		}

		// Log successful requests at info level
		event.Msg(message)

		// Log performance metrics if enabled
		if config.EnablePerformanceMetrics {
			logPerformanceMetrics(config, c, duration, requestBody)
		}
	}
}

// logPerformanceMetrics logs detailed performance metrics for the request
func logPerformanceMetrics(config LoggerConfig, c *gin.Context, duration time.Duration, requestBody []byte) {
	performanceLogger := logging.NewPerformanceLogger(config.LoggerManager.GetLogger())
	
	// Add request context if available
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			performanceLogger = performanceLogger.WithRequestID(id)
		}
	}
	
	// Add user context if available
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(int64); ok {
			if userEmail, exists := c.Get("user_email"); exists {
				if email, ok := userEmail.(string); ok {
					performanceLogger = performanceLogger.WithUserContext(id, email)
				}
			}
		}
	}

	// Calculate request and response sizes
	requestSize := int64(len(requestBody))
	responseSize := int64(c.Writer.Size())
	
	// Log detailed HTTP request performance
	performanceLogger.LogHTTPRequestWithDetails(
		c.Request.Method,
		c.Request.URL.Path,
		duration,
		c.Writer.Status(),
		requestSize,
		responseSize,
		c.ClientIP(),
	)
	
	// Log slow requests with additional details
	if duration > config.SlowRequestThreshold {
		details := map[string]interface{}{
			"method":        c.Request.Method,
			"path":          c.Request.URL.Path,
			"status":        c.Writer.Status(),
			"client_ip":     c.ClientIP(),
			"user_agent":    c.Request.UserAgent(),
			"request_size":  requestSize,
			"response_size": responseSize,
		}
		
		// Add query parameters if present
		if c.Request.URL.RawQuery != "" {
			details["query"] = c.Request.URL.RawQuery
		}
		
		// Add error information if present
		if len(c.Errors) > 0 {
			details["error"] = c.Errors.String()
		}
		
		performanceLogger.LogSlowOperation(
			"http_request",
			c.Request.Method+" "+c.Request.URL.Path,
			duration,
			config.SlowRequestThreshold,
			details,
		)
	}
	
	// Log error rate metrics for failed requests
	if c.Writer.Status() >= 400 {
		logErrorRateMetrics(performanceLogger, c, duration)
	}
}

// logErrorRateMetrics logs error-specific performance metrics
func logErrorRateMetrics(performanceLogger *logging.PerformanceLogger, c *gin.Context, duration time.Duration) {
	// Create error context details
	errorDetails := map[string]interface{}{
		"method":       c.Request.Method,
		"path":         c.Request.URL.Path,
		"status":       c.Writer.Status(),
		"duration_ms":  duration.Milliseconds(),
		"client_ip":    c.ClientIP(),
		"user_agent":   c.Request.UserAgent(),
	}
	
	// Add query parameters if present
	if c.Request.URL.RawQuery != "" {
		errorDetails["query"] = c.Request.URL.RawQuery
	}
	
	// Add error information if present
	if len(c.Errors) > 0 {
		errorDetails["error"] = c.Errors.String()
	}
	
	// Add user context if available
	if userID, exists := c.Get("user_id"); exists {
		errorDetails["user_id"] = userID
	}
	if userEmail, exists := c.Get("user_email"); exists {
		errorDetails["user_email"] = userEmail
	}
	
	// Log as a performance metric with error context
	logger := performanceLogger.GetLogger()
	logger.Warn().
		Str("metric_type", "error_rate").
		Str("error_category", getErrorCategory(c.Writer.Status())).
		Interface("error_details", errorDetails).
		Time("timestamp", time.Now()).
		Msg("HTTP error performance metric")
}

// getErrorCategory categorizes HTTP status codes for error rate monitoring
func getErrorCategory(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "server_error"
	case statusCode >= 400:
		return "client_error"
	case statusCode >= 300:
		return "redirect"
	default:
		return "success"
	}
}

// isSensitiveHeader checks if a header name is considered sensitive
func isSensitiveHeader(headerName string, sensitiveHeaders []string) bool {
	lowerName := strings.ToLower(headerName)
	for _, sensitive := range sensitiveHeaders {
		if lowerName == strings.ToLower(sensitive) {
			return true
		}
	}
	return false
}

// isJSONContent checks if the content type indicates JSON
func isJSONContent(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "application/json")
}

// sanitizeRequestBody removes sensitive fields from request body
func sanitizeRequestBody(body []byte, sensitiveFields []string) interface{} {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil
	}

	return sanitizeData(data, sensitiveFields)
}

// sanitizeData recursively removes sensitive fields from data structures
func sanitizeData(data interface{}, sensitiveFields []string) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		sanitized := make(map[string]interface{})
		for key, value := range v {
			if isSensitiveField(key, sensitiveFields) {
				sanitized[key] = "[REDACTED]"
			} else {
				sanitized[key] = sanitizeData(value, sensitiveFields)
			}
		}
		return sanitized
	case []interface{}:
		sanitized := make([]interface{}, len(v))
		for i, item := range v {
			sanitized[i] = sanitizeData(item, sensitiveFields)
		}
		return sanitized
	default:
		return v
	}
}

// isSensitiveField checks if a field name is considered sensitive
func isSensitiveField(fieldName string, sensitiveFields []string) bool {
	lowerName := strings.ToLower(fieldName)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(lowerName, strings.ToLower(sensitive)) {
			return true
		}
	}
	return false
}

// RequestID middleware adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// generateRequestID generates a simple request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	now := time.Now().UnixNano()
	for i := range b {
		// Use different seed for each character to avoid repetition
		b[i] = charset[(now+int64(i*7))%int64(len(charset))]
	}
	return string(b)
}