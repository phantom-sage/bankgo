package integration

import (
	"context"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/phantom-sage/bankgo/internal/admin/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminSystemHealthMonitoring_Integration(t *testing.T) {
	// Skip integration test if running in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test the system monitoring service directly (without HTTP layer)
	service := services.NewSystemMonitoringService(nil, nil, "http://localhost:8080")
	
	t.Run("SystemHealthCollection", func(t *testing.T) {
		ctx := context.Background()
		
		// Test getting system health
		health, err := service.GetSystemHealth(ctx)
		require.NoError(t, err)
		require.NotNil(t, health)

		// Verify health structure
		assert.NotEmpty(t, health.Status)
		assert.NotZero(t, health.Timestamp)
		assert.NotNil(t, health.Services)
		assert.GreaterOrEqual(t, health.AlertCount, 0)

		// Verify services are checked
		assert.Contains(t, health.Services, "database")
		assert.Contains(t, health.Services, "redis")
		assert.Contains(t, health.Services, "banking_api")

		// Verify metrics are present
		assert.GreaterOrEqual(t, health.Metrics.CPUUsage, 0.0)
		assert.GreaterOrEqual(t, health.Metrics.MemoryUsage, 0.0)
	})

	t.Run("SystemMetricsRetrieval", func(t *testing.T) {
		ctx := context.Background()
		
		// Define time range
		now := time.Now()
		timeRange := interfaces.TimeRange{
			Start: now.Add(-time.Hour),
			End:   now,
		}
		
		// Test getting metrics
		metrics, err := service.GetMetrics(ctx, timeRange)
		require.NoError(t, err)
		require.NotNil(t, metrics)

		// Verify metrics structure
		assert.Greater(t, metrics.Interval, time.Duration(0))
		assert.NotNil(t, metrics.DataPoints)
		assert.True(t, metrics.TimeRange.End.After(metrics.TimeRange.Start) || metrics.TimeRange.End.Equal(metrics.TimeRange.Start))
	})

	t.Run("SystemAlertsManagement", func(t *testing.T) {
		ctx := context.Background()
		
		// Test getting alerts
		params := interfaces.AlertParams{
			PaginationParams: interfaces.PaginationParams{Page: 1, PageSize: 20},
		}
		
		alerts, err := service.GetAlerts(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, alerts)

		// Verify pagination structure
		assert.Equal(t, 1, alerts.Pagination.Page)
		assert.Equal(t, 20, alerts.Pagination.PageSize)
		assert.GreaterOrEqual(t, alerts.Pagination.TotalItems, 0)
		assert.NotNil(t, alerts.Alerts)
	})

	t.Run("AlertFiltering", func(t *testing.T) {
		ctx := context.Background()
		
		// Test filtering by severity
		params := interfaces.AlertParams{
			PaginationParams: interfaces.PaginationParams{Page: 1, PageSize: 20},
			Severity:         "critical",
		}
		
		alerts, err := service.GetAlerts(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, alerts)

		// Verify filtering works
		assert.NotNil(t, alerts.Alerts)
		assert.GreaterOrEqual(t, alerts.Pagination.TotalItems, 0)
	})
}

func TestAdminSystemHealthMonitoring_ServiceHealthChecks(t *testing.T) {
	// Test individual service health checks
	service := services.NewSystemMonitoringService(nil, nil, "http://localhost:8080")
	serviceImpl := service.(*services.SystemMonitoringServiceImpl)

	t.Run("DatabaseHealthCheck", func(t *testing.T) {
		// Test database health check with nil connection
		health := serviceImpl.CheckDatabaseHealth(context.Background())
		
		// Should return critical status for nil connection
		assert.Equal(t, "critical", health.Status)
		assert.Contains(t, health.Error, "database connection not initialized")
	})

	t.Run("RedisHealthCheck", func(t *testing.T) {
		// Test Redis health check with nil connection
		health := serviceImpl.CheckRedisHealth(context.Background())
		
		// Should return warning status for nil connection (Redis is not critical)
		assert.Equal(t, "warning", health.Status)
		assert.Contains(t, health.Error, "redis connection not initialized")
	})

	t.Run("BankingAPIHealthCheck", func(t *testing.T) {
		// Test banking API health check
		health := serviceImpl.CheckBankingAPIHealth(context.Background())
		
		// Should return healthy status (mocked)
		assert.Equal(t, "healthy", health.Status)
		assert.Empty(t, health.Error)
	})
}

func TestAdminSystemHealthMonitoring_AlertLifecycle(t *testing.T) {
	// Test complete alert lifecycle
	service := services.NewSystemMonitoringService(nil, nil, "http://localhost:8080")
	serviceImpl := service.(*services.SystemMonitoringServiceImpl)

	// Generate some test alerts by triggering high metrics
	highCPUMetrics := interfaces.SystemMetricsSnapshot{
		CPUUsage:        95.0, // Above critical threshold
		MemoryUsage:     50.0,
		DBConnections:   10,
		APIResponseTime: 100.0,
		ActiveSessions:  5,
	}

	// Trigger alert generation
	serviceImpl.CheckMetricsForAlerts(highCPUMetrics)

	t.Run("AlertGeneration", func(t *testing.T) {
		// Verify alert was generated
		alerts, err := service.GetAlerts(context.Background(), interfaces.AlertParams{
			PaginationParams: interfaces.PaginationParams{Page: 1, PageSize: 10},
		})
		require.NoError(t, err)
		assert.Greater(t, alerts.Pagination.TotalItems, 0)
	})

	t.Run("AlertAcknowledgment", func(t *testing.T) {
		// Get first alert
		alerts, err := service.GetAlerts(context.Background(), interfaces.AlertParams{
			PaginationParams: interfaces.PaginationParams{Page: 1, PageSize: 1},
		})
		require.NoError(t, err)
		require.Greater(t, len(alerts.Alerts), 0)

		alertID := alerts.Alerts[0].ID

		// Acknowledge the alert
		err = service.AcknowledgeAlert(context.Background(), alertID)
		require.NoError(t, err)

		// Verify alert is acknowledged
		acknowledgedAlerts, err := service.GetAlerts(context.Background(), interfaces.AlertParams{
			PaginationParams: interfaces.PaginationParams{Page: 1, PageSize: 10},
			Acknowledged:     &[]bool{true}[0],
		})
		require.NoError(t, err)
		assert.Greater(t, acknowledgedAlerts.Pagination.TotalItems, 0)
	})

	t.Run("AlertResolution", func(t *testing.T) {
		// Get first alert
		alerts, err := service.GetAlerts(context.Background(), interfaces.AlertParams{
			PaginationParams: interfaces.PaginationParams{Page: 1, PageSize: 1},
		})
		require.NoError(t, err)
		require.Greater(t, len(alerts.Alerts), 0)

		alertID := alerts.Alerts[0].ID

		// Resolve the alert
		notes := "Resolved during testing"
		err = service.ResolveAlert(context.Background(), alertID, notes)
		require.NoError(t, err)

		// Verify alert is resolved
		resolvedAlerts, err := service.GetAlerts(context.Background(), interfaces.AlertParams{
			PaginationParams: interfaces.PaginationParams{Page: 1, PageSize: 10},
			Resolved:         &[]bool{true}[0],
		})
		require.NoError(t, err)
		assert.Greater(t, resolvedAlerts.Pagination.TotalItems, 0)
	})
}

func TestAdminSystemHealthMonitoring_MetricsHistoryStorage(t *testing.T) {
	// Test metrics history storage and retrieval
	service := services.NewSystemMonitoringService(nil, nil, "http://localhost:8080")
	serviceImpl := service.(*services.SystemMonitoringServiceImpl)

	// Simulate metrics collection over time
	testMetrics := []interfaces.SystemMetricsSnapshot{
		{CPUUsage: 45.0, MemoryUsage: 60.0, DBConnections: 10, APIResponseTime: 120.0, ActiveSessions: 5},
		{CPUUsage: 50.0, MemoryUsage: 65.0, DBConnections: 12, APIResponseTime: 130.0, ActiveSessions: 6},
		{CPUUsage: 55.0, MemoryUsage: 70.0, DBConnections: 14, APIResponseTime: 140.0, ActiveSessions: 7},
	}

	// Add metrics to history
	serviceImpl.AddMetricsToHistory(testMetrics...)

	// Test retrieving metrics
	now := time.Now()
	timeRange := interfaces.TimeRange{
		Start: now.Add(-time.Hour),
		End:   now,
	}

	metrics, err := service.GetMetrics(context.Background(), timeRange)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	// Verify metrics were stored and retrieved
	assert.GreaterOrEqual(t, len(metrics.DataPoints), len(testMetrics))
	assert.Equal(t, timeRange, metrics.TimeRange)
	assert.Greater(t, metrics.Interval, time.Duration(0))
}

