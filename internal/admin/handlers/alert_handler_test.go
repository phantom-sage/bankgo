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

func setupAlertHandler() (*AlertHandlerImpl, *MockAlertService) {
	mockService := &MockAlertService{}
	handler := NewAlertHandler(mockService).(*AlertHandlerImpl)
	return handler, mockService
}

func TestAlertHandler_CreateAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockService := setupAlertHandler()

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		mockSetup      func(*MockAlertService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid alert creation",
			requestBody: map[string]interface{}{
				"severity": "critical",
				"title":    "Database Connection Failed",
				"message":  "Unable to connect to primary database",
				"source":   "database_monitor",
				"metadata": map[string]interface{}{
					"error_code": "DB_CONN_FAILED",
				},
			},
			mockSetup: func(m *MockAlertService) {
				alert := &interfaces.Alert{
					ID:        "test-alert-id",
					Severity:  "critical",
					Title:     "Database Connection Failed",
					Message:   "Unable to connect to primary database",
					Source:    "database_monitor",
					Timestamp: time.Now(),
				}
				m.On("CreateAlert", mock.Anything, "critical", "Database Connection Failed", "Unable to connect to primary database", "database_monitor", mock.AnythingOfType("map[string]interface {}")).Return(alert, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Invalid severity",
			requestBody: map[string]interface{}{
				"severity": "invalid",
				"title":    "Test Alert",
				"message":  "Test message",
				"source":   "test_source",
			},
			mockSetup:      func(m *MockAlertService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_request_body",
		},
		{
			name: "Missing required fields",
			requestBody: map[string]interface{}{
				"severity": "warning",
			},
			mockSetup:      func(m *MockAlertService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_request_body",
		},
		{
			name: "Service error",
			requestBody: map[string]interface{}{
				"severity": "warning",
				"title":    "Test Alert",
				"message":  "Test message",
				"source":   "test_source",
			},
			mockSetup: func(m *MockAlertService) {
				m.On("CreateAlert", mock.Anything, "warning", "Test Alert", "Test message", "test_source", mock.Anything).Return(nil, fmt.Errorf("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "failed_to_create_alert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(mockService)

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/alerts", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			handler.CreateAlert(c)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAlertHandler_GetAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockService := setupAlertHandler()

	tests := []struct {
		name           string
		alertID        string
		mockSetup      func(*MockAlertService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:    "Valid alert retrieval",
			alertID: "test-alert-id",
			mockSetup: func(m *MockAlertService) {
				alert := &interfaces.Alert{
					ID:        "test-alert-id",
					Severity:  "warning",
					Title:     "Test Alert",
					Message:   "Test message",
					Source:    "test_source",
					Timestamp: time.Now(),
				}
				m.On("GetAlert", mock.Anything, "test-alert-id").Return(alert, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "Alert not found",
			alertID: "non-existent-id",
			mockSetup: func(m *MockAlertService) {
				m.On("GetAlert", mock.Anything, "non-existent-id").Return(nil, fmt.Errorf("alert not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "alert_not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(mockService)

			// Create request
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/alerts/%s", tt.alertID), nil)
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: tt.alertID}}

			// Call handler
			handler.GetAlert(c)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAlertHandler_ListAlerts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockService := setupAlertHandler()

	mockService.On("ListAlerts", mock.Anything, mock.AnythingOfType("interfaces.AlertParams")).Return(&interfaces.PaginatedAlerts{
		Alerts: []interfaces.Alert{
			{
				ID:        "alert-1",
				Severity:  "critical",
				Title:     "Alert 1",
				Message:   "Message 1",
				Source:    "source1",
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
	}, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/alerts?page=1&page_size=20&severity=critical", nil)
	w := httptest.NewRecorder()

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call handler
	handler.ListAlerts(c)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response interfaces.PaginatedAlerts
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response.Alerts, 1)
	assert.Equal(t, "alert-1", response.Alerts[0].ID)

	mockService.AssertExpectations(t)
}

func TestAlertHandler_SearchAlerts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockService := setupAlertHandler()

	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(*MockAlertService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:        "Valid search",
			queryParams: "?q=database&page=1&page_size=10",
			mockSetup: func(m *MockAlertService) {
				m.On("SearchAlerts", mock.Anything, "database", mock.AnythingOfType("interfaces.AlertParams")).Return(&interfaces.PaginatedAlerts{
					Alerts: []interfaces.Alert{
						{
							ID:        "alert-1",
							Severity:  "critical",
							Title:     "Database Alert",
							Message:   "Database connection failed",
							Source:    "database_monitor",
							Timestamp: time.Now(),
						},
					},
					Pagination: interfaces.PaginationInfo{
						Page:       1,
						PageSize:   10,
						TotalItems: 1,
						TotalPages: 1,
						HasNext:    false,
						HasPrev:    false,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing search query",
			queryParams:    "?page=1",
			mockSetup:      func(m *MockAlertService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "missing_search_query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(mockService)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/alerts/search"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			handler.SearchAlerts(c)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAlertHandler_AcknowledgeAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockService := setupAlertHandler()

	tests := []struct {
		name           string
		alertID        string
		requestBody    map[string]interface{}
		mockSetup      func(*MockAlertService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:    "Valid acknowledgment",
			alertID: "test-alert-id",
			requestBody: map[string]interface{}{
				"acknowledged_by": "admin_user",
			},
			mockSetup: func(m *MockAlertService) {
				alert := &interfaces.Alert{
					ID:             "test-alert-id",
					Severity:       "warning",
					Title:          "Test Alert",
					Acknowledged:   true,
					AcknowledgedBy: "admin_user",
				}
				m.On("AcknowledgeAlert", mock.Anything, "test-alert-id", "admin_user").Return(alert, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "Default acknowledged_by",
			alertID: "test-alert-id",
			requestBody: map[string]interface{}{},
			mockSetup: func(m *MockAlertService) {
				alert := &interfaces.Alert{
					ID:             "test-alert-id",
					Severity:       "warning",
					Title:          "Test Alert",
					Acknowledged:   true,
					AcknowledgedBy: "admin",
				}
				m.On("AcknowledgeAlert", mock.Anything, "test-alert-id", "admin").Return(alert, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing alert ID",
			alertID:        "",
			requestBody:    map[string]interface{}{},
			mockSetup:      func(m *MockAlertService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "missing_alert_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(mockService)

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/alerts/%s/acknowledge", tt.alertID), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: tt.alertID}}

			// Call handler
			handler.AcknowledgeAlert(c)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAlertHandler_ResolveAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockService := setupAlertHandler()

	tests := []struct {
		name           string
		alertID        string
		requestBody    map[string]interface{}
		mockSetup      func(*MockAlertService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:    "Valid resolution",
			alertID: "test-alert-id",
			requestBody: map[string]interface{}{
				"resolved_by": "admin_user",
				"notes":       "Issue has been fixed",
			},
			mockSetup: func(m *MockAlertService) {
				alert := &interfaces.Alert{
					ID:            "test-alert-id",
					Severity:      "warning",
					Title:         "Test Alert",
					Resolved:      true,
					ResolvedBy:    "admin_user",
					ResolvedNotes: "Issue has been fixed",
				}
				m.On("ResolveAlert", mock.Anything, "test-alert-id", "admin_user", "Issue has been fixed").Return(alert, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "Default resolved_by",
			alertID: "test-alert-id",
			requestBody: map[string]interface{}{
				"notes": "Fixed automatically",
			},
			mockSetup: func(m *MockAlertService) {
				alert := &interfaces.Alert{
					ID:            "test-alert-id",
					Severity:      "warning",
					Title:         "Test Alert",
					Resolved:      true,
					ResolvedBy:    "admin",
					ResolvedNotes: "Fixed automatically",
				}
				m.On("ResolveAlert", mock.Anything, "test-alert-id", "admin", "Fixed automatically").Return(alert, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid request body",
			alertID:        "test-alert-id",
			requestBody:    nil,
			mockSetup:      func(m *MockAlertService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_request_body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(mockService)

			// Create request
			var body *bytes.Buffer
			if tt.requestBody != nil {
				bodyBytes, _ := json.Marshal(tt.requestBody)
				body = bytes.NewBuffer(bodyBytes)
			} else {
				body = bytes.NewBuffer([]byte("invalid json"))
			}

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/alerts/%s/resolve", tt.alertID), body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: tt.alertID}}

			// Call handler
			handler.ResolveAlert(c)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAlertHandler_GetAlertStatistics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockService := setupAlertHandler()

	mockService.On("GetAlertStatistics", mock.Anything, mock.Anything).Return(&interfaces.AlertStatistics{
		TotalAlerts:       10,
		CriticalCount:     2,
		WarningCount:      5,
		InfoCount:         3,
		AcknowledgedCount: 6,
		ResolvedCount:     4,
		UnresolvedCount:   6,
	}, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/alerts/statistics", nil)
	w := httptest.NewRecorder()

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Call handler
	handler.GetAlertStatistics(c)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response interfaces.AlertStatistics
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 10, response.TotalAlerts)
	assert.Equal(t, 2, response.CriticalCount)
	assert.Equal(t, 5, response.WarningCount)

	mockService.AssertExpectations(t)
}

func TestAlertHandler_GetAlertsBySource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockService := setupAlertHandler()

	mockService.On("GetAlertsBySource", mock.Anything, "database_monitor", 10).Return([]interfaces.Alert{
		{
			ID:        "alert-1",
			Severity:  "critical",
			Title:     "Database Alert",
			Source:    "database_monitor",
			Timestamp: time.Now(),
		},
	}, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/alerts/by-source/database_monitor?limit=10", nil)
	w := httptest.NewRecorder()

	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "source", Value: "database_monitor"}}

	// Call handler
	handler.GetAlertsBySource(c)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "database_monitor", response["source"])
	assert.Equal(t, float64(10), response["limit"])

	alerts := response["alerts"].([]interface{})
	assert.Len(t, alerts, 1)

	mockService.AssertExpectations(t)
}

func TestAlertHandler_CleanupOldResolvedAlerts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockService := setupAlertHandler()

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		mockSetup      func(*MockAlertService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid cleanup",
			requestBody: map[string]interface{}{
				"older_than_days": 30,
			},
			mockSetup: func(m *MockAlertService) {
				m.On("CleanupOldResolvedAlerts", mock.Anything, mock.AnythingOfType("time.Time")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid days value",
			requestBody: map[string]interface{}{
				"older_than_days": 0,
			},
			mockSetup:      func(m *MockAlertService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_request_body",
		},
		{
			name:           "Missing required field",
			requestBody:    map[string]interface{}{},
			mockSetup:      func(m *MockAlertService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_request_body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(mockService)

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodDelete, "/alerts/cleanup", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			handler.CleanupOldResolvedAlerts(c)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}