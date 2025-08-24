package services

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// SystemMonitoringServiceImpl implements the SystemMonitoringService interface
type SystemMonitoringServiceImpl struct {
	db          *pgxpool.Pool
	redis       *redis.Client
	bankingAPI  string
	alertService interfaces.AlertService
	
	// Metrics storage
	metricsHistory []interfaces.SystemMetricsSnapshot
	historyMutex   sync.RWMutex
	maxHistorySize int
}

// NewSystemMonitoringService creates a new system monitoring service
func NewSystemMonitoringService(db *pgxpool.Pool, redis *redis.Client, bankingAPIURL string, alertService interfaces.AlertService) interfaces.SystemMonitoringService {
	service := &SystemMonitoringServiceImpl{
		db:             db,
		redis:          redis,
		bankingAPI:     bankingAPIURL,
		alertService:   alertService,
		metricsHistory: make([]interfaces.SystemMetricsSnapshot, 0),
		maxHistorySize: 1000, // Keep last 1000 metrics snapshots
	}
	
	// Start background metrics collection
	go service.startMetricsCollection()
	
	return service
}

// GetSystemHealth returns current system health status
func (s *SystemMonitoringServiceImpl) GetSystemHealth(ctx context.Context) (*interfaces.SystemHealth, error) {
	// Collect current metrics
	metrics, err := s.collectCurrentMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}
	
	// Check service health
	services := s.checkServiceHealth(ctx)
	
	// Determine overall status
	status := s.determineOverallStatus(services, metrics)
	
	// Count unresolved alerts
	alertCount, err := s.alertService.GetUnresolvedAlertsCount(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get unresolved alerts count")
		alertCount = 0
	}
	
	health := &interfaces.SystemHealth{
		Status:     status,
		Timestamp:  time.Now(),
		Services:   services,
		Metrics:    *metrics,
		AlertCount: alertCount,
	}
	
	return health, nil
}

// GetMetrics returns system performance metrics for a time range
func (s *SystemMonitoringServiceImpl) GetMetrics(ctx context.Context, timeRange interfaces.TimeRange) (*interfaces.SystemMetrics, error) {
	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()
	
	// Filter metrics by time range
	filteredMetrics := make([]interfaces.SystemMetricsSnapshot, 0)
	for _, metric := range s.metricsHistory {
		// Note: We'll need to add timestamp to SystemMetricsSnapshot
		// For now, we'll return recent metrics
		filteredMetrics = append(filteredMetrics, metric)
	}
	
	// Calculate interval based on time range
	interval := time.Minute // Default interval
	if len(filteredMetrics) > 0 {
		interval = timeRange.End.Sub(timeRange.Start) / time.Duration(len(filteredMetrics))
		if interval < time.Minute {
			interval = time.Minute
		}
	}
	
	metrics := &interfaces.SystemMetrics{
		TimeRange:  timeRange,
		Interval:   interval,
		DataPoints: filteredMetrics,
	}
	
	return metrics, nil
}

// GetAlerts returns system alerts with pagination and filtering
func (s *SystemMonitoringServiceImpl) GetAlerts(ctx context.Context, params interfaces.AlertParams) (*interfaces.PaginatedAlerts, error) {
	return s.alertService.ListAlerts(ctx, params)
}

// AcknowledgeAlert marks an alert as acknowledged
func (s *SystemMonitoringServiceImpl) AcknowledgeAlert(ctx context.Context, alertID string) error {
	// TODO: Get admin ID from context
	adminID := "admin"
	_, err := s.alertService.AcknowledgeAlert(ctx, alertID, adminID)
	return err
}

// ResolveAlert marks an alert as resolved
func (s *SystemMonitoringServiceImpl) ResolveAlert(ctx context.Context, alertID string, notes string) error {
	// TODO: Get admin ID from context
	adminID := "admin"
	_, err := s.alertService.ResolveAlert(ctx, alertID, adminID, notes)
	return err
}

// collectCurrentMetrics collects current system performance metrics
func (s *SystemMonitoringServiceImpl) collectCurrentMetrics(ctx context.Context) (*interfaces.SystemMetricsSnapshot, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Calculate CPU usage (simplified)
	cpuUsage := s.calculateCPUUsage()
	
	// Calculate memory usage percentage
	memoryUsage := float64(m.Alloc) / float64(m.Sys) * 100
	
	// Get database connection count
	dbConnections := 0
	if s.db != nil {
		stat := s.db.Stat()
		dbConnections = int(stat.AcquiredConns())
	}
	
	// Calculate API response time (simplified - would need actual measurement)
	apiResponseTime := s.calculateAPIResponseTime(ctx)
	
	// Get active sessions count (simplified)
	activeSessions := s.getActiveSessionsCount(ctx)
	
	metrics := &interfaces.SystemMetricsSnapshot{
		CPUUsage:        cpuUsage,
		MemoryUsage:     memoryUsage,
		DBConnections:   dbConnections,
		APIResponseTime: apiResponseTime,
		ActiveSessions:  activeSessions,
	}
	
	return metrics, nil
}

// checkServiceHealth checks the health of individual services
func (s *SystemMonitoringServiceImpl) checkServiceHealth(ctx context.Context) map[string]interfaces.ServiceHealth {
	services := make(map[string]interfaces.ServiceHealth)
	
	// Check database health
	services["database"] = s.checkDatabaseHealth(ctx)
	
	// Check Redis health
	services["redis"] = s.checkRedisHealth(ctx)
	
	// Check banking API health
	services["banking_api"] = s.checkBankingAPIHealth(ctx)
	
	return services
}

// CheckDatabaseHealth checks PostgreSQL database health (exported for testing)
func (s *SystemMonitoringServiceImpl) CheckDatabaseHealth(ctx context.Context) interfaces.ServiceHealth {
	return s.checkDatabaseHealth(ctx)
}

// checkDatabaseHealth checks PostgreSQL database health
func (s *SystemMonitoringServiceImpl) checkDatabaseHealth(ctx context.Context) interfaces.ServiceHealth {
	start := time.Now()
	
	if s.db == nil {
		return interfaces.ServiceHealth{
			Status:       "critical",
			LastCheck:    start,
			ResponseTime: 0,
			Error:        "database connection not initialized",
		}
	}
	
	// Simple ping to check database connectivity
	err := s.db.Ping(ctx)
	responseTime := time.Since(start)
	
	if err != nil {
		return interfaces.ServiceHealth{
			Status:       "critical",
			LastCheck:    start,
			ResponseTime: responseTime,
			Error:        err.Error(),
		}
	}
	
	// Check if response time is acceptable
	status := "healthy"
	if responseTime > 5*time.Second {
		status = "warning"
	}
	
	return interfaces.ServiceHealth{
		Status:       status,
		LastCheck:    start,
		ResponseTime: responseTime,
	}
}

// CheckRedisHealth checks Redis health (exported for testing)
func (s *SystemMonitoringServiceImpl) CheckRedisHealth(ctx context.Context) interfaces.ServiceHealth {
	return s.checkRedisHealth(ctx)
}

// checkRedisHealth checks Redis health
func (s *SystemMonitoringServiceImpl) checkRedisHealth(ctx context.Context) interfaces.ServiceHealth {
	start := time.Now()
	
	if s.redis == nil {
		return interfaces.ServiceHealth{
			Status:       "warning",
			LastCheck:    start,
			ResponseTime: 0,
			Error:        "redis connection not initialized",
		}
	}
	
	// Simple ping to check Redis connectivity
	err := s.redis.Ping(ctx).Err()
	responseTime := time.Since(start)
	
	if err != nil {
		return interfaces.ServiceHealth{
			Status:       "warning",
			LastCheck:    start,
			ResponseTime: responseTime,
			Error:        err.Error(),
		}
	}
	
	// Check if response time is acceptable
	status := "healthy"
	if responseTime > 2*time.Second {
		status = "warning"
	}
	
	return interfaces.ServiceHealth{
		Status:       status,
		LastCheck:    start,
		ResponseTime: responseTime,
	}
}

// CheckBankingAPIHealth checks banking API health (exported for testing)
func (s *SystemMonitoringServiceImpl) CheckBankingAPIHealth(ctx context.Context) interfaces.ServiceHealth {
	return s.checkBankingAPIHealth(ctx)
}

// AddMetricsToHistory adds metrics to history for testing
func (s *SystemMonitoringServiceImpl) AddMetricsToHistory(metrics ...interfaces.SystemMetricsSnapshot) {
	s.historyMutex.Lock()
	defer s.historyMutex.Unlock()
	
	for _, metric := range metrics {
		s.metricsHistory = append(s.metricsHistory, metric)
	}
}

// checkBankingAPIHealth checks banking API health
func (s *SystemMonitoringServiceImpl) checkBankingAPIHealth(ctx context.Context) interfaces.ServiceHealth {
	start := time.Now()
	
	// TODO: Implement actual HTTP health check to banking API
	// For now, return a mock healthy status
	responseTime := 100 * time.Millisecond
	
	return interfaces.ServiceHealth{
		Status:       "healthy",
		LastCheck:    start,
		ResponseTime: responseTime,
	}
}

// determineOverallStatus determines overall system status based on service health
func (s *SystemMonitoringServiceImpl) determineOverallStatus(services map[string]interfaces.ServiceHealth, metrics *interfaces.SystemMetricsSnapshot) string {
	hasCritical := false
	hasWarning := false
	
	// Check service statuses
	for _, service := range services {
		switch service.Status {
		case "critical":
			hasCritical = true
		case "warning":
			hasWarning = true
		}
	}
	
	// Check metrics thresholds
	if metrics.CPUUsage > 90 || metrics.MemoryUsage > 90 {
		hasCritical = true
	} else if metrics.CPUUsage > 70 || metrics.MemoryUsage > 70 {
		hasWarning = true
	}
	
	if hasCritical {
		return "critical"
	}
	if hasWarning {
		return "warning"
	}
	return "healthy"
}

// Helper methods for metrics calculation
func (s *SystemMonitoringServiceImpl) calculateCPUUsage() float64 {
	// Simplified CPU usage calculation
	// In a real implementation, you would use system calls or libraries
	// to get actual CPU usage
	return float64(runtime.NumGoroutine()) / 100.0 * 10 // Mock calculation
}

func (s *SystemMonitoringServiceImpl) calculateAPIResponseTime(ctx context.Context) float64 {
	// TODO: Implement actual API response time measurement
	// This would involve making test requests and measuring response times
	return 150.0 // Mock response time in milliseconds
}

func (s *SystemMonitoringServiceImpl) getActiveSessionsCount(ctx context.Context) int {
	// TODO: Implement actual session counting
	// This would query the session store or cache
	return 5 // Mock active sessions count
}



// startMetricsCollection starts background metrics collection
func (s *SystemMonitoringServiceImpl) startMetricsCollection() {
	ticker := time.NewTicker(30 * time.Second) // Collect metrics every 30 seconds
	defer ticker.Stop()
	
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		
		metrics, err := s.collectCurrentMetrics(ctx)
		if err != nil {
			// Log error but continue
			cancel()
			continue
		}
		
		// Store metrics in history
		s.historyMutex.Lock()
		s.metricsHistory = append(s.metricsHistory, *metrics)
		
		// Keep only the last maxHistorySize entries
		if len(s.metricsHistory) > s.maxHistorySize {
			s.metricsHistory = s.metricsHistory[1:]
		}
		s.historyMutex.Unlock()
		
		// Check for alerts based on metrics
		s.checkMetricsForAlerts(*metrics)
		
		cancel()
	}
}

// CheckMetricsForAlerts generates alerts based on metrics thresholds (exported for testing)
func (s *SystemMonitoringServiceImpl) CheckMetricsForAlerts(metrics interfaces.SystemMetricsSnapshot) {
	s.checkMetricsForAlerts(metrics)
}

// checkMetricsForAlerts generates alerts based on metrics thresholds
func (s *SystemMonitoringServiceImpl) checkMetricsForAlerts(metrics interfaces.SystemMetricsSnapshot) {
	ctx := context.Background()
	
	// Check CPU usage
	if metrics.CPUUsage > 90 {
		metadata := map[string]interface{}{
			"cpu_usage":  metrics.CPUUsage,
			"threshold":  90.0,
			"metric_type": "cpu",
		}
		_, err := s.alertService.CreateAlert(
			ctx,
			"critical",
			"High CPU Usage",
			fmt.Sprintf("CPU usage is %.1f%%, which exceeds the critical threshold of 90%%", metrics.CPUUsage),
			"system_monitor",
			metadata,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create CPU high usage alert")
		}
	} else if metrics.CPUUsage > 70 {
		metadata := map[string]interface{}{
			"cpu_usage":  metrics.CPUUsage,
			"threshold":  70.0,
			"metric_type": "cpu",
		}
		_, err := s.alertService.CreateAlert(
			ctx,
			"warning",
			"Elevated CPU Usage",
			fmt.Sprintf("CPU usage is %.1f%%, which exceeds the warning threshold of 70%%", metrics.CPUUsage),
			"system_monitor",
			metadata,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create CPU elevated usage alert")
		}
	}
	
	// Check memory usage
	if metrics.MemoryUsage > 90 {
		metadata := map[string]interface{}{
			"memory_usage": metrics.MemoryUsage,
			"threshold":    90.0,
			"metric_type":  "memory",
		}
		_, err := s.alertService.CreateAlert(
			ctx,
			"critical",
			"High Memory Usage",
			fmt.Sprintf("Memory usage is %.1f%%, which exceeds the critical threshold of 90%%", metrics.MemoryUsage),
			"system_monitor",
			metadata,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create memory high usage alert")
		}
	} else if metrics.MemoryUsage > 70 {
		metadata := map[string]interface{}{
			"memory_usage": metrics.MemoryUsage,
			"threshold":    70.0,
			"metric_type":  "memory",
		}
		_, err := s.alertService.CreateAlert(
			ctx,
			"warning",
			"Elevated Memory Usage",
			fmt.Sprintf("Memory usage is %.1f%%, which exceeds the warning threshold of 70%%", metrics.MemoryUsage),
			"system_monitor",
			metadata,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create memory elevated usage alert")
		}
	}
	
	// Check API response time
	if metrics.APIResponseTime > 5000 { // 5 seconds
		metadata := map[string]interface{}{
			"response_time": metrics.APIResponseTime,
			"threshold":     5000.0,
			"metric_type":   "api_response_time",
		}
		_, err := s.alertService.CreateAlert(
			ctx,
			"warning",
			"Slow API Response Time",
			fmt.Sprintf("API response time is %.1fms, which exceeds the threshold of 5000ms", metrics.APIResponseTime),
			"system_monitor",
			metadata,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create API slow response alert")
		}
	}
}