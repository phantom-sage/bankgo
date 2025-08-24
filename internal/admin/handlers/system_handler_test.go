package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSystemMonitoringService is a mock implementation of SystemMonitoringService
type MockSystemMonitoringService struct {
	mock.Mock
}

func (m *MockSystemMonitoringService) GetSystemHealth(ctx context.Context) (*interfaces.SystemHealth, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.SystemHealth), args.Error(1)
}

func (m *MockSystemMonitoringService) GetMetrics(ctx context.Context, timeRange interfaces.TimeRange) (*interfaces.SystemMetrics, error) {
	args := m.Called(ctx, timeRange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.SystemMetrics), args.Error(1)
}

func (m *MockSystemMonitoringService) GetAlerts(ctx context.Context, params interfaces.AlertParams) (*interfaces.PaginatedAlerts, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.PaginatedAlerts), args.Error(1)
}

func (m *MockSystemMonitoringService) AcknowledgeAlert(ctx context.Context, alertID string) error {
	args := m.Called(ctx, alertID)
	return args.Error(0)
}

func (m *MockSystemMonitoringService) ResolveAlert(ctx context.Context, alertID string, notes string) error {
	args := m.Called(ctx, alertID, notes)
	return args.Error(0)
}

func setupSystemHandler() (*SystemHandlerImpl, *MockSystemMonitoringService) {
	mockService := &MockSystemMonitoringService{}
	handler := NewSystemHandler(mockService).(*SystemHandlerImpl)
	return handler, mockService
}

func TestSystemHandler_GetHealth_Success(t *testing.T) {
	handler, mockService := setupSystemHandler()
	
	// Setup mock response
	expectedHealth := &interfaces.SystemHealth{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services: map[string]interfaces.ServiceHealth{
			"database": {Status: "healthy", LastCheck: time.Now(), ResponseTime: 50 * time.Millisecond},
		},
		Metrics: interfaces.SystemMetricsSnapshot{
			CPUUsage:        45.5,
			MemoryUsage:     60.2,
			DBConnections:   10,
			APIResponseTime: 120.5,
			ActiveSessions:  5,
		},
		AlertCount: 2,
	}
	
	mockService.On("GetSystemHealth", mock.Anything).Return(expectedHealth, nil)
	
	// Setup request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/system/health", nil)
	
	// Execute
	handler.GetHealth(c)
	
	// Verify
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response interfaces.SystemHealth
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, expectedHealth.Status, response.Status)
	assert.Equal(t, expectedHealth.AlertCount, response.AlertCount)
	assert.Equal(t, expectedHealth.Metrics.CPUUsage, response.Metrics.CPUUsage)
	
	mockService.AssertExpectations(t)
}

func TestSystemHandler_GetHealth_ServiceError(t *testing.T) {
	handler, mockService := setupSystemHandler()
	
	// Setup mock error
	mockService.On("GetSystemHealth", mock.Anything).Return(nil, fmt.Errorf("service error"))
	
	// Setup request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/system/health", nil)
	
	// Execute
	handler.GetHealth(c)
	
	// Verify
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "failed_to_get_health", response["error"])
	assert.Contains(t, response["details"], "service error")
	
	mockService.AssertExpectations(t)
}

func TestSystemHandler_GetMetrics_Success(t *testing.T) {
	handler, mockService := setupSystemHandler()
	
	// Setup mock response
	now := time.Now()
	expectedMetrics := &interfaces.SystemMetrics{
		TimeRange: interfaces.TimeRange{
			Start: now.Add(-time.Hour),
			End:   now,
		},
		Interval: time.Minute,
		DataPoints: []interfaces.SystemMetricsSnapshot{
			{CPUUsage: 45.5, MemoryUsage: 60.2},
			{CPUUsage: 50.1, MemoryUsage: 65.8},
		},
	}
	
	mockService.On("GetMetrics", mock.Anything, mock.AnythingOfType("interfaces.TimeRange")).Return(expectedMetrics, nil)
	
	// Setup request with time range parameters
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/system/metrics?duration=1h", nil)
	
	// Execute
	handler.GetMetrics(c)
	
	// Verify
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response interfaces.SystemMetrics
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, expectedMetrics.Interval, response.Interval)
	assert.Len(t, response.DataPoints, 2)
	
	mockService.AssertExpectations(t)
}

func TestSystemHandler_GetMetrics_InvalidTimeRange(t *testing.T) {
	handler, _ := setupSystemHandler()
	
	// Setup request with invalid time format
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/system/metrics?start=invalid-time", nil)
	
	// Execute
	handler.GetMetrics(c)
	
	// Verify
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "invalid_time_range", response["error"])
}

func TestSystemHandler_GetAlerts_Success(t *testing.T) {
	handler, mockService := setupSystemHandler()
	
	// Setup mock response
	expectedAlerts := &interfaces.PaginatedAlerts{
		Alerts: []interfaces.Alert{
			{
				ID:        "alert-1",
				Severity:  "critical",
				Title:     "High CPU Usage",
				Message:   "CPU usage is above 90%",
				Source:    "system_monitor",
				Timestamp: time.Now(),
			},
		},
		Pagination: interfaces.PaginationInfo{
			Page:       1,
			PageSize:   20,
			TotalItems: 1,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}
	
	mockService.On("GetAlerts", mock.Anything, mock.AnythingOfType("interfaces.AlertParams")).Return(expectedAlerts, nil)
	
	// Setup request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/system/alerts?severity=critical&page=1&page_size=20", nil)
	
	// Execute
	handler.GetAlerts(c)
	
	// Verify
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response interfaces.PaginatedAlerts
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Len(t, response.Alerts, 1)
	assert.Equal(t, "critical", response.Alerts[0].Severity)
	assert.Equal(t, 1, response.Pagination.Page)
	
	mockService.AssertExpectations(t)
}

func TestSystemHandler_GetAlerts_InvalidParameters(t *testing.T) {
	handler, _ := setupSystemHandler()
	
	// Setup request with invalid page parameter
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/system/alerts?page=invalid", nil)
	
	// Execute
	handler.GetAlerts(c)
	
	// Verify
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "invalid_parameters", response["error"])
}

func TestSystemHandler_AcknowledgeAlert_Success(t *testing.T) {
	handler, mockService := setupSystemHandler()
	
	// Setup mock
	alertID := "test-alert-123"
	mockService.On("AcknowledgeAlert", mock.Anything, alertID).Return(nil)
	
	// Setup request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", fmt.Sprintf("/system/alerts/%s/acknowledge", alertID), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID}}
	
	// Execute
	handler.AcknowledgeAlert(c)
	
	// Verify
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Alert acknowledged successfully", response["message"])
	assert.Equal(t, alertID, response["alert_id"])
	
	mockService.AssertExpectations(t)
}

func TestSystemHandler_AcknowledgeAlert_MissingID(t *testing.T) {
	handler, _ := setupSystemHandler()
	
	// Setup request without alert ID
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/system/alerts//acknowledge", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	
	// Execute
	handler.AcknowledgeAlert(c)
	
	// Verify
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "missing_alert_id", response["error"])
}

func TestSystemHandler_AcknowledgeAlert_ServiceError(t *testing.T) {
	handler, mockService := setupSystemHandler()
	
	// Setup mock error
	alertID := "test-alert-123"
	mockService.On("AcknowledgeAlert", mock.Anything, alertID).Return(fmt.Errorf("alert not found"))
	
	// Setup request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", fmt.Sprintf("/system/alerts/%s/acknowledge", alertID), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID}}
	
	// Execute
	handler.AcknowledgeAlert(c)
	
	// Verify
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "failed_to_acknowledge_alert", response["error"])
	assert.Contains(t, response["details"], "alert not found")
	
	mockService.AssertExpectations(t)
}

func TestSystemHandler_ResolveAlert_Success(t *testing.T) {
	handler, mockService := setupSystemHandler()
	
	// Setup mock
	alertID := "test-alert-123"
	notes := "Issue resolved by restarting service"
	mockService.On("ResolveAlert", mock.Anything, alertID, notes).Return(nil)
	
	// Setup request body
	requestBody := map[string]string{
		"notes": notes,
	}
	bodyBytes, _ := json.Marshal(requestBody)
	
	// Setup request
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", fmt.Sprintf("/system/alerts/%s/resolve", alertID), bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: alertID}}
	
	// Execute
	handler.ResolveAlert(c)
	
	// Verify
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Alert resolved successfully", response["message"])
	assert.Equal(t, alertID, response["alert_id"])
	assert.Equal(t, notes, response["notes"])
	
	mockService.AssertExpectations(t)
}

func TestSystemHandler_ResolveAlert_InvalidRequestBody(t *testing.T) {
	handler, _ := setupSystemHandler()
	
	alertID := "test-alert-123"
	
	// Setup request with invalid JSON
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", fmt.Sprintf("/system/alerts/%s/resolve", alertID), bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: alertID}}
	
	// Execute
	handler.ResolveAlert(c)
	
	// Verify
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "invalid_request_body", response["error"])
}

func TestSystemHandler_ParseTimeRange(t *testing.T) {
	handler, _ := setupSystemHandler()
	
	// Test default time range (no parameters)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/system/metrics", nil)
	
	timeRange, err := handler.parseTimeRange(c)
	require.NoError(t, err)
	
	// Should default to last hour
	assert.True(t, timeRange.End.After(timeRange.Start))
	assert.True(t, timeRange.End.Sub(timeRange.Start) <= time.Hour+time.Minute) // Allow some tolerance
	
	// Test with duration parameter
	req := httptest.NewRequest("GET", "/system/metrics?duration=2h", nil)
	c, _ = gin.CreateTestContext(w)
	c.Request = req
	
	timeRange, err = handler.parseTimeRange(c)
	require.NoError(t, err)
	
	expectedDuration := 2 * time.Hour
	actualDuration := timeRange.End.Sub(timeRange.Start)
	assert.True(t, actualDuration >= expectedDuration-time.Minute && actualDuration <= expectedDuration+time.Minute)
	
	// Test with explicit start and end times
	now := time.Now().UTC()
	start := now.Add(-3 * time.Hour)
	end := now.Add(-time.Hour)
	
	req = httptest.NewRequest("GET", fmt.Sprintf("/system/metrics?start=%s&end=%s", 
		start.Format(time.RFC3339), end.Format(time.RFC3339)), nil)
	c, _ = gin.CreateTestContext(w)
	c.Request = req
	
	timeRange, err = handler.parseTimeRange(c)
	require.NoError(t, err)
	
	assert.True(t, timeRange.Start.Equal(start) || timeRange.Start.Sub(start) < time.Second)
	assert.True(t, timeRange.End.Equal(end) || timeRange.End.Sub(end) < time.Second)
}

func TestSystemHandler_ParseAlertParams(t *testing.T) {
	handler, _ := setupSystemHandler()
	
	// Test default parameters
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/system/alerts", nil)
	
	params, err := handler.parseAlertParams(c)
	require.NoError(t, err)
	
	assert.Equal(t, 1, params.Page)
	assert.Equal(t, 20, params.PageSize)
	
	// Test with all parameters
	req := httptest.NewRequest("GET", "/system/alerts?page=2&page_size=50&severity=critical&acknowledged=true&resolved=false&source=system", nil)
	c, _ = gin.CreateTestContext(w)
	c.Request = req
	
	params, err = handler.parseAlertParams(c)
	require.NoError(t, err)
	
	assert.Equal(t, 2, params.Page)
	assert.Equal(t, 50, params.PageSize)
	assert.Equal(t, "critical", params.Severity)
	assert.Equal(t, "system", params.Source)
	assert.NotNil(t, params.Acknowledged)
	assert.True(t, *params.Acknowledged)
	assert.NotNil(t, params.Resolved)
	assert.False(t, *params.Resolved)
}