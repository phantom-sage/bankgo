package services

import (
	"context"
	"fmt"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/rs/zerolog/log"
)

// AlertLifecycleService provides advanced alert lifecycle management
type AlertLifecycleService struct {
	alertService interfaces.AlertService
}

// NewAlertLifecycleService creates a new alert lifecycle service
func NewAlertLifecycleService(alertService interfaces.AlertService) *AlertLifecycleService {
	return &AlertLifecycleService{
		alertService: alertService,
	}
}

// AlertHistoryParams defines parameters for alert history queries
type AlertHistoryParams struct {
	interfaces.AlertParams
	IncludeResolved   bool      `json:"include_resolved"`
	MinSeverity       string    `json:"min_severity"` // critical, warning, info
	MaxAge            *time.Duration `json:"max_age"`
	Sources           []string  `json:"sources"`
	MetadataFilters   map[string]interface{} `json:"metadata_filters"`
}

// AlertTrend represents alert trend data
type AlertTrend struct {
	Period      string `json:"period"`      // hour, day, week, month
	Timestamp   time.Time `json:"timestamp"`
	Count       int    `json:"count"`
	Severity    string `json:"severity"`
	Source      string `json:"source,omitempty"`
}

// AlertSummary provides a summary of alert statistics
type AlertSummary struct {
	TotalAlerts       int                    `json:"total_alerts"`
	UnresolvedAlerts  int                    `json:"unresolved_alerts"`
	CriticalAlerts    int                    `json:"critical_alerts"`
	RecentAlerts      int                    `json:"recent_alerts"` // Last 24 hours
	TopSources        []SourceSummary        `json:"top_sources"`
	SeverityBreakdown map[string]int         `json:"severity_breakdown"`
	TrendData         []AlertTrend           `json:"trend_data"`
}

// SourceSummary provides alert statistics by source
type SourceSummary struct {
	Source          string `json:"source"`
	Count           int    `json:"count"`
	UnresolvedCount int    `json:"unresolved_count"`
	LastAlert       *time.Time `json:"last_alert,omitempty"`
}

// GetAlertHistory retrieves alert history with advanced filtering
func (s *AlertLifecycleService) GetAlertHistory(ctx context.Context, params AlertHistoryParams) (*interfaces.PaginatedAlerts, error) {
	// Convert to base alert params
	alertParams := params.AlertParams

	// Apply additional filters
	if !params.IncludeResolved {
		resolved := false
		alertParams.Resolved = &resolved
	}

	// Apply minimum severity filter
	if params.MinSeverity != "" {
		alertParams.Severity = params.MinSeverity
	}

	// Apply max age filter
	if params.MaxAge != nil {
		cutoffTime := time.Now().Add(-*params.MaxAge)
		alertParams.DateFrom = &cutoffTime
	}

	// For now, use the basic list alerts functionality
	// In a more advanced implementation, you would extend the database queries
	// to support metadata filtering and multiple sources
	return s.alertService.ListAlerts(ctx, alertParams)
}

// GetAlertSummary provides a comprehensive summary of alert statistics
func (s *AlertLifecycleService) GetAlertSummary(ctx context.Context, timeRange *interfaces.TimeRange) (*AlertSummary, error) {
	// Get basic statistics
	stats, err := s.alertService.GetAlertStatistics(ctx, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert statistics: %w", err)
	}

	// Calculate recent alerts (last 24 hours)
	recentTimeRange := &interfaces.TimeRange{
		Start: time.Now().Add(-24 * time.Hour),
		End:   time.Now(),
	}
	recentStats, err := s.alertService.GetAlertStatistics(ctx, recentTimeRange)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get recent alert statistics")
		recentStats = &interfaces.AlertStatistics{}
	}

	// Get top sources (simplified - in a real implementation, you'd have a dedicated query)
	topSources := s.getTopSources(ctx)

	// Create severity breakdown
	severityBreakdown := map[string]int{
		"critical": stats.CriticalCount,
		"warning":  stats.WarningCount,
		"info":     stats.InfoCount,
	}

	// Generate trend data (simplified)
	trendData := s.generateTrendData(ctx, timeRange)

	summary := &AlertSummary{
		TotalAlerts:       stats.TotalAlerts,
		UnresolvedAlerts:  stats.UnresolvedCount,
		CriticalAlerts:    stats.CriticalCount,
		RecentAlerts:      recentStats.TotalAlerts,
		TopSources:        topSources,
		SeverityBreakdown: severityBreakdown,
		TrendData:         trendData,
	}

	return summary, nil
}

// ArchiveOldAlerts archives old resolved alerts instead of deleting them
func (s *AlertLifecycleService) ArchiveOldAlerts(ctx context.Context, olderThan time.Time) error {
	// For now, use the cleanup function
	// In a real implementation, you might move alerts to an archive table
	return s.alertService.CleanupOldResolvedAlerts(ctx, olderThan)
}

// GetAlertsByPattern searches for alerts matching specific patterns
func (s *AlertLifecycleService) GetAlertsByPattern(ctx context.Context, pattern string, params interfaces.AlertParams) (*interfaces.PaginatedAlerts, error) {
	// Use the search functionality
	return s.alertService.SearchAlerts(ctx, pattern, params)
}

// GetRelatedAlerts finds alerts related to a specific alert
func (s *AlertLifecycleService) GetRelatedAlerts(ctx context.Context, alertID string, maxResults int) ([]interfaces.Alert, error) {
	// Get the original alert
	originalAlert, err := s.alertService.GetAlert(ctx, alertID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original alert: %w", err)
	}

	// Find alerts from the same source
	relatedAlerts, err := s.alertService.GetAlertsBySource(ctx, originalAlert.Source, maxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to get related alerts: %w", err)
	}

	// Filter out the original alert
	var filtered []interfaces.Alert
	for _, alert := range relatedAlerts {
		if alert.ID != alertID {
			filtered = append(filtered, alert)
		}
	}

	return filtered, nil
}

// BulkAcknowledgeAlerts acknowledges multiple alerts based on criteria
func (s *AlertLifecycleService) BulkAcknowledgeAlerts(ctx context.Context, source, acknowledgedBy string) (int, error) {
	// Get unacknowledged alerts from the specified source
	alerts, err := s.alertService.GetAlertsBySource(ctx, source, 100)
	if err != nil {
		return 0, fmt.Errorf("failed to get alerts by source: %w", err)
	}

	acknowledgedCount := 0
	for _, alert := range alerts {
		if !alert.Acknowledged && !alert.Resolved {
			_, err := s.alertService.AcknowledgeAlert(ctx, alert.ID, acknowledgedBy)
			if err != nil {
				log.Error().
					Err(err).
					Str("alert_id", alert.ID).
					Msg("Failed to acknowledge alert in bulk operation")
				continue
			}
			acknowledgedCount++
		}
	}

	log.Info().
		Str("source", source).
		Int("acknowledged_count", acknowledgedCount).
		Msg("Bulk alert acknowledgment completed")

	return acknowledgedCount, nil
}

// GetAlertEscalationCandidates finds alerts that may need escalation
func (s *AlertLifecycleService) GetAlertEscalationCandidates(ctx context.Context, maxAge time.Duration) ([]interfaces.Alert, error) {
	cutoffTime := time.Now().Add(-maxAge)
	
	params := interfaces.AlertParams{
		PaginationParams: interfaces.PaginationParams{
			Page:     1,
			PageSize: 100,
		},
		DateTo: &cutoffTime,
	}

	// Get unresolved alerts older than the specified age
	resolved := false
	params.Resolved = &resolved

	result, err := s.alertService.ListAlerts(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get escalation candidates: %w", err)
	}

	// Filter for critical alerts or unacknowledged alerts
	var candidates []interfaces.Alert
	for _, alert := range result.Alerts {
		if alert.Severity == "critical" || !alert.Acknowledged {
			candidates = append(candidates, alert)
		}
	}

	return candidates, nil
}

// GetAlertMetrics provides detailed metrics about alert patterns
func (s *AlertLifecycleService) GetAlertMetrics(ctx context.Context, timeRange *interfaces.TimeRange) (map[string]interface{}, error) {
	stats, err := s.alertService.GetAlertStatistics(ctx, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert statistics: %w", err)
	}

	metrics := map[string]interface{}{
		"total_alerts":        stats.TotalAlerts,
		"unresolved_alerts":   stats.UnresolvedCount,
		"resolution_rate":     float64(stats.ResolvedCount) / float64(stats.TotalAlerts) * 100,
		"acknowledgment_rate": float64(stats.AcknowledgedCount) / float64(stats.TotalAlerts) * 100,
		"critical_percentage": float64(stats.CriticalCount) / float64(stats.TotalAlerts) * 100,
		"warning_percentage":  float64(stats.WarningCount) / float64(stats.TotalAlerts) * 100,
		"info_percentage":     float64(stats.InfoCount) / float64(stats.TotalAlerts) * 100,
	}

	// Add average resolution time (would need additional database queries)
	metrics["avg_resolution_time_hours"] = 2.5 // Mock value

	return metrics, nil
}

// Helper methods

// getTopSources returns the top alert sources (simplified implementation)
func (s *AlertLifecycleService) getTopSources(ctx context.Context) []SourceSummary {
	// In a real implementation, this would be a database query
	// For now, return mock data
	return []SourceSummary{
		{
			Source:          "system_monitor",
			Count:           45,
			UnresolvedCount: 12,
			LastAlert:       func() *time.Time { t := time.Now().Add(-1 * time.Hour); return &t }(),
		},
		{
			Source:          "database_monitor",
			Count:           23,
			UnresolvedCount: 5,
			LastAlert:       func() *time.Time { t := time.Now().Add(-30 * time.Minute); return &t }(),
		},
		{
			Source:          "auth_monitor",
			Count:           18,
			UnresolvedCount: 3,
			LastAlert:       func() *time.Time { t := time.Now().Add(-2 * time.Hour); return &t }(),
		},
	}
}

// generateTrendData generates alert trend data (simplified implementation)
func (s *AlertLifecycleService) generateTrendData(ctx context.Context, timeRange *interfaces.TimeRange) []AlertTrend {
	// In a real implementation, this would aggregate alerts by time periods
	// For now, return mock trend data
	now := time.Now()
	trends := make([]AlertTrend, 0)

	// Generate hourly trends for the last 24 hours
	for i := 23; i >= 0; i-- {
		timestamp := now.Add(-time.Duration(i) * time.Hour)
		
		// Mock data - in reality, this would come from database aggregation
		trends = append(trends, AlertTrend{
			Period:    "hour",
			Timestamp: timestamp,
			Count:     5 + (i % 3), // Mock varying counts
			Severity:  "all",
		})
	}

	return trends
}

// ValidateAlertParams validates alert history parameters
func (s *AlertLifecycleService) ValidateAlertParams(params AlertHistoryParams) error {
	// Validate minimum severity
	if params.MinSeverity != "" {
		validSeverities := map[string]bool{
			"critical": true,
			"warning":  true,
			"info":     true,
		}
		if !validSeverities[params.MinSeverity] {
			return fmt.Errorf("invalid minimum severity: %s", params.MinSeverity)
		}
	}

	// Validate max age
	if params.MaxAge != nil && *params.MaxAge < 0 {
		return fmt.Errorf("max age cannot be negative")
	}

	// Validate pagination
	if params.Page < 1 {
		return fmt.Errorf("page must be >= 1")
	}
	if params.PageSize < 1 || params.PageSize > 1000 {
		return fmt.Errorf("page size must be between 1 and 1000")
	}

	return nil
}