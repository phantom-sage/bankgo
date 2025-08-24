package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAlertGeneratorService_DatabaseConnectionAlert(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	dbError := errors.New("connection timeout")
	connectionType := "primary"

	expectedAlert := &interfaces.Alert{
		ID:       "test-alert-id",
		Severity: "critical",
		Title:    "Database Connection Failed",
		Message:  "Failed to connect to primary database: connection timeout",
		Source:   "database_monitor",
	}

	mockAlertService.On("CreateAlert", 
		ctx, 
		"critical", 
		"Database Connection Failed", 
		"Failed to connect to primary database: connection timeout", 
		"database_monitor", 
		mock.AnythingOfType("map[string]interface {}")).Return(expectedAlert, nil)

	err := generator.DatabaseConnectionAlert(ctx, dbError, connectionType)
	require.NoError(t, err)

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_AuthenticationFailureAlert(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	tests := []struct {
		name           string
		username       string
		ipAddress      string
		failureCount   int
		expectedSeverity string
		expectedTitle    string
	}{
		{
			name:             "Low failure count",
			username:         "testuser",
			ipAddress:        "192.168.1.100",
			failureCount:     3,
			expectedSeverity: "warning",
			expectedTitle:    "Authentication Failure",
		},
		{
			name:             "High failure count",
			username:         "testuser",
			ipAddress:        "192.168.1.100",
			failureCount:     5,
			expectedSeverity: "critical",
			expectedTitle:    "Multiple Authentication Failures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedAlert := &interfaces.Alert{
				ID:       "test-alert-id",
				Severity: tt.expectedSeverity,
				Title:    tt.expectedTitle,
				Source:   "auth_monitor",
			}

			mockAlertService.On("CreateAlert", 
				ctx, 
				tt.expectedSeverity, 
				tt.expectedTitle, 
				mock.AnythingOfType("string"), 
				"auth_monitor", 
				mock.AnythingOfType("map[string]interface {}")).Return(expectedAlert, nil).Once()

			err := generator.AuthenticationFailureAlert(ctx, tt.username, tt.ipAddress, tt.failureCount)
			require.NoError(t, err)
		})
	}

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_TransactionAnomalyAlert(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	tests := []struct {
		name             string
		userID           string
		accountID        string
		amount           float64
		threshold        float64
		anomalyType      string
		expectedSeverity string
	}{
		{
			name:             "Warning level anomaly",
			userID:           "user123",
			accountID:        "acc456",
			amount:           1500.0,
			threshold:        1000.0,
			anomalyType:      "large_transaction",
			expectedSeverity: "warning",
		},
		{
			name:             "Critical level anomaly",
			userID:           "user123",
			accountID:        "acc456",
			amount:           2500.0,
			threshold:        1000.0,
			anomalyType:      "large_transaction",
			expectedSeverity: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedAlert := &interfaces.Alert{
				ID:       "test-alert-id",
				Severity: tt.expectedSeverity,
				Title:    "Transaction Anomaly Detected",
				Source:   "transaction_monitor",
			}

			mockAlertService.On("CreateAlert", 
				ctx, 
				tt.expectedSeverity, 
				"Transaction Anomaly Detected", 
				mock.AnythingOfType("string"), 
				"transaction_monitor", 
				mock.AnythingOfType("map[string]interface {}")).Return(expectedAlert, nil).Once()

			err := generator.TransactionAnomalyAlert(ctx, tt.userID, tt.accountID, tt.amount, tt.threshold, tt.anomalyType)
			require.NoError(t, err)
		})
	}

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_SystemResourceAlert(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	tests := []struct {
		name             string
		resourceType     string
		currentValue     float64
		threshold        float64
		unit             string
		expectedSeverity string
	}{
		{
			name:             "Warning level resource usage",
			resourceType:     "CPU",
			currentValue:     75.0,
			threshold:        70.0,
			unit:             "%",
			expectedSeverity: "warning",
		},
		{
			name:             "Critical level resource usage",
			resourceType:     "Memory",
			currentValue:     95.0,
			threshold:        70.0,
			unit:             "%",
			expectedSeverity: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedAlert := &interfaces.Alert{
				ID:       "test-alert-id",
				Severity: tt.expectedSeverity,
				Title:    "High " + tt.resourceType + " Usage",
				Source:   "system_monitor",
			}

			mockAlertService.On("CreateAlert", 
				ctx, 
				tt.expectedSeverity, 
				"High "+tt.resourceType+" Usage", 
				mock.AnythingOfType("string"), 
				"system_monitor", 
				mock.AnythingOfType("map[string]interface {}")).Return(expectedAlert, nil).Once()

			err := generator.SystemResourceAlert(ctx, tt.resourceType, tt.currentValue, tt.threshold, tt.unit)
			require.NoError(t, err)
		})
	}

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_ServiceDownAlert(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	serviceName := "Banking API"
	serviceURL := "http://localhost:8080"
	serviceError := errors.New("connection refused")

	expectedAlert := &interfaces.Alert{
		ID:       "test-alert-id",
		Severity: "critical",
		Title:    "Service Unavailable",
		Source:   "service_monitor",
	}

	mockAlertService.On("CreateAlert", 
		ctx, 
		"critical", 
		"Service Unavailable", 
		mock.AnythingOfType("string"), 
		"service_monitor", 
		mock.AnythingOfType("map[string]interface {}")).Return(expectedAlert, nil)

	err := generator.ServiceDownAlert(ctx, serviceName, serviceURL, serviceError)
	require.NoError(t, err)

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_DataIntegrityAlert(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	tests := []struct {
		name             string
		tableName        string
		issueType        string
		description      string
		affectedRecords  int
		expectedSeverity string
	}{
		{
			name:             "Minor data inconsistency",
			tableName:        "users",
			issueType:        "inconsistency",
			description:      "Duplicate email addresses found",
			affectedRecords:  5,
			expectedSeverity: "warning",
		},
		{
			name:             "Data corruption",
			tableName:        "accounts",
			issueType:        "corruption",
			description:      "Invalid balance values detected",
			affectedRecords:  10,
			expectedSeverity: "critical",
		},
		{
			name:             "Large number of affected records",
			tableName:        "transactions",
			issueType:        "inconsistency",
			description:      "Missing transaction references",
			affectedRecords:  150,
			expectedSeverity: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedAlert := &interfaces.Alert{
				ID:       "test-alert-id",
				Severity: tt.expectedSeverity,
				Title:    "Data Integrity Issue",
				Source:   "data_monitor",
			}

			mockAlertService.On("CreateAlert", 
				ctx, 
				tt.expectedSeverity, 
				"Data Integrity Issue", 
				mock.AnythingOfType("string"), 
				"data_monitor", 
				mock.AnythingOfType("map[string]interface {}")).Return(expectedAlert, nil).Once()

			err := generator.DataIntegrityAlert(ctx, tt.tableName, tt.issueType, tt.description, tt.affectedRecords)
			require.NoError(t, err)
		})
	}

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_SecurityAlert(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	tests := []struct {
		name             string
		eventType        string
		description      string
		ipAddress        string
		userAgent        string
		expectedSeverity string
	}{
		{
			name:             "Suspicious login attempt",
			eventType:        "suspicious_login",
			description:      "Login from unusual location",
			ipAddress:        "192.168.1.100",
			userAgent:        "Mozilla/5.0",
			expectedSeverity: "warning",
		},
		{
			name:             "SQL injection attempt",
			eventType:        "sql_injection",
			description:      "Malicious SQL detected in request",
			ipAddress:        "10.0.0.1",
			userAgent:        "curl/7.68.0",
			expectedSeverity: "critical",
		},
		{
			name:             "Unauthorized access attempt",
			eventType:        "unauthorized_access",
			description:      "Access to restricted endpoint without authorization",
			ipAddress:        "172.16.0.1",
			userAgent:        "Python-requests/2.25.1",
			expectedSeverity: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedAlert := &interfaces.Alert{
				ID:       "test-alert-id",
				Severity: tt.expectedSeverity,
				Title:    "Security Event Detected",
				Source:   "security_monitor",
			}

			mockAlertService.On("CreateAlert", 
				ctx, 
				tt.expectedSeverity, 
				"Security Event Detected", 
				mock.AnythingOfType("string"), 
				"security_monitor", 
				mock.AnythingOfType("map[string]interface {}")).Return(expectedAlert, nil).Once()

			err := generator.SecurityAlert(ctx, tt.eventType, tt.description, tt.ipAddress, tt.userAgent)
			require.NoError(t, err)
		})
	}

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_PerformanceAlert(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	tests := []struct {
		name             string
		component        string
		metric           string
		currentValue     float64
		threshold        float64
		unit             string
		expectedSeverity string
	}{
		{
			name:             "Warning level performance",
			component:        "Database",
			metric:           "response_time",
			currentValue:     1200.0,
			threshold:        1000.0,
			unit:             "ms",
			expectedSeverity: "warning",
		},
		{
			name:             "Critical level performance",
			component:        "API",
			metric:           "response_time",
			currentValue:     3000.0,
			threshold:        1000.0,
			unit:             "ms",
			expectedSeverity: "critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedAlert := &interfaces.Alert{
				ID:       "test-alert-id",
				Severity: tt.expectedSeverity,
				Title:    "Performance Degradation: " + tt.component,
				Source:   "performance_monitor",
			}

			mockAlertService.On("CreateAlert", 
				ctx, 
				tt.expectedSeverity, 
				"Performance Degradation: "+tt.component, 
				mock.AnythingOfType("string"), 
				"performance_monitor", 
				mock.AnythingOfType("map[string]interface {}")).Return(expectedAlert, nil).Once()

			err := generator.PerformanceAlert(ctx, tt.component, tt.metric, tt.currentValue, tt.threshold, tt.unit)
			require.NoError(t, err)
		})
	}

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_MaintenanceAlert(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	tests := []struct {
		name          string
		eventType     string
		description   string
		scheduledTime *time.Time
	}{
		{
			name:          "Immediate maintenance",
			eventType:     "emergency_restart",
			description:   "Emergency system restart required",
			scheduledTime: nil,
		},
		{
			name:          "Scheduled maintenance",
			eventType:     "database_upgrade",
			description:   "Database will be upgraded to version 15.2",
			scheduledTime: func() *time.Time { t := time.Now().Add(24 * time.Hour); return &t }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedAlert := &interfaces.Alert{
				ID:       "test-alert-id",
				Severity: "info",
				Title:    "Maintenance Event",
				Source:   "maintenance_system",
			}

			mockAlertService.On("CreateAlert", 
				ctx, 
				"info", 
				"Maintenance Event", 
				mock.AnythingOfType("string"), 
				"maintenance_system", 
				mock.AnythingOfType("map[string]interface {}")).Return(expectedAlert, nil).Once()

			err := generator.MaintenanceAlert(ctx, tt.eventType, tt.description, tt.scheduledTime)
			require.NoError(t, err)
		})
	}

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_CustomAlert(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	severity := "warning"
	title := "Custom Test Alert"
	message := "This is a custom alert for testing"
	source := "test_system"
	metadata := map[string]interface{}{
		"test_key": "test_value",
	}

	expectedAlert := &interfaces.Alert{
		ID:       "test-alert-id",
		Severity: severity,
		Title:    title,
		Message:  message,
		Source:   source,
	}

	mockAlertService.On("CreateAlert", 
		ctx, 
		severity, 
		title, 
		message, 
		source, 
		mock.AnythingOfType("map[string]interface {}")).Return(expectedAlert, nil)

	err := generator.CustomAlert(ctx, severity, title, message, source, metadata)
	require.NoError(t, err)

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_CustomAlert_NilMetadata(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	severity := "info"
	title := "Custom Alert Without Metadata"
	message := "This alert has no initial metadata"
	source := "test_system"

	expectedAlert := &interfaces.Alert{
		ID:       "test-alert-id",
		Severity: severity,
		Title:    title,
		Message:  message,
		Source:   source,
	}

	mockAlertService.On("CreateAlert", 
		ctx, 
		severity, 
		title, 
		message, 
		source, 
		mock.AnythingOfType("map[string]interface {}")).Return(expectedAlert, nil)

	err := generator.CustomAlert(ctx, severity, title, message, source, nil)
	require.NoError(t, err)

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_BulkAlertResolution(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	source := "system_monitor"
	resolvedBy := "admin"
	notes := "System maintenance completed"

	// Mock alerts from the source
	alerts := []interfaces.Alert{
		{
			ID:       "alert-1",
			Severity: "warning",
			Title:    "High CPU Usage",
			Source:   source,
			Resolved: false,
		},
		{
			ID:       "alert-2",
			Severity: "critical",
			Title:    "High Memory Usage",
			Source:   source,
			Resolved: false,
		},
		{
			ID:       "alert-3",
			Severity: "info",
			Title:    "System Info",
			Source:   source,
			Resolved: true, // Already resolved
		},
	}

	mockAlertService.On("GetAlertsBySource", ctx, source, 100).Return(alerts, nil)

	// Expect resolution calls only for unresolved alerts
	for _, alert := range alerts {
		if !alert.Resolved {
			resolvedAlert := alert
			resolvedAlert.Resolved = true
			resolvedAlert.ResolvedBy = resolvedBy
			resolvedAlert.ResolvedNotes = notes
			mockAlertService.On("ResolveAlert", ctx, alert.ID, resolvedBy, notes).Return(&resolvedAlert, nil)
		}
	}

	err := generator.BulkAlertResolution(ctx, source, resolvedBy, notes)
	require.NoError(t, err)

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_BulkAlertResolution_GetAlertsError(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	source := "system_monitor"
	resolvedBy := "admin"
	notes := "System maintenance completed"

	mockAlertService.On("GetAlertsBySource", ctx, source, 100).Return(nil, errors.New("database error"))

	err := generator.BulkAlertResolution(ctx, source, resolvedBy, notes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get alerts by source")

	mockAlertService.AssertExpectations(t)
}

func TestAlertGeneratorService_ErrorHandling(t *testing.T) {
	mockAlertService := &MockAlertService{}
	generator := NewAlertGeneratorService(mockAlertService)
	ctx := context.Background()

	// Test error handling when alert service fails
	mockAlertService.On("CreateAlert", 
		mock.Anything, 
		mock.Anything, 
		mock.Anything, 
		mock.Anything, 
		mock.Anything, 
		mock.Anything).Return(nil, errors.New("alert service error"))

	err := generator.DatabaseConnectionAlert(ctx, errors.New("db error"), "primary")
	assert.Error(t, err)
	assert.Equal(t, "alert service error", err.Error())

	mockAlertService.AssertExpectations(t)
}