package services

import (
	"context"
	"fmt"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/rs/zerolog/log"
)

// AlertGeneratorService provides a centralized service for generating alerts
// from various system events and conditions
type AlertGeneratorService struct {
	alertService interfaces.AlertService
}

// NewAlertGeneratorService creates a new alert generator service
func NewAlertGeneratorService(alertService interfaces.AlertService) *AlertGeneratorService {
	return &AlertGeneratorService{
		alertService: alertService,
	}
}

// DatabaseConnectionAlert generates an alert for database connection issues
func (s *AlertGeneratorService) DatabaseConnectionAlert(ctx context.Context, err error, connectionType string) error {
	metadata := map[string]interface{}{
		"error":           err.Error(),
		"connection_type": connectionType,
		"timestamp":       time.Now().Unix(),
		"alert_type":      "database_connection",
	}

	_, alertErr := s.alertService.CreateAlert(
		ctx,
		"critical",
		"Database Connection Failed",
		fmt.Sprintf("Failed to connect to %s database: %s", connectionType, err.Error()),
		"database_monitor",
		metadata,
	)

	if alertErr != nil {
		log.Error().
			Err(alertErr).
			Str("connection_type", connectionType).
			Msg("Failed to create database connection alert")
		return alertErr
	}

	return nil
}

// AuthenticationFailureAlert generates an alert for authentication failures
func (s *AlertGeneratorService) AuthenticationFailureAlert(ctx context.Context, username, ipAddress string, failureCount int) error {
	severity := "warning"
	if failureCount >= 5 {
		severity = "critical"
	}

	metadata := map[string]interface{}{
		"username":      username,
		"ip_address":    ipAddress,
		"failure_count": failureCount,
		"timestamp":     time.Now().Unix(),
		"alert_type":    "authentication_failure",
	}

	title := "Authentication Failure"
	message := fmt.Sprintf("Authentication failed for user %s from IP %s (%d attempts)", username, ipAddress, failureCount)

	if failureCount >= 5 {
		title = "Multiple Authentication Failures"
		message = fmt.Sprintf("Critical: %d authentication failures for user %s from IP %s", failureCount, username, ipAddress)
	}

	_, err := s.alertService.CreateAlert(
		ctx,
		severity,
		title,
		message,
		"auth_monitor",
		metadata,
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("username", username).
			Str("ip_address", ipAddress).
			Msg("Failed to create authentication failure alert")
		return err
	}

	return nil
}

// TransactionAnomalyAlert generates an alert for suspicious transaction patterns
func (s *AlertGeneratorService) TransactionAnomalyAlert(ctx context.Context, userID, accountID string, amount, threshold float64, anomalyType string) error {
	metadata := map[string]interface{}{
		"user_id":       userID,
		"account_id":    accountID,
		"amount":        amount,
		"threshold":     threshold,
		"anomaly_type":  anomalyType,
		"timestamp":     time.Now().Unix(),
		"alert_type":    "transaction_anomaly",
	}

	severity := "warning"
	if amount > threshold*2 {
		severity = "critical"
	}

	_, err := s.alertService.CreateAlert(
		ctx,
		severity,
		"Transaction Anomaly Detected",
		fmt.Sprintf("Suspicious %s transaction detected: $%.2f (threshold: $%.2f) for account %s", anomalyType, amount, threshold, accountID),
		"transaction_monitor",
		metadata,
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("account_id", accountID).
			Msg("Failed to create transaction anomaly alert")
		return err
	}

	return nil
}

// SystemResourceAlert generates an alert for system resource issues
func (s *AlertGeneratorService) SystemResourceAlert(ctx context.Context, resourceType string, currentValue, threshold float64, unit string) error {
	severity := "warning"
	if currentValue > threshold*1.2 {
		severity = "critical"
	}

	metadata := map[string]interface{}{
		"resource_type":  resourceType,
		"current_value":  currentValue,
		"threshold":      threshold,
		"unit":          unit,
		"timestamp":     time.Now().Unix(),
		"alert_type":    "system_resource",
	}

	title := fmt.Sprintf("High %s Usage", resourceType)
	message := fmt.Sprintf("%s usage is %.1f%s, exceeding threshold of %.1f%s", resourceType, currentValue, unit, threshold, unit)

	_, err := s.alertService.CreateAlert(
		ctx,
		severity,
		title,
		message,
		"system_monitor",
		metadata,
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("resource_type", resourceType).
			Float64("current_value", currentValue).
			Msg("Failed to create system resource alert")
		return err
	}

	return nil
}

// ServiceDownAlert generates an alert when a service becomes unavailable
func (s *AlertGeneratorService) ServiceDownAlert(ctx context.Context, serviceName, serviceURL string, err error) error {
	metadata := map[string]interface{}{
		"service_name": serviceName,
		"service_url":  serviceURL,
		"error":        err.Error(),
		"timestamp":    time.Now().Unix(),
		"alert_type":   "service_down",
	}

	_, alertErr := s.alertService.CreateAlert(
		ctx,
		"critical",
		"Service Unavailable",
		fmt.Sprintf("Service %s is unavailable at %s: %s", serviceName, serviceURL, err.Error()),
		"service_monitor",
		metadata,
	)

	if alertErr != nil {
		log.Error().
			Err(alertErr).
			Str("service_name", serviceName).
			Msg("Failed to create service down alert")
		return alertErr
	}

	return nil
}

// DataIntegrityAlert generates an alert for data integrity issues
func (s *AlertGeneratorService) DataIntegrityAlert(ctx context.Context, tableName, issueType, description string, affectedRecords int) error {
	severity := "warning"
	if affectedRecords > 100 || issueType == "corruption" {
		severity = "critical"
	}

	metadata := map[string]interface{}{
		"table_name":       tableName,
		"issue_type":       issueType,
		"affected_records": affectedRecords,
		"timestamp":        time.Now().Unix(),
		"alert_type":       "data_integrity",
	}

	_, err := s.alertService.CreateAlert(
		ctx,
		severity,
		"Data Integrity Issue",
		fmt.Sprintf("Data integrity issue detected in table %s: %s (%d records affected)", tableName, description, affectedRecords),
		"data_monitor",
		metadata,
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("table_name", tableName).
			Str("issue_type", issueType).
			Msg("Failed to create data integrity alert")
		return err
	}

	return nil
}

// SecurityAlert generates an alert for security-related events
func (s *AlertGeneratorService) SecurityAlert(ctx context.Context, eventType, description, ipAddress, userAgent string) error {
	metadata := map[string]interface{}{
		"event_type":  eventType,
		"ip_address":  ipAddress,
		"user_agent":  userAgent,
		"timestamp":   time.Now().Unix(),
		"alert_type":  "security",
	}

	severity := "warning"
	if eventType == "sql_injection" || eventType == "unauthorized_access" || eventType == "privilege_escalation" {
		severity = "critical"
	}

	_, err := s.alertService.CreateAlert(
		ctx,
		severity,
		"Security Event Detected",
		fmt.Sprintf("Security event (%s): %s from IP %s", eventType, description, ipAddress),
		"security_monitor",
		metadata,
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("event_type", eventType).
			Str("ip_address", ipAddress).
			Msg("Failed to create security alert")
		return err
	}

	return nil
}

// PerformanceAlert generates an alert for performance degradation
func (s *AlertGeneratorService) PerformanceAlert(ctx context.Context, component string, metric string, currentValue, threshold float64, unit string) error {
	severity := "warning"
	if currentValue > threshold*1.5 {
		severity = "critical"
	}

	metadata := map[string]interface{}{
		"component":     component,
		"metric":        metric,
		"current_value": currentValue,
		"threshold":     threshold,
		"unit":          unit,
		"timestamp":     time.Now().Unix(),
		"alert_type":    "performance",
	}

	title := fmt.Sprintf("Performance Degradation: %s", component)
	message := fmt.Sprintf("%s %s is %.2f%s, exceeding threshold of %.2f%s", component, metric, currentValue, unit, threshold, unit)

	_, err := s.alertService.CreateAlert(
		ctx,
		severity,
		title,
		message,
		"performance_monitor",
		metadata,
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("component", component).
			Str("metric", metric).
			Msg("Failed to create performance alert")
		return err
	}

	return nil
}

// MaintenanceAlert generates an alert for maintenance events
func (s *AlertGeneratorService) MaintenanceAlert(ctx context.Context, eventType, description string, scheduledTime *time.Time) error {
	metadata := map[string]interface{}{
		"event_type":  eventType,
		"timestamp":   time.Now().Unix(),
		"alert_type":  "maintenance",
	}

	if scheduledTime != nil {
		metadata["scheduled_time"] = scheduledTime.Unix()
	}

	severity := "info"
	title := "Maintenance Event"
	message := fmt.Sprintf("Maintenance event (%s): %s", eventType, description)

	if scheduledTime != nil {
		message = fmt.Sprintf("Scheduled maintenance (%s) at %s: %s", eventType, scheduledTime.Format("2006-01-02 15:04:05"), description)
	}

	_, err := s.alertService.CreateAlert(
		ctx,
		severity,
		title,
		message,
		"maintenance_system",
		metadata,
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("event_type", eventType).
			Msg("Failed to create maintenance alert")
		return err
	}

	return nil
}

// CustomAlert generates a custom alert with specified parameters
func (s *AlertGeneratorService) CustomAlert(ctx context.Context, severity, title, message, source string, metadata map[string]interface{}) error {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	metadata["timestamp"] = time.Now().Unix()
	metadata["alert_type"] = "custom"

	_, err := s.alertService.CreateAlert(
		ctx,
		severity,
		title,
		message,
		source,
		metadata,
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("severity", severity).
			Str("source", source).
			Msg("Failed to create custom alert")
		return err
	}

	return nil
}

// BulkAlertResolution resolves multiple alerts based on criteria
func (s *AlertGeneratorService) BulkAlertResolution(ctx context.Context, source, resolvedBy, notes string) error {
	// Get unresolved alerts from the specified source
	alerts, err := s.alertService.GetAlertsBySource(ctx, source, 100)
	if err != nil {
		return fmt.Errorf("failed to get alerts by source: %w", err)
	}

	resolvedCount := 0
	for _, alert := range alerts {
		if !alert.Resolved {
			_, err := s.alertService.ResolveAlert(ctx, alert.ID, resolvedBy, notes)
			if err != nil {
				log.Error().
					Err(err).
					Str("alert_id", alert.ID).
					Msg("Failed to resolve alert in bulk operation")
				continue
			}
			resolvedCount++
		}
	}

	log.Info().
		Str("source", source).
		Int("resolved_count", resolvedCount).
		Msg("Bulk alert resolution completed")

	return nil
}