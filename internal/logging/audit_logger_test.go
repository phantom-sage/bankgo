package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	
	auditLogger := NewAuditLogger(logger)
	
	assert.NotNil(t, auditLogger)
	assert.NotNil(t, auditLogger.logger)
}

func TestAuditLogger_LogAuthentication(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	userID := int64(123)
	email := "test@example.com"
	action := "login"
	result := "success"
	
	auditLogger.LogAuthentication(userID, email, action, result)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "authentication", logEntry["event_type"])
	assert.Equal(t, float64(userID), logEntry["user_id"])
	assert.Equal(t, email, logEntry["user_email"])
	assert.Equal(t, action, logEntry["action"])
	assert.Equal(t, result, logEntry["result"])
	assert.Equal(t, "Authentication event", logEntry["message"])
	assert.Contains(t, logEntry, "timestamp")
}

func TestAuditLogger_LogAuthenticationWithIP(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	userID := int64(123)
	email := "test@example.com"
	action := "login"
	result := "success"
	clientIP := "192.168.1.100"
	
	auditLogger.LogAuthenticationWithIP(userID, email, action, result, clientIP)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "authentication", logEntry["event_type"])
	assert.Equal(t, float64(userID), logEntry["user_id"])
	assert.Equal(t, email, logEntry["user_email"])
	assert.Equal(t, action, logEntry["action"])
	assert.Equal(t, result, logEntry["result"])
	assert.Equal(t, clientIP, logEntry["client_ip"])
	assert.Equal(t, "Authentication event", logEntry["message"])
}

func TestAuditLogger_LogAccountOperation(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	userID := int64(123)
	accountID := int64(456)
	operation := "create"
	result := "success"
	
	auditLogger.LogAccountOperation(userID, accountID, operation, result)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "account_operation", logEntry["event_type"])
	assert.Equal(t, float64(userID), logEntry["user_id"])
	assert.Equal(t, float64(accountID), logEntry["account_id"])
	assert.Equal(t, operation, logEntry["operation"])
	assert.Equal(t, result, logEntry["result"])
	assert.Equal(t, "Account operation", logEntry["message"])
}

func TestAuditLogger_LogAccountOperationWithDetails(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	userID := int64(123)
	accountID := int64(456)
	operation := "create"
	result := "success"
	currency := "USD"
	balance := decimal.NewFromFloat(1000.50)
	
	auditLogger.LogAccountOperationWithDetails(userID, accountID, operation, result, currency, balance)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "account_operation", logEntry["event_type"])
	assert.Equal(t, float64(userID), logEntry["user_id"])
	assert.Equal(t, float64(accountID), logEntry["account_id"])
	assert.Equal(t, operation, logEntry["operation"])
	assert.Equal(t, result, logEntry["result"])
	assert.Equal(t, currency, logEntry["currency"])
	assert.Equal(t, "1000.50", logEntry["balance"])
}

func TestAuditLogger_LogTransfer(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	fromAccountID := int64(123)
	toAccountID := int64(456)
	amount := decimal.NewFromFloat(250.75)
	result := "success"
	
	auditLogger.LogTransfer(fromAccountID, toAccountID, amount, result)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "transfer", logEntry["event_type"])
	assert.Equal(t, float64(fromAccountID), logEntry["from_account_id"])
	assert.Equal(t, float64(toAccountID), logEntry["to_account_id"])
	assert.Equal(t, "250.75", logEntry["amount"])
	assert.Equal(t, result, logEntry["result"])
	assert.Equal(t, "Money transfer", logEntry["message"])
}

func TestAuditLogger_LogTransferWithDetails(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	transferID := int64(789)
	fromAccountID := int64(123)
	toAccountID := int64(456)
	amount := decimal.NewFromFloat(250.75)
	currency := "USD"
	description := "Payment for services"
	result := "success"
	userID := int64(999)
	
	auditLogger.LogTransferWithDetails(transferID, fromAccountID, toAccountID, amount, currency, description, result, userID)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "transfer", logEntry["event_type"])
	assert.Equal(t, float64(transferID), logEntry["transfer_id"])
	assert.Equal(t, float64(fromAccountID), logEntry["from_account_id"])
	assert.Equal(t, float64(toAccountID), logEntry["to_account_id"])
	assert.Equal(t, "250.75", logEntry["amount"])
	assert.Equal(t, currency, logEntry["currency"])
	assert.Equal(t, description, logEntry["description"])
	assert.Equal(t, result, logEntry["result"])
	assert.Equal(t, float64(userID), logEntry["user_id"])
}

func TestAuditLogger_LogAdminAction(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	adminID := int64(1)
	action := "delete_user"
	target := "user_123"
	result := "success"
	
	auditLogger.LogAdminAction(adminID, action, target, result)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "admin_action", logEntry["event_type"])
	assert.Equal(t, float64(adminID), logEntry["admin_id"])
	assert.Equal(t, action, logEntry["action"])
	assert.Equal(t, target, logEntry["target"])
	assert.Equal(t, result, logEntry["result"])
	assert.Equal(t, "Administrative action", logEntry["message"])
}

func TestAuditLogger_LogAdminActionWithDetails(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	adminID := int64(1)
	adminEmail := "admin@example.com"
	action := "delete_user"
	target := "user_123"
	result := "success"
	details := "User violated terms of service"
	
	auditLogger.LogAdminActionWithDetails(adminID, adminEmail, action, target, result, details)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "admin_action", logEntry["event_type"])
	assert.Equal(t, float64(adminID), logEntry["admin_id"])
	assert.Equal(t, adminEmail, logEntry["admin_email"])
	assert.Equal(t, action, logEntry["action"])
	assert.Equal(t, target, logEntry["target"])
	assert.Equal(t, result, logEntry["result"])
	assert.Equal(t, details, logEntry["details"])
}

func TestAuditLogger_LogSecurityEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	event := "brute_force_attempt"
	source := "login_endpoint"
	details := "Multiple failed login attempts detected"
	
	auditLogger.LogSecurityEvent(event, source, details)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "security_event", logEntry["event_type"])
	assert.Equal(t, event, logEntry["event"])
	assert.Equal(t, source, logEntry["source"])
	assert.Equal(t, details, logEntry["details"])
	assert.Equal(t, "Security event", logEntry["message"])
	assert.Equal(t, "warn", logEntry["level"])
}

func TestAuditLogger_LogSecurityEventWithIP(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	event := "brute_force_attempt"
	source := "login_endpoint"
	details := "Multiple failed login attempts detected"
	clientIP := "192.168.1.100"
	
	auditLogger.LogSecurityEventWithIP(event, source, details, clientIP)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "security_event", logEntry["event_type"])
	assert.Equal(t, event, logEntry["event"])
	assert.Equal(t, source, logEntry["source"])
	assert.Equal(t, details, logEntry["details"])
	assert.Equal(t, clientIP, logEntry["client_ip"])
	assert.Equal(t, "warn", logEntry["level"])
}

func TestAuditLogger_LogFailedAuthentication(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	email := "test@example.com"
	reason := "invalid_password"
	clientIP := "192.168.1.100"
	
	auditLogger.LogFailedAuthentication(email, reason, clientIP)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "failed_authentication", logEntry["event_type"])
	assert.Equal(t, email, logEntry["user_email"])
	assert.Equal(t, reason, logEntry["reason"])
	assert.Equal(t, clientIP, logEntry["client_ip"])
	assert.Equal(t, "warn", logEntry["level"])
}

func TestAuditLogger_LogRateLimitExceeded(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	clientIP := "192.168.1.100"
	endpoint := "/api/v1/login"
	requestCount := 10
	
	auditLogger.LogRateLimitExceeded(clientIP, endpoint, requestCount)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "rate_limit_exceeded", logEntry["event_type"])
	assert.Equal(t, clientIP, logEntry["client_ip"])
	assert.Equal(t, endpoint, logEntry["endpoint"])
	assert.Equal(t, float64(requestCount), logEntry["request_count"])
	assert.Equal(t, "warn", logEntry["level"])
}

func TestAuditLogger_LogUserRegistration(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	userID := int64(123)
	email := "newuser@example.com"
	result := "success"
	
	auditLogger.LogUserRegistration(userID, email, result)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "user_registration", logEntry["event_type"])
	assert.Equal(t, float64(userID), logEntry["user_id"])
	assert.Equal(t, email, logEntry["user_email"])
	assert.Equal(t, result, logEntry["result"])
}

func TestAuditLogger_LogPasswordChange(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	userID := int64(123)
	email := "user@example.com"
	result := "success"
	
	auditLogger.LogPasswordChange(userID, email, result)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "password_change", logEntry["event_type"])
	assert.Equal(t, float64(userID), logEntry["user_id"])
	assert.Equal(t, email, logEntry["user_email"])
	assert.Equal(t, result, logEntry["result"])
}

func TestAuditLogger_LogSuspiciousActivity(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	userID := int64(123)
	activityType := "unusual_transfer_pattern"
	description := "Multiple large transfers in short time"
	severity := "high"
	
	auditLogger.LogSuspiciousActivity(userID, activityType, description, severity)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "suspicious_activity", logEntry["event_type"])
	assert.Equal(t, float64(userID), logEntry["user_id"])
	assert.Equal(t, activityType, logEntry["activity_type"])
	assert.Equal(t, description, logEntry["description"])
	assert.Equal(t, severity, logEntry["severity"])
	assert.Equal(t, "warn", logEntry["level"])
}

func TestAuditLogger_LogComplianceEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	eventType := "kyc_verification"
	description := "Customer identity verification completed"
	complianceRule := "KYC_RULE_001"
	userID := int64(123)
	
	auditLogger.LogComplianceEvent(eventType, description, complianceRule, userID)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "compliance", logEntry["event_type"])
	assert.Equal(t, eventType, logEntry["compliance_event_type"])
	assert.Equal(t, description, logEntry["description"])
	assert.Equal(t, complianceRule, logEntry["compliance_rule"])
	assert.Equal(t, float64(userID), logEntry["user_id"])
}

func TestAuditLogger_WithRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	requestID := "req-123-456"
	auditLoggerWithReqID := auditLogger.WithRequestID(requestID)
	
	auditLoggerWithReqID.LogAuthentication(123, "test@example.com", "login", "success")
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, requestID, logEntry["request_id"])
	assert.Equal(t, "audit", logEntry["log_type"])
}

func TestAuditLogger_WithUserContext(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	userID := int64(123)
	userEmail := "test@example.com"
	auditLoggerWithUser := auditLogger.WithUserContext(userID, userEmail)
	
	auditLoggerWithUser.LogSecurityEvent("test_event", "test_source", "test_details")
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, float64(userID), logEntry["user_id"])
	assert.Equal(t, userEmail, logEntry["user_email"])
	assert.Equal(t, "audit", logEntry["log_type"])
}

func TestAuditLogger_WithCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	correlationID := "corr-123-456"
	auditLoggerWithCorr := auditLogger.WithCorrelationID(correlationID)
	
	auditLoggerWithCorr.LogAuthentication(123, "test@example.com", "login", "success")
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, correlationID, logEntry["correlation_id"])
	assert.Equal(t, "audit", logEntry["log_type"])
}

func TestAuditLogger_GetLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	underlyingLogger := auditLogger.GetLogger()
	assert.NotNil(t, underlyingLogger)
	
	// Test that the underlying logger has the audit log_type
	underlyingLogger.Info().Msg("test message")
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "audit", logEntry["log_type"])
	assert.Equal(t, "test message", logEntry["message"])
}

func TestAuditLogger_TimestampFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	auditLogger.LogAuthentication(123, "test@example.com", "login", "success")
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	
	timestampStr, ok := logEntry["timestamp"].(string)
	require.True(t, ok, "timestamp should be a string")
	
	// Parse the timestamp to ensure it's in the correct format
	_, err = time.Parse(time.RFC3339Nano, timestampStr)
	assert.NoError(t, err, "timestamp should be in RFC3339Nano format")
}

func TestAuditLogger_LogLevels(t *testing.T) {
	tests := []struct {
		name          string
		logFunc       func(*AuditLogger)
		expectedLevel string
	}{
		{
			name: "Authentication uses info level",
			logFunc: func(al *AuditLogger) {
				al.LogAuthentication(123, "test@example.com", "login", "success")
			},
			expectedLevel: "info",
		},
		{
			name: "Security event uses warn level",
			logFunc: func(al *AuditLogger) {
				al.LogSecurityEvent("test_event", "test_source", "test_details")
			},
			expectedLevel: "warn",
		},
		{
			name: "Failed authentication uses warn level",
			logFunc: func(al *AuditLogger) {
				al.LogFailedAuthentication("test@example.com", "invalid_password", "192.168.1.1")
			},
			expectedLevel: "warn",
		},
		{
			name: "Suspicious activity uses warn level",
			logFunc: func(al *AuditLogger) {
				al.LogSuspiciousActivity(123, "test_activity", "test_description", "high")
			},
			expectedLevel: "warn",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf)
			auditLogger := NewAuditLogger(logger)
			
			tt.logFunc(auditLogger)
			
			var logEntry map[string]interface{}
			err := json.Unmarshal(buf.Bytes(), &logEntry)
			require.NoError(t, err)
			
			assert.Equal(t, tt.expectedLevel, logEntry["level"])
		})
	}
}

func TestAuditLogger_DecimalFormatting(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	auditLogger := NewAuditLogger(logger)
	
	// Test with various decimal amounts
	testCases := []struct {
		amount   decimal.Decimal
		expected string
	}{
		{decimal.NewFromFloat(100.00), "100.00"},
		{decimal.NewFromFloat(100.5), "100.50"},
		{decimal.NewFromFloat(100.123), "100.12"}, // Should round to 2 decimal places
		{decimal.NewFromFloat(0.01), "0.01"},
		{decimal.NewFromFloat(1000000.99), "1000000.99"},
	}
	
	for _, tc := range testCases {
		buf.Reset()
		auditLogger.LogTransfer(123, 456, tc.amount, "success")
		
		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)
		
		assert.Equal(t, tc.expected, logEntry["amount"])
	}
}

func TestAuditLogger_ConcurrentLogging(t *testing.T) {
	// Use a synchronized buffer to handle concurrent writes
	var buf bytes.Buffer
	var mu sync.Mutex
	
	syncWriter := &syncWriter{buf: &buf, mu: &mu}
	logger := zerolog.New(syncWriter)
	auditLogger := NewAuditLogger(logger)
	
	// Test concurrent logging to ensure thread safety
	const numGoroutines = 10
	const logsPerGoroutine = 10
	
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < logsPerGoroutine; j++ {
				auditLogger.LogAuthentication(int64(id), "test@example.com", "login", "success")
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
		assert.Equal(t, "audit", logEntry["log_type"])
	}
}

// syncWriter is a thread-safe writer for testing concurrent logging
type syncWriter struct {
	buf *bytes.Buffer
	mu  *sync.Mutex
}

func (sw *syncWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.buf.Write(p)
}