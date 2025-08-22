package logging

import (
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// PerformanceLogger provides specialized logging for performance metrics and monitoring
type PerformanceLogger struct {
	logger zerolog.Logger
}

// NewPerformanceLogger creates a new performance logger instance
func NewPerformanceLogger(logger zerolog.Logger) *PerformanceLogger {
	return &PerformanceLogger{
		logger: logger.With().Str("log_type", "performance").Logger(),
	}
}

// LogHTTPRequest logs HTTP request performance metrics
func (pl *PerformanceLogger) LogHTTPRequest(method, path string, duration time.Duration, statusCode int) {
	pl.logger.Info().
		Str("metric_type", "http_request").
		Str("method", method).
		Str("path", path).
		Int64("duration_ms", duration.Milliseconds()).
		Float64("duration_seconds", duration.Seconds()).
		Int("status_code", statusCode).
		Time("timestamp", time.Now()).
		Msg("HTTP request performance")
}

// LogHTTPRequestWithDetails logs HTTP request with additional details
func (pl *PerformanceLogger) LogHTTPRequestWithDetails(method, path string, duration time.Duration, statusCode int, requestSize, responseSize int64, clientIP string) {
	pl.logger.Info().
		Str("metric_type", "http_request").
		Str("method", method).
		Str("path", path).
		Int64("duration_ms", duration.Milliseconds()).
		Float64("duration_seconds", duration.Seconds()).
		Int("status_code", statusCode).
		Int64("request_size_bytes", requestSize).
		Int64("response_size_bytes", responseSize).
		Str("client_ip", clientIP).
		Time("timestamp", time.Now()).
		Msg("HTTP request performance")
}

// LogDatabaseQuery logs database query performance metrics
func (pl *PerformanceLogger) LogDatabaseQuery(query string, duration time.Duration, rowsAffected int64) {
	pl.logger.Debug().
		Str("metric_type", "database_query").
		Str("query_type", extractQueryType(query)).
		Int64("duration_ms", duration.Milliseconds()).
		Float64("duration_seconds", duration.Seconds()).
		Int64("rows_affected", rowsAffected).
		Time("timestamp", time.Now()).
		Msg("Database query performance")
}

// LogDatabaseQueryWithDetails logs database query with additional context
func (pl *PerformanceLogger) LogDatabaseQueryWithDetails(query, table string, duration time.Duration, rowsAffected int64, connectionPoolSize int) {
	pl.logger.Debug().
		Str("metric_type", "database_query").
		Str("query_type", extractQueryType(query)).
		Str("table", table).
		Int64("duration_ms", duration.Milliseconds()).
		Float64("duration_seconds", duration.Seconds()).
		Int64("rows_affected", rowsAffected).
		Int("connection_pool_size", connectionPoolSize).
		Time("timestamp", time.Now()).
		Msg("Database query performance")
}

// LogDatabaseTransaction logs database transaction performance
func (pl *PerformanceLogger) LogDatabaseTransaction(duration time.Duration, operationCount int, success bool) {
	level := pl.logger.Info()
	if !success {
		level = pl.logger.Warn()
	}
	
	level.
		Str("metric_type", "database_transaction").
		Int64("duration_ms", duration.Milliseconds()).
		Float64("duration_seconds", duration.Seconds()).
		Int("operation_count", operationCount).
		Bool("success", success).
		Time("timestamp", time.Now()).
		Msg("Database transaction performance")
}

// LogBackgroundJob logs background job performance metrics
func (pl *PerformanceLogger) LogBackgroundJob(jobType string, duration time.Duration, success bool) {
	level := pl.logger.Info()
	if !success {
		level = pl.logger.Warn()
	}
	
	level.
		Str("metric_type", "background_job").
		Str("job_type", jobType).
		Int64("duration_ms", duration.Milliseconds()).
		Float64("duration_seconds", duration.Seconds()).
		Bool("success", success).
		Time("timestamp", time.Now()).
		Msg("Background job performance")
}

// LogBackgroundJobWithDetails logs background job with queue metrics
func (pl *PerformanceLogger) LogBackgroundJobWithDetails(jobType string, duration time.Duration, success bool, queueSize, retryCount int) {
	level := pl.logger.Info()
	if !success {
		level = pl.logger.Warn()
	}
	
	level.
		Str("metric_type", "background_job").
		Str("job_type", jobType).
		Int64("duration_ms", duration.Milliseconds()).
		Float64("duration_seconds", duration.Seconds()).
		Bool("success", success).
		Int("queue_size", queueSize).
		Int("retry_count", retryCount).
		Time("timestamp", time.Now()).
		Msg("Background job performance")
}

// LogResourceUsage logs system resource usage metrics
func (pl *PerformanceLogger) LogResourceUsage(cpuPercent, memoryPercent float64) {
	pl.logger.Info().
		Str("metric_type", "resource_usage").
		Float64("cpu_percent", cpuPercent).
		Float64("memory_percent", memoryPercent).
		Time("timestamp", time.Now()).
		Msg("System resource usage")
}

// LogResourceUsageWithDetails logs detailed system resource metrics
func (pl *PerformanceLogger) LogResourceUsageWithDetails(cpuPercent, memoryPercent float64, memoryUsedMB, memoryTotalMB int64, goroutineCount int) {
	pl.logger.Info().
		Str("metric_type", "resource_usage").
		Float64("cpu_percent", cpuPercent).
		Float64("memory_percent", memoryPercent).
		Int64("memory_used_mb", memoryUsedMB).
		Int64("memory_total_mb", memoryTotalMB).
		Int("goroutine_count", goroutineCount).
		Time("timestamp", time.Now()).
		Msg("System resource usage")
}

// LogExternalService logs external service call performance
func (pl *PerformanceLogger) LogExternalService(serviceName string, duration time.Duration, success bool) {
	level := pl.logger.Info()
	if !success {
		level = pl.logger.Warn()
	}
	
	level.
		Str("metric_type", "external_service").
		Str("service_name", serviceName).
		Int64("duration_ms", duration.Milliseconds()).
		Float64("duration_seconds", duration.Seconds()).
		Bool("success", success).
		Time("timestamp", time.Now()).
		Msg("External service call performance")
}

// LogExternalServiceWithDetails logs external service call with additional context
func (pl *PerformanceLogger) LogExternalServiceWithDetails(serviceName, endpoint string, duration time.Duration, success bool, statusCode int, requestSize, responseSize int64) {
	level := pl.logger.Info()
	if !success {
		level = pl.logger.Warn()
	}
	
	level.
		Str("metric_type", "external_service").
		Str("service_name", serviceName).
		Str("endpoint", endpoint).
		Int64("duration_ms", duration.Milliseconds()).
		Float64("duration_seconds", duration.Seconds()).
		Bool("success", success).
		Int("status_code", statusCode).
		Int64("request_size_bytes", requestSize).
		Int64("response_size_bytes", responseSize).
		Time("timestamp", time.Now()).
		Msg("External service call performance")
}

// LogCacheOperation logs cache operation performance
func (pl *PerformanceLogger) LogCacheOperation(operation, key string, duration time.Duration, hit bool) {
	pl.logger.Debug().
		Str("metric_type", "cache_operation").
		Str("operation", operation).
		Str("cache_key", key).
		Int64("duration_ms", duration.Milliseconds()).
		Bool("cache_hit", hit).
		Time("timestamp", time.Now()).
		Msg("Cache operation performance")
}

// LogQueueMetrics logs message queue performance metrics
func (pl *PerformanceLogger) LogQueueMetrics(queueName string, size, processingRate int, avgProcessingTime time.Duration) {
	pl.logger.Info().
		Str("metric_type", "queue_metrics").
		Str("queue_name", queueName).
		Int("queue_size", size).
		Int("processing_rate_per_minute", processingRate).
		Int64("avg_processing_time_ms", avgProcessingTime.Milliseconds()).
		Time("timestamp", time.Now()).
		Msg("Queue performance metrics")
}

// LogConnectionPoolMetrics logs database connection pool metrics
func (pl *PerformanceLogger) LogConnectionPoolMetrics(activeConnections, idleConnections, maxConnections int, waitDuration time.Duration) {
	pl.logger.Debug().
		Str("metric_type", "connection_pool").
		Int("active_connections", activeConnections).
		Int("idle_connections", idleConnections).
		Int("max_connections", maxConnections).
		Float64("utilization_percent", float64(activeConnections)/float64(maxConnections)*100).
		Int64("wait_duration_ms", waitDuration.Milliseconds()).
		Time("timestamp", time.Now()).
		Msg("Connection pool metrics")
}

// LogMemoryAllocation logs memory allocation metrics
func (pl *PerformanceLogger) LogMemoryAllocation(allocatedBytes, freedBytes uint64, gcCount uint32) {
	pl.logger.Debug().
		Str("metric_type", "memory_allocation").
		Uint64("allocated_bytes", allocatedBytes).
		Uint64("freed_bytes", freedBytes).
		Uint64("net_allocated_bytes", allocatedBytes-freedBytes).
		Uint32("gc_count", gcCount).
		Time("timestamp", time.Now()).
		Msg("Memory allocation metrics")
}

// LogSlowOperation logs operations that exceed performance thresholds
func (pl *PerformanceLogger) LogSlowOperation(operationType, operationName string, duration, threshold time.Duration, details map[string]interface{}) {
	event := pl.logger.Warn().
		Str("metric_type", "slow_operation").
		Str("operation_type", operationType).
		Str("operation_name", operationName).
		Int64("duration_ms", duration.Milliseconds()).
		Int64("threshold_ms", threshold.Milliseconds()).
		Float64("threshold_exceeded_ratio", float64(duration)/float64(threshold)).
		Time("timestamp", time.Now())
	
	// Add additional details if provided
	for key, value := range details {
		event = event.Interface(key, value)
	}
	
	event.Msg("Slow operation detected")
}

// LogThroughputMetrics logs throughput metrics for various operations
func (pl *PerformanceLogger) LogThroughputMetrics(operationType string, requestCount int, duration time.Duration, errorCount int) {
	requestsPerSecond := float64(requestCount) / duration.Seconds()
	errorRate := float64(errorCount) / float64(requestCount) * 100
	
	pl.logger.Info().
		Str("metric_type", "throughput").
		Str("operation_type", operationType).
		Int("request_count", requestCount).
		Int("error_count", errorCount).
		Float64("requests_per_second", requestsPerSecond).
		Float64("error_rate_percent", errorRate).
		Int64("measurement_duration_ms", duration.Milliseconds()).
		Time("timestamp", time.Now()).
		Msg("Throughput metrics")
}

// LogLatencyPercentiles logs latency percentile metrics
func (pl *PerformanceLogger) LogLatencyPercentiles(operationType string, p50, p90, p95, p99 time.Duration, sampleCount int) {
	pl.logger.Info().
		Str("metric_type", "latency_percentiles").
		Str("operation_type", operationType).
		Int64("p50_ms", p50.Milliseconds()).
		Int64("p90_ms", p90.Milliseconds()).
		Int64("p95_ms", p95.Milliseconds()).
		Int64("p99_ms", p99.Milliseconds()).
		Int("sample_count", sampleCount).
		Time("timestamp", time.Now()).
		Msg("Latency percentile metrics")
}

// WithRequestID returns a new PerformanceLogger with request ID context
func (pl *PerformanceLogger) WithRequestID(requestID string) *PerformanceLogger {
	return &PerformanceLogger{
		logger: pl.logger.With().Str("request_id", requestID).Logger(),
	}
}

// WithUserContext returns a new PerformanceLogger with user context
func (pl *PerformanceLogger) WithUserContext(userID int64, userEmail string) *PerformanceLogger {
	return &PerformanceLogger{
		logger: pl.logger.With().
			Int64("user_id", userID).
			Str("user_email", userEmail).
			Logger(),
	}
}

// WithCorrelationID returns a new PerformanceLogger with correlation ID
func (pl *PerformanceLogger) WithCorrelationID(correlationID string) *PerformanceLogger {
	return &PerformanceLogger{
		logger: pl.logger.With().Str("correlation_id", correlationID).Logger(),
	}
}

// WithComponent returns a new PerformanceLogger with component context
func (pl *PerformanceLogger) WithComponent(component string) *PerformanceLogger {
	return &PerformanceLogger{
		logger: pl.logger.With().Str("component", component).Logger(),
	}
}

// GetLogger returns the underlying zerolog logger
func (pl *PerformanceLogger) GetLogger() zerolog.Logger {
	return pl.logger
}

// extractQueryType extracts the query type (SELECT, INSERT, UPDATE, DELETE) from SQL query
func extractQueryType(query string) string {
	if len(query) == 0 {
		return "unknown"
	}
	
	// Convert to uppercase and get first word
	query = strings.ToUpper(strings.TrimSpace(query))
	words := strings.Fields(query)
	if len(words) == 0 {
		return "unknown"
	}
	
	switch words[0] {
	case "SELECT":
		return "select"
	case "INSERT":
		return "insert"
	case "UPDATE":
		return "update"
	case "DELETE":
		return "delete"
	case "CREATE":
		return "create"
	case "DROP":
		return "drop"
	case "ALTER":
		return "alter"
	case "BEGIN", "START":
		return "transaction"
	case "COMMIT":
		return "commit"
	case "ROLLBACK":
		return "rollback"
	default:
		return "other"
	}
}