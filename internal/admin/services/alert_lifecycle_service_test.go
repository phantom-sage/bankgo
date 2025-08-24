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

func TestAlertLifecycleService_GetAlertHistory(t *testing.T) {
	mockAlertService := &MockAlertService{}
	lifecycle := NewAlertLifecycleService(mockAlertService)
	ctx := context.Background()

	tests := []struct {
		name   string
		params AlertHistoryParams
		setup  func(*MockAlertService)
	}{
		{
			name: "Get all alerts including resolved",
			params: AlertHistoryParams{
				AlertParams: interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     1,
						PageSize: 20,
					},
				},
				IncludeResolved: true,
			},
			setup: func(m *MockAlertService) {
				expectedParams := interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     1,
						PageSize: 20,
					},
				}
				m.On("ListAlerts", ctx, expectedParams).Return(&interfaces.PaginatedAlerts{
					Alerts: []interfaces.Alert{
						{ID: "alert-1", Severity: "critical", Resolved: false},
						{ID: "alert-2", Severity: "warning", Resolved: true},
					},
					Pagination: interfaces.PaginationInfo{Page: 1, PageSize: 20, TotalItems: 2},
				}, nil)
			},
		},
		{
			name: "Get only unresolved alerts",
			params: AlertHistoryParams{
				AlertParams: interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     1,
						PageSize: 20,
					},
				},
				IncludeResolved: false,
			},
			setup: func(m *MockAlertService) {
				resolved := false
				expectedParams := interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     1,
						PageSize: 20,
					},
					Resolved: &resolved,
				}
				m.On("ListAlerts", ctx, expectedParams).Return(&interfaces.PaginatedAlerts{
					Alerts: []interfaces.Alert{
						{ID: "alert-1", Severity: "critical", Resolved: false},
					},
					Pagination: interfaces.PaginationInfo{Page: 1, PageSize: 20, TotalItems: 1},
				}, nil)
			},
		},
		{
			name: "Filter by minimum severity",
			params: AlertHistoryParams{
				AlertParams: interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     1,
						PageSize: 20,
					},
				},
				MinSeverity: "critical",
			},
			setup: func(m *MockAlertService) {
				resolved := false
				expectedParams := interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     1,
						PageSize: 20,
					},
					Severity: "critical",
					Resolved: &resolved,
				}
				m.On("ListAlerts", ctx, expectedParams).Return(&interfaces.PaginatedAlerts{
					Alerts: []interfaces.Alert{
						{ID: "alert-1", Severity: "critical", Resolved: false},
					},
					Pagination: interfaces.PaginationInfo{Page: 1, PageSize: 20, TotalItems: 1},
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(mockAlertService)

			result, err := lifecycle.GetAlertHistory(ctx, tt.params)
			require.NoError(t, err)
			require.NotNil(t, result)

			mockAlertService.AssertExpectations(t)
		})
	}
}

func TestAlertLifecycleService_GetAlertSummary(t *testing.T) {
	mockAlertService := &MockAlertService{}
	lifecycle := NewAlertLifecycleService(mockAlertService)
	ctx := context.Background()

	timeRange := &interfaces.TimeRange{
		Start: time.Now().Add(-24 * time.Hour),
		End:   time.Now(),
	}

	// Mock main statistics
	mockAlertService.On("GetAlertStatistics", ctx, timeRange).Return(&interfaces.AlertStatistics{
		TotalAlerts:       100,
		CriticalCount:     20,
		WarningCount:      50,
		InfoCount:         30,
		AcknowledgedCount: 80,
		ResolvedCount:     70,
		UnresolvedCount:   30,
	}, nil)

	// Mock recent statistics (last 24 hours)
	mockAlertService.On("GetAlertStatistics", ctx, mock.AnythingOfType("*interfaces.TimeRange")).Return(&interfaces.AlertStatistics{
		TotalAlerts: 15,
	}, nil)

	summary, err := lifecycle.GetAlertSummary(ctx, timeRange)
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Equal(t, 100, summary.TotalAlerts)
	assert.Equal(t, 30, summary.UnresolvedAlerts)
	assert.Equal(t, 20, summary.CriticalAlerts)
	assert.Equal(t, 15, summary.RecentAlerts)
	assert.NotEmpty(t, summary.TopSources)
	assert.NotEmpty(t, summary.SeverityBreakdown)
	assert.NotEmpty(t, summary.TrendData)

	mockAlertService.AssertExpectations(t)
}

func TestAlertLifecycleService_GetAlertsByPattern(t *testing.T) {
	mockAlertService := &MockAlertService{}
	lifecycle := NewAlertLifecycleService(mockAlertService)
	ctx := context.Background()

	pattern := "database"
	params := interfaces.AlertParams{
		PaginationParams: interfaces.PaginationParams{
			Page:     1,
			PageSize: 10,
		},
	}

	expectedResult := &interfaces.PaginatedAlerts{
		Alerts: []interfaces.Alert{
			{ID: "alert-1", Title: "Database Connection Failed", Message: "Database is unreachable"},
		},
		Pagination: interfaces.PaginationInfo{Page: 1, PageSize: 10, TotalItems: 1},
	}

	mockAlertService.On("SearchAlerts", ctx, pattern, params).Return(expectedResult, nil)

	result, err := lifecycle.GetAlertsByPattern(ctx, pattern, params)
	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)

	mockAlertService.AssertExpectations(t)
}

func TestAlertLifecycleService_GetRelatedAlerts(t *testing.T) {
	mockAlertService := &MockAlertService{}
	lifecycle := NewAlertLifecycleService(mockAlertService)
	ctx := context.Background()

	alertID := "alert-1"
	maxResults := 5

	originalAlert := &interfaces.Alert{
		ID:       alertID,
		Severity: "critical",
		Title:    "Database Connection Failed",
		Source:   "database_monitor",
	}

	relatedAlerts := []interfaces.Alert{
		{ID: "alert-1", Source: "database_monitor", Title: "Original Alert"},
		{ID: "alert-2", Source: "database_monitor", Title: "Related Alert 1"},
		{ID: "alert-3", Source: "database_monitor", Title: "Related Alert 2"},
	}

	mockAlertService.On("GetAlert", ctx, alertID).Return(originalAlert, nil)
	mockAlertService.On("GetAlertsBySource", ctx, "database_monitor", maxResults).Return(relatedAlerts, nil)

	result, err := lifecycle.GetRelatedAlerts(ctx, alertID, maxResults)
	require.NoError(t, err)
	
	// Should exclude the original alert
	assert.Len(t, result, 2)
	for _, alert := range result {
		assert.NotEqual(t, alertID, alert.ID)
	}

	mockAlertService.AssertExpectations(t)
}

func TestAlertLifecycleService_BulkAcknowledgeAlerts(t *testing.T) {
	mockAlertService := &MockAlertService{}
	lifecycle := NewAlertLifecycleService(mockAlertService)
	ctx := context.Background()

	source := "system_monitor"
	acknowledgedBy := "admin"

	alerts := []interfaces.Alert{
		{ID: "alert-1", Source: source, Acknowledged: false, Resolved: false},
		{ID: "alert-2", Source: source, Acknowledged: false, Resolved: false},
		{ID: "alert-3", Source: source, Acknowledged: true, Resolved: false},  // Already acknowledged
		{ID: "alert-4", Source: source, Acknowledged: false, Resolved: true}, // Already resolved
	}

	mockAlertService.On("GetAlertsBySource", ctx, source, 100).Return(alerts, nil)

	// Expect acknowledgment calls only for unacknowledged, unresolved alerts
	for _, alert := range alerts {
		if !alert.Acknowledged && !alert.Resolved {
			acknowledgedAlert := alert
			acknowledgedAlert.Acknowledged = true
			acknowledgedAlert.AcknowledgedBy = acknowledgedBy
			mockAlertService.On("AcknowledgeAlert", ctx, alert.ID, acknowledgedBy).Return(&acknowledgedAlert, nil)
		}
	}

	count, err := lifecycle.BulkAcknowledgeAlerts(ctx, source, acknowledgedBy)
	require.NoError(t, err)
	assert.Equal(t, 2, count) // Only 2 alerts should be acknowledged

	mockAlertService.AssertExpectations(t)
}

func TestAlertLifecycleService_GetAlertEscalationCandidates(t *testing.T) {
	mockAlertService := &MockAlertService{}
	lifecycle := NewAlertLifecycleService(mockAlertService)
	ctx := context.Background()

	maxAge := 2 * time.Hour

	alerts := []interfaces.Alert{
		{ID: "alert-1", Severity: "critical", Acknowledged: false, Resolved: false}, // Should be included
		{ID: "alert-2", Severity: "warning", Acknowledged: false, Resolved: false},  // Should be included (unacknowledged)
		{ID: "alert-3", Severity: "info", Acknowledged: true, Resolved: false},      // Should not be included
		{ID: "alert-4", Severity: "critical", Acknowledged: true, Resolved: false}, // Should be included (critical)
	}

	mockAlertService.On("ListAlerts", ctx, mock.AnythingOfType("interfaces.AlertParams")).Return(&interfaces.PaginatedAlerts{
		Alerts: alerts,
	}, nil)

	candidates, err := lifecycle.GetAlertEscalationCandidates(ctx, maxAge)
	require.NoError(t, err)
	
	// Should include critical alerts and unacknowledged alerts
	assert.Len(t, candidates, 3)
	
	// Verify the correct alerts are included
	candidateIDs := make(map[string]bool)
	for _, candidate := range candidates {
		candidateIDs[candidate.ID] = true
	}
	
	assert.True(t, candidateIDs["alert-1"]) // Critical and unacknowledged
	assert.True(t, candidateIDs["alert-2"]) // Unacknowledged
	assert.True(t, candidateIDs["alert-4"]) // Critical
	assert.False(t, candidateIDs["alert-3"]) // Info and acknowledged

	mockAlertService.AssertExpectations(t)
}

func TestAlertLifecycleService_GetAlertMetrics(t *testing.T) {
	mockAlertService := &MockAlertService{}
	lifecycle := NewAlertLifecycleService(mockAlertService)
	ctx := context.Background()

	timeRange := &interfaces.TimeRange{
		Start: time.Now().Add(-24 * time.Hour),
		End:   time.Now(),
	}

	stats := &interfaces.AlertStatistics{
		TotalAlerts:       100,
		CriticalCount:     20,
		WarningCount:      50,
		InfoCount:         30,
		AcknowledgedCount: 80,
		ResolvedCount:     70,
		UnresolvedCount:   30,
	}

	mockAlertService.On("GetAlertStatistics", ctx, timeRange).Return(stats, nil)

	metrics, err := lifecycle.GetAlertMetrics(ctx, timeRange)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	assert.Equal(t, 100, metrics["total_alerts"])
	assert.Equal(t, 30, metrics["unresolved_alerts"])
	assert.Equal(t, 70.0, metrics["resolution_rate"])
	assert.Equal(t, 80.0, metrics["acknowledgment_rate"])
	assert.Equal(t, 20.0, metrics["critical_percentage"])
	assert.Equal(t, 50.0, metrics["warning_percentage"])
	assert.Equal(t, 30.0, metrics["info_percentage"])
	assert.Contains(t, metrics, "avg_resolution_time_hours")

	mockAlertService.AssertExpectations(t)
}

func TestAlertLifecycleService_ValidateAlertParams(t *testing.T) {
	lifecycle := NewAlertLifecycleService(nil)

	tests := []struct {
		name    string
		params  AlertHistoryParams
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid parameters",
			params: AlertHistoryParams{
				AlertParams: interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     1,
						PageSize: 20,
					},
				},
				MinSeverity: "warning",
				MaxAge:      func() *time.Duration { d := 24 * time.Hour; return &d }(),
			},
			wantErr: false,
		},
		{
			name: "Invalid minimum severity",
			params: AlertHistoryParams{
				AlertParams: interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     1,
						PageSize: 20,
					},
				},
				MinSeverity: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid minimum severity",
		},
		{
			name: "Negative max age",
			params: AlertHistoryParams{
				AlertParams: interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     1,
						PageSize: 20,
					},
				},
				MaxAge: func() *time.Duration { d := -1 * time.Hour; return &d }(),
			},
			wantErr: true,
			errMsg:  "max age cannot be negative",
		},
		{
			name: "Invalid page number",
			params: AlertHistoryParams{
				AlertParams: interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     0,
						PageSize: 20,
					},
				},
			},
			wantErr: true,
			errMsg:  "page must be >= 1",
		},
		{
			name: "Invalid page size - too small",
			params: AlertHistoryParams{
				AlertParams: interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     1,
						PageSize: 0,
					},
				},
			},
			wantErr: true,
			errMsg:  "page size must be between 1 and 1000",
		},
		{
			name: "Invalid page size - too large",
			params: AlertHistoryParams{
				AlertParams: interfaces.AlertParams{
					PaginationParams: interfaces.PaginationParams{
						Page:     1,
						PageSize: 1001,
					},
				},
			},
			wantErr: true,
			errMsg:  "page size must be between 1 and 1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lifecycle.ValidateAlertParams(tt.params)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAlertLifecycleService_ErrorHandling(t *testing.T) {
	mockAlertService := &MockAlertService{}
	lifecycle := NewAlertLifecycleService(mockAlertService)
	ctx := context.Background()

	t.Run("GetAlertHistory error", func(t *testing.T) {
		params := AlertHistoryParams{
			AlertParams: interfaces.AlertParams{
				PaginationParams: interfaces.PaginationParams{
					Page:     1,
					PageSize: 20,
				},
			},
		}

		resolved := false
		expectedParams := interfaces.AlertParams{
			PaginationParams: interfaces.PaginationParams{
				Page:     1,
				PageSize: 20,
			},
			Resolved: &resolved,
		}

		mockAlertService.On("ListAlerts", ctx, expectedParams).Return(nil, errors.New("database error"))

		result, err := lifecycle.GetAlertHistory(ctx, params)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("GetRelatedAlerts - original alert not found", func(t *testing.T) {
		alertID := "non-existent"
		mockAlertService.On("GetAlert", ctx, alertID).Return(nil, errors.New("alert not found"))

		result, err := lifecycle.GetRelatedAlerts(ctx, alertID, 5)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get original alert")
	})

	mockAlertService.AssertExpectations(t)
}