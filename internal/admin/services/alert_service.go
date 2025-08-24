package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/rs/zerolog/log"
)

// AlertServiceImpl implements alert management with database persistence
type AlertServiceImpl struct {
	db      *pgxpool.Pool
	queries *queries.Queries
	notificationService interfaces.NotificationService
}

// NewAlertService creates a new alert service
func NewAlertService(db *pgxpool.Pool, notificationService interfaces.NotificationService) interfaces.AlertService {
	return &AlertServiceImpl{
		db:      db,
		queries: queries.New(db),
		notificationService: notificationService,
	}
}

// CreateAlert creates a new alert and broadcasts it
func (s *AlertServiceImpl) CreateAlert(ctx context.Context, severity, title, message, source string, metadata map[string]interface{}) (*interfaces.Alert, error) {
	// Validate severity
	if !isValidSeverity(severity) {
		return nil, fmt.Errorf("invalid severity: %s. Must be one of: critical, warning, info", severity)
	}

	// Convert metadata to JSON
	var metadataBytes []byte
	if metadata != nil {
		var err error
		metadataBytes, err = json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Create alert in database
	params := queries.CreateAlertParams{
		Severity:  severity,
		Title:     title,
		Message:   message,
		Source:    source,
		Timestamp: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Metadata:  metadataBytes,
	}

	dbAlert, err := s.queries.CreateAlert(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	// Convert to interface type
	alert := s.convertDBAlertToInterface(dbAlert)

	// Broadcast notification
	notification := &interfaces.Notification{
		ID:        fmt.Sprintf("alert_%s", alert.ID),
		Type:      "alert",
		Title:     fmt.Sprintf("New %s Alert", severity),
		Message:   fmt.Sprintf("%s: %s", title, message),
		Severity:  severity,
		Timestamp: alert.Timestamp,
		Data: map[string]interface{}{
			"alert_id": alert.ID,
			"source":   source,
			"metadata": metadata,
		},
	}

	if err := s.notificationService.Broadcast(ctx, notification); err != nil {
		log.Warn().
			Err(err).
			Str("alert_id", alert.ID).
			Msg("Failed to broadcast alert notification")
	}

	log.Info().
		Str("alert_id", alert.ID).
		Str("severity", severity).
		Str("source", source).
		Msg("Alert created successfully")

	return alert, nil
}

// GetAlert retrieves a specific alert by ID
func (s *AlertServiceImpl) GetAlert(ctx context.Context, alertID string) (*interfaces.Alert, error) {
	alertUUID, err := uuid.Parse(alertID)
	if err != nil {
		return nil, fmt.Errorf("invalid alert ID format: %w", err)
	}

	dbAlert, err := s.queries.GetAlert(ctx, pgtype.UUID{Bytes: alertUUID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	return s.convertDBAlertToInterface(dbAlert), nil
}

// ListAlerts returns paginated alerts with filtering
func (s *AlertServiceImpl) ListAlerts(ctx context.Context, params interfaces.AlertParams) (*interfaces.PaginatedAlerts, error) {
	// Set defaults
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	// Convert parameters for database query
	listParams := queries.ListAlertsParams{
		Column1: params.Severity,
		Limit:   int32(params.PageSize),
		Offset:  int32((params.Page - 1) * params.PageSize),
	}

	// Handle optional boolean parameters
	if params.Acknowledged != nil {
		listParams.Column2 = *params.Acknowledged
	}
	if params.Resolved != nil {
		listParams.Column3 = *params.Resolved
	}

	// Handle optional string parameters
	listParams.Column4 = params.Source

	// Handle date range
	if params.DateFrom != nil {
		listParams.Column5 = pgtype.Timestamptz{Time: *params.DateFrom, Valid: true}
	}
	if params.DateTo != nil {
		listParams.Column6 = pgtype.Timestamptz{Time: *params.DateTo, Valid: true}
	}

	// Get alerts
	dbAlerts, err := s.queries.ListAlerts(ctx, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}

	// Get total count
	countParams := queries.CountAlertsParams{
		Column1: params.Severity,
		Column4: params.Source,
	}
	if params.Acknowledged != nil {
		countParams.Column2 = *params.Acknowledged
	}
	if params.Resolved != nil {
		countParams.Column3 = *params.Resolved
	}
	if params.DateFrom != nil {
		countParams.Column5 = pgtype.Timestamptz{Time: *params.DateFrom, Valid: true}
	}
	if params.DateTo != nil {
		countParams.Column6 = pgtype.Timestamptz{Time: *params.DateTo, Valid: true}
	}

	totalCount, err := s.queries.CountAlerts(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("failed to count alerts: %w", err)
	}

	// Convert to interface types
	alerts := make([]interfaces.Alert, len(dbAlerts))
	for i, dbAlert := range dbAlerts {
		alerts[i] = *s.convertDBAlertToInterface(dbAlert)
	}

	// Calculate pagination info
	totalPages := int((totalCount + int64(params.PageSize) - 1) / int64(params.PageSize))
	pagination := interfaces.PaginationInfo{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: int(totalCount),
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}

	return &interfaces.PaginatedAlerts{
		Alerts:     alerts,
		Pagination: pagination,
	}, nil
}

// SearchAlerts searches alerts with text search and filtering
func (s *AlertServiceImpl) SearchAlerts(ctx context.Context, searchText string, params interfaces.AlertParams) (*interfaces.PaginatedAlerts, error) {
	// Set defaults
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	// Convert parameters for database query
	searchParams := queries.SearchAlertsParams{
		Column1: searchText,
		Column2: params.Severity,
		Limit:   int32(params.PageSize),
		Offset:  int32((params.Page - 1) * params.PageSize),
		Column7: "timestamp", // Default sort by timestamp
		Column8: true,        // Default descending order
	}

	// Handle optional boolean parameters
	if params.Acknowledged != nil {
		searchParams.Column3 = *params.Acknowledged
	}
	if params.Resolved != nil {
		searchParams.Column4 = *params.Resolved
	}

	// Handle date range
	if params.DateFrom != nil {
		searchParams.Column5 = pgtype.Timestamptz{Time: *params.DateFrom, Valid: true}
	}
	if params.DateTo != nil {
		searchParams.Column6 = pgtype.Timestamptz{Time: *params.DateTo, Valid: true}
	}

	// Get alerts
	dbAlerts, err := s.queries.SearchAlerts(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to search alerts: %w", err)
	}

	// Convert to interface types
	alerts := make([]interfaces.Alert, len(dbAlerts))
	for i, dbAlert := range dbAlerts {
		alerts[i] = *s.convertDBAlertToInterface(dbAlert)
	}

	// For search, we'll use the count of returned results as an approximation
	// In a production system, you might want a separate count query for search
	totalCount := len(alerts)
	totalPages := 1
	if totalCount == params.PageSize {
		totalPages = params.Page + 1 // Assume there might be more pages
	}

	pagination := interfaces.PaginationInfo{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: totalCount,
		TotalPages: totalPages,
		HasNext:    totalCount == params.PageSize,
		HasPrev:    params.Page > 1,
	}

	return &interfaces.PaginatedAlerts{
		Alerts:     alerts,
		Pagination: pagination,
	}, nil
}

// AcknowledgeAlert marks an alert as acknowledged
func (s *AlertServiceImpl) AcknowledgeAlert(ctx context.Context, alertID, acknowledgedBy string) (*interfaces.Alert, error) {
	alertUUID, err := uuid.Parse(alertID)
	if err != nil {
		return nil, fmt.Errorf("invalid alert ID format: %w", err)
	}

	params := queries.AcknowledgeAlertParams{
		ID:             pgtype.UUID{Bytes: alertUUID, Valid: true},
		AcknowledgedBy: pgtype.Text{String: acknowledgedBy, Valid: true},
	}

	dbAlert, err := s.queries.AcknowledgeAlert(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	alert := s.convertDBAlertToInterface(dbAlert)

	// Broadcast acknowledgment notification
	notification := &interfaces.Notification{
		ID:        fmt.Sprintf("alert_ack_%s", alert.ID),
		Type:      "alert_update",
		Title:     "Alert Acknowledged",
		Message:   fmt.Sprintf("Alert '%s' has been acknowledged by %s", alert.Title, acknowledgedBy),
		Severity:  "info",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"alert_id":        alert.ID,
			"action":          "acknowledged",
			"acknowledged_by": acknowledgedBy,
		},
	}

	if err := s.notificationService.Broadcast(ctx, notification); err != nil {
		log.Warn().
			Err(err).
			Str("alert_id", alert.ID).
			Msg("Failed to broadcast alert acknowledgment notification")
	}

	log.Info().
		Str("alert_id", alert.ID).
		Str("acknowledged_by", acknowledgedBy).
		Msg("Alert acknowledged successfully")

	return alert, nil
}

// ResolveAlert marks an alert as resolved
func (s *AlertServiceImpl) ResolveAlert(ctx context.Context, alertID, resolvedBy, notes string) (*interfaces.Alert, error) {
	alertUUID, err := uuid.Parse(alertID)
	if err != nil {
		return nil, fmt.Errorf("invalid alert ID format: %w", err)
	}

	params := queries.ResolveAlertParams{
		ID:            pgtype.UUID{Bytes: alertUUID, Valid: true},
		ResolvedBy:    pgtype.Text{String: resolvedBy, Valid: true},
		ResolvedNotes: pgtype.Text{String: notes, Valid: true},
	}

	dbAlert, err := s.queries.ResolveAlert(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve alert: %w", err)
	}

	alert := s.convertDBAlertToInterface(dbAlert)

	// Broadcast resolution notification
	notification := &interfaces.Notification{
		ID:        fmt.Sprintf("alert_resolve_%s", alert.ID),
		Type:      "alert_update",
		Title:     "Alert Resolved",
		Message:   fmt.Sprintf("Alert '%s' has been resolved by %s", alert.Title, resolvedBy),
		Severity:  "info",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"alert_id":    alert.ID,
			"action":      "resolved",
			"resolved_by": resolvedBy,
			"notes":       notes,
		},
	}

	if err := s.notificationService.Broadcast(ctx, notification); err != nil {
		log.Warn().
			Err(err).
			Str("alert_id", alert.ID).
			Msg("Failed to broadcast alert resolution notification")
	}

	log.Info().
		Str("alert_id", alert.ID).
		Str("resolved_by", resolvedBy).
		Msg("Alert resolved successfully")

	return alert, nil
}

// GetAlertStatistics returns alert statistics for a time range
func (s *AlertServiceImpl) GetAlertStatistics(ctx context.Context, timeRange *interfaces.TimeRange) (*interfaces.AlertStatistics, error) {
	params := queries.GetAlertStatisticsParams{}

	if timeRange != nil {
		if !timeRange.Start.IsZero() {
			params.Column1 = pgtype.Timestamptz{Time: timeRange.Start, Valid: true}
		}
		if !timeRange.End.IsZero() {
			params.Column2 = pgtype.Timestamptz{Time: timeRange.End, Valid: true}
		}
	}

	stats, err := s.queries.GetAlertStatistics(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert statistics: %w", err)
	}

	return &interfaces.AlertStatistics{
		TotalAlerts:       int(stats.TotalAlerts),
		CriticalCount:     int(stats.CriticalCount),
		WarningCount:      int(stats.WarningCount),
		InfoCount:         int(stats.InfoCount),
		AcknowledgedCount: int(stats.AcknowledgedCount),
		ResolvedCount:     int(stats.ResolvedCount),
		UnresolvedCount:   int(stats.UnresolvedCount),
	}, nil
}

// GetUnresolvedAlertsCount returns the count of unresolved alerts
func (s *AlertServiceImpl) GetUnresolvedAlertsCount(ctx context.Context) (int, error) {
	count, err := s.queries.GetUnresolvedAlertsCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get unresolved alerts count: %w", err)
	}
	return int(count), nil
}

// GetAlertsBySource returns alerts from a specific source
func (s *AlertServiceImpl) GetAlertsBySource(ctx context.Context, source string, limit int) ([]interfaces.Alert, error) {
	if limit <= 0 {
		limit = 10
	}

	params := queries.GetAlertsBySourceParams{
		Source: source,
		Limit:  int32(limit),
	}

	dbAlerts, err := s.queries.GetAlertsBySource(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get alerts by source: %w", err)
	}

	alerts := make([]interfaces.Alert, len(dbAlerts))
	for i, dbAlert := range dbAlerts {
		alerts[i] = *s.convertDBAlertToInterface(dbAlert)
	}

	return alerts, nil
}

// CleanupOldResolvedAlerts removes old resolved alerts
func (s *AlertServiceImpl) CleanupOldResolvedAlerts(ctx context.Context, olderThan time.Time) error {
	err := s.queries.DeleteOldResolvedAlerts(ctx, pgtype.Timestamptz{Time: olderThan, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to cleanup old resolved alerts: %w", err)
	}

	log.Info().
		Time("older_than", olderThan).
		Msg("Cleaned up old resolved alerts")

	return nil
}

// Helper methods

// convertDBAlertToInterface converts database Alert to interface Alert
func (s *AlertServiceImpl) convertDBAlertToInterface(dbAlert queries.Alert) *interfaces.Alert {
	alert := &interfaces.Alert{
		ID:           uuid.UUID(dbAlert.ID.Bytes).String(),
		Severity:     dbAlert.Severity,
		Title:        dbAlert.Title,
		Message:      dbAlert.Message,
		Source:       dbAlert.Source,
		Timestamp:    dbAlert.Timestamp.Time,
		Acknowledged: dbAlert.Acknowledged,
		Resolved:     dbAlert.Resolved,
	}

	if dbAlert.AcknowledgedBy.Valid {
		alert.AcknowledgedBy = dbAlert.AcknowledgedBy.String
	}
	if dbAlert.AcknowledgedAt.Valid {
		alert.AcknowledgedAt = &dbAlert.AcknowledgedAt.Time
	}
	if dbAlert.ResolvedBy.Valid {
		alert.ResolvedBy = dbAlert.ResolvedBy.String
	}
	if dbAlert.ResolvedAt.Valid {
		alert.ResolvedAt = &dbAlert.ResolvedAt.Time
	}
	if dbAlert.ResolvedNotes.Valid {
		alert.ResolvedNotes = dbAlert.ResolvedNotes.String
	}

	// Parse metadata JSON
	if len(dbAlert.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(dbAlert.Metadata, &metadata); err == nil {
			alert.Metadata = metadata
		}
	}

	return alert
}

// isValidSeverity checks if the severity is valid
func isValidSeverity(severity string) bool {
	switch severity {
	case "critical", "warning", "info":
		return true
	default:
		return false
	}
}