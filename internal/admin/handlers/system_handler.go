package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/gin-gonic/gin"
)

// SystemHandlerImpl implements the SystemHandler interface
type SystemHandlerImpl struct {
	systemService interfaces.SystemMonitoringService
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(systemService interfaces.SystemMonitoringService) interfaces.SystemHandler {
	return &SystemHandlerImpl{
		systemService: systemService,
	}
}

// RegisterRoutes registers HTTP routes for system monitoring
func (h *SystemHandlerImpl) RegisterRoutes(router gin.IRouter) {
	systemGroup := router.Group("/system")
	{
		systemGroup.GET("/health", h.GetHealth)
		systemGroup.GET("/metrics", h.GetMetrics)
		systemGroup.GET("/alerts", h.GetAlerts)
		systemGroup.POST("/alerts/:id/acknowledge", h.AcknowledgeAlert)
		systemGroup.POST("/alerts/:id/resolve", h.ResolveAlert)
	}
}

// GetHealth returns current system health status
func (h *SystemHandlerImpl) GetHealth(c *gin.Context) {
	health, err := h.systemService.GetSystemHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_get_health",
			"message": "Failed to retrieve system health status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, health)
}

// GetMetrics returns system performance metrics
func (h *SystemHandlerImpl) GetMetrics(c *gin.Context) {
	// Parse time range parameters
	timeRange, err := h.parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_time_range",
			"message": "Invalid time range parameters",
			"details": err.Error(),
		})
		return
	}

	metrics, err := h.systemService.GetMetrics(c.Request.Context(), timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_get_metrics",
			"message": "Failed to retrieve system metrics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetAlerts returns system alerts with pagination and filtering
func (h *SystemHandlerImpl) GetAlerts(c *gin.Context) {
	// Parse alert parameters
	params, err := h.parseAlertParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameters",
			"message": "Invalid alert parameters",
			"details": err.Error(),
		})
		return
	}

	alerts, err := h.systemService.GetAlerts(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_get_alerts",
			"message": "Failed to retrieve system alerts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

// AcknowledgeAlert marks an alert as acknowledged
func (h *SystemHandlerImpl) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_alert_id",
			"message": "Alert ID is required",
		})
		return
	}

	err := h.systemService.AcknowledgeAlert(c.Request.Context(), alertID)
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
		"alert_id": alertID,
	})
}

// ResolveAlert marks an alert as resolved
func (h *SystemHandlerImpl) ResolveAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_alert_id",
			"message": "Alert ID is required",
		})
		return
	}

	// Parse request body for resolution notes
	var req struct {
		Notes string `json:"notes"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request_body",
			"message": "Invalid request body format",
			"details": err.Error(),
		})
		return
	}

	err := h.systemService.ResolveAlert(c.Request.Context(), alertID, req.Notes)
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
		"alert_id": alertID,
		"notes": req.Notes,
	})
}

// parseTimeRange parses time range parameters from query string
func (h *SystemHandlerImpl) parseTimeRange(c *gin.Context) (interfaces.TimeRange, error) {
	var timeRange interfaces.TimeRange
	
	// Default to last hour if no parameters provided
	now := time.Now()
	timeRange.End = now
	timeRange.Start = now.Add(-time.Hour)
	
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
		timeRange.Start = timeRange.End.Add(-duration)
	}
	
	return timeRange, nil
}

// parseAlertParams parses alert filtering and pagination parameters
func (h *SystemHandlerImpl) parseAlertParams(c *gin.Context) (interfaces.AlertParams, error) {
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