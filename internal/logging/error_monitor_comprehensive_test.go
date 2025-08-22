package logging

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorMonitorComprehensive tests comprehensive error monitoring functionality
func TestErrorMonitorComprehensive(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Test error tracking
	errorCtx1 := ErrorContext{
		Operation: "create_user",
		Component: "user_service",
		Category:  ValidationError,
		Severity:  LowSeverity,
	}
	
	errorCtx2 := ErrorContext{
		Operation: "authenticate_user",
		Component: "auth_service",
		Category:  AuthenticationError,
		Severity:  MediumSeverity,
	}
	
	// Track multiple errors
	for i := 0; i < 5; i++ {
		monitor.TrackError(errorCtx1)
	}
	
	for i := 0; i < 3; i++ {
		monitor.TrackError(errorCtx2)
	}
	
	// Get error frequencies
	frequencies := monitor.GetErrorFrequencies()
	assert.Len(t, frequencies, 2, "Should track 2 different error patterns")
	
	// Verify frequencies
	var validationFreq, authFreq *ErrorFrequency
	for _, freq := range frequencies {
		if freq.Category == ValidationError {
			validationFreq = freq
		} else if freq.Category == AuthenticationError {
			authFreq = freq
		}
	}
	
	require.NotNil(t, validationFreq, "Should have validation error frequency")
	require.NotNil(t, authFreq, "Should have auth error frequency")
	
	assert.Equal(t, int64(5), validationFreq.Count)
	assert.Equal(t, "user_service", validationFreq.Component)
	assert.Equal(t, "create_user", validationFreq.Operation)
	
	assert.Equal(t, int64(3), authFreq.Count)
	assert.Equal(t, "auth_service", authFreq.Component)
	assert.Equal(t, "authenticate_user", authFreq.Operation)
	
	// Test error statistics
	stats := monitor.GetErrorStats()
	assert.Equal(t, int64(5), stats[ValidationError])
	assert.Equal(t, int64(3), stats[AuthenticationError])
	
	// Test top errors
	topErrors := monitor.GetTopErrors(5)
	assert.Len(t, topErrors, 2)
	
	// Should be sorted by count (descending)
	assert.True(t, topErrors[0].Count >= topErrors[1].Count)
	assert.Equal(t, int64(5), topErrors[0].Count) // Validation errors should be first
}

// TestErrorMonitorAlerting tests error alerting functionality
func TestErrorMonitorAlerting(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Add alert threshold
	threshold := AlertThreshold{
		Category:      ValidationError,
		Component:     "user_service",
		Operation:     "create_user",
		MaxCount:      3,
		TimeWindow:    1 * time.Minute,
		Severity:      MediumSeverity,
		AlertInterval: 100 * time.Millisecond,
	}
	
	monitor.AddThreshold(threshold)
	
	// Verify threshold was added
	thresholds := monitor.GetThresholds()
	assert.Len(t, thresholds, 1)
	
	// Find the threshold in the map
	found := false
	for _, th := range thresholds {
		if th.Category == threshold.Category {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find the added threshold")
	
	// Clear buffer to focus on alert messages
	buf.Reset()
	
	// Generate errors that should trigger alert
	errorCtx := ErrorContext{
		Operation: "create_user",
		Component: "user_service",
		Category:  ValidationError,
		Severity:  LowSeverity,
	}
	
	// Generate errors up to threshold
	for i := 0; i < 4; i++ { // Exceed threshold of 3
		monitor.TrackError(errorCtx)
	}
	
	// Give time for alert processing
	time.Sleep(200 * time.Millisecond)
	
	// Check for alert in logs
	logOutput := buf.String()
	assert.Contains(t, logOutput, "error_threshold_exceeded", "Should contain threshold exceeded alert")
	assert.Contains(t, logOutput, "validation_error", "Should mention error category")
	assert.Contains(t, logOutput, "user_service", "Should mention component")
	
	// Remove threshold
	monitor.RemoveThreshold(ValidationError, "user_service", "create_user")
	
	thresholds = monitor.GetThresholds()
	assert.Len(t, thresholds, 0, "Should have no thresholds after removal")
}

// TestErrorMonitorConcurrency tests concurrent error tracking
func TestErrorMonitorConcurrency(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	const numGoroutines = 10
	const errorsPerGoroutine = 100
	
	var wg sync.WaitGroup
	
	// Concurrent error tracking
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			errorCtx := ErrorContext{
				Operation: "concurrent_operation",
				Component: "test_component",
				Category:  SystemError,
				Severity:  MediumSeverity,
			}
			
			for j := 0; j < errorsPerGoroutine; j++ {
				monitor.TrackError(errorCtx)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify all errors were tracked
	stats := monitor.GetErrorStats()
	expectedTotal := int64(numGoroutines * errorsPerGoroutine)
	assert.Equal(t, expectedTotal, stats[SystemError], "Should track all concurrent errors")
	
	frequencies := monitor.GetErrorFrequencies()
	assert.Len(t, frequencies, 1, "Should have one error pattern")
	
	// Find the frequency in the map
	var totalCount int64
	for _, freq := range frequencies {
		totalCount += freq.Count
	}
	assert.Equal(t, expectedTotal, totalCount)
}

// TestErrorMonitorTimeWindows tests time window functionality for alerts
func TestErrorMonitorTimeWindows(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Add threshold with short time window
	threshold := AlertThreshold{
		Category:      ValidationError,
		MaxCount:      2,
		TimeWindow:    200 * time.Millisecond,
		Severity:      MediumSeverity,
		AlertInterval: 50 * time.Millisecond,
	}
	
	monitor.AddThreshold(threshold)
	
	errorCtx := ErrorContext{
		Category: ValidationError,
		Severity: LowSeverity,
	}
	
	// Clear buffer
	buf.Reset()
	
	// Generate errors within time window
	monitor.TrackError(errorCtx)
	monitor.TrackError(errorCtx)
	monitor.TrackError(errorCtx) // Should trigger alert
	
	// Give time for alert processing
	time.Sleep(100 * time.Millisecond)
	
	// Should have alert
	logOutput := buf.String()
	assert.Contains(t, logOutput, "error_threshold_exceeded", "Should trigger alert within time window")
	
	// Clear buffer and wait for time window to expire
	buf.Reset()
	time.Sleep(300 * time.Millisecond) // Wait longer than time window
	
	// Generate more errors after time window
	monitor.TrackError(errorCtx)
	monitor.TrackError(errorCtx)
	
	// Give time for potential alert processing
	time.Sleep(100 * time.Millisecond)
	
	// Should not trigger alert (errors are outside time window)
	logOutput = buf.String()
	assert.NotContains(t, logOutput, "error_threshold_exceeded", "Should not trigger alert outside time window")
}

// TestErrorMonitorAlertInterval tests alert interval functionality
func TestErrorMonitorAlertInterval(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Add threshold with alert interval
	threshold := AlertThreshold{
		Category:      ValidationError,
		MaxCount:      1,
		TimeWindow:    1 * time.Minute,
		Severity:      MediumSeverity,
		AlertInterval: 200 * time.Millisecond,
	}
	
	monitor.AddThreshold(threshold)
	
	errorCtx := ErrorContext{
		Category: ValidationError,
		Severity: LowSeverity,
	}
	
	// Clear buffer
	buf.Reset()
	
	// Generate multiple errors quickly
	for i := 0; i < 5; i++ {
		monitor.TrackError(errorCtx)
		time.Sleep(10 * time.Millisecond)
	}
	
	// Give time for alert processing
	time.Sleep(100 * time.Millisecond)
	
	// Count alerts in log output
	logOutput := buf.String()
	alertCount := strings.Count(logOutput, "error_threshold_exceeded")
	
	// Should have limited number of alerts due to alert interval
	assert.LessOrEqual(t, alertCount, 2, "Should limit alerts based on alert interval")
	assert.GreaterOrEqual(t, alertCount, 1, "Should have at least one alert")
}

// TestErrorMonitorSeverityFiltering tests severity-based filtering
func TestErrorMonitorSeverityFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Add threshold that only alerts on high severity
	threshold := AlertThreshold{
		Category:      SystemError,
		MaxCount:      1,
		TimeWindow:    1 * time.Minute,
		Severity:      HighSeverity, // Only alert on high severity
		AlertInterval: 50 * time.Millisecond,
	}
	
	monitor.AddThreshold(threshold)
	
	// Clear buffer
	buf.Reset()
	
	// Generate low severity errors (should not alert)
	lowSeverityCtx := ErrorContext{
		Category: SystemError,
		Severity: LowSeverity,
	}
	
	for i := 0; i < 3; i++ {
		monitor.TrackError(lowSeverityCtx)
	}
	
	time.Sleep(100 * time.Millisecond)
	
	// Should not have alerts for low severity
	logOutput := buf.String()
	assert.NotContains(t, logOutput, "error_threshold_exceeded", "Should not alert on low severity errors")
	
	// Generate high severity error (should alert)
	highSeverityCtx := ErrorContext{
		Category: SystemError,
		Severity: HighSeverity,
	}
	
	monitor.TrackError(highSeverityCtx)
	
	time.Sleep(100 * time.Millisecond)
	
	// Should have alert for high severity
	logOutput = buf.String()
	assert.Contains(t, logOutput, "error_threshold_exceeded", "Should alert on high severity errors")
}

// TestErrorMonitorPatternMatching tests error pattern matching
func TestErrorMonitorPatternMatching(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Add specific threshold for component and operation
	specificThreshold := AlertThreshold{
		Category:      ValidationError,
		Component:     "user_service",
		Operation:     "create_user",
		MaxCount:      2,
		TimeWindow:    1 * time.Minute,
		Severity:      LowSeverity,
		AlertInterval: 50 * time.Millisecond,
	}
	
	// Add general threshold for category only
	generalThreshold := AlertThreshold{
		Category:      ValidationError,
		MaxCount:      5,
		TimeWindow:    1 * time.Minute,
		Severity:      LowSeverity,
		AlertInterval: 50 * time.Millisecond,
	}
	
	monitor.AddThreshold(specificThreshold)
	monitor.AddThreshold(generalThreshold)
	
	// Clear buffer
	buf.Reset()
	
	// Generate errors that match specific threshold
	specificCtx := ErrorContext{
		Category:  ValidationError,
		Component: "user_service",
		Operation: "create_user",
		Severity:  LowSeverity,
	}
	
	for i := 0; i < 3; i++ { // Exceed specific threshold of 2
		monitor.TrackError(specificCtx)
	}
	
	time.Sleep(100 * time.Millisecond)
	
	// Should trigger specific threshold
	logOutput := buf.String()
	assert.Contains(t, logOutput, "error_threshold_exceeded", "Should trigger specific threshold")
	assert.Contains(t, logOutput, "user_service", "Should mention specific component")
	assert.Contains(t, logOutput, "create_user", "Should mention specific operation")
	
	// Clear buffer
	buf.Reset()
	
	// Generate errors that match general threshold but not specific
	generalCtx := ErrorContext{
		Category:  ValidationError,
		Component: "account_service", // Different component
		Operation: "create_account",  // Different operation
		Severity:  LowSeverity,
	}
	
	for i := 0; i < 6; i++ { // Exceed general threshold of 5
		monitor.TrackError(generalCtx)
	}
	
	time.Sleep(100 * time.Millisecond)
	
	// Should trigger general threshold
	logOutput = buf.String()
	assert.Contains(t, logOutput, "error_threshold_exceeded", "Should trigger general threshold")
	assert.Contains(t, logOutput, "account_service", "Should mention general component")
}

// TestErrorMonitorCleanup tests cleanup of old error data
func TestErrorMonitorCleanup(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
		CleanupInterval: 100 * time.Millisecond,
		MaxAge:         200 * time.Millisecond,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	errorCtx := ErrorContext{
		Category:  ValidationError,
		Component: "test_service",
		Operation: "test_operation",
		Severity:  LowSeverity,
	}
	
	// Generate some errors
	for i := 0; i < 5; i++ {
		monitor.TrackError(errorCtx)
	}
	
	// Verify errors are tracked
	stats := monitor.GetErrorStats()
	assert.Equal(t, int64(5), stats[ValidationError], "Should track all errors initially")
	
	// Wait for cleanup to occur
	time.Sleep(300 * time.Millisecond)
	
	// Errors should be cleaned up
	stats = monitor.GetErrorStats()
	assert.Equal(t, int64(0), stats[ValidationError], "Should clean up old errors")
	
	frequencies := monitor.GetErrorFrequencies()
	assert.Len(t, frequencies, 0, "Should have no error frequencies after cleanup")
}

// TestErrorMonitorDisabled tests behavior when monitoring is disabled
func TestErrorMonitorDisabled(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: false,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	errorCtx := ErrorContext{
		Category:  ValidationError,
		Component: "test_service",
		Severity:  LowSeverity,
	}
	
	// Try to track errors
	for i := 0; i < 5; i++ {
		monitor.TrackError(errorCtx)
	}
	
	// Should not track anything
	stats := monitor.GetErrorStats()
	assert.Equal(t, int64(0), stats[ValidationError], "Should not track errors when disabled")
	
	frequencies := monitor.GetErrorFrequencies()
	assert.Len(t, frequencies, 0, "Should have no error frequencies when disabled")
	
	// Try to add threshold
	threshold := AlertThreshold{
		Category: ValidationError,
		MaxCount: 1,
	}
	
	monitor.AddThreshold(threshold)
	
	// Should not add threshold when alerting is disabled
	thresholds := monitor.GetThresholds()
	assert.Len(t, thresholds, 0, "Should not add thresholds when alerting is disabled")
}

// TestErrorMonitorMetrics tests error metrics calculation
func TestErrorMonitorMetrics(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Generate different types of errors
	errorTypes := []struct {
		category ErrorCategory
		count    int
	}{
		{ValidationError, 10},
		{AuthenticationError, 5},
		{DatabaseError, 3},
		{SystemError, 7},
		{BusinessLogicError, 2},
	}
	
	for _, et := range errorTypes {
		errorCtx := ErrorContext{
			Category:  et.category,
			Component: "test_service",
			Operation: "test_operation",
			Severity:  MediumSeverity,
		}
		
		for i := 0; i < et.count; i++ {
			monitor.TrackError(errorCtx)
		}
	}
	
	// Verify statistics
	stats := monitor.GetErrorStats()
	for _, et := range errorTypes {
		assert.Equal(t, int64(et.count), stats[et.category], "Should track correct count for %s", et.category)
	}
	
	// Verify top errors
	topErrors := monitor.GetTopErrors(3)
	assert.Len(t, topErrors, 3, "Should return top 3 errors")
	
	// Should be sorted by count (descending)
	assert.True(t, topErrors[0].Count >= topErrors[1].Count)
	assert.True(t, topErrors[1].Count >= topErrors[2].Count)
	
	// Top error should be ValidationError (10 occurrences)
	assert.Equal(t, int64(10), topErrors[0].Count)
	assert.Equal(t, ValidationError, topErrors[0].Category)
	
	// Verify total error count
	totalErrors := int64(0)
	for _, count := range stats {
		totalErrors += count
	}
	
	expectedTotal := int64(10 + 5 + 3 + 7 + 2)
	assert.Equal(t, expectedTotal, totalErrors, "Should track correct total error count")
}

// TestErrorMonitorComplexScenarios tests complex real-world scenarios
func TestErrorMonitorComplexScenarios(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Set up multiple thresholds for different scenarios
	thresholds := []AlertThreshold{
		{
			Category:      ValidationError,
			Component:     "user_service",
			MaxCount:      5,
			TimeWindow:    1 * time.Minute,
			Severity:      LowSeverity,
			AlertInterval: 100 * time.Millisecond,
		},
		{
			Category:      DatabaseError,
			MaxCount:      2, // Any component
			TimeWindow:    1 * time.Minute,
			Severity:      MediumSeverity,
			AlertInterval: 100 * time.Millisecond,
		},
		{
			Category:      SystemError,
			MaxCount:      1, // Very sensitive
			TimeWindow:    1 * time.Minute,
			Severity:      HighSeverity,
			AlertInterval: 50 * time.Millisecond,
		},
	}
	
	for _, threshold := range thresholds {
		monitor.AddThreshold(threshold)
	}
	
	// Clear buffer
	buf.Reset()
	
	// Scenario 1: Gradual increase in validation errors
	validationCtx := ErrorContext{
		Category:  ValidationError,
		Component: "user_service",
		Operation: "create_user",
		Severity:  LowSeverity,
	}
	
	for i := 0; i < 6; i++ { // Exceed threshold of 5
		monitor.TrackError(validationCtx)
		time.Sleep(10 * time.Millisecond)
	}
	
	// Scenario 2: Database errors from different components
	dbCtx1 := ErrorContext{
		Category:  DatabaseError,
		Component: "user_repository",
		Operation: "save_user",
		Severity:  HighSeverity,
	}
	
	dbCtx2 := ErrorContext{
		Category:  DatabaseError,
		Component: "account_repository",
		Operation: "save_account",
		Severity:  HighSeverity,
	}
	
	monitor.TrackError(dbCtx1)
	monitor.TrackError(dbCtx2)
	monitor.TrackError(dbCtx1) // Should trigger alert
	
	// Scenario 3: Critical system error
	systemCtx := ErrorContext{
		Category:  SystemError,
		Component: "core_service",
		Operation: "initialize",
		Severity:  CriticalSeverity,
	}
	
	monitor.TrackError(systemCtx) // Should trigger immediate alert
	
	// Give time for alert processing
	time.Sleep(200 * time.Millisecond)
	
	// Verify alerts were triggered
	logOutput := buf.String()
	
	// Should have alerts for all three scenarios
	alertCount := strings.Count(logOutput, "error_threshold_exceeded")
	assert.GreaterOrEqual(t, alertCount, 3, "Should have alerts for all scenarios")
	
	// Verify specific alert content
	assert.Contains(t, logOutput, "validation_error", "Should alert on validation errors")
	assert.Contains(t, logOutput, "database_error", "Should alert on database errors")
	assert.Contains(t, logOutput, "system_error", "Should alert on system errors")
	
	// Verify error statistics
	stats := monitor.GetErrorStats()
	assert.Equal(t, int64(6), stats[ValidationError])
	assert.Equal(t, int64(3), stats[DatabaseError])
	assert.Equal(t, int64(1), stats[SystemError])
	
	// Verify error frequencies show different patterns
	frequencies := monitor.GetErrorFrequencies()
	assert.GreaterOrEqual(t, len(frequencies), 3, "Should track different error patterns")
	
	// Find specific patterns
	var userServicePattern, dbPattern, systemPattern *ErrorFrequency
	for _, freq := range frequencies {
		if freq.Component == "user_service" && freq.Category == ValidationError {
			userServicePattern = freq
		} else if freq.Category == DatabaseError {
			if dbPattern == nil || freq.Count > dbPattern.Count {
				dbPattern = freq // Get the most frequent DB error pattern
			}
		} else if freq.Category == SystemError {
			systemPattern = freq
		}
	}
	
	assert.NotNil(t, userServicePattern, "Should track user service validation pattern")
	assert.NotNil(t, dbPattern, "Should track database error pattern")
	assert.NotNil(t, systemPattern, "Should track system error pattern")
	
	assert.Equal(t, int64(6), userServicePattern.Count)
	assert.GreaterOrEqual(t, dbPattern.Count, int64(2))
	assert.Equal(t, int64(1), systemPattern.Count)
}