package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/database"
	"github.com/phantom-sage/bankgo/internal/logging"
	"github.com/phantom-sage/bankgo/internal/queue"
)

// HealthStatus represents the status of a service component
type HealthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents the overall health check response
type HealthResponse struct {
	Status    string                  `json:"status"`
	Timestamp time.Time               `json:"timestamp"`
	Services  map[string]HealthStatus `json:"services"`
	Version   string                  `json:"version,omitempty"`
}

// HealthHandlers handles health check related HTTP requests
type HealthHandlers struct {
	db            *database.DB
	queueManager  *queue.QueueManager
	loggerManager *logging.LoggerManager
	version       string
}

// NewHealthHandlers creates a new health handlers instance
func NewHealthHandlers(db *database.DB, queueManager *queue.QueueManager, loggerManager *logging.LoggerManager, version string) *HealthHandlers {
	return &HealthHandlers{
		db:            db,
		queueManager:  queueManager,
		loggerManager: loggerManager,
		version:       version,
	}
}

// HealthCheck handles the health check endpoint
// GET /health
func (h *HealthHandlers) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response := HealthResponse{
		Timestamp: time.Now(),
		Services:  make(map[string]HealthStatus),
		Version:   h.version,
	}

	overallHealthy := true

	// Check database connectivity
	dbStatus := h.checkDatabaseHealth(ctx)
	response.Services["database"] = dbStatus
	if dbStatus.Status != "healthy" {
		overallHealthy = false
	}

	// Check Redis connectivity
	redisStatus := h.checkRedisHealth(ctx)
	response.Services["redis"] = redisStatus
	if redisStatus.Status != "healthy" {
		overallHealthy = false
	}

	// Check logging system health
	loggingStatus := h.checkLoggingHealth(ctx)
	response.Services["logging"] = loggingStatus
	if loggingStatus.Status != "healthy" {
		overallHealthy = false
	}

	// Set overall status
	if overallHealthy {
		response.Status = "healthy"
	} else {
		response.Status = "unhealthy"
	}

	// Return appropriate HTTP status code
	statusCode := http.StatusOK
	if !overallHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// checkDatabaseHealth performs database connectivity and query execution checks
func (h *HealthHandlers) checkDatabaseHealth(ctx context.Context) HealthStatus {
	if h.db == nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "database connection not initialized",
		}
	}

	// Perform comprehensive health check
	if err := h.db.HealthCheck(ctx); err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}

	// Get connection pool statistics for additional reporting
	stats := h.db.Stats()
	if stats.TotalConns() == 0 {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "no database connections available",
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "database connectivity verified",
	}
}

// checkRedisHealth performs Redis connectivity validation
func (h *HealthHandlers) checkRedisHealth(ctx context.Context) HealthStatus {
	if h.queueManager == nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "redis connection not initialized",
		}
	}

	// Perform Redis health check
	if err := h.queueManager.HealthCheck(ctx); err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "redis connectivity verified",
	}
}

// checkLoggingHealth performs logging system health validation
func (h *HealthHandlers) checkLoggingHealth(ctx context.Context) HealthStatus {
	if h.loggerManager == nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "logger manager not initialized",
		}
	}

	// Perform logging system health check
	if err := h.loggerManager.HealthCheck(); err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "logging system operational",
	}
}