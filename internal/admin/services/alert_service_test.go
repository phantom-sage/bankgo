package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAlertService is a mock implementation of AlertService
type MockAlertService struct {
	mock.Mock
}

func (m *MockAlertService) CreateAlert(ctx context.Context, severity, title, message, source string, metadata map[string]interface{}) (*interfaces.Alert, error) {
	args := m.Called(ctx, severity, title, message, source, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.Alert), args.Error(1)
}

func (m *MockAlertService) GetAlert(ctx context.Context, alertID string) (*interfaces.Alert, error) {
	args := m.Called(ctx, alertID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.Alert), args.Error(1)
}

func (m *MockAlertService) ListAlerts(ctx context.Context, params interfaces.AlertParams) (*interfaces.PaginatedAlerts, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.PaginatedAlerts), args.Error(1)
}

func (m *MockAlertService) SearchAlerts(ctx context.Context, searchText string, params interfaces.AlertParams) (*interfaces.PaginatedAlerts, error) {
	args := m.Called(ctx, searchText, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.PaginatedAlerts), args.Error(1)
}

func (m *MockAlertService) AcknowledgeAlert(ctx context.Context, alertID, acknowledgedBy string) (*interfaces.Alert, error) {
	args := m.Called(ctx, alertID, acknowledgedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.Alert), args.Error(1)
}

func (m *MockAlertService) ResolveAlert(ctx context.Context, alertID, resolvedBy, notes string) (*interfaces.Alert, error) {
	args := m.Called(ctx, alertID, resolvedBy, notes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.Alert), args.Error(1)
}

func (m *MockAlertService) GetAlertStatistics(ctx context.Context, timeRange *interfaces.TimeRange) (*interfaces.AlertStatistics, error) {
	args := m.Called(ctx, timeRange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.AlertStatistics), args.Error(1)
}

func (m *MockAlertService) GetUnresolvedAlertsCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockAlertService) GetAlertsBySource(ctx context.Context, source string, limit int) ([]interfaces.Alert, error) {
	args := m.Called(ctx, source, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]interfaces.Alert), args.Error(1)
}

func (m *MockAlertService) CleanupOldResolvedAlerts(ctx context.Context, olderThan time.Time) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

// MockNotificationService is a mock implementation of NotificationService
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) Subscribe(ctx context.Context, conn *websocket.Conn, adminID string) error {
	args := m.Called(ctx, conn, adminID)
	return args.Error(0)
}

func (m *MockNotificationService) Unsubscribe(adminID string, conn *websocket.Conn) error {
	args := m.Called(adminID, conn)
	return args.Error(0)
}

func (m *MockNotificationService) Broadcast(ctx context.Context, notification *interfaces.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationService) SendToAdmin(ctx context.Context, adminID string, notification *interfaces.Notification) error {
	args := m.Called(ctx, adminID, notification)
	return args.Error(0)
}

func (m *MockNotificationService) GetConnectionCount() int {
	args := m.Called()
	return args.Int(0)
}

func TestAlertService_CreateAlert(t *testing.T) {
	// Skip if no database connection available
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer db.Close()

	mockNotificationService := &MockNotificationService{}
	mockNotificationService.On("Broadcast", mock.Anything, mock.AnythingOfType("*interfaces.Notification")).Return(nil)

	alertService := NewAlertService(db, mockNotificationService)
	ctx := context.Background()

	tests := []struct {
		name     string
		severity string
		title    string
		message  string
		source   string
		metadata map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "Valid critical alert",
			severity: "critical",
			title:    "Database Connection Failed",
			message:  "Unable to connect to primary database",
			source:   "database_monitor",
			metadata: map[string]interface{}{
				"error_code": "DB_CONN_FAILED",
				"retry_count": 3,
			},
			wantErr: false,
		},
		{
			name:     "Valid warning alert",
			severity: "warning",
			title:    "High CPU Usage",
			message:  "CPU usage exceeded 80%",
			source:   "system_monitor",
			metadata: map[string]interface{}{
				"cpu_usage": 85.5,
				"threshold": 80.0,
			},
			wantErr: false,
		},
		{
			name:     "Valid info alert",
			severity: "info",
			title:    "System Maintenance",
			message:  "Scheduled maintenance completed successfully",
			source:   "maintenance_system",
			metadata: nil,
			wantErr:  false,
		},
		{
			name:     "Invalid severity",
			severity: "invalid",
			title:    "Test Alert",
			message:  "Test message",
			source:   "test",
			metadata: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert, err := alertService.CreateAlert(ctx, tt.severity, tt.title, tt.message, tt.source, tt.metadata)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, alert)
			} else {
				require.NoError(t, err)
				require.NotNil(t, alert)

				assert.NotEmpty(t, alert.ID)
				assert.Equal(t, tt.severity, alert.Severity)
				assert.Equal(t, tt.title, alert.Title)
				assert.Equal(t, tt.message, alert.Message)
				assert.Equal(t, tt.source, alert.Source)
				assert.False(t, alert.Acknowledged)
				assert.False(t, alert.Resolved)
				assert.WithinDuration(t, time.Now(), alert.Timestamp, 5*time.Second)

				if tt.metadata != nil {
					assert.NotNil(t, alert.Metadata)
				}
			}
		})
	}

	// Verify notification service was called for valid alerts
	mockNotificationService.AssertNumberOfCalls(t, "Broadcast", 3) // 3 valid alerts
}

func TestAlertService_GetAlert(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer db.Close()

	mockNotificationService := &MockNotificationService{}
	mockNotificationService.On("Broadcast", mock.Anything, mock.AnythingOfType("*interfaces.Notification")).Return(nil)

	alertService := NewAlertService(db, mockNotificationService)
	ctx := context.Background()

	// Create a test alert first
	createdAlert, err := alertService.CreateAlert(ctx, "warning", "Test Alert", "Test message", "test_source", nil)
	require.NoError(t, err)
	require.NotNil(t, createdAlert)

	// Test getting the alert
	retrievedAlert, err := alertService.GetAlert(ctx, createdAlert.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedAlert)

	assert.Equal(t, createdAlert.ID, retrievedAlert.ID)
	assert.Equal(t, createdAlert.Severity, retrievedAlert.Severity)
	assert.Equal(t, createdAlert.Title, retrievedAlert.Title)
	assert.Equal(t, createdAlert.Message, retrievedAlert.Message)
	assert.Equal(t, createdAlert.Source, retrievedAlert.Source)

	// Test getting non-existent alert
	_, err = alertService.GetAlert(ctx, "00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)

	// Test invalid UUID format
	_, err = alertService.GetAlert(ctx, "invalid-uuid")
	assert.Error(t, err)
}

func TestAlertService_ListAlerts(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer db.Close()

	mockNotificationService := &MockNotificationService{}
	mockNotificationService.On("Broadcast", mock.Anything, mock.AnythingOfType("*interfaces.Notification")).Return(nil)

	alertService := NewAlertService(db, mockNotificationService)
	ctx := context.Background()

	// Create test alerts
	testAlerts := []struct {
		severity string
		title    string
		source   string
	}{
		{"critical", "Critical Alert 1", "source1"},
		{"warning", "Warning Alert 1", "source1"},
		{"info", "Info Alert 1", "source2"},
		{"critical", "Critical Alert 2", "source2"},
	}

	var createdAlerts []*interfaces.Alert
	for _, ta := range testAlerts {
		alert, err := alertService.CreateAlert(ctx, ta.severity, ta.title, "Test message", ta.source, nil)
		require.NoError(t, err)
		createdAlerts = append(createdAlerts, alert)
	}

	tests := []struct {
		name           string
		params         interfaces.AlertParams
		expectedCount  int
		expectedSeverity string
	}{
		{
			name: "List all alerts",
			params: interfaces.AlertParams{
				PaginationParams: interfaces.PaginationParams{
					Page:     1,
					PageSize: 10,
				},
			},
			expectedCount: 4,
		},
		{
			name: "Filter by severity - critical",
			params: interfaces.AlertParams{
				PaginationParams: interfaces.PaginationParams{
					Page:     1,
					PageSize: 10,
				},
				Severity: "critical",
			},
			expectedCount:    2,
			expectedSeverity: "critical",
		},
		{
			name: "Filter by source",
			params: interfaces.AlertParams{
				PaginationParams: interfaces.PaginationParams{
					Page:     1,
					PageSize: 10,
				},
				Source: "source1",
			},
			expectedCount: 2,
		},
		{
			name: "Pagination - page 1",
			params: interfaces.AlertParams{
				PaginationParams: interfaces.PaginationParams{
					Page:     1,
					PageSize: 2,
				},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := alertService.ListAlerts(ctx, tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Len(t, result.Alerts, tt.expectedCount)
			assert.Equal(t, tt.params.Page, result.Pagination.Page)
			assert.Equal(t, tt.params.PageSize, result.Pagination.PageSize)

			if tt.expectedSeverity != "" {
				for _, alert := range result.Alerts {
					assert.Equal(t, tt.expectedSeverity, alert.Severity)
				}
			}
		})
	}
}

func TestAlertService_AcknowledgeAlert(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer db.Close()

	mockNotificationService := &MockNotificationService{}
	mockNotificationService.On("Broadcast", mock.Anything, mock.AnythingOfType("*interfaces.Notification")).Return(nil)

	alertService := NewAlertService(db, mockNotificationService)
	ctx := context.Background()

	// Create a test alert
	createdAlert, err := alertService.CreateAlert(ctx, "warning", "Test Alert", "Test message", "test_source", nil)
	require.NoError(t, err)
	require.NotNil(t, createdAlert)
	assert.False(t, createdAlert.Acknowledged)

	// Acknowledge the alert
	acknowledgedBy := "admin_user"
	acknowledgedAlert, err := alertService.AcknowledgeAlert(ctx, createdAlert.ID, acknowledgedBy)
	require.NoError(t, err)
	require.NotNil(t, acknowledgedAlert)

	assert.True(t, acknowledgedAlert.Acknowledged)
	assert.Equal(t, acknowledgedBy, acknowledgedAlert.AcknowledgedBy)
	assert.NotNil(t, acknowledgedAlert.AcknowledgedAt)
	assert.WithinDuration(t, time.Now(), *acknowledgedAlert.AcknowledgedAt, 5*time.Second)

	// Test acknowledging non-existent alert
	_, err = alertService.AcknowledgeAlert(ctx, "00000000-0000-0000-0000-000000000000", acknowledgedBy)
	assert.Error(t, err)

	// Test invalid UUID format
	_, err = alertService.AcknowledgeAlert(ctx, "invalid-uuid", acknowledgedBy)
	assert.Error(t, err)

	// Verify notification service was called (once for creation, once for acknowledgment)
	mockNotificationService.AssertNumberOfCalls(t, "Broadcast", 2)
}

func TestAlertService_ResolveAlert(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer db.Close()

	mockNotificationService := &MockNotificationService{}
	mockNotificationService.On("Broadcast", mock.Anything, mock.AnythingOfType("*interfaces.Notification")).Return(nil)

	alertService := NewAlertService(db, mockNotificationService)
	ctx := context.Background()

	// Create a test alert
	createdAlert, err := alertService.CreateAlert(ctx, "critical", "Test Alert", "Test message", "test_source", nil)
	require.NoError(t, err)
	require.NotNil(t, createdAlert)
	assert.False(t, createdAlert.Resolved)

	// Resolve the alert
	resolvedBy := "admin_user"
	notes := "Issue has been fixed by restarting the service"
	resolvedAlert, err := alertService.ResolveAlert(ctx, createdAlert.ID, resolvedBy, notes)
	require.NoError(t, err)
	require.NotNil(t, resolvedAlert)

	assert.True(t, resolvedAlert.Resolved)
	assert.Equal(t, resolvedBy, resolvedAlert.ResolvedBy)
	assert.Equal(t, notes, resolvedAlert.ResolvedNotes)
	assert.NotNil(t, resolvedAlert.ResolvedAt)
	assert.WithinDuration(t, time.Now(), *resolvedAlert.ResolvedAt, 5*time.Second)

	// Test resolving non-existent alert
	_, err = alertService.ResolveAlert(ctx, "00000000-0000-0000-0000-000000000000", resolvedBy, notes)
	assert.Error(t, err)

	// Test invalid UUID format
	_, err = alertService.ResolveAlert(ctx, "invalid-uuid", resolvedBy, notes)
	assert.Error(t, err)

	// Verify notification service was called (once for creation, once for resolution)
	mockNotificationService.AssertNumberOfCalls(t, "Broadcast", 2)
}

func TestAlertService_SearchAlerts(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer db.Close()

	mockNotificationService := &MockNotificationService{}
	mockNotificationService.On("Broadcast", mock.Anything, mock.AnythingOfType("*interfaces.Notification")).Return(nil)

	alertService := NewAlertService(db, mockNotificationService)
	ctx := context.Background()

	// Create test alerts with searchable content
	testAlerts := []struct {
		severity string
		title    string
		message  string
		source   string
	}{
		{"critical", "Database Connection Failed", "Primary database is unreachable", "database_monitor"},
		{"warning", "High Memory Usage", "Memory usage exceeded threshold", "system_monitor"},
		{"info", "Database Backup Complete", "Daily backup completed successfully", "backup_system"},
		{"critical", "API Response Timeout", "API calls are timing out", "api_monitor"},
	}

	for _, ta := range testAlerts {
		_, err := alertService.CreateAlert(ctx, ta.severity, ta.title, ta.message, ta.source, nil)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		searchText    string
		expectedCount int
		description   string
	}{
		{
			name:          "Search by title - database",
			searchText:    "database",
			expectedCount: 2,
			description:   "Should find alerts with 'database' in title",
		},
		{
			name:          "Search by message - timeout",
			searchText:    "timeout",
			expectedCount: 1,
			description:   "Should find alerts with 'timeout' in message",
		},
		{
			name:          "Search by source - monitor",
			searchText:    "monitor",
			expectedCount: 3,
			description:   "Should find alerts with 'monitor' in source",
		},
		{
			name:          "Search non-existent term",
			searchText:    "nonexistent",
			expectedCount: 0,
			description:   "Should find no alerts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := interfaces.AlertParams{
				PaginationParams: interfaces.PaginationParams{
					Page:     1,
					PageSize: 10,
				},
			}

			result, err := alertService.SearchAlerts(ctx, tt.searchText, params)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Len(t, result.Alerts, tt.expectedCount, tt.description)
		})
	}
}

func TestAlertService_GetAlertStatistics(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer db.Close()

	mockNotificationService := &MockNotificationService{}
	mockNotificationService.On("Broadcast", mock.Anything, mock.AnythingOfType("*interfaces.Notification")).Return(nil)

	alertService := NewAlertService(db, mockNotificationService)
	ctx := context.Background()

	// Create test alerts with different severities and states
	testAlerts := []struct {
		severity string
		title    string
	}{
		{"critical", "Critical Alert 1"},
		{"critical", "Critical Alert 2"},
		{"warning", "Warning Alert 1"},
		{"warning", "Warning Alert 2"},
		{"warning", "Warning Alert 3"},
		{"info", "Info Alert 1"},
	}

	var createdAlerts []*interfaces.Alert
	for _, ta := range testAlerts {
		alert, err := alertService.CreateAlert(ctx, ta.severity, ta.title, "Test message", "test_source", nil)
		require.NoError(t, err)
		createdAlerts = append(createdAlerts, alert)
	}

	// Acknowledge some alerts
	_, err := alertService.AcknowledgeAlert(ctx, createdAlerts[0].ID, "admin")
	require.NoError(t, err)
	_, err = alertService.AcknowledgeAlert(ctx, createdAlerts[1].ID, "admin")
	require.NoError(t, err)

	// Resolve some alerts
	_, err = alertService.ResolveAlert(ctx, createdAlerts[0].ID, "admin", "Fixed")
	require.NoError(t, err)

	// Get statistics
	stats, err := alertService.GetAlertStatistics(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, 6, stats.TotalAlerts)
	assert.Equal(t, 2, stats.CriticalCount)
	assert.Equal(t, 3, stats.WarningCount)
	assert.Equal(t, 1, stats.InfoCount)
	assert.Equal(t, 2, stats.AcknowledgedCount)
	assert.Equal(t, 1, stats.ResolvedCount)
	assert.Equal(t, 5, stats.UnresolvedCount)
}

func TestAlertService_GetUnresolvedAlertsCount(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer db.Close()

	mockNotificationService := &MockNotificationService{}
	mockNotificationService.On("Broadcast", mock.Anything, mock.AnythingOfType("*interfaces.Notification")).Return(nil)

	alertService := NewAlertService(db, mockNotificationService)
	ctx := context.Background()

	// Initially should be 0
	count, err := alertService.GetUnresolvedAlertsCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Create some alerts
	alert1, err := alertService.CreateAlert(ctx, "critical", "Alert 1", "Message 1", "source1", nil)
	require.NoError(t, err)
	alert2, err := alertService.CreateAlert(ctx, "warning", "Alert 2", "Message 2", "source2", nil)
	require.NoError(t, err)
	_, err = alertService.CreateAlert(ctx, "info", "Alert 3", "Message 3", "source3", nil)
	require.NoError(t, err)

	// Should be 3 unresolved
	count, err = alertService.GetUnresolvedAlertsCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Resolve one alert
	_, err = alertService.ResolveAlert(ctx, alert1.ID, "admin", "Fixed")
	require.NoError(t, err)

	// Should be 2 unresolved
	count, err = alertService.GetUnresolvedAlertsCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Resolve another alert
	_, err = alertService.ResolveAlert(ctx, alert2.ID, "admin", "Fixed")
	require.NoError(t, err)

	// Should be 1 unresolved
	count, err = alertService.GetUnresolvedAlertsCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestAlertService_GetAlertsBySource(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer db.Close()

	mockNotificationService := &MockNotificationService{}
	mockNotificationService.On("Broadcast", mock.Anything, mock.AnythingOfType("*interfaces.Notification")).Return(nil)

	alertService := NewAlertService(db, mockNotificationService)
	ctx := context.Background()

	// Create alerts from different sources
	sources := []string{"database_monitor", "system_monitor", "database_monitor", "api_monitor"}
	var createdAlerts []*interfaces.Alert

	for i, source := range sources {
		alert, err := alertService.CreateAlert(ctx, "warning", fmt.Sprintf("Alert %d", i+1), "Test message", source, nil)
		require.NoError(t, err)
		createdAlerts = append(createdAlerts, alert)
	}

	// Resolve one database_monitor alert
	_, err := alertService.ResolveAlert(ctx, createdAlerts[0].ID, "admin", "Fixed")
	require.NoError(t, err)

	// Get alerts by source (should only return unresolved ones)
	alerts, err := alertService.GetAlertsBySource(ctx, "database_monitor", 10)
	require.NoError(t, err)
	assert.Len(t, alerts, 1) // Only 1 unresolved database_monitor alert

	alerts, err = alertService.GetAlertsBySource(ctx, "system_monitor", 10)
	require.NoError(t, err)
	assert.Len(t, alerts, 1)

	alerts, err = alertService.GetAlertsBySource(ctx, "nonexistent_source", 10)
	require.NoError(t, err)
	assert.Len(t, alerts, 0)
}

func TestAlertService_CleanupOldResolvedAlerts(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database not available for testing")
		return
	}
	defer db.Close()

	mockNotificationService := &MockNotificationService{}
	mockNotificationService.On("Broadcast", mock.Anything, mock.AnythingOfType("*interfaces.Notification")).Return(nil)

	alertService := NewAlertService(db, mockNotificationService)
	ctx := context.Background()

	// Create and resolve some alerts
	alert1, err := alertService.CreateAlert(ctx, "warning", "Old Alert 1", "Message 1", "source1", nil)
	require.NoError(t, err)
	alert2, err := alertService.CreateAlert(ctx, "info", "Old Alert 2", "Message 2", "source2", nil)
	require.NoError(t, err)
	alert3, err := alertService.CreateAlert(ctx, "critical", "Recent Alert", "Message 3", "source3", nil)
	require.NoError(t, err)

	// Resolve the first two alerts
	_, err = alertService.ResolveAlert(ctx, alert1.ID, "admin", "Fixed")
	require.NoError(t, err)
	_, err = alertService.ResolveAlert(ctx, alert2.ID, "admin", "Fixed")
	require.NoError(t, err)

	// Get initial count
	initialCount, err := alertService.GetAlertStatistics(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, initialCount.TotalAlerts)
	assert.Equal(t, 2, initialCount.ResolvedCount)

	// Cleanup alerts older than now (should remove the resolved ones)
	// Note: In a real scenario, you'd use a more realistic time threshold
	err = alertService.CleanupOldResolvedAlerts(ctx, time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	// Check that resolved alerts were cleaned up
	finalCount, err := alertService.GetAlertStatistics(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, finalCount.TotalAlerts) // Only the unresolved alert should remain
	assert.Equal(t, 0, finalCount.ResolvedCount)
	assert.Equal(t, 1, finalCount.UnresolvedCount)

	// Verify the remaining alert is the unresolved one
	remainingAlert, err := alertService.GetAlert(ctx, alert3.ID)
	require.NoError(t, err)
	assert.Equal(t, "Recent Alert", remainingAlert.Title)
	assert.False(t, remainingAlert.Resolved)
}

// Helper function to set up test database
// This is a simplified version - in a real test, you'd set up a proper test database
func setupTestDB(t *testing.T) *pgxpool.Pool {
	// This would typically connect to a test database
	// For now, we'll skip tests that require a database
	return nil
}