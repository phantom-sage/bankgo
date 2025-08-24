package services

import (
	"context"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAlertService for testing
type MockAlertServiceForSystemMonitoring struct {
	mock.Mock
}

func (m *MockAlertServiceForSystemMonitoring) CreateAlert(ctx context.Context, severity, title, message, source string, metadata map[string]interface{}) (*interfaces.Alert, error) {
	args := m.Called(ctx, severity, title, message, source, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.Alert), args.Error(1)
}

func (m *MockAlertServiceForSystemMonitoring) GetAlert(ctx context.Context, alertID string) (*interfaces.Alert, error) {
	args := m.Called(ctx, alertID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.Alert), args.Error(1)
}

func (m *MockAlertServiceForSystemMonitoring) ListAlerts(ctx context.Context, params interfaces.AlertParams) (*interfaces.PaginatedAlerts, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.PaginatedAlerts), args.Error(1)
}

func (m *MockAlertServiceForSystemMonitoring) SearchAlerts(ctx context.Context, searchText string, params interfaces.AlertParams) (*interfaces.PaginatedAlerts, error) {
	args := m.Called(ctx, searchText, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.PaginatedAlerts), args.Error(1)
}

func (m *MockAlertServiceForSystemMonitoring) AcknowledgeAlert(ctx context.Context, alertID, acknowledgedBy string) (*interfaces.Alert, error) {
	args := m.Called(ctx, alertID, acknowledgedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.Alert), args.Error(1)
}

func (m *MockAlertServiceForSystemMonitoring) ResolveAlert(ctx context.Context, alertID, resolvedBy, notes string) (*interfaces.Alert, error) {
	args := m.Called(ctx, alertID, resolvedBy, notes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.Alert), args.Error(1)
}

func (m *MockAlertServiceForSystemMonitoring) GetAlertStatistics(ctx context.Context, timeRange *interfaces.TimeRange) (*interfaces.AlertStatistics, error) {
	args := m.Called(ctx, timeRange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.AlertStatistics), args.Error(1)
}

func (m *MockAlertServiceForSystemMonitoring) GetUnresolvedAlertsCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockAlertServiceForSystemMonitoring) GetAlertsBySource(ctx context.Context, source string, limit int) ([]interfaces.Alert, error) {
	args := m.Called(ctx, source, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]interfaces.Alert), args.Error(1)
}

func (m *MockAlertServiceForSystemMonitoring) CleanupOldResolvedAlerts(ctx context.Context, olderThan time.Time) error {
	args := m.Called(ctx, olderThan)
	return args.Error(0)
}

func TestSystemMonitoringService_BasicFunctionality(t *testing.T) {
	// Create mock alert service
	mockAlertService := &MockAlertServiceForSystemMonitoring{}
	mockAlertService.On("GetUnresolvedAlertsCount", mock.Anything).Return(5, nil)
	
	// Create service
	service := NewSystemMonitoringService(nil, nil, "http://localhost:8080", mockAlertService)
	ctx := context.Background()
	
	// Test getting system health
	health, err := service.GetSystemHealth(ctx)
	require.NoError(t, err)
	require.NotNil(t, health)
	
	// Verify health structure
	assert.NotEmpty(t, health.Status)
	assert.NotZero(t, health.Timestamp)
	assert.NotNil(t, health.Services)
	assert.Equal(t, 5, health.AlertCount) // From mock
	
	// Verify services are checked
	assert.Contains(t, health.Services, "database")
	assert.Contains(t, health.Services, "redis")
	assert.Contains(t, health.Services, "banking_api")
	
	mockAlertService.AssertExpectations(t)
}

func TestSystemMonitoringService_GetMetrics(t *testing.T) {
	mockAlertService := &MockAlertServiceForSystemMonitoring{}
	service := NewSystemMonitoringService(nil, nil, "http://localhost:8080", mockAlertService)
	
	ctx := context.Background()
	
	// Define time range
	timeRange := interfaces.TimeRange{
		Start: time.Now().Add(-1 * time.Hour),
		End:   time.Now(),
	}
	
	// Test getting metrics
	metrics, err := service.GetMetrics(ctx, timeRange)
	require.NoError(t, err)
	require.NotNil(t, metrics)
	
	// Verify metrics structure
	assert.Equal(t, timeRange.Start, metrics.TimeRange.Start)
	assert.Equal(t, timeRange.End, metrics.TimeRange.End)
	assert.Greater(t, metrics.Interval, time.Duration(0))
	assert.NotNil(t, metrics.DataPoints)
}

func TestSystemMonitoringService_AlertDelegation(t *testing.T) {
	// Test that alert operations are properly delegated to the alert service
	mockAlertService := &MockAlertServiceForSystemMonitoring{}
	
	// Set up expectations for alert operations
	mockAlertService.On("ListAlerts", mock.Anything, mock.AnythingOfType("interfaces.AlertParams")).Return(&interfaces.PaginatedAlerts{
		Alerts: []interfaces.Alert{
			{
				ID:        "test-alert",
				Severity:  "warning",
				Title:     "Test Alert",
				Message:   "Test message",
				Source:    "test",
				Timestamp: time.Now(),
			},
		},
		Pagination: interfaces.PaginationInfo{
			Page:       1,
			PageSize:   20,
			TotalItems: 1,
			TotalPages: 1,
		},
	}, nil)
	
	mockAlertService.On("AcknowledgeAlert", mock.Anything, "test-alert", "admin").Return(&interfaces.Alert{
		ID:             "test-alert",
		Acknowledged:   true,
		AcknowledgedBy: "admin",
	}, nil)
	
	mockAlertService.On("ResolveAlert", mock.Anything, "test-alert", "admin", "Fixed").Return(&interfaces.Alert{
		ID:            "test-alert",
		Resolved:      true,
		ResolvedBy:    "admin",
		ResolvedNotes: "Fixed",
	}, nil)
	
	service := NewSystemMonitoringService(nil, nil, "http://localhost:8080", mockAlertService)
	ctx := context.Background()
	
	// Test GetAlerts delegation
	params := interfaces.AlertParams{
		PaginationParams: interfaces.PaginationParams{
			Page:     1,
			PageSize: 20,
		},
	}
	alerts, err := service.GetAlerts(ctx, params)
	require.NoError(t, err)
	assert.Len(t, alerts.Alerts, 1)
	assert.Equal(t, "test-alert", alerts.Alerts[0].ID)
	
	// Test AcknowledgeAlert delegation
	err = service.AcknowledgeAlert(ctx, "test-alert")
	require.NoError(t, err)
	
	// Test ResolveAlert delegation
	err = service.ResolveAlert(ctx, "test-alert", "Fixed")
	require.NoError(t, err)
	
	mockAlertService.AssertExpectations(t)
}

func TestSystemMonitoringService_MetricsCollection(t *testing.T) {
	mockAlertService := &MockAlertServiceForSystemMonitoring{}
	service := NewSystemMonitoringService(nil, nil, "http://localhost:8080", mockAlertService)
	serviceImpl := service.(*SystemMonitoringServiceImpl)
	
	ctx := context.Background()
	
	// Test metrics collection
	metrics, err := serviceImpl.collectCurrentMetrics(ctx)
	require.NoError(t, err)
	require.NotNil(t, metrics)
	
	// Verify metrics are reasonable
	assert.GreaterOrEqual(t, metrics.CPUUsage, 0.0)
	assert.GreaterOrEqual(t, metrics.MemoryUsage, 0.0)
	assert.GreaterOrEqual(t, metrics.DBConnections, 0)
	assert.GreaterOrEqual(t, metrics.APIResponseTime, 0.0)
	assert.GreaterOrEqual(t, metrics.ActiveSessions, 0)
}

func TestSystemMonitoringService_ServiceHealthChecks(t *testing.T) {
	mockAlertService := &MockAlertServiceForSystemMonitoring{}
	service := NewSystemMonitoringService(nil, nil, "http://localhost:8080", mockAlertService)
	serviceImpl := service.(*SystemMonitoringServiceImpl)
	
	ctx := context.Background()
	
	// Test individual service health checks
	dbHealth := serviceImpl.CheckDatabaseHealth(ctx)
	assert.NotEmpty(t, dbHealth.Status)
	assert.NotZero(t, dbHealth.LastCheck)
	
	redisHealth := serviceImpl.CheckRedisHealth(ctx)
	assert.NotEmpty(t, redisHealth.Status)
	assert.NotZero(t, redisHealth.LastCheck)
	
	apiHealth := serviceImpl.CheckBankingAPIHealth(ctx)
	assert.NotEmpty(t, apiHealth.Status)
	assert.NotZero(t, apiHealth.LastCheck)
}

func TestSystemMonitoringService_AlertGeneration(t *testing.T) {
	mockAlertService := &MockAlertServiceForSystemMonitoring{}
	
	// Set up expectations for alert creation
	mockAlertService.On("CreateAlert", mock.Anything, "critical", "High CPU Usage", mock.AnythingOfType("string"), "system_monitor", mock.AnythingOfType("map[string]interface {}")).Return(&interfaces.Alert{
		ID:       "cpu-alert",
		Severity: "critical",
		Title:    "High CPU Usage",
	}, nil)
	
	mockAlertService.On("CreateAlert", mock.Anything, "critical", "High Memory Usage", mock.AnythingOfType("string"), "system_monitor", mock.AnythingOfType("map[string]interface {}")).Return(&interfaces.Alert{
		ID:       "memory-alert",
		Severity: "critical",
		Title:    "High Memory Usage",
	}, nil)
	
	service := NewSystemMonitoringService(nil, nil, "http://localhost:8080", mockAlertService)
	serviceImpl := service.(*SystemMonitoringServiceImpl)
	
	// Test alert generation for high CPU
	highCPUMetrics := interfaces.SystemMetricsSnapshot{
		CPUUsage:        95.0, // Above critical threshold
		MemoryUsage:     50.0,
		DBConnections:   10,
		APIResponseTime: 100.0,
		ActiveSessions:  5,
	}
	
	serviceImpl.CheckMetricsForAlerts(highCPUMetrics)
	
	// Test alert generation for high memory
	highMemoryMetrics := interfaces.SystemMetricsSnapshot{
		CPUUsage:        50.0,
		MemoryUsage:     95.0, // Above critical threshold
		DBConnections:   10,
		APIResponseTime: 100.0,
		ActiveSessions:  5,
	}
	
	serviceImpl.CheckMetricsForAlerts(highMemoryMetrics)
	
	mockAlertService.AssertExpectations(t)
}

func TestSystemMonitoringService_OverallStatusDetermination(t *testing.T) {
	mockAlertService := &MockAlertServiceForSystemMonitoring{}
	service := NewSystemMonitoringService(nil, nil, "http://localhost:8080", mockAlertService)
	serviceImpl := service.(*SystemMonitoringServiceImpl)
	
	// Test healthy status
	healthyServices := map[string]interfaces.ServiceHealth{
		"database": {Status: "healthy"},
		"redis":    {Status: "healthy"},
		"api":      {Status: "healthy"},
	}
	healthyMetrics := &interfaces.SystemMetricsSnapshot{
		CPUUsage:    50.0,
		MemoryUsage: 60.0,
	}
	
	status := serviceImpl.determineOverallStatus(healthyServices, healthyMetrics)
	assert.Equal(t, "healthy", status)
	
	// Test warning status
	warningServices := map[string]interfaces.ServiceHealth{
		"database": {Status: "healthy"},
		"redis":    {Status: "warning"},
		"api":      {Status: "healthy"},
	}
	
	status = serviceImpl.determineOverallStatus(warningServices, healthyMetrics)
	assert.Equal(t, "warning", status)
	
	// Test critical status
	criticalServices := map[string]interfaces.ServiceHealth{
		"database": {Status: "critical"},
		"redis":    {Status: "healthy"},
		"api":      {Status: "healthy"},
	}
	
	status = serviceImpl.determineOverallStatus(criticalServices, healthyMetrics)
	assert.Equal(t, "critical", status)
	
	// Test critical status from metrics
	criticalMetrics := &interfaces.SystemMetricsSnapshot{
		CPUUsage:    95.0, // Above critical threshold
		MemoryUsage: 60.0,
	}
	
	status = serviceImpl.determineOverallStatus(healthyServices, criticalMetrics)
	assert.Equal(t, "critical", status)
}