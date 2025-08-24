package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/gin-gonic/gin"
)

// AlertHandlerImpl implements comprehensive alert management
type AlertHandlerImpl struct {
	alertService interfaces.AlertService
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(alertService interfaces.AlertService) interfaces.AlertHandler {
	return &AlertHandlerImpl{
		alertService: alertService,
	}
}

// RegisterRoutes registers HTTP routes for alert management
func (h *AlertHandlerImpl) RegisterRoutes(router gin.IRouter) {
	alertGroup := router.Group("/alerts")
	{
		// Basic CRUD operations
		alertGroup.POST("", h.CreateAlert)
		alertGroup.GET("", h.ListAlerts)
		alertGroup.GET("/:id", h.GetAlert)
		
		// Alert lifecycle management
		alertGroup.POST("/:id/acknowledge", h.AcknowledgeAlert)
		alertGroup.POST("/:id/resolve", h.ResolveAlert)
		
		// Search and filtering
		alertGroup.GET("/search", h.SearchAlerts)
		alertGroup.GET("/statistics", h.GetAlertStatistics)
		alertGroup.GET("/by-source/:source", h.GetAlertsBySource)
		
		// Maintenance operations
		alertGroup.DELETE("/cleanup", h.CleanupOldResolvedAlerts)
	}
}

// CreateAlert creates a new alert
func (h *AlertHandlerImpl) CreateAlert(c *gin.Context) {
	var req struct {
		Severity string                 `json:"severity" binding:"required,oneof=critical warning info"`
		Title    string                 `json:"title" binding:"required,max=255"`
		Message  string                 `json:"message" binding:"required"`
		Source   string                 `json:"source" binding:"required,max=100"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request_body",
			"message": "Invalid request body format",
			"details": err.Error(),
		})
		return
	}

	alert, err := h.alertService.CreateAlert(
		c.Request.Context(),
		req.Severity,
		req.Title,
		req.Message,
		req.Source,
		req.Metadata,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_create_alert",
			"message": "Failed to create alert",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, alert)
}

// GetAlert retrieves a specific alert by ID
func (h *AlertHandlerImpl) GetAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_alert_id",
			"message": "Alert ID is required",
		})
		return
	}

	alert, err := h.alertService.GetAlert(c.Request.Context(), alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "alert_not_found",
			"message": "Alert not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// ListAlerts returns paginated alerts with filtering
func (h *AlertHandlerImpl) ListAlerts(c *gin.Context) {
	params, err := h.parseAlertParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameters",
			"message": "Invalid alert parameters",
			"details": err.Error(),
		})
		return
	}

	alerts, err := h.alertService.ListAlerts(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_list_alerts",
			"message": "Failed to retrieve alerts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

// SearchAlerts searches alerts with text search and filtering
func (h *AlertHandlerImpl) SearchAlerts(c *gin.Context) {
	searchText := c.Query("q")
	if searchText == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_search_query",
			"message": "Search query parameter 'q' is required",
		})
		return
	}

	params, err := h.parseAlertParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameters",
			"message": "Invalid alert parameters",
			"details": err.Error(),
		})
		return
	}

	alerts, err := h.alertService.SearchAlerts(c.Request.Context(), searchText, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_search_alerts",
			"message": "Failed to search alerts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

// AcknowledgeAlert marks an alert as acknowledged
func (h *AlertHandlerImpl) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_alert_id",
			"message": "Alert ID is required",
		})
		return
	}

	var req struct {
		AcknowledgedBy string `json:"acknowledged_by"`
	}

	// Parse request body (optional)
	c.ShouldBindJSON(&req)

	// If not provided in body, use a default or get from auth context
	acknowledgedBy := req.AcknowledgedBy
	if acknowledgedBy == "" {
		// TODO: Get from authentication context
		acknowledgedBy = "admin"
	}

	alert, err := h.alertService.AcknowledgeAlert(c.Request.Context(), alertID, acknowledgedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_acknowledge_alert",
			"message": "Failed to acknowledge alert",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Alert acknowledged successfully",
		"alert":   alert,
	})
}

// ResolveAlert marks an alert as resolved
func (h *AlertHandlerImpl) ResolveAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_alert_id",
			"message": "Alert ID is required",
		})
		return
	}

	var req struct {
		ResolvedBy string `json:"resolved_by"`
		Notes      string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request_body",
			"message": "Invalid request body format",
			"details": err.Error(),
		})
		return
	}

	// If not provided in body, use a default or get from auth context
	resolvedBy := req.ResolvedBy
	if resolvedBy == "" {
		// TODO: Get from authentication context
		resolvedBy = "admin"
	}

	alert, err := h.alertService.ResolveAlert(c.Request.Context(), alertID, resolvedBy, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_resolve_alert",
			"message": "Failed to resolve alert",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Alert resolved successfully",
		"alert":   alert,
	})
}

// GetAlertStatistics returns alert statistics for a time range
func (h *AlertHandlerImpl) GetAlertStatistics(c *gin.Context) {
	timeRange, err := h.parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_time_range",
			"message": "Invalid time range parameters",
			"details": err.Error(),
		})
		return
	}

	var timeRangePtr *interfaces.TimeRange
	if !timeRange.Start.IsZero() || !timeRange.End.IsZero() {
		timeRangePtr = &timeRange
	}

	stats, err := h.alertService.GetAlertStatistics(c.Request.Context(), timeRangePtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_get_statistics",
			"message": "Failed to retrieve alert statistics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetAlertsBySource returns alerts from a specific source
func (h *AlertHandlerImpl) GetAlertsBySource(c *gin.Context) {
	source := c.Param("source")
	if source == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_source",
			"message": "Source parameter is required",
		})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	alerts, err := h.alertService.GetAlertsBySource(c.Request.Context(), source, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_get_alerts_by_source",
			"message": "Failed to retrieve alerts by source",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"source": source,
		"limit":  limit,
		"alerts": alerts,
	})
}

// CleanupOldResolvedAlerts removes old resolved alerts
func (h *AlertHandlerImpl) CleanupOldResolvedAlerts(c *gin.Context) {
	var req struct {
		OlderThanDays int `json:"older_than_days" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request_body",
			"message": "Invalid request body format. 'older_than_days' is required and must be >= 1",
			"details": err.Error(),
		})
		return
	}

	olderThan := time.Now().AddDate(0, 0, -req.OlderThanDays)

	err := h.alertService.CleanupOldResolvedAlerts(c.Request.Context(), olderThan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_cleanup_alerts",
			"message": "Failed to cleanup old resolved alerts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Old resolved alerts cleaned up successfully",
		"older_than_days": req.OlderThanDays,
		"cutoff_date":     olderThan.Format(time.RFC3339),
	})
}

// Helper methods

// parseAlertParams parses alert filtering and pagination parameters
func (h *AlertHandlerImpl) parseAlertParams(c *gin.Context) (interfaces.AlertParams, error) {
	var params interfaces.AlertParams

	// Parse pagination parameters
	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			return params, err
		}
		params.Page = page
	} else {
		params.Page = 1
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			return params, err
		}
		params.PageSize = pageSize
	} else {
		params.PageSize = 20
	}

	// Parse filter parameters
	params.Severity = c.Query("severity")
	params.Source = c.Query("source")

	// Parse boolean parameters
	if acknowledgedStr := c.Query("acknowledged"); acknowledgedStr != "" {
		acknowledged, err := strconv.ParseBool(acknowledgedStr)
		if err != nil {
			return params, err
		}
		params.Acknowledged = &acknowledged
	}

	if resolvedStr := c.Query("resolved"); resolvedStr != "" {
		resolved, err := strconv.ParseBool(resolvedStr)
		if err != nil {
			return params, err
		}
		params.Resolved = &resolved
	}

	// Parse date range parameters
	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		dateFrom, err := time.Parse(time.RFC3339, dateFromStr)
		if err != nil {
			return params, err
		}
		params.DateFrom = &dateFrom
	}

	if dateToStr := c.Query("date_to"); dateToStr != "" {
		dateTo, err := time.Parse(time.RFC3339, dateToStr)
		if err != nil {
			return params, err
		}
		params.DateTo = &dateTo
	}

	return params, nil
}

// parseTimeRange parses time range parameters from query string
func (h *AlertHandlerImpl) parseTimeRange(c *gin.Context) (interfaces.TimeRange, error) {
	var timeRange interfaces.TimeRange

	// Parse start time
	if startStr := c.Query("start"); startStr != "" {
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return timeRange, err
		}
		timeRange.Start = start
	}

	// Parse end time
	if endStr := c.Query("end"); endStr != "" {
		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			return timeRange, err
		}
		timeRange.End = end
	}

	// Parse duration (alternative to start/end)
	if durationStr := c.Query("duration"); durationStr != "" {
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return timeRange, err
		}
		if timeRange.End.IsZero() {
			timeRange.End = time.Now()
		}
		timeRange.Start = timeRange.End.Add(-duration)
	}

	return timeRange, nil
}