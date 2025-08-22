package logging

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
)

// AuditLogger provides specialized logging for security and audit events
type AuditLogger struct {
	logger zerolog.Logger
}

// NewAuditLogger creates a new audit logger instance
func NewAuditLogger(logger zerolog.Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger.With().Str("log_type", "audit").Logger(),
	}
}

// LogAuthentication logs authentication events with user context
func (al *AuditLogger) LogAuthentication(userID int64, email, action, result string) {
	al.logger.Info().
		Str("event_type", "authentication").
		Int64("user_id", userID).
		Str("user_email", email).
		Str("action", action).
		Str("result", result).
		Time("timestamp", time.Now()).
		Msg("Authentication event")
}

// LogAuthenticationWithIP logs authentication events with IP address
func (al *AuditLogger) LogAuthenticationWithIP(userID int64, email, action, result, clientIP string) {
	al.logger.Info().
		Str("event_type", "authentication").
		Int64("user_id", userID).
		Str("user_email", email).
		Str("action", action).
		Str("result", result).
		Str("client_ip", clientIP).
		Time("timestamp", time.Now()).
		Msg("Authentication event")
}

// LogAccountOperation logs account-related operations
func (al *AuditLogger) LogAccountOperation(userID int64, accountID int64, operation, result string) {
	al.logger.Info().
		Str("event_type", "account_operation").
		Int64("user_id", userID).
		Int64("account_id", accountID).
		Str("operation", operation).
		Str("result", result).
		Time("timestamp", time.Now()).
		Msg("Account operation")
}

// LogAccountOperationWithDetails logs account operations with additional details
func (al *AuditLogger) LogAccountOperationWithDetails(userID int64, accountID int64, operation, result, currency string, balance decimal.Decimal) {
	al.logger.Info().
		Str("event_type", "account_operation").
		Int64("user_id", userID).
		Int64("account_id", accountID).
		Str("operation", operation).
		Str("result", result).
		Str("currency", currency).
		Str("balance", balance.StringFixed(2)).
		Time("timestamp", time.Now()).
		Msg("Account operation")
}

// LogTransfer logs money transfer transactions for audit compliance
func (al *AuditLogger) LogTransfer(fromAccountID, toAccountID int64, amount decimal.Decimal, result string) {
	al.logger.Info().
		Str("event_type", "transfer").
		Int64("from_account_id", fromAccountID).
		Int64("to_account_id", toAccountID).
		Str("amount", amount.StringFixed(2)).
		Str("result", result).
		Time("timestamp", time.Now()).
		Msg("Money transfer")
}

// LogTransferWithDetails logs transfer with comprehensive details
func (al *AuditLogger) LogTransferWithDetails(transferID int64, fromAccountID, toAccountID int64, amount decimal.Decimal, currency, description, result string, userID int64) {
	al.logger.Info().
		Str("event_type", "transfer").
		Int64("transfer_id", transferID).
		Int64("from_account_id", fromAccountID).
		Int64("to_account_id", toAccountID).
		Str("amount", amount.StringFixed(2)).
		Str("currency", currency).
		Str("description", description).
		Str("result", result).
		Int64("user_id", userID).
		Time("timestamp", time.Now()).
		Msg("Money transfer")
}

// LogAdminAction logs administrative actions with user identification
func (al *AuditLogger) LogAdminAction(adminID int64, action, target, result string) {
	al.logger.Info().
		Str("event_type", "admin_action").
		Int64("admin_id", adminID).
		Str("action", action).
		Str("target", target).
		Str("result", result).
		Time("timestamp", time.Now()).
		Msg("Administrative action")
}

// LogAdminActionWithDetails logs admin actions with additional context
func (al *AuditLogger) LogAdminActionWithDetails(adminID int64, adminEmail, action, target, result, details string) {
	al.logger.Info().
		Str("event_type", "admin_action").
		Int64("admin_id", adminID).
		Str("admin_email", adminEmail).
		Str("action", action).
		Str("target", target).
		Str("result", result).
		Str("details", details).
		Time("timestamp", time.Now()).
		Msg("Administrative action")
}

// LogSecurityEvent logs security events and failures
func (al *AuditLogger) LogSecurityEvent(event, source, details string) {
	al.logger.Warn().
		Str("event_type", "security_event").
		Str("event", event).
		Str("source", source).
		Str("details", details).
		Time("timestamp", time.Now()).
		Msg("Security event")
}

// LogSecurityEventWithIP logs security events with IP address
func (al *AuditLogger) LogSecurityEventWithIP(event, source, details, clientIP string) {
	al.logger.Warn().
		Str("event_type", "security_event").
		Str("event", event).
		Str("source", source).
		Str("details", details).
		Str("client_ip", clientIP).
		Time("timestamp", time.Now()).
		Msg("Security event")
}

// LogFailedAuthentication logs failed authentication attempts
func (al *AuditLogger) LogFailedAuthentication(email, reason, clientIP string) {
	al.logger.Warn().
		Str("event_type", "failed_authentication").
		Str("user_email", email).
		Str("reason", reason).
		Str("client_ip", clientIP).
		Time("timestamp", time.Now()).
		Msg("Failed authentication attempt")
}

// LogRateLimitExceeded logs rate limiting events
func (al *AuditLogger) LogRateLimitExceeded(clientIP, endpoint string, requestCount int) {
	al.logger.Warn().
		Str("event_type", "rate_limit_exceeded").
		Str("client_ip", clientIP).
		Str("endpoint", endpoint).
		Int("request_count", requestCount).
		Time("timestamp", time.Now()).
		Msg("Rate limit exceeded")
}

// LogUserRegistration logs user registration events
func (al *AuditLogger) LogUserRegistration(userID int64, email, result string) {
	al.logger.Info().
		Str("event_type", "user_registration").
		Int64("user_id", userID).
		Str("user_email", email).
		Str("result", result).
		Time("timestamp", time.Now()).
		Msg("User registration")
}

// LogPasswordChange logs password change events
func (al *AuditLogger) LogPasswordChange(userID int64, email, result string) {
	al.logger.Info().
		Str("event_type", "password_change").
		Int64("user_id", userID).
		Str("user_email", email).
		Str("result", result).
		Time("timestamp", time.Now()).
		Msg("Password change")
}

// LogAccountCreation logs account creation events
func (al *AuditLogger) LogAccountCreation(userID int64, accountID int64, currency, result string) {
	al.logger.Info().
		Str("event_type", "account_creation").
		Int64("user_id", userID).
		Int64("account_id", accountID).
		Str("currency", currency).
		Str("result", result).
		Time("timestamp", time.Now()).
		Msg("Account creation")
}

// LogAccountDeletion logs account deletion events
func (al *AuditLogger) LogAccountDeletion(userID int64, accountID int64, result string) {
	al.logger.Info().
		Str("event_type", "account_deletion").
		Int64("user_id", userID).
		Int64("account_id", accountID).
		Str("result", result).
		Time("timestamp", time.Now()).
		Msg("Account deletion")
}

// LogSuspiciousActivity logs suspicious activity detection
func (al *AuditLogger) LogSuspiciousActivity(userID int64, activityType, description, severity string) {
	al.logger.Warn().
		Str("event_type", "suspicious_activity").
		Int64("user_id", userID).
		Str("activity_type", activityType).
		Str("description", description).
		Str("severity", severity).
		Time("timestamp", time.Now()).
		Msg("Suspicious activity detected")
}

// LogDataAccess logs sensitive data access events
func (al *AuditLogger) LogDataAccess(userID int64, dataType, operation, result string) {
	al.logger.Info().
		Str("event_type", "data_access").
		Int64("user_id", userID).
		Str("data_type", dataType).
		Str("operation", operation).
		Str("result", result).
		Time("timestamp", time.Now()).
		Msg("Data access")
}

// LogComplianceEvent logs compliance-related events
func (al *AuditLogger) LogComplianceEvent(eventType, description, complianceRule string, userID int64) {
	al.logger.Info().
		Str("event_type", "compliance").
		Str("compliance_event_type", eventType).
		Str("description", description).
		Str("compliance_rule", complianceRule).
		Int64("user_id", userID).
		Time("timestamp", time.Now()).
		Msg("Compliance event")
}

// WithRequestID returns a new AuditLogger with request ID context
func (al *AuditLogger) WithRequestID(requestID string) *AuditLogger {
	return &AuditLogger{
		logger: al.logger.With().Str("request_id", requestID).Logger(),
	}
}

// WithUserContext returns a new AuditLogger with user context
func (al *AuditLogger) WithUserContext(userID int64, userEmail string) *AuditLogger {
	return &AuditLogger{
		logger: al.logger.With().
			Int64("user_id", userID).
			Str("user_email", userEmail).
			Logger(),
	}
}

// WithCorrelationID returns a new AuditLogger with correlation ID
func (al *AuditLogger) WithCorrelationID(correlationID string) *AuditLogger {
	return &AuditLogger{
		logger: al.logger.With().Str("correlation_id", correlationID).Logger(),
	}
}

// GetLogger returns the underlying zerolog logger
func (al *AuditLogger) GetLogger() zerolog.Logger {
	return al.logger
}