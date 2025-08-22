package logging

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDailyFileWriterThreadSafety tests thread safety of DailyFileWriter
func TestDailyFileWriterThreadSafety(t *testing.T) {
	tempDir := t.TempDir()
	
	config := DailyFileConfig{
		Directory:  tempDir,
		MaxAge:     1,
		MaxBackups: 5,
		Compress:   false,
		LocalTime:  true,
	}
	
	writer, err := NewDailyFileWriter(config)
	require.NoError(t, err)
	defer writer.Close()
	
	const numGoroutines = 20
	const writesPerGoroutine = 100
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < writesPerGoroutine; j++ {
				data := fmt.Sprintf("goroutine-%d-write-%d\n", id, j)
				n, err := writer.Write([]byte(data))
				assert.NoError(t, err)
				assert.Equal(t, len(data), n)
			}
		}(i)
	}
	
	// Concurrent rotations
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		for i := 0; i < 5; i++ {
			time.Sleep(50 * time.Millisecond)
			err := writer.Rotate()
			// Rotation might fail if already rotated, that's ok
			if err != nil {
				t.Logf("Rotation %d failed: %v", i, err)
			}
		}
	}()
	
	// Concurrent syncs
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		for i := 0; i < 10; i++ {
			time.Sleep(25 * time.Millisecond)
			err := writer.Sync()
			assert.NoError(t, err)
		}
	}()
	
	wg.Wait()
	
	// Final sync and verification
	err = writer.Sync()
	require.NoError(t, err)
	
	// Count total writes across all files
	files, err := writer.GetLogFiles()
	require.NoError(t, err)
	
	totalWrites := 0
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue // File might be in rotation
		}
		
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.Contains(line, "goroutine-") && strings.Contains(line, "-write-") {
				totalWrites++
			}
		}
	}
	
	expectedWrites := numGoroutines * writesPerGoroutine
	assert.Equal(t, expectedWrites, totalWrites, "All writes should be present across all files")
}

// TestLoggerManagerThreadSafety tests thread safety of LoggerManager
func TestLoggerManagerThreadSafety(t *testing.T) {
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
	
	const numGoroutines = 15
	const operationsPerGoroutine = 50
	
	var wg sync.WaitGroup
	
	// Concurrent logging
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			logger := lm.GetLogger()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				logger.Info().
					Int("goroutine", id).
					Int("operation", j).
					Msg("thread safety test")
			}
		}(i)
	}
	
	// Concurrent context logger creation
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			ctx := context.Background()
			ctx = context.WithValue(ctx, "request_id", fmt.Sprintf("req-%d", id))
			ctx = context.WithValue(ctx, "user_id", int64(id))
			
			for j := 0; j < operationsPerGoroutine; j++ {
				contextLogger := lm.GetContextLogger(ctx)
				contextLogger.Info().
					Int("operation", j).
					Msg("context logger thread safety test")
			}
		}(i)
	}
	
	// Concurrent field additions
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			fields := map[string]interface{}{
				"goroutine": id,
				"timestamp": time.Now().Unix(),
			}
			
			for j := 0; j < operationsPerGoroutine; j++ {
				logger := lm.WithFields(fields)
				logger.Info().
					Int("operation", j).
					Msg("fields thread safety test")
			}
		}(i)
	}
	
	// Concurrent health checks
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		for i := 0; i < 10; i++ {
			err := lm.HealthCheck()
			assert.NoError(t, err)
			time.Sleep(20 * time.Millisecond)
		}
	}()
	
	// Concurrent syncs
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		for i := 0; i < 10; i++ {
			err := lm.Sync()
			assert.NoError(t, err)
			time.Sleep(30 * time.Millisecond)
		}
	}()
	
	wg.Wait()
	
	// Final verification
	err = lm.Sync()
	require.NoError(t, err)
	
	// Verify log file integrity
	logFile := lm.GetCurrentLogFile()
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	validLines := 0
	
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			validLines++
		}
	}
	
	expectedLogs := numGoroutines * operationsPerGoroutine * 3 // 3 types of concurrent operations
	assert.Equal(t, expectedLogs, validLines, "All concurrent operations should be logged")
}

// TestSpecializedLoggersThreadSafety tests thread safety of specialized loggers
func TestSpecializedLoggersThreadSafety(t *testing.T) {
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
	
	// Create specialized loggers
	auditLogger := NewAuditLogger(lm.GetLogger())
	perfLogger := NewPerformanceLogger(lm.GetLogger())
	errorLogger := NewErrorLogger(lm.GetLogger())
	
	const numGoroutines = 10
	const operationsPerGoroutine = 30
	
	var wg sync.WaitGroup
	
	// Concurrent audit logging
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix different audit operations
				switch j % 4 {
				case 0:
					auditLogger.LogAuthentication(
						int64(id*1000+j),
						fmt.Sprintf("user%d@example.com", id),
						"login",
						"success",
					)
				case 1:
					auditLogger.LogAccountOperation(
						int64(id*1000+j),
						int64(id*100+j),
						"create",
						"success",
					)
				case 2:
					amount := decimal.NewFromFloat(float64(j) * 10.5)
					auditLogger.LogTransfer(
						int64(id*100+j),
						int64(id*100+j+1),
						amount,
						"success",
					)
				case 3:
					auditLogger.LogSecurityEvent(
						"test_event",
						"test_source",
						fmt.Sprintf("details-%d-%d", id, j),
					)
				}
			}
		}(i)
	}
	
	// Concurrent performance logging
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix different performance operations
				switch j % 3 {
				case 0:
					perfLogger.LogHTTPRequest(
						"GET",
						fmt.Sprintf("/api/test/%d", j),
						time.Duration(100+j)*time.Millisecond,
						200,
					)
				case 1:
					perfLogger.LogDatabaseQuery(
						fmt.Sprintf("SELECT * FROM table_%d", j),
						time.Duration(50+j)*time.Millisecond,
						int64(j+1),
					)
				case 2:
					perfLogger.LogBackgroundJob(
						fmt.Sprintf("job_%d", j),
						time.Duration(200+j)*time.Millisecond,
						j%2 == 0,
					)
				}
			}
		}(i)
	}
	
	// Concurrent error logging
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				errorCtx := ErrorContext{
					RequestID: fmt.Sprintf("req-%d-%d", id, j),
					UserID:    int64(id),
					Operation: fmt.Sprintf("operation_%d", j),
					Component: fmt.Sprintf("component_%d", id),
					Category:  ValidationError,
					Severity:  MediumSeverity,
				}
				
				err := fmt.Errorf("test error %d-%d", id, j)
				errorLogger.LogError(err, errorCtx)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Final verification
	err = lm.Sync()
	require.NoError(t, err)
	
	// Verify log file integrity and content
	logFile := lm.GetCurrentLogFile()
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	logContent := string(content)
	
	// Verify different log types are present
	assert.Contains(t, logContent, `"log_type":"audit"`)
	assert.Contains(t, logContent, `"log_type":"performance"`)
	assert.Contains(t, logContent, `"log_type":"error"`)
	
	// Count log entries by type
	auditCount := strings.Count(logContent, `"log_type":"audit"`)
	perfCount := strings.Count(logContent, `"log_type":"performance"`)
	errorCount := strings.Count(logContent, `"log_type":"error"`)
	
	expectedPerType := numGoroutines * operationsPerGoroutine
	
	assert.Equal(t, expectedPerType, auditCount, "Should have all audit log entries")
	assert.Equal(t, expectedPerType, perfCount, "Should have all performance log entries")
	assert.Equal(t, expectedPerType, errorCount, "Should have all error log entries")
}

// TestContextLoggerThreadSafety tests thread safety of ContextLogger
func TestContextLoggerThreadSafety(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	
	const numGoroutines = 20
	const operationsPerGoroutine = 50
	
	var wg sync.WaitGroup
	
	// Test concurrent context logger creation and modification
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			ctx := context.Background()
			ctx = context.WithValue(ctx, "request_id", fmt.Sprintf("req-%d", id))
			ctx = context.WithValue(ctx, "user_id", int64(id))
			
			baseLogger := NewContextLogger(logger, ctx)
			
			for j := 0; j < operationsPerGoroutine; j++ {
				// Create modified loggers concurrently
				modifiedLogger := baseLogger.
					WithRequestID(fmt.Sprintf("req-%d-%d", id, j)).
					WithUser(int64(id*1000+j), fmt.Sprintf("user%d@example.com", id)).
					WithComponent(fmt.Sprintf("component_%d", j)).
					WithOperation(fmt.Sprintf("operation_%d", j)).
					WithField("iteration", j)
				
				// Log with modified logger
				modifiedLogger.Info().
					Int("goroutine", id).
					Int("operation", j).
					Msg("context logger thread safety test")
				
				// Verify original logger is unchanged
				assert.Equal(t, fmt.Sprintf("req-%d", id), baseLogger.GetRequestID())
				assert.Equal(t, int64(id), baseLogger.GetUserID())
			}
		}(i)
	}
	
	wg.Wait()
}

// TestErrorMonitorThreadSafety tests thread safety of ErrorMonitor
func TestErrorMonitorThreadSafety(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	const numGoroutines = 15
	const operationsPerGoroutine = 40
	
	var wg sync.WaitGroup
	
	// Concurrent error tracking
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				errorCtx := ErrorContext{
					Operation: fmt.Sprintf("operation_%d", j%5), // Create patterns
					Component: fmt.Sprintf("component_%d", id%3),
					Category:  ErrorCategory([]ErrorCategory{ValidationError, AuthenticationError, DatabaseError}[j%3]),
					Severity:  ErrorSeverity([]ErrorSeverity{LowSeverity, MediumSeverity, HighSeverity}[j%3]),
				}
				
				monitor.TrackError(errorCtx)
			}
		}(i)
	}
	
	// Concurrent threshold management
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		for i := 0; i < 10; i++ {
			threshold := AlertThreshold{
				Category:      ValidationError,
				Component:     fmt.Sprintf("component_%d", i%3),
				MaxCount:      10,
				TimeWindow:    1 * time.Minute,
				Severity:      MediumSeverity,
				AlertInterval: 100 * time.Millisecond,
			}
			
			monitor.AddThreshold(threshold)
			
			time.Sleep(50 * time.Millisecond)
			
			// Remove some thresholds
			if i%2 == 0 {
				monitor.RemoveThreshold(ValidationError, fmt.Sprintf("component_%d", i%3), "")
			}
		}
	}()
	
	// Concurrent statistics reading
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		for i := 0; i < 20; i++ {
			stats := monitor.GetErrorStats()
			frequencies := monitor.GetErrorFrequencies()
			topErrors := monitor.GetTopErrors(5)
			
			// Basic sanity checks
			assert.NotNil(t, stats)
			assert.NotNil(t, frequencies)
			assert.NotNil(t, topErrors)
			
			time.Sleep(25 * time.Millisecond)
		}
	}()
	
	wg.Wait()
	
	// Final verification
	stats := monitor.GetErrorStats()
	totalExpected := int64(numGoroutines * operationsPerGoroutine)
	
	var totalActual int64
	for _, count := range stats {
		totalActual += count
	}
	
	assert.Equal(t, totalExpected, totalActual, "All errors should be tracked")
	
	frequencies := monitor.GetErrorFrequencies()
	assert.Greater(t, len(frequencies), 0, "Should have error frequencies")
	
	topErrors := monitor.GetTopErrors(10)
	assert.Greater(t, len(topErrors), 0, "Should have top errors")
}

// TestConcurrentRotationAndLogging tests concurrent rotation and logging
func TestConcurrentRotationAndLogging(t *testing.T) {
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
	
	const testDuration = 1 * time.Second
	const numLoggers = 8
	
	stopChan := make(chan bool)
	var wg sync.WaitGroup
	var totalLogs int64
	var mu sync.Mutex
	
	// Start concurrent loggers
	wg.Add(numLoggers)
	for i := 0; i < numLoggers; i++ {
		go func(loggerID int) {
			defer wg.Done()
			
			logCount := 0
			for {
				select {
				case <-stopChan:
					mu.Lock()
					totalLogs += int64(logCount)
					mu.Unlock()
					return
				default:
					logger.Info().
						Int("logger_id", loggerID).
						Int("count", logCount).
						Time("timestamp", time.Now()).
						Msg("concurrent rotation test")
					logCount++
				}
			}
		}(i)
	}
	
	// Start rotation goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		rotationCount := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-stopChan:
				t.Logf("Performed %d rotations during test", rotationCount)
				return
			case <-ticker.C:
				err := lm.RotateLogFile()
				if err != nil {
					t.Logf("Rotation error: %v", err)
				} else {
					rotationCount++
				}
			}
		}
	}()
	
	// Run test
	time.Sleep(testDuration)
	close(stopChan)
	wg.Wait()
	
	// Final sync
	err = lm.Sync()
	require.NoError(t, err)
	
	t.Logf("Concurrent rotation test completed:")
	t.Logf("  Total logs: %d", totalLogs)
	t.Logf("  Logs per second: %.2f", float64(totalLogs)/testDuration.Seconds())
	
	// Verify all logs are present across all files
	files, err := lm.GetLogFiles()
	require.NoError(t, err)
	
	totalFoundLogs := int64(0)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		for _, line := range lines {
			if strings.Contains(line, "concurrent rotation test") {
				totalFoundLogs++
			}
		}
	}
	
	assert.Equal(t, totalLogs, totalFoundLogs, "All logs should be found across all files")
	assert.Greater(t, totalLogs, int64(500), "Should generate reasonable number of logs")
}