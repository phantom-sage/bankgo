package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPerformanceLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	perfLogger := NewPerformanceLogger(logger)
	
	assert.NotNil(t, perfLogger)
	assert.NotNil(t, perfLogger.logger)
}

func TestPerformanceLogger_LogHTTPRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	method := "GET"
	path := "/api/v1/users"
	duration := 150 * time.Millisecond
	statusCode := 200
	
	perfLogger.LogHTTPRequest(method, path, duration, statusCode)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "http_request", logEntry["metric_type"])
	assert.Equal(t, method, logEntry["method"])
	assert.Equal(t, path, logEntry["path"])
	assert.Equal(t, float64(150), logEntry["duration_ms"])
	assert.Equal(t, 0.15, logEntry["duration_seconds"])
	assert.Equal(t, float64(statusCode), logEntry["status_code"])
	assert.Equal(t, "HTTP request performance", logEntry["message"])
	assert.Contains(t, logEntry, "timestamp")
}

func TestPerformanceLogger_LogHTTPRequestWithDetails(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	method := "POST"
	path := "/api/v1/transfers"
	duration := 250 * time.Millisecond
	statusCode := 201
	requestSize := int64(1024)
	responseSize := int64(512)
	clientIP := "192.168.1.100"
	
	perfLogger.LogHTTPRequestWithDetails(method, path, duration, statusCode, requestSize, responseSize, clientIP)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "http_request", logEntry["metric_type"])
	assert.Equal(t, method, logEntry["method"])
	assert.Equal(t, path, logEntry["path"])
	assert.Equal(t, float64(250), logEntry["duration_ms"])
	assert.Equal(t, 0.25, logEntry["duration_seconds"])
	assert.Equal(t, float64(statusCode), logEntry["status_code"])
	assert.Equal(t, float64(requestSize), logEntry["request_size_bytes"])
	assert.Equal(t, float64(responseSize), logEntry["response_size_bytes"])
	assert.Equal(t, clientIP, logEntry["client_ip"])
}

func TestPerformanceLogger_LogDatabaseQuery(t *testing.T) {
	// Save and restore global level
	originalLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(originalLevel)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	query := "SELECT * FROM users WHERE id = $1"
	duration := 50 * time.Millisecond
	rowsAffected := int64(1)
	
	perfLogger.LogDatabaseQuery(query, duration, rowsAffected)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "database_query", logEntry["metric_type"])
	assert.Equal(t, "select", logEntry["query_type"])
	assert.Equal(t, float64(50), logEntry["duration_ms"])
	assert.Equal(t, 0.05, logEntry["duration_seconds"])
	assert.Equal(t, float64(rowsAffected), logEntry["rows_affected"])
	assert.Equal(t, "Database query performance", logEntry["message"])
	assert.Equal(t, "debug", logEntry["level"])
}

func TestPerformanceLogger_LogDatabaseQueryWithDetails(t *testing.T) {
	// Save and restore global level
	originalLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(originalLevel)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	query := "INSERT INTO accounts (user_id, currency, balance) VALUES ($1, $2, $3)"
	table := "accounts"
	duration := 75 * time.Millisecond
	rowsAffected := int64(1)
	connectionPoolSize := 10
	
	perfLogger.LogDatabaseQueryWithDetails(query, table, duration, rowsAffected, connectionPoolSize)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "database_query", logEntry["metric_type"])
	assert.Equal(t, "insert", logEntry["query_type"])
	assert.Equal(t, table, logEntry["table"])
	assert.Equal(t, float64(75), logEntry["duration_ms"])
	assert.Equal(t, float64(connectionPoolSize), logEntry["connection_pool_size"])
}

func TestPerformanceLogger_LogDatabaseTransaction(t *testing.T) {
	tests := []struct {
		name            string
		duration        time.Duration
		operationCount  int
		success         bool
		expectedLevel   string
	}{
		{
			name:           "successful transaction",
			duration:       100 * time.Millisecond,
			operationCount: 3,
			success:        true,
			expectedLevel:  "info",
		},
		{
			name:           "failed transaction",
			duration:       200 * time.Millisecond,
			operationCount: 2,
			success:        false,
			expectedLevel:  "warn",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			perfLogger := NewPerformanceLogger(logger)
			
			perfLogger.LogDatabaseTransaction(tt.duration, tt.operationCount, tt.success)
			
			var logEntry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			require.NoError(t, err)
			
			assert.Equal(t, "performance", logEntry["log_type"])
			assert.Equal(t, "database_transaction", logEntry["metric_type"])
			assert.Equal(t, float64(tt.duration.Milliseconds()), logEntry["duration_ms"])
			assert.Equal(t, float64(tt.operationCount), logEntry["operation_count"])
			assert.Equal(t, tt.success, logEntry["success"])
			assert.Equal(t, tt.expectedLevel, logEntry["level"])
		})
	}
}

func TestPerformanceLogger_LogBackgroundJob(t *testing.T) {
	tests := []struct {
		name          string
		jobType       string
		duration      time.Duration
		success       bool
		expectedLevel string
	}{
		{
			name:          "successful job",
			jobType:       "email_sender",
			duration:      500 * time.Millisecond,
			success:       true,
			expectedLevel: "info",
		},
		{
			name:          "failed job",
			jobType:       "report_generator",
			duration:      1000 * time.Millisecond,
			success:       false,
			expectedLevel: "warn",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			perfLogger := NewPerformanceLogger(logger)
			
			perfLogger.LogBackgroundJob(tt.jobType, tt.duration, tt.success)
			
			var logEntry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			require.NoError(t, err)
			
			assert.Equal(t, "performance", logEntry["log_type"])
			assert.Equal(t, "background_job", logEntry["metric_type"])
			assert.Equal(t, tt.jobType, logEntry["job_type"])
			assert.Equal(t, float64(tt.duration.Milliseconds()), logEntry["duration_ms"])
			assert.Equal(t, tt.success, logEntry["success"])
			assert.Equal(t, tt.expectedLevel, logEntry["level"])
		})
	}
}

func TestPerformanceLogger_LogBackgroundJobWithDetails(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	jobType := "email_sender"
	duration := 300 * time.Millisecond
	success := true
	queueSize := 25
	retryCount := 2
	
	perfLogger.LogBackgroundJobWithDetails(jobType, duration, success, queueSize, retryCount)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "background_job", logEntry["metric_type"])
	assert.Equal(t, jobType, logEntry["job_type"])
	assert.Equal(t, float64(queueSize), logEntry["queue_size"])
	assert.Equal(t, float64(retryCount), logEntry["retry_count"])
}

func TestPerformanceLogger_LogResourceUsage(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	cpuPercent := 75.5
	memoryPercent := 60.2
	
	perfLogger.LogResourceUsage(cpuPercent, memoryPercent)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "resource_usage", logEntry["metric_type"])
	assert.Equal(t, cpuPercent, logEntry["cpu_percent"])
	assert.Equal(t, memoryPercent, logEntry["memory_percent"])
	assert.Equal(t, "System resource usage", logEntry["message"])
}

func TestPerformanceLogger_LogResourceUsageWithDetails(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	cpuPercent := 75.5
	memoryPercent := 60.2
	memoryUsedMB := int64(1024)
	memoryTotalMB := int64(2048)
	goroutineCount := 150
	
	perfLogger.LogResourceUsageWithDetails(cpuPercent, memoryPercent, memoryUsedMB, memoryTotalMB, goroutineCount)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "resource_usage", logEntry["metric_type"])
	assert.Equal(t, cpuPercent, logEntry["cpu_percent"])
	assert.Equal(t, memoryPercent, logEntry["memory_percent"])
	assert.Equal(t, float64(memoryUsedMB), logEntry["memory_used_mb"])
	assert.Equal(t, float64(memoryTotalMB), logEntry["memory_total_mb"])
	assert.Equal(t, float64(goroutineCount), logEntry["goroutine_count"])
}

func TestPerformanceLogger_LogExternalService(t *testing.T) {
	tests := []struct {
		name          string
		serviceName   string
		duration      time.Duration
		success       bool
		expectedLevel string
	}{
		{
			name:          "successful service call",
			serviceName:   "payment_gateway",
			duration:      200 * time.Millisecond,
			success:       true,
			expectedLevel: "info",
		},
		{
			name:          "failed service call",
			serviceName:   "email_service",
			duration:      5000 * time.Millisecond,
			success:       false,
			expectedLevel: "warn",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			perfLogger := NewPerformanceLogger(logger)
			
			perfLogger.LogExternalService(tt.serviceName, tt.duration, tt.success)
			
			var logEntry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			require.NoError(t, err)
			
			assert.Equal(t, "performance", logEntry["log_type"])
			assert.Equal(t, "external_service", logEntry["metric_type"])
			assert.Equal(t, tt.serviceName, logEntry["service_name"])
			assert.Equal(t, float64(tt.duration.Milliseconds()), logEntry["duration_ms"])
			assert.Equal(t, tt.success, logEntry["success"])
			assert.Equal(t, tt.expectedLevel, logEntry["level"])
		})
	}
}

func TestPerformanceLogger_LogExternalServiceWithDetails(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	serviceName := "payment_gateway"
	endpoint := "/api/v1/charge"
	duration := 300 * time.Millisecond
	success := true
	statusCode := 200
	requestSize := int64(256)
	responseSize := int64(128)
	
	perfLogger.LogExternalServiceWithDetails(serviceName, endpoint, duration, success, statusCode, requestSize, responseSize)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "external_service", logEntry["metric_type"])
	assert.Equal(t, serviceName, logEntry["service_name"])
	assert.Equal(t, endpoint, logEntry["endpoint"])
	assert.Equal(t, float64(statusCode), logEntry["status_code"])
	assert.Equal(t, float64(requestSize), logEntry["request_size_bytes"])
	assert.Equal(t, float64(responseSize), logEntry["response_size_bytes"])
}

func TestPerformanceLogger_LogCacheOperation(t *testing.T) {
	// Save and restore global level
	originalLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(originalLevel)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	operation := "GET"
	key := "user:123"
	duration := 5 * time.Millisecond
	hit := true
	
	perfLogger.LogCacheOperation(operation, key, duration, hit)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "cache_operation", logEntry["metric_type"])
	assert.Equal(t, operation, logEntry["operation"])
	assert.Equal(t, key, logEntry["cache_key"])
	assert.Equal(t, float64(5), logEntry["duration_ms"])
	assert.Equal(t, hit, logEntry["cache_hit"])
	assert.Equal(t, "debug", logEntry["level"])
}

func TestPerformanceLogger_LogSlowOperation(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	operationType := "database_query"
	operationName := "complex_report_query"
	duration := 5 * time.Second
	threshold := 1 * time.Second
	details := map[string]interface{}{
		"table":      "transactions",
		"row_count":  10000,
		"complexity": "high",
	}
	
	perfLogger.LogSlowOperation(operationType, operationName, duration, threshold, details)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "slow_operation", logEntry["metric_type"])
	assert.Equal(t, operationType, logEntry["operation_type"])
	assert.Equal(t, operationName, logEntry["operation_name"])
	assert.Equal(t, float64(5000), logEntry["duration_ms"])
	assert.Equal(t, float64(1000), logEntry["threshold_ms"])
	assert.Equal(t, 5.0, logEntry["threshold_exceeded_ratio"])
	assert.Equal(t, "warn", logEntry["level"])
	
	// Check details
	assert.Equal(t, "transactions", logEntry["table"])
	assert.Equal(t, float64(10000), logEntry["row_count"])
	assert.Equal(t, "high", logEntry["complexity"])
}

func TestPerformanceLogger_LogThroughputMetrics(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	operationType := "http_requests"
	requestCount := 1000
	duration := 60 * time.Second
	errorCount := 50
	
	perfLogger.LogThroughputMetrics(operationType, requestCount, duration, errorCount)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "throughput", logEntry["metric_type"])
	assert.Equal(t, operationType, logEntry["operation_type"])
	assert.Equal(t, float64(requestCount), logEntry["request_count"])
	assert.Equal(t, float64(errorCount), logEntry["error_count"])
	assert.InDelta(t, 16.67, logEntry["requests_per_second"], 0.01) // 1000/60
	assert.Equal(t, 5.0, logEntry["error_rate_percent"])             // 50/1000 * 100
}

func TestPerformanceLogger_LogLatencyPercentiles(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	operationType := "api_requests"
	p50 := 100 * time.Millisecond
	p90 := 200 * time.Millisecond
	p95 := 300 * time.Millisecond
	p99 := 500 * time.Millisecond
	sampleCount := 10000
	
	perfLogger.LogLatencyPercentiles(operationType, p50, p90, p95, p99, sampleCount)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "latency_percentiles", logEntry["metric_type"])
	assert.Equal(t, operationType, logEntry["operation_type"])
	assert.Equal(t, float64(100), logEntry["p50_ms"])
	assert.Equal(t, float64(200), logEntry["p90_ms"])
	assert.Equal(t, float64(300), logEntry["p95_ms"])
	assert.Equal(t, float64(500), logEntry["p99_ms"])
	assert.Equal(t, float64(sampleCount), logEntry["sample_count"])
}

func TestPerformanceLogger_WithRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	requestID := "req-123-456"
	perfLoggerWithReqID := perfLogger.WithRequestID(requestID)
	
	perfLoggerWithReqID.LogHTTPRequest("GET", "/api/test", 100*time.Millisecond, 200)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, requestID, logEntry["request_id"])
	assert.Equal(t, "performance", logEntry["log_type"])
}

func TestPerformanceLogger_WithUserContext(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	userID := int64(123)
	userEmail := "test@example.com"
	perfLoggerWithUser := perfLogger.WithUserContext(userID, userEmail)
	
	perfLoggerWithUser.LogResourceUsage(50.0, 60.0)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, float64(userID), logEntry["user_id"])
	assert.Equal(t, userEmail, logEntry["user_email"])
	assert.Equal(t, "performance", logEntry["log_type"])
}

func TestPerformanceLogger_WithCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	correlationID := "corr-123-456"
	perfLoggerWithCorr := perfLogger.WithCorrelationID(correlationID)
	
	perfLoggerWithCorr.LogBackgroundJob("test_job", 100*time.Millisecond, true)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, correlationID, logEntry["correlation_id"])
	assert.Equal(t, "performance", logEntry["log_type"])
}

func TestPerformanceLogger_WithComponent(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	component := "user_service"
	perfLoggerWithComponent := perfLogger.WithComponent(component)
	
	perfLoggerWithComponent.LogExternalService("payment_api", 200*time.Millisecond, true)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, component, logEntry["component"])
	assert.Equal(t, "performance", logEntry["log_type"])
}

func TestPerformanceLogger_GetLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	underlyingLogger := perfLogger.GetLogger()
	assert.NotNil(t, underlyingLogger)
	
	// Test that the underlying logger has the performance log_type
	underlyingLogger.Info().Msg("test message")
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "performance", logEntry["log_type"])
	assert.Equal(t, "test message", logEntry["message"])
}

func TestExtractQueryType(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"SELECT * FROM users", "select"},
		{"select id from accounts", "select"},
		{"INSERT INTO users (name) VALUES ('test')", "insert"},
		{"UPDATE users SET name = 'test'", "update"},
		{"DELETE FROM users WHERE id = 1", "delete"},
		{"CREATE TABLE test (id INT)", "create"},
		{"DROP TABLE test", "drop"},
		{"ALTER TABLE users ADD COLUMN email VARCHAR(255)", "alter"},
		{"BEGIN TRANSACTION", "transaction"},
		{"START TRANSACTION", "transaction"},
		{"COMMIT", "commit"},
		{"ROLLBACK", "rollback"},
		{"EXPLAIN SELECT * FROM users", "other"},
		{"", "unknown"},
		{"   ", "unknown"},
		{"INVALID QUERY", "other"},
	}
	
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := extractQueryType(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPerformanceLogger_TimestampFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	perfLogger.LogHTTPRequest("GET", "/test", 100*time.Millisecond, 200)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	timestampStr, ok := logEntry["timestamp"].(string)
	require.True(t, ok, "timestamp should be a string")
	
	// Parse the timestamp to ensure it's in the correct format
	_, err = time.Parse(time.RFC3339Nano, timestampStr)
	assert.NoError(t, err, "timestamp should be in RFC3339Nano format")
}

func TestPerformanceLogger_LogLevels(t *testing.T) {
	tests := []struct {
		name          string
		logFunc       func(*PerformanceLogger)
		expectedLevel string
	}{
		{
			name: "HTTP request uses info level",
			logFunc: func(pl *PerformanceLogger) {
				pl.LogHTTPRequest("GET", "/test", 100*time.Millisecond, 200)
			},
			expectedLevel: "info",
		},
		{
			name: "Database query uses debug level",
			logFunc: func(pl *PerformanceLogger) {
				pl.LogDatabaseQuery("SELECT * FROM users", 50*time.Millisecond, 1)
			},
			expectedLevel: "debug",
		},
		{
			name: "Successful background job uses info level",
			logFunc: func(pl *PerformanceLogger) {
				pl.LogBackgroundJob("test_job", 100*time.Millisecond, true)
			},
			expectedLevel: "info",
		},
		{
			name: "Failed background job uses warn level",
			logFunc: func(pl *PerformanceLogger) {
				pl.LogBackgroundJob("test_job", 100*time.Millisecond, false)
			},
			expectedLevel: "warn",
		},
		{
			name: "Slow operation uses warn level",
			logFunc: func(pl *PerformanceLogger) {
				pl.LogSlowOperation("test", "test_op", 2*time.Second, 1*time.Second, nil)
			},
			expectedLevel: "warn",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore global level
			originalLevel := zerolog.GlobalLevel()
			defer zerolog.SetGlobalLevel(originalLevel)
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
			
			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			perfLogger := NewPerformanceLogger(logger)
			
			tt.logFunc(perfLogger)
			
			var logEntry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			require.NoError(t, err)
			
			assert.Equal(t, tt.expectedLevel, logEntry["level"])
		})
	}
}

func TestPerformanceLogger_ConcurrentLogging(t *testing.T) {
	// Use a synchronized buffer to handle concurrent writes
	var buf bytes.Buffer
	var mu sync.Mutex
	
	syncWriter := &syncWriter{buf: &buf, mu: &mu}
	logger := zerolog.New(syncWriter)
	perfLogger := NewPerformanceLogger(logger)
	
	// Test concurrent logging to ensure thread safety
	const numGoroutines = 10
	const logsPerGoroutine = 10
	
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < logsPerGoroutine; j++ {
				perfLogger.LogHTTPRequest("GET", "/test", 100*time.Millisecond, 200)
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Count the number of log entries
	mu.Lock()
	logOutput := buf.String()
	mu.Unlock()
	
	logLines := strings.Split(strings.TrimSpace(logOutput), "\n")
	
	// Filter out empty lines
	var validLines []string
	for _, line := range logLines {
		if strings.TrimSpace(line) != "" {
			validLines = append(validLines, line)
		}
	}
	
	// Should have numGoroutines * logsPerGoroutine log entries
	expectedLogs := numGoroutines * logsPerGoroutine
	assert.Equal(t, expectedLogs, len(validLines))
	
	// Verify each log entry is valid JSON
	for _, line := range validLines {
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(line), &logEntry)
		assert.NoError(t, err, "Each log line should be valid JSON: %s", line)
		assert.Equal(t, "performance", logEntry["log_type"])
	}
}

func TestPerformanceLogger_DurationCalculations(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	perfLogger := NewPerformanceLogger(logger)
	
	// Test various duration calculations
	testCases := []struct {
		duration        time.Duration
		expectedMS      float64
		expectedSeconds float64
	}{
		{100 * time.Millisecond, 100, 0.1},
		{1 * time.Second, 1000, 1.0},
		{1500 * time.Millisecond, 1500, 1.5},
		{50 * time.Microsecond, 0, 0.00005}, // Less than 1ms
	}
	
	for _, tc := range testCases {
		buf.Reset()
		perfLogger.LogHTTPRequest("GET", "/test", tc.duration, 200)
		
		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)
		
		assert.Equal(t, tc.expectedMS, logEntry["duration_ms"])
		assert.InDelta(t, tc.expectedSeconds, logEntry["duration_seconds"], 0.00001)
	}
}