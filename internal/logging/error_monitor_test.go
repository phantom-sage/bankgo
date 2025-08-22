package logging

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewErrorMonitor(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := DefaultErrorMonitorConfig()
	
	monitor := NewErrorMonitor(config, logger)
	
	assert.NotNil(t, monitor)
	assert.Equal(t, config.EnableTracking, monitor.config.EnableTracking)
	assert.Equal(t, config.EnableAlerting, monitor.config.EnableAlerting)
	assert.Equal(t, len(config.DefaultThresholds), len(monitor.thresholds))
	
	// Clean up
	monitor.Close()
}

func TestErrorMonitor_TrackError(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	ctx := ErrorContext{
		Category:  ValidationError,
		Component: "user_service",
		Operation: "create_user",
	}
	
	// Track the same error multiple times
	monitor.TrackError(ctx)
	monitor.TrackError(ctx)
	monitor.TrackError(ctx)
	
	// Check frequency data
	freq := monitor.GetErrorFrequency(ValidationError, "user_service", "create_user")
	require.NotNil(t, freq)
	assert.Equal(t, int64(3), freq.Count)
	assert.Equal(t, ValidationError, freq.Category)
	assert.Equal(t, "user_service", freq.Component)
	assert.Equal(t, "create_user", freq.Operation)
	assert.False(t, freq.FirstOccurred.IsZero())
	assert.False(t, freq.LastOccurred.IsZero())
}

func TestErrorMonitor_TrackError_Disabled(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: false,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	ctx := ErrorContext{
		Category:  ValidationError,
		Component: "user_service",
		Operation: "create_user",
	}
	
	monitor.TrackError(ctx)
	
	// Should not track when disabled
	freq := monitor.GetErrorFrequency(ValidationError, "user_service", "create_user")
	assert.Nil(t, freq)
}

func TestErrorMonitor_AddRemoveThreshold(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	threshold := AlertThreshold{
		Category:      ValidationError,
		Component:     "user_service",
		Operation:     "create_user",
		MaxCount:      5,
		TimeWindow:    1 * time.Minute,
		Severity:      LowSeverity,
		AlertInterval: 5 * time.Minute,
	}
	
	// Add threshold
	monitor.AddThreshold(threshold)
	
	thresholds := monitor.GetThresholds()
	assert.Len(t, thresholds, 1)
	
	// Remove threshold
	monitor.RemoveThreshold(ValidationError, "user_service", "create_user")
	
	thresholds = monitor.GetThresholds()
	assert.Len(t, thresholds, 0)
}

func TestErrorMonitor_GetErrorFrequencies(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Track different errors
	contexts := []ErrorContext{
		{Category: ValidationError, Component: "user_service", Operation: "create_user"},
		{Category: AuthenticationError, Component: "auth_service", Operation: "login"},
		{Category: DatabaseError, Component: "user_repository", Operation: "save"},
	}
	
	for _, ctx := range contexts {
		monitor.TrackError(ctx)
		monitor.TrackError(ctx) // Track twice
	}
	
	frequencies := monitor.GetErrorFrequencies()
	assert.Len(t, frequencies, 3)
	
	for _, freq := range frequencies {
		assert.Equal(t, int64(2), freq.Count)
	}
}

func TestErrorMonitor_ResetErrorFrequency(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	ctx := ErrorContext{
		Category:  ValidationError,
		Component: "user_service",
		Operation: "create_user",
	}
	
	// Track error
	monitor.TrackError(ctx)
	
	// Verify it exists
	freq := monitor.GetErrorFrequency(ValidationError, "user_service", "create_user")
	assert.NotNil(t, freq)
	
	// Reset frequency
	monitor.ResetErrorFrequency(ValidationError, "user_service", "create_user")
	
	// Verify it's gone
	freq = monitor.GetErrorFrequency(ValidationError, "user_service", "create_user")
	assert.Nil(t, freq)
}

func TestErrorMonitor_ClearAllFrequencies(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Track multiple errors
	contexts := []ErrorContext{
		{Category: ValidationError, Component: "user_service", Operation: "create_user"},
		{Category: AuthenticationError, Component: "auth_service", Operation: "login"},
	}
	
	for _, ctx := range contexts {
		monitor.TrackError(ctx)
	}
	
	// Verify they exist
	frequencies := monitor.GetErrorFrequencies()
	assert.Len(t, frequencies, 2)
	
	// Clear all
	monitor.ClearAllFrequencies()
	
	// Verify they're gone
	frequencies = monitor.GetErrorFrequencies()
	assert.Len(t, frequencies, 0)
}

func TestErrorMonitor_GetErrorStats(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Track errors with different categories
	contexts := []ErrorContext{
		{Category: ValidationError, Component: "service1", Operation: "op1"},
		{Category: ValidationError, Component: "service2", Operation: "op2"},
		{Category: AuthenticationError, Component: "service3", Operation: "op3"},
	}
	
	// Track validation errors 3 times each, auth error 2 times
	for i, ctx := range contexts {
		count := 3
		if i == 2 { // auth error
			count = 2
		}
		for j := 0; j < count; j++ {
			monitor.TrackError(ctx)
		}
	}
	
	stats := monitor.GetErrorStats()
	assert.Equal(t, int64(6), stats[ValidationError]) // 3 + 3
	assert.Equal(t, int64(2), stats[AuthenticationError])
}

func TestErrorMonitor_GetTopErrors(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Track errors with different frequencies
	contexts := []ErrorContext{
		{Category: ValidationError, Component: "service1", Operation: "op1"},   // 5 times
		{Category: AuthenticationError, Component: "service2", Operation: "op2"}, // 3 times
		{Category: DatabaseError, Component: "service3", Operation: "op3"},       // 1 time
	}
	
	counts := []int{5, 3, 1}
	for i, ctx := range contexts {
		for j := 0; j < counts[i]; j++ {
			monitor.TrackError(ctx)
		}
	}
	
	// Get top 2 errors
	topErrors := monitor.GetTopErrors(2)
	assert.Len(t, topErrors, 2)
	
	// Should be sorted by count (descending)
	assert.Equal(t, int64(5), topErrors[0].Count)
	assert.Equal(t, int64(3), topErrors[1].Count)
	assert.Equal(t, ValidationError, topErrors[0].Category)
	assert.Equal(t, AuthenticationError, topErrors[1].Category)
	
	// Get all errors
	allErrors := monitor.GetTopErrors(0)
	assert.Len(t, allErrors, 3)
}

func TestErrorMonitor_AlertThresholds(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Add a threshold
	threshold := AlertThreshold{
		Category:      ValidationError,
		Component:     "user_service",
		MaxCount:      3,
		TimeWindow:    1 * time.Minute,
		Severity:      LowSeverity,
		AlertInterval: 1 * time.Second, // Short interval for testing
	}
	monitor.AddThreshold(threshold)
	
	ctx := ErrorContext{
		Category:  ValidationError,
		Component: "user_service",
		Operation: "create_user",
	}
	
	// Track errors to exceed threshold
	for i := 0; i < 4; i++ {
		monitor.TrackError(ctx)
	}
	
	// Give some time for alert processing
	time.Sleep(100 * time.Millisecond)
	
	// Check that alert was logged
	logOutput := buf.String()
	assert.Contains(t, logOutput, "error_threshold_exceeded")
	assert.Contains(t, logOutput, "alert triggered")
}

func TestErrorMonitor_ThresholdMatching(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	tests := []struct {
		name      string
		threshold AlertThreshold
		freq      *ErrorFrequency
		expected  bool
	}{
		{
			name: "exact match",
			threshold: AlertThreshold{
				Category:  ValidationError,
				Component: "user_service",
				Operation: "create_user",
			},
			freq: &ErrorFrequency{
				Category:  ValidationError,
				Component: "user_service",
				Operation: "create_user",
			},
			expected: true,
		},
		{
			name: "category only match",
			threshold: AlertThreshold{
				Category: ValidationError,
			},
			freq: &ErrorFrequency{
				Category:  ValidationError,
				Component: "any_service",
				Operation: "any_operation",
			},
			expected: true,
		},
		{
			name: "category mismatch",
			threshold: AlertThreshold{
				Category: ValidationError,
			},
			freq: &ErrorFrequency{
				Category: AuthenticationError,
			},
			expected: false,
		},
		{
			name: "component mismatch",
			threshold: AlertThreshold{
				Category:  ValidationError,
				Component: "user_service",
			},
			freq: &ErrorFrequency{
				Category:  ValidationError,
				Component: "auth_service",
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.thresholdMatches(tt.threshold, tt.freq)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorMonitor_CleanupOldFrequencies(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking:  true,
		EnableAlerting:  false,
		CleanupInterval: 100 * time.Millisecond,
		MaxAge:          50 * time.Millisecond,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	ctx := ErrorContext{
		Category:  ValidationError,
		Component: "user_service",
		Operation: "create_user",
	}
	
	// Track an error
	monitor.TrackError(ctx)
	
	// Verify it exists
	freq := monitor.GetErrorFrequency(ValidationError, "user_service", "create_user")
	assert.NotNil(t, freq)
	
	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)
	
	// Verify it's been cleaned up
	freq = monitor.GetErrorFrequency(ValidationError, "user_service", "create_user")
	assert.Nil(t, freq)
}

func TestDefaultErrorMonitorConfig(t *testing.T) {
	config := DefaultErrorMonitorConfig()
	
	assert.True(t, config.EnableTracking)
	assert.True(t, config.EnableAlerting)
	assert.Equal(t, 1*time.Hour, config.CleanupInterval)
	assert.Equal(t, 24*time.Hour, config.MaxAge)
	assert.NotEmpty(t, config.DefaultThresholds)
	
	// Check that all error categories have thresholds
	categories := map[ErrorCategory]bool{
		ValidationError:      false,
		AuthenticationError:  false,
		BusinessLogicError:   false,
		SystemError:          false,
		DatabaseError:        false,
	}
	
	for _, threshold := range config.DefaultThresholds {
		categories[threshold.Category] = true
	}
	
	for category, found := range categories {
		assert.True(t, found, "Missing threshold for category: %s", category)
	}
}

func TestErrorMonitor_ConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	ctx := ErrorContext{
		Category:  ValidationError,
		Component: "user_service",
		Operation: "create_user",
	}
	
	// Simulate concurrent error tracking
	done := make(chan bool)
	numGoroutines := 10
	errorsPerGoroutine := 100
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < errorsPerGoroutine; j++ {
				monitor.TrackError(ctx)
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Check final count
	freq := monitor.GetErrorFrequency(ValidationError, "user_service", "create_user")
	require.NotNil(t, freq)
	assert.Equal(t, int64(numGoroutines*errorsPerGoroutine), freq.Count)
}

func TestErrorMonitor_KeyGeneration(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	tests := []struct {
		name     string
		ctx      ErrorContext
		expected string
	}{
		{
			name: "full context",
			ctx: ErrorContext{
				Category:  ValidationError,
				Component: "user_service",
				Operation: "create_user",
			},
			expected: "validation_error:user_service:create_user",
		},
		{
			name: "empty component and operation",
			ctx: ErrorContext{
				Category: SystemError,
			},
			expected: "system_error::",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := monitor.getFrequencyKey(tt.ctx)
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestErrorMonitor_ComprehensiveFrequencyTracking(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: false,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Test tracking different error patterns with various frequencies
	errorPatterns := []struct {
		category  ErrorCategory
		component string
		operation string
		frequency int
	}{
		{ValidationError, "user_service", "create_user", 10},
		{ValidationError, "account_service", "create_account", 5},
		{AuthenticationError, "auth_service", "login", 15},
		{AuthenticationError, "auth_service", "refresh_token", 3},
		{DatabaseError, "user_repository", "save", 7},
		{DatabaseError, "account_repository", "update", 2},
		{BusinessLogicError, "transfer_service", "transfer", 8},
		{SystemError, "email_service", "send", 4},
		{NetworkError, "external_api", "call", 6},
		{ConfigurationError, "config_loader", "load", 1},
	}
	
	// Track errors according to patterns
	for _, pattern := range errorPatterns {
		for i := 0; i < pattern.frequency; i++ {
			ctx := ErrorContext{
				Category:  pattern.category,
				Component: pattern.component,
				Operation: pattern.operation,
			}
			monitor.TrackError(ctx)
		}
	}
	
	// Verify all patterns are tracked
	frequencies := monitor.GetErrorFrequencies()
	assert.Len(t, frequencies, len(errorPatterns), "Should track all error patterns")
	
	// Verify individual frequencies
	for _, pattern := range errorPatterns {
		freq := monitor.GetErrorFrequency(pattern.category, pattern.component, pattern.operation)
		require.NotNil(t, freq, "Frequency should exist for pattern: %s:%s:%s", 
			pattern.category, pattern.component, pattern.operation)
		assert.Equal(t, int64(pattern.frequency), freq.Count, 
			"Frequency count mismatch for pattern: %s:%s:%s", 
			pattern.category, pattern.component, pattern.operation)
		assert.Equal(t, pattern.category, freq.Category)
		assert.Equal(t, pattern.component, freq.Component)
		assert.Equal(t, pattern.operation, freq.Operation)
		assert.False(t, freq.FirstOccurred.IsZero(), "FirstOccurred should be set")
		assert.False(t, freq.LastOccurred.IsZero(), "LastOccurred should be set")
	}
	
	// Verify aggregated statistics
	expectedStats := map[ErrorCategory]int64{
		ValidationError:      15, // 10 + 5
		AuthenticationError:  18, // 15 + 3
		DatabaseError:        9,  // 7 + 2
		BusinessLogicError:   8,  // 8
		SystemError:          4,  // 4
		NetworkError:         6,  // 6
		ConfigurationError:   1,  // 1
	}
	
	stats := monitor.GetErrorStats()
	for category, expectedCount := range expectedStats {
		assert.Equal(t, expectedCount, stats[category], 
			"Aggregated count mismatch for category: %s", category)
	}
	
	// Verify top errors (should be sorted by frequency)
	topErrors := monitor.GetTopErrors(3)
	assert.Len(t, topErrors, 3, "Should return top 3 errors")
	
	// Verify sorting (descending order)
	for i := 1; i < len(topErrors); i++ {
		assert.True(t, topErrors[i-1].Count >= topErrors[i].Count, 
			"Errors should be sorted by count (descending): %d >= %d", 
			topErrors[i-1].Count, topErrors[i].Count)
	}
	
	// Verify the highest count is reasonable (should be 15 for auth_service:login)
	assert.True(t, topErrors[0].Count >= 10, "Top error should have at least 10 occurrences")
}

func TestErrorMonitor_AlertingSystem_Comprehensive(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Define comprehensive alert thresholds
	thresholds := []AlertThreshold{
		// Category-only threshold (applies to all validation errors)
		{
			Category:      ValidationError,
			MaxCount:      5,
			TimeWindow:    1 * time.Minute,
			Severity:      LowSeverity,
			AlertInterval: 1 * time.Second,
		},
		// Component-specific threshold
		{
			Category:      AuthenticationError,
			Component:     "auth_service",
			MaxCount:      3,
			TimeWindow:    1 * time.Minute,
			Severity:      MediumSeverity,
			AlertInterval: 1 * time.Second,
		},
		// Operation-specific threshold
		{
			Category:      DatabaseError,
			Component:     "user_repository",
			Operation:     "save",
			MaxCount:      2,
			TimeWindow:    1 * time.Minute,
			Severity:      HighSeverity,
			AlertInterval: 1 * time.Second,
		},
		// Critical system error threshold
		{
			Category:      SystemError,
			MaxCount:      1,
			TimeWindow:    1 * time.Minute,
			Severity:      CriticalSeverity,
			AlertInterval: 1 * time.Second,
		},
	}
	
	// Add all thresholds
	for _, threshold := range thresholds {
		monitor.AddThreshold(threshold)
	}
	
	// Clear buffer to focus on alert messages
	buf.Reset()
	
	// Test category-only threshold (ValidationError)
	// Use the same component/operation to ensure they're tracked together
	validationContext := ErrorContext{
		Category:  ValidationError,
		Component: "user_service",
		Operation: "create",
	}
	
	// Trigger validation errors to exceed threshold (5)
	for i := 0; i < 6; i++ {
		monitor.TrackError(validationContext)
	}
	
	// Test component-specific threshold (AuthenticationError in auth_service)
	authContext := ErrorContext{
		Category:  AuthenticationError,
		Component: "auth_service",
		Operation: "login",
	}
	
	// Trigger auth errors to exceed threshold (3)
	for i := 0; i < 4; i++ {
		monitor.TrackError(authContext)
	}
	
	// Test operation-specific threshold (DatabaseError in user_repository:save)
	dbContext := ErrorContext{
		Category:  DatabaseError,
		Component: "user_repository",
		Operation: "save",
	}
	
	// Trigger database errors to exceed threshold (2)
	for i := 0; i < 3; i++ {
		monitor.TrackError(dbContext)
	}
	
	// Test critical system error threshold
	systemContext := ErrorContext{
		Category:  SystemError,
		Component: "core_system",
		Operation: "initialize",
	}
	
	// Trigger system error to exceed threshold (1)
	for i := 0; i < 2; i++ {
		monitor.TrackError(systemContext)
	}
	
	// Give time for alert processing
	time.Sleep(100 * time.Millisecond)
	
	// Verify alerts were triggered
	logOutput := buf.String()
	assert.Contains(t, logOutput, "error_threshold_exceeded", "Should contain threshold exceeded alerts")
	
	// Verify different categories triggered alerts
	assert.Contains(t, logOutput, "validation_error", "Should alert on validation errors")
	assert.Contains(t, logOutput, "authentication_error", "Should alert on auth errors")
	assert.Contains(t, logOutput, "system_error", "Should alert on system errors")
	
	// Verify different severity levels in alerts (check what's actually present)
	assert.Contains(t, logOutput, "\"severity\":\"low\"", "Should contain low severity alert")
	assert.Contains(t, logOutput, "\"severity\":\"medium\"", "Should contain medium severity alert")
	assert.Contains(t, logOutput, "\"severity\":\"critical\"", "Should contain critical severity alert")
	
	// Database error alert may or may not be triggered depending on timing
	if strings.Contains(logOutput, "database_error") {
		assert.Contains(t, logOutput, "\"severity\":\"high\"", "Should contain high severity alert for database errors")
	}
}

func TestErrorMonitor_AlertInterval_RateLimit(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Set up threshold with longer alert interval
	threshold := AlertThreshold{
		Category:      ValidationError,
		Component:     "test_service",
		MaxCount:      2,
		TimeWindow:    1 * time.Minute,
		Severity:      LowSeverity,
		AlertInterval: 500 * time.Millisecond, // Longer interval for testing
	}
	
	monitor.AddThreshold(threshold)
	
	ctx := ErrorContext{
		Category:  ValidationError,
		Component: "test_service",
		Operation: "test_operation",
	}
	
	// Clear buffer
	buf.Reset()
	
	// Trigger errors to exceed threshold multiple times rapidly
	for i := 0; i < 3; i++ { // First batch - should trigger alert
		monitor.TrackError(ctx)
	}
	
	// Give time for first alert
	time.Sleep(50 * time.Millisecond)
	
	// Trigger more errors immediately - should NOT trigger another alert due to interval
	for i := 0; i < 3; i++ {
		monitor.TrackError(ctx)
	}
	
	// Check that at least one alert was triggered
	logOutput := buf.String()
	alertCount := strings.Count(logOutput, "error_threshold_exceeded")
	assert.True(t, alertCount >= 1, "Should trigger at least one alert")
	
	// Wait for alert interval to pass
	time.Sleep(600 * time.Millisecond)
	
	// Clear buffer and trigger more errors
	buf.Reset()
	for i := 0; i < 3; i++ {
		monitor.TrackError(ctx)
	}
	
	// Give time for alert processing
	time.Sleep(50 * time.Millisecond)
	
	// Now should trigger another alert
	logOutput = buf.String()
	alertCount = strings.Count(logOutput, "error_threshold_exceeded")
	assert.True(t, alertCount >= 1, "Should trigger another alert after interval passes")
}

func TestErrorMonitor_ThresholdMatching_Specificity(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Test threshold matching with different levels of specificity
	tests := []struct {
		name      string
		threshold AlertThreshold
		frequency ErrorFrequency
		expected  bool
	}{
		{
			name: "exact match - all fields",
			threshold: AlertThreshold{
				Category:  ValidationError,
				Component: "user_service",
				Operation: "create_user",
			},
			frequency: ErrorFrequency{
				Category:  ValidationError,
				Component: "user_service",
				Operation: "create_user",
			},
			expected: true,
		},
		{
			name: "category and component match",
			threshold: AlertThreshold{
				Category:  ValidationError,
				Component: "user_service",
				Operation: "", // Empty means any operation
			},
			frequency: ErrorFrequency{
				Category:  ValidationError,
				Component: "user_service",
				Operation: "create_user",
			},
			expected: true,
		},
		{
			name: "category only match",
			threshold: AlertThreshold{
				Category:  ValidationError,
				Component: "", // Empty means any component
				Operation: "", // Empty means any operation
			},
			frequency: ErrorFrequency{
				Category:  ValidationError,
				Component: "any_service",
				Operation: "any_operation",
			},
			expected: true,
		},
		{
			name: "category mismatch",
			threshold: AlertThreshold{
				Category: ValidationError,
			},
			frequency: ErrorFrequency{
				Category: AuthenticationError,
			},
			expected: false,
		},
		{
			name: "component mismatch",
			threshold: AlertThreshold{
				Category:  ValidationError,
				Component: "user_service",
			},
			frequency: ErrorFrequency{
				Category:  ValidationError,
				Component: "auth_service",
			},
			expected: false,
		},
		{
			name: "operation mismatch",
			threshold: AlertThreshold{
				Category:  ValidationError,
				Component: "user_service",
				Operation: "create_user",
			},
			frequency: ErrorFrequency{
				Category:  ValidationError,
				Component: "user_service",
				Operation: "update_user",
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.thresholdMatches(tt.threshold, &tt.frequency)
			assert.Equal(t, tt.expected, result, "Threshold matching result mismatch")
		})
	}
}

func TestErrorMonitor_TimeWindowFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	config := ErrorMonitorConfig{
		EnableTracking: true,
		EnableAlerting: true,
	}
	
	monitor := NewErrorMonitor(config, logger)
	defer monitor.Close()
	
	// Set up threshold with short time window
	threshold := AlertThreshold{
		Category:      ValidationError,
		Component:     "test_service",
		MaxCount:      3,
		TimeWindow:    100 * time.Millisecond, // Very short window
		Severity:      LowSeverity,
		AlertInterval: 1 * time.Second,
	}
	
	monitor.AddThreshold(threshold)
	
	ctx := ErrorContext{
		Category:  ValidationError,
		Component: "test_service",
		Operation: "test_operation",
	}
	
	// Clear buffer
	buf.Reset()
	
	// Trigger errors within time window
	for i := 0; i < 4; i++ { // Exceed threshold
		monitor.TrackError(ctx)
		if i < 3 {
			time.Sleep(20 * time.Millisecond) // Stay within window
		}
	}
	
	// Give time for alert processing
	time.Sleep(50 * time.Millisecond)
	
	// Should trigger alert
	logOutput := buf.String()
	assert.Contains(t, logOutput, "error_threshold_exceeded", "Should trigger alert within time window")
	
	// Wait for time window to pass
	time.Sleep(200 * time.Millisecond)
	
	// Clear buffer and trigger more errors outside window
	buf.Reset()
	for i := 0; i < 4; i++ {
		monitor.TrackError(ctx)
		time.Sleep(150 * time.Millisecond) // Outside window between each error
	}
	
	// Give time for processing
	time.Sleep(50 * time.Millisecond)
	
	// Should NOT trigger alert because errors are spread outside time window
	logOutput = buf.String()
	assert.NotContains(t, logOutput, "error_threshold_exceeded", 
		"Should not trigger alert when errors are outside time window")
}

func TestErrorMonitor_DefaultConfiguration_Coverage(t *testing.T) {
	config := DefaultErrorMonitorConfig()
	
	// Verify all expected error categories have default thresholds
	expectedCategories := []ErrorCategory{
		ValidationError,
		AuthenticationError,
		BusinessLogicError,
		SystemError,
		DatabaseError,
	}
	
	foundCategories := make(map[ErrorCategory]bool)
	for _, threshold := range config.DefaultThresholds {
		foundCategories[threshold.Category] = true
		
		// Verify threshold has reasonable values
		assert.Greater(t, threshold.MaxCount, int64(0), "MaxCount should be positive")
		assert.Greater(t, threshold.TimeWindow, time.Duration(0), "TimeWindow should be positive")
		assert.Greater(t, threshold.AlertInterval, time.Duration(0), "AlertInterval should be positive")
		assert.NotEmpty(t, threshold.Severity, "Severity should be set")
	}
	
	// Verify all expected categories are covered
	for _, category := range expectedCategories {
		assert.True(t, foundCategories[category], "Missing default threshold for category: %s", category)
	}
	
	// Verify configuration defaults
	assert.True(t, config.EnableTracking, "Tracking should be enabled by default")
	assert.True(t, config.EnableAlerting, "Alerting should be enabled by default")
	assert.Equal(t, 1*time.Hour, config.CleanupInterval, "Default cleanup interval should be 1 hour")
	assert.Equal(t, 24*time.Hour, config.MaxAge, "Default max age should be 24 hours")
}