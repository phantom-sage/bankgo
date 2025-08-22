package logging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEndLoggingFlow tests complete request logging flow from HTTP to file
func TestEndToEndLoggingFlow(t *testing.T) {
	tempDir := t.TempDir()
	
	// Set up logger manager
	config := LogConfig{
		Level:     "debug",
		Format:    "json",
		Output:    "file",
		Directory: tempDir,
		MaxAge:    1,
		MaxBackups: 3,
		Compress:  false,
	}
	
	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()
	
	// Create specialized loggers
	auditLogger := NewAuditLogger(lm.GetLogger())
	perfLogger := NewPerformanceLogger(lm.GetLogger())
	errorLogger := NewErrorLogger(lm.GetLogger())
	
	// Simulate a complete request flow
	requestID := "req-integration-test-123"
	userID := int64(12345)
	userEmail := "integration.test@example.com"
	
	// 1. Start of request - performance logging
	startTime := time.Now()
	
	// 2. Authentication - audit logging
	auditLogger.WithRequestID(requestID).LogAuthentication(userID, userEmail, "login", "success")
	
	// 3. Business operation - context logging with performance
	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", requestID)
	ctx = context.WithValue(ctx, "user_id", userID)
	ctx = context.WithValue(ctx, "user_email", userEmail)
	
	contextLogger := NewContextLogger(lm.GetLogger(), ctx)
	contextLogger.WithComponent("transfer_service").Info().Msg("Processing money transfer")
	
	// 4. Database operation - performance logging
	perfLogger.WithRequestID(requestID).LogDatabaseQuery(
		"INSERT INTO transfers (from_account, to_account, amount) VALUES ($1, $2, $3)",
		25*time.Millisecond,
		1,
	)
	
	// 5. Business logic - audit logging
	amount := decimal.NewFromFloat(250.75)
	auditLogger.WithRequestID(requestID).LogTransferWithDetails(
		789, 123, 456, amount, "USD", "Payment for services", "success", userID,
	)
	
	// 6. Error scenario - error logging
	errorCtx := ErrorContext{
		RequestID: requestID,
		UserID:    userID,
		UserEmail: userEmail,
		Operation: "validate_transfer_limits",
		Component: "transfer_service",
		Method:    "ValidateTransferLimits",
		Category:  BusinessLogicError,
		Severity:  MediumSeverity,
		Details: map[string]interface{}{
			"daily_limit":     1000.00,
			"current_amount":  250.75,
			"remaining_limit": 749.25,
		},
	}
	errorLogger.LogBusinessLogicError(errors.New("daily limit check warning"), errorCtx)
	
	// 7. End of request - performance logging
	duration := time.Since(startTime)
	perfLogger.WithRequestID(requestID).LogHTTPRequest("POST", "/api/v1/transfers", duration, 201)
	
	// 8. Background job - performance logging
	perfLogger.WithRequestID(requestID).LogBackgroundJobWithDetails(
		"send_transfer_notification", 150*time.Millisecond, true, 5, 0,
	)
	
	// Sync all logs
	err = lm.Sync()
	require.NoError(t, err)
	
	// Read and verify log file
	logFile := lm.GetCurrentLogFile()
	require.NotEmpty(t, logFile)
	
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	logContent := string(content)
	lines := strings.Split(strings.TrimSpace(logContent), "\n")
	
	// Should have 7 log entries
	validLines := 0
	var logEntries []map[string]interface{}
	
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			validLines++
			var entry map[string]interface{}
			err := json.Unmarshal([]byte(line), &entry)
			require.NoError(t, err, "Each log line should be valid JSON")
			logEntries = append(logEntries, entry)
		}
	}
	
	assert.Equal(t, 7, validLines, "Should have 7 log entries")
	
	// Verify request ID correlation across all entries
	for i, entry := range logEntries {
		assert.Equal(t, requestID, entry["request_id"], "Entry %d should have correct request ID", i)
	}
	
	// Verify specific log types and content
	logTypeCount := make(map[string]int)
	for _, entry := range logEntries {
		if logType, ok := entry["log_type"].(string); ok {
			logTypeCount[logType]++
		}
	}
	
	assert.Equal(t, 2, logTypeCount["audit"], "Should have 2 audit log entries")
	assert.Equal(t, 3, logTypeCount["performance"], "Should have 3 performance log entries")
	assert.Equal(t, 1, logTypeCount["error"], "Should have 1 error log entry")
	
	// Verify specific content
	assert.Contains(t, logContent, `"event_type":"authentication"`)
	assert.Contains(t, logContent, `"event_type":"transfer"`)
	assert.Contains(t, logContent, `"metric_type":"database_query"`)
	assert.Contains(t, logContent, `"metric_type":"http_request"`)
	assert.Contains(t, logContent, `"metric_type":"background_job"`)
	assert.Contains(t, logContent, `"category":"business_logic_error"`)
	assert.Contains(t, logContent, `"amount":"250.75"`)
	assert.Contains(t, logContent, userEmail)
}

// TestLogFileRotationAndCleanup tests log file creation, rotation, and cleanup
func TestLogFileRotationAndCleanup(t *testing.T) {
	tempDir := t.TempDir()
	
	config := LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "file",
		Directory:  tempDir,
		MaxAge:     1, // Keep files for 1 day
		MaxBackups: 3, // Keep 3 backup files
		Compress:   false,
	}
	
	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()
	
	logger := lm.GetLogger()
	
	// Write initial logs
	for i := 0; i < 50; i++ {
		logger.Info().Int("iteration", i).Msg("initial log entry")
	}
	
	// Get initial file list
	initialFiles, err := lm.GetLogFiles()
	require.NoError(t, err)
	assert.Len(t, initialFiles, 1, "Should start with 1 log file")
	
	// Force rotation
	err = lm.RotateLogFile()
	require.NoError(t, err)
	
	// Write more logs after rotation
	for i := 50; i < 100; i++ {
		logger.Info().Int("iteration", i).Msg("post-rotation log entry")
	}
	
	// Check files after rotation
	postRotationFiles, err := lm.GetLogFiles()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(postRotationFiles), 1, "Should have at least 1 log file after rotation")
	
	// Create old files to test cleanup
	now := time.Now()
	oldFiles := []struct {
		name    string
		age     time.Duration
	}{
		{"app-2023-01-01.log", 365 * 24 * time.Hour}, // Very old
		{"app-2023-06-01.log", 180 * 24 * time.Hour}, // Old
		{"app-2023-12-01.log", 30 * 24 * time.Hour},  // Should be cleaned up (older than MaxAge)
	}
	
	for _, of := range oldFiles {
		filePath := filepath.Join(tempDir, of.name)
		err := os.WriteFile(filePath, []byte("old log content"), 0644)
		require.NoError(t, err)
		
		// Set file modification time
		modTime := now.Add(-of.age)
		err = os.Chtimes(filePath, modTime, modTime)
		require.NoError(t, err)
	}
	
	// Trigger cleanup
	err = lm.CleanupLogFiles()
	require.NoError(t, err)
	
	// Verify cleanup worked
	finalFiles, err := lm.GetLogFiles()
	require.NoError(t, err)
	
	// Should not contain the very old files
	for _, file := range finalFiles {
		fileName := filepath.Base(file)
		assert.NotEqual(t, "app-2023-01-01.log", fileName, "Very old file should be cleaned up")
		assert.NotEqual(t, "app-2023-06-01.log", fileName, "Old file should be cleaned up")
		assert.NotEqual(t, "app-2023-12-01.log", fileName, "Old file should be cleaned up")
	}
	
	// Verify current log file still exists and has content
	currentFile := lm.GetCurrentLogFile()
	content, err := os.ReadFile(currentFile)
	require.NoError(t, err)
	
	logContent := string(content)
	assert.Contains(t, logContent, "post-rotation log entry")
}

// TestConcurrentLoggingAndRotation tests concurrent logging with rotation
func TestConcurrentLoggingAndRotation(t *testing.T) {
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
	
	logger := lm.GetLogger()
	
	const numGoroutines = 10
	const logsPerGoroutine = 100
	const rotationInterval = 50 * time.Millisecond
	
	var wg sync.WaitGroup
	stopRotation := make(chan bool)
	
	// Start concurrent logging
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info().
					Int("goroutine", id).
					Int("iteration", j).
					Str("data", fmt.Sprintf("data-%d-%d", id, j)).
					Msg("concurrent log entry")
				
				// Small delay to spread out the logging
				time.Sleep(time.Millisecond)
			}
		}(i)
	}
	
	// Start periodic rotation
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(rotationInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				err := lm.RotateLogFile()
				if err != nil {
					t.Logf("Rotation error: %v", err)
				}
			case <-stopRotation:
				return
			}
		}
	}()
	
	// Wait for logging to complete
	wg.Wait()
	close(stopRotation)
	
	// Final sync
	err = lm.Sync()
	require.NoError(t, err)
	
	// Verify all logs were written
	files, err := lm.GetLogFiles()
	require.NoError(t, err)
	
	totalLogCount := 0
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that might be in rotation
		}
		
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" && strings.Contains(line, "concurrent log entry") {
				totalLogCount++
			}
		}
	}
	
	expectedLogs := numGoroutines * logsPerGoroutine
	assert.Equal(t, expectedLogs, totalLogCount, "Should have all concurrent log entries across all files")
}

// TestLoggingPerformanceUnderLoad tests logging performance under high load
func TestLoggingPerformanceUnderLoad(t *testing.T) {
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
	
	logger := lm.GetLogger()
	
	// Measure logging performance
	const numLogs = 10000
	
	startTime := time.Now()
	
	for i := 0; i < numLogs; i++ {
		logger.Info().
			Int("iteration", i).
			Str("request_id", fmt.Sprintf("req-%d", i)).
			Float64("amount", float64(i)*1.5).
			Bool("success", i%2 == 0).
			Msg("performance test log entry")
	}
	
	// Sync to ensure all logs are written
	err = lm.Sync()
	require.NoError(t, err)
	
	duration := time.Since(startTime)
	
	// Calculate performance metrics
	logsPerSecond := float64(numLogs) / duration.Seconds()
	avgTimePerLog := duration / numLogs
	
	t.Logf("Performance metrics:")
	t.Logf("  Total logs: %d", numLogs)
	t.Logf("  Total time: %v", duration)
	t.Logf("  Logs per second: %.2f", logsPerSecond)
	t.Logf("  Average time per log: %v", avgTimePerLog)
	
	// Performance assertions (these are reasonable expectations for zerolog)
	assert.Greater(t, logsPerSecond, 1000.0, "Should log at least 1000 entries per second")
	assert.Less(t, avgTimePerLog, time.Millisecond, "Average time per log should be less than 1ms")
	
	// Verify log file integrity
	logFile := lm.GetCurrentLogFile()
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Equal(t, numLogs, len(lines), "Should have all log entries in file")
	
	// Verify a few random entries
	for i := 0; i < 10; i++ {
		lineIndex := i * (numLogs / 10)
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(lines[lineIndex]), &logEntry)
		assert.NoError(t, err, "Log entry %d should be valid JSON", lineIndex)
		assert.Equal(t, float64(lineIndex), logEntry["iteration"])
	}
}

// TestMemoryUsageUnderLoad tests memory usage during sustained logging
func TestMemoryUsageUnderLoad(t *testing.T) {
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
	
	logger := lm.GetLogger()
	
	// Get initial memory stats
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// Log many entries with varying sizes
	const numLogs = 5000
	
	for i := 0; i < numLogs; i++ {
		// Create varying sized log entries
		dataSize := (i % 100) + 1 // 1 to 100 characters
		data := strings.Repeat("x", dataSize)
		
		logger.Info().
			Int("iteration", i).
			Str("variable_data", data).
			Str("request_id", fmt.Sprintf("req-%d", i)).
			Int("data_size", dataSize).
			Msg("memory test log entry")
		
		// Periodic sync to prevent excessive buffering
		if i%1000 == 0 {
			lm.Sync()
		}
	}
	
	// Final sync
	err = lm.Sync()
	require.NoError(t, err)
	
	// Get final memory stats
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	// Calculate memory usage
	allocDiff := m2.Alloc - m1.Alloc
	totalAllocDiff := m2.TotalAlloc - m1.TotalAlloc
	
	t.Logf("Memory usage:")
	t.Logf("  Current alloc diff: %d bytes", allocDiff)
	t.Logf("  Total alloc diff: %d bytes", totalAllocDiff)
	t.Logf("  Bytes per log (current): %.2f", float64(allocDiff)/numLogs)
	t.Logf("  Bytes per log (total): %.2f", float64(totalAllocDiff)/numLogs)
	
	// Memory usage should be reasonable (these are rough estimates)
	bytesPerLogCurrent := float64(allocDiff) / numLogs
	bytesPerLogTotal := float64(totalAllocDiff) / numLogs
	
	// Current allocation should be low (most memory should be freed)
	assert.Less(t, bytesPerLogCurrent, 100.0, "Current memory per log should be less than 100 bytes")
	
	// Total allocation should be reasonable for structured logging
	assert.Less(t, bytesPerLogTotal, 1000.0, "Total memory per log should be less than 1000 bytes")
	
	// Verify log file was created correctly
	logFile := lm.GetCurrentLogFile()
	stat, err := os.Stat(logFile)
	require.NoError(t, err)
	
	// File should have reasonable size
	assert.Greater(t, stat.Size(), int64(numLogs*50), "Log file should be at least 50 bytes per entry")
}

// TestHTTPMiddlewareIntegration tests integration with HTTP middleware
func TestHTTPMiddlewareIntegration(t *testing.T) {
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
	
	// Create a simple HTTP middleware that uses our logging
	loggingMiddleware := func(logger zerolog.Logger) gin.HandlerFunc {
		return func(c *gin.Context) {
			start := time.Now()
			requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
			
			// Add request context
			c.Set("request_id", requestID)
			c.Set("user_id", int64(123))
			c.Set("user_email", "test@example.com")
			
			// Create context logger
			ctx := context.WithValue(c.Request.Context(), "request_id", requestID)
			ctx = context.WithValue(ctx, "user_id", int64(123))
			ctx = context.WithValue(ctx, "user_email", "test@example.com")
			
			contextLogger := NewContextLogger(logger, ctx)
			
			// Log request start
			contextLogger.Info().
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Str("client_ip", c.ClientIP()).
				Msg("Request started")
			
			// Process request
			c.Next()
			
			// Log request completion
			duration := time.Since(start)
			status := c.Writer.Status()
			
			perfLogger := NewPerformanceLogger(logger)
			perfLogger.WithRequestID(requestID).LogHTTPRequest(
				c.Request.Method,
				c.Request.URL.Path,
				duration,
				status,
			)
			
			contextLogger.Info().
				Int("status", status).
				Dur("duration", duration).
				Msg("Request completed")
		}
	}
	
	// Set up Gin router with logging middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(loggingMiddleware(lm.GetLogger()))
	
	// Add test routes
	router.GET("/api/users/:id", func(c *gin.Context) {
		// Simulate some business logic with logging
		userID := c.Param("id")
		
		ctx := c.Request.Context()
		contextLogger := NewContextLogger(lm.GetLogger(), ctx)
		
		contextLogger.WithComponent("user_service").Info().
			Str("user_id", userID).
			Msg("Fetching user details")
		
		c.JSON(200, gin.H{"id": userID, "name": "Test User"})
	})
	
	router.POST("/api/transfers", func(c *gin.Context) {
		// Simulate transfer with audit logging
		auditLogger := NewAuditLogger(lm.GetLogger())
		
		if requestID, exists := c.Get("request_id"); exists {
			auditLogger = auditLogger.WithRequestID(requestID.(string))
		}
		
		auditLogger.LogTransfer(123, 456, decimal.NewFromFloat(100.50), "success")
		
		c.JSON(201, gin.H{"status": "success", "transfer_id": "txn_123"})
	})
	
	// Test the endpoints
	testCases := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/api/users/123", 200},
		{"POST", "/api/transfers", 201},
		{"GET", "/api/users/456", 200},
	}
	
	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		assert.Equal(t, tc.status, w.Code, "Request %s %s should return status %d", tc.method, tc.path, tc.status)
	}
	
	// Sync logs and verify
	err = lm.Sync()
	require.NoError(t, err)
	
	logFile := lm.GetCurrentLogFile()
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	logContent := string(content)
	
	// Verify request logging
	assert.Contains(t, logContent, "Request started")
	assert.Contains(t, logContent, "Request completed")
	assert.Contains(t, logContent, "Fetching user details")
	
	// Verify performance logging
	assert.Contains(t, logContent, `"metric_type":"http_request"`)
	assert.Contains(t, logContent, `"method":"GET"`)
	assert.Contains(t, logContent, `"method":"POST"`)
	assert.Contains(t, logContent, `"path":"/api/users/123"`)
	assert.Contains(t, logContent, `"path":"/api/transfers"`)
	
	// Verify audit logging
	assert.Contains(t, logContent, `"log_type":"audit"`)
	assert.Contains(t, logContent, `"event_type":"transfer"`)
	assert.Contains(t, logContent, `"amount":"100.50"`)
	
	// Count log entries
	lines := strings.Split(strings.TrimSpace(logContent), "\n")
	validLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			validLines++
		}
	}
	
	// Should have multiple log entries per request
	assert.GreaterOrEqual(t, validLines, len(testCases)*2, "Should have at least 2 log entries per request")
}

// TestErrorRecoveryAndResilience tests logging system resilience
func TestErrorRecoveryAndResilience(t *testing.T) {
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
	
	logger := lm.GetLogger()
	
	// Test 1: Log before any issues
	logger.Info().Msg("initial log entry")
	
	// Test 2: Simulate disk space issues by filling up the directory
	// (This is a simulation - actual disk space issues are hard to test reliably)
	
	// Test 3: Log with invalid JSON characters
	logger.Info().
		Str("field_with_quotes", `"quoted string"`).
		Str("field_with_newlines", "line1\nline2\nline3").
		Str("field_with_unicode", "测试 🚀 émojis").
		Msg("log with special characters")
	
	// Test 4: Log with very large data
	largeData := strings.Repeat("x", 10*1024) // 10KB
	logger.Info().
		Str("large_field", largeData).
		Msg("log with large data")
	
	// Test 5: Concurrent logging during rotation
	var wg sync.WaitGroup
	
	// Start background logging
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			logger.Info().Int("bg_iteration", i).Msg("background log entry")
			time.Sleep(time.Millisecond)
		}
	}()
	
	// Trigger rotation during logging
	time.Sleep(50 * time.Millisecond)
	err = lm.RotateLogFile()
	assert.NoError(t, err, "Rotation should succeed even during concurrent logging")
	
	wg.Wait()
	
	// Test 6: Health check
	err = lm.HealthCheck()
	assert.NoError(t, err, "Health check should pass")
	
	// Sync and verify
	err = lm.Sync()
	require.NoError(t, err)
	
	// Verify log files exist and contain data
	files, err := lm.GetLogFiles()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1, "Should have at least one log file")
	
	// Read all log files and verify content
	totalLogCount := 0
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				totalLogCount++
				
				// Verify each line is valid JSON
				var logEntry map[string]interface{}
				err := json.Unmarshal([]byte(line), &logEntry)
				assert.NoError(t, err, "Each log line should be valid JSON")
			}
		}
	}
	
	assert.GreaterOrEqual(t, totalLogCount, 103, "Should have at least 103 log entries") // initial + special chars + large data + 100 background
}

// TestLoggerHealthCheck tests the health check functionality
func TestLoggerHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() LogConfig
		expectError bool
	}{
		{
			name: "healthy file logging",
			setupConfig: func() LogConfig {
				return LogConfig{
					Level:     "info",
					Format:    "json",
					Output:    "file",
					Directory: t.TempDir(),
				}
			},
			expectError: false,
		},
		{
			name: "healthy console logging",
			setupConfig: func() LogConfig {
				return LogConfig{
					Level:  "info",
					Format: "json",
					Output: "console",
				}
			},
			expectError: false,
		},
		{
			name: "invalid directory",
			setupConfig: func() LogConfig {
				return LogConfig{
					Level:     "info",
					Format:    "json",
					Output:    "file",
					Directory: "/nonexistent/directory/that/should/not/exist",
				}
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setupConfig()
			
			lm, err := NewLoggerManager(config)
			if tt.expectError && err != nil {
				// Expected error during creation
				return
			}
			require.NoError(t, err)
			defer lm.Close()
			
			// Test health check
			err = lm.HealthCheck()
			if tt.expectError {
				assert.Error(t, err, "Health check should fail for invalid configuration")
			} else {
				assert.NoError(t, err, "Health check should pass for valid configuration")
			}
		})
	}
}