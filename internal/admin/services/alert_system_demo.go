package services

import (
	"context"
	"fmt"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/rs/zerolog/log"
)

// AlertSystemDemo demonstrates the complete alert system functionality
// This is for demonstration and testing purposes only
type AlertSystemDemo struct {
	alertService     interfaces.AlertService
	generator        *AlertGeneratorService
	lifecycle        *AlertLifecycleService
	systemMonitoring interfaces.SystemMonitoringService
}

// NewAlertSystemDemo creates a new alert system demonstration
func NewAlertSystemDemo(
	alertService interfaces.AlertService,
	systemMonitoring interfaces.SystemMonitoringService,
) *AlertSystemDemo {
	return &AlertSystemDemo{
		alertService:     alertService,
		generator:        NewAlertGeneratorService(alertService),
		lifecycle:        NewAlertLifecycleService(alertService),
		systemMonitoring: systemMonitoring,
	}
}

// DemonstrateAlertLifecycle shows a complete alert lifecycle
func (d *AlertSystemDemo) DemonstrateAlertLifecycle(ctx context.Context) error {
	log.Info().Msg("Starting alert system lifecycle demonstration")

	// 1. Generate various types of alerts
	log.Info().Msg("Step 1: Generating different types of alerts")
	
	// Database connection alert
	err := d.generator.DatabaseConnectionAlert(ctx, fmt.Errorf("connection timeout"), "primary")
	if err != nil {
		return fmt.Errorf("failed to create database alert: %w", err)
	}

	// Authentication failure alert
	err = d.generator.AuthenticationFailureAlert(ctx, "suspicious_user", "192.168.1.100", 3)
	if err != nil {
		return fmt.Errorf("failed to create auth alert: %w", err)
	}

	// System resource alert
	err = d.generator.SystemResourceAlert(ctx, "CPU", 85.5, 80.0, "%")
	if err != nil {
		return fmt.Errorf("failed to create resource alert: %w", err)
	}

	// Security alert
	err = d.generator.SecurityAlert(ctx, "sql_injection", "Malicious SQL detected in request", "10.0.0.1", "curl/7.68.0")
	if err != nil {
		return fmt.Errorf("failed to create security alert: %w", err)
	}

	// Performance alert
	err = d.generator.PerformanceAlert(ctx, "Database", "response_time", 2500.0, 1000.0, "ms")
	if err != nil {
		return fmt.Errorf("failed to create performance alert: %w", err)
	}

	// 2. List all alerts
	log.Info().Msg("Step 2: Listing all alerts")
	params := interfaces.AlertParams{
		PaginationParams: interfaces.PaginationParams{
			Page:     1,
			PageSize: 10,
		},
	}

	alerts, err := d.alertService.ListAlerts(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list alerts: %w", err)
	}

	log.Info().
		Int("total_alerts", len(alerts.Alerts)).
		Int("total_count", alerts.Pagination.TotalItems).
		Msg("Retrieved alerts")

	// 3. Search for specific alerts
	log.Info().Msg("Step 3: Searching for database-related alerts")
	searchResults, err := d.alertService.SearchAlerts(ctx, "database", params)
	if err != nil {
		return fmt.Errorf("failed to search alerts: %w", err)
	}

	log.Info().
		Int("search_results", len(searchResults.Alerts)).
		Msg("Found database-related alerts")

	// 4. Acknowledge some alerts
	log.Info().Msg("Step 4: Acknowledging alerts")
	if len(alerts.Alerts) > 0 {
		for i, alert := range alerts.Alerts {
			if i >= 2 { // Acknowledge first 2 alerts
				break
			}
			_, err := d.alertService.AcknowledgeAlert(ctx, alert.ID, "demo_admin")
			if err != nil {
				log.Warn().
					Err(err).
					Str("alert_id", alert.ID).
					Msg("Failed to acknowledge alert")
				continue
			}
			log.Info().
				Str("alert_id", alert.ID).
				Str("title", alert.Title).
				Msg("Alert acknowledged")
		}
	}

	// 5. Resolve some alerts
	log.Info().Msg("Step 5: Resolving alerts")
	if len(alerts.Alerts) > 0 {
		alert := alerts.Alerts[0]
		_, err := d.alertService.ResolveAlert(ctx, alert.ID, "demo_admin", "Issue resolved during demonstration")
		if err != nil {
			log.Warn().
				Err(err).
				Str("alert_id", alert.ID).
				Msg("Failed to resolve alert")
		} else {
			log.Info().
				Str("alert_id", alert.ID).
				Str("title", alert.Title).
				Msg("Alert resolved")
		}
	}

	// 6. Get alert statistics
	log.Info().Msg("Step 6: Getting alert statistics")
	stats, err := d.alertService.GetAlertStatistics(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get statistics: %w", err)
	}

	log.Info().
		Int("total_alerts", stats.TotalAlerts).
		Int("critical_count", stats.CriticalCount).
		Int("warning_count", stats.WarningCount).
		Int("info_count", stats.InfoCount).
		Int("acknowledged_count", stats.AcknowledgedCount).
		Int("resolved_count", stats.ResolvedCount).
		Int("unresolved_count", stats.UnresolvedCount).
		Msg("Alert statistics")

	// 7. Demonstrate lifecycle service features
	log.Info().Msg("Step 7: Demonstrating lifecycle service features")
	
	// Get alert summary
	summary, err := d.lifecycle.GetAlertSummary(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get alert summary: %w", err)
	}

	log.Info().
		Int("total_alerts", summary.TotalAlerts).
		Int("unresolved_alerts", summary.UnresolvedAlerts).
		Int("critical_alerts", summary.CriticalAlerts).
		Int("recent_alerts", summary.RecentAlerts).
		Int("top_sources_count", len(summary.TopSources)).
		Msg("Alert summary")

	// Get escalation candidates
	escalationCandidates, err := d.lifecycle.GetAlertEscalationCandidates(ctx, 1*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to get escalation candidates: %w", err)
	}

	log.Info().
		Int("escalation_candidates", len(escalationCandidates)).
		Msg("Alerts requiring escalation")

	// Get alert metrics
	metrics, err := d.lifecycle.GetAlertMetrics(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get alert metrics: %w", err)
	}

	log.Info().
		Interface("metrics", metrics).
		Msg("Alert metrics")

	// 8. Demonstrate bulk operations
	log.Info().Msg("Step 8: Demonstrating bulk operations")
	
	// Bulk acknowledge alerts from system_monitor source
	acknowledgedCount, err := d.lifecycle.BulkAcknowledgeAlerts(ctx, "system_monitor", "demo_admin")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to bulk acknowledge alerts")
	} else {
		log.Info().
			Int("acknowledged_count", acknowledgedCount).
			Msg("Bulk acknowledgment completed")
	}

	// Bulk resolve alerts from a specific source
	err = d.generator.BulkAlertResolution(ctx, "performance_monitor", "demo_admin", "Performance issues resolved")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to bulk resolve alerts")
	} else {
		log.Info().Msg("Bulk resolution completed")
	}

	log.Info().Msg("Alert system lifecycle demonstration completed successfully")
	return nil
}

// DemonstrateSystemIntegration shows how alerts integrate with system monitoring
func (d *AlertSystemDemo) DemonstrateSystemIntegration(ctx context.Context) error {
	log.Info().Msg("Starting system integration demonstration")

	// 1. Get system health (which may generate alerts)
	health, err := d.systemMonitoring.GetSystemHealth(ctx)
	if err != nil {
		return fmt.Errorf("failed to get system health: %w", err)
	}

	log.Info().
		Str("status", health.Status).
		Int("alert_count", health.AlertCount).
		Float64("cpu_usage", health.Metrics.CPUUsage).
		Float64("memory_usage", health.Metrics.MemoryUsage).
		Msg("System health status")

	// 2. Simulate system monitoring generating alerts
	log.Info().Msg("Simulating system monitoring alerts")
	
	// Simulate high CPU usage
	if systemMonitor, ok := d.systemMonitoring.(*SystemMonitoringServiceImpl); ok {
		metrics := interfaces.SystemMetricsSnapshot{
			CPUUsage:        95.0, // High CPU usage
			MemoryUsage:     85.0, // High memory usage
			DBConnections:   50,
			APIResponseTime: 2500.0, // Slow API response
			ActiveSessions:  100,
		}
		
		// This would normally be called by the background metrics collection
		systemMonitor.CheckMetricsForAlerts(metrics)
		
		log.Info().Msg("System monitoring alerts generated based on metrics")
	}

	// 3. Get alerts generated by system monitoring
	monitoringAlerts, err := d.alertService.GetAlertsBySource(ctx, "system_monitor", 10)
	if err != nil {
		return fmt.Errorf("failed to get monitoring alerts: %w", err)
	}

	log.Info().
		Int("monitoring_alerts", len(monitoringAlerts)).
		Msg("Alerts generated by system monitoring")

	// 4. Demonstrate alert categorization
	log.Info().Msg("Demonstrating alert categorization")
	
	criticalParams := interfaces.AlertParams{
		PaginationParams: interfaces.PaginationParams{Page: 1, PageSize: 10},
		Severity:         "critical",
	}
	
	criticalAlerts, err := d.alertService.ListAlerts(ctx, criticalParams)
	if err != nil {
		return fmt.Errorf("failed to get critical alerts: %w", err)
	}

	log.Info().
		Int("critical_alerts", len(criticalAlerts.Alerts)).
		Msg("Critical alerts requiring immediate attention")

	log.Info().Msg("System integration demonstration completed successfully")
	return nil
}

// DemonstrateAlertHistory shows alert history and search capabilities
func (d *AlertSystemDemo) DemonstrateAlertHistory(ctx context.Context) error {
	log.Info().Msg("Starting alert history demonstration")

	// 1. Get alert history with different filters
	log.Info().Msg("Getting alert history with various filters")

	// Get all alerts from last 24 hours
	maxAge := 24 * time.Hour
	historyParams := AlertHistoryParams{
		AlertParams: interfaces.AlertParams{
			PaginationParams: interfaces.PaginationParams{Page: 1, PageSize: 20},
		},
		IncludeResolved: true,
		MaxAge:          &maxAge,
	}

	history, err := d.lifecycle.GetAlertHistory(ctx, historyParams)
	if err != nil {
		return fmt.Errorf("failed to get alert history: %w", err)
	}

	log.Info().
		Int("history_count", len(history.Alerts)).
		Msg("Alert history retrieved")

	// 2. Get only unresolved alerts
	historyParams.IncludeResolved = false
	unresolvedHistory, err := d.lifecycle.GetAlertHistory(ctx, historyParams)
	if err != nil {
		return fmt.Errorf("failed to get unresolved history: %w", err)
	}

	log.Info().
		Int("unresolved_count", len(unresolvedHistory.Alerts)).
		Msg("Unresolved alerts in history")

	// 3. Search for patterns
	log.Info().Msg("Searching for alert patterns")
	
	patterns := []string{"database", "timeout", "failed", "high"}
	for _, pattern := range patterns {
		patternResults, err := d.lifecycle.GetAlertsByPattern(ctx, pattern, interfaces.AlertParams{
			PaginationParams: interfaces.PaginationParams{Page: 1, PageSize: 5},
		})
		if err != nil {
			log.Warn().
				Err(err).
				Str("pattern", pattern).
				Msg("Failed to search for pattern")
			continue
		}

		log.Info().
			Str("pattern", pattern).
			Int("matches", len(patternResults.Alerts)).
			Msg("Pattern search results")
	}

	// 4. Get related alerts
	if len(history.Alerts) > 0 {
		alertID := history.Alerts[0].ID
		relatedAlerts, err := d.lifecycle.GetRelatedAlerts(ctx, alertID, 5)
		if err != nil {
			log.Warn().
				Err(err).
				Str("alert_id", alertID).
				Msg("Failed to get related alerts")
		} else {
			log.Info().
				Str("alert_id", alertID).
				Int("related_count", len(relatedAlerts)).
				Msg("Related alerts found")
		}
	}

	log.Info().Msg("Alert history demonstration completed successfully")
	return nil
}

// CleanupDemo cleans up alerts created during demonstration
func (d *AlertSystemDemo) CleanupDemo(ctx context.Context) error {
	log.Info().Msg("Cleaning up demonstration alerts")

	// Clean up old resolved alerts (older than 1 minute for demo purposes)
	cutoffTime := time.Now().Add(-1 * time.Minute)
	err := d.alertService.CleanupOldResolvedAlerts(ctx, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old alerts: %w", err)
	}

	log.Info().Msg("Demonstration cleanup completed")
	return nil
}

// RunFullDemo runs the complete alert system demonstration
func (d *AlertSystemDemo) RunFullDemo(ctx context.Context) error {
	log.Info().Msg("Starting complete alert system demonstration")

	// Run all demonstration phases
	if err := d.DemonstrateAlertLifecycle(ctx); err != nil {
		return fmt.Errorf("alert lifecycle demo failed: %w", err)
	}

	if err := d.DemonstrateSystemIntegration(ctx); err != nil {
		return fmt.Errorf("system integration demo failed: %w", err)
	}

	if err := d.DemonstrateAlertHistory(ctx); err != nil {
		return fmt.Errorf("alert history demo failed: %w", err)
	}

	// Optional cleanup
	if err := d.CleanupDemo(ctx); err != nil {
		log.Warn().Err(err).Msg("Demo cleanup failed, but continuing")
	}

	log.Info().Msg("Complete alert system demonstration finished successfully")
	return nil
}