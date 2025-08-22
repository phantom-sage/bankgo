package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/phantom-sage/bankgo/internal/config"
	"github.com/rs/zerolog"
)

// Task types
const (
	TypeWelcomeEmail = "email:welcome"
)

// WelcomeEmailPayload represents the payload for welcome email tasks
type WelcomeEmailPayload struct {
	UserID    int    `json:"user_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// QueueManager manages task queuing and processing
type QueueManager struct {
	client           *AsyncqClient
	server           *AsyncqServer
	redis            *RedisClient
	logger           zerolog.Logger
	performanceLogger *PerformanceLogger
}

// NewQueueManager creates a new queue manager
func NewQueueManager(cfg config.RedisConfig, logger zerolog.Logger) (*QueueManager, error) {
	// Create Redis client
	redisClient, err := NewRedisClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	// Create Asynq client for task queuing
	asyncqClient, err := NewAsyncqClient(cfg)
	if err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("failed to create Asynq client: %w", err)
	}

	// Create Asynq server for task processing
	asyncqServer, err := NewAsyncqServer(cfg, logger)
	if err != nil {
		redisClient.Close()
		asyncqClient.Close()
		return nil, fmt.Errorf("failed to create Asynq server: %w", err)
	}

	queueLogger := logger.With().Str("component", "queue").Logger()
	performanceLogger := NewPerformanceLogger(logger)

	return &QueueManager{
		client:            asyncqClient,
		server:            asyncqServer,
		redis:             redisClient,
		logger:            queueLogger,
		performanceLogger: performanceLogger,
	}, nil
}

// QueueWelcomeEmail queues a welcome email task
func (qm *QueueManager) QueueWelcomeEmail(ctx context.Context, payload WelcomeEmailPayload) error {
	startTime := time.Now()
	
	// Get correlation ID from context if available
	correlationID := getCorrelationID(ctx)
	
	logger := qm.logger.With().
		Str("operation", "queue_welcome_email").
		Str("job_type", TypeWelcomeEmail).
		Int("user_id", payload.UserID).
		Str("email", payload.Email).
		Str("correlation_id", correlationID).
		Logger()

	logger.Info().Msg("Queuing welcome email task")

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error().
			Err(err).
			Dur("duration", time.Since(startTime)).
			Msg("Failed to marshal welcome email payload")
		return fmt.Errorf("failed to marshal welcome email payload: %w", err)
	}

	// Create task with retry options
	task := asynq.NewTask(TypeWelcomeEmail, payloadBytes)
	
	// Queue task with options
	opts := []asynq.Option{
		asynq.Queue("email"),           // Use email queue for high priority
		asynq.MaxRetry(3),              // Retry up to 3 times
		asynq.Timeout(30 * time.Second), // 30 second timeout
		asynq.ProcessIn(5 * time.Second), // Process after 5 seconds to allow for immediate response
	}

	info, err := qm.client.Client().EnqueueContext(ctx, task, opts...)
	if err != nil {
		logger.Error().
			Err(err).
			Dur("duration", time.Since(startTime)).
			Msg("Failed to enqueue welcome email task")
		return fmt.Errorf("failed to enqueue welcome email task: %w", err)
	}

	logger.Info().
		Str("task_id", info.ID).
		Str("queue", info.Queue).
		Dur("duration", time.Since(startTime)).
		Msg("Welcome email task enqueued successfully")
	
	return nil
}

// RegisterHandlers registers task handlers with the server
func (qm *QueueManager) RegisterHandlers(emailProcessor EmailProcessor) {
	// Register welcome email handler
	qm.server.RegisterHandler(TypeWelcomeEmail, func(ctx context.Context, t *asynq.Task) error {
		startTime := time.Now()
		
		// Generate correlation ID for this job execution
		correlationID := generateCorrelationID()
		ctx = context.WithValue(ctx, "correlation_id", correlationID)
		
		logger := qm.logger.With().
			Str("operation", "process_welcome_email").
			Str("job_type", TypeWelcomeEmail).
			Str("task_id", t.Type()).
			Str("correlation_id", correlationID).
			Logger()

		logger.Info().Msg("Starting welcome email task processing")

		var payload WelcomeEmailPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			logger.Error().
				Err(err).
				Dur("duration", time.Since(startTime)).
				Msg("Failed to unmarshal welcome email payload")
			return fmt.Errorf("failed to unmarshal welcome email payload: %w", err)
		}

		// Add payload details to logger
		logger = logger.With().
			Int("user_id", payload.UserID).
			Str("email", payload.Email).
			Str("first_name", payload.FirstName).
			Str("last_name", payload.LastName).
			Logger()

		// Process the email
		err := emailProcessor.ProcessWelcomeEmail(ctx, payload)
		duration := time.Since(startTime)
		success := err == nil
		
		// Log performance metrics
		qm.performanceLogger.LogJobExecution(TypeWelcomeEmail, correlationID, duration, success, 0)
		
		if err != nil {
			logger.Error().
				Err(err).
				Dur("duration", duration).
				Msg("Welcome email task processing failed")
			return err
		}

		logger.Info().
			Dur("duration", duration).
			Msg("Welcome email task processing completed successfully")
		
		return nil
	})
}

// StartServer starts the task processing server
func (qm *QueueManager) StartServer() error {
	return qm.server.Start()
}

// StopServer stops the task processing server
func (qm *QueueManager) StopServer() {
	qm.server.Stop()
}

// ShutdownServer shuts down the task processing server gracefully
func (qm *QueueManager) ShutdownServer() {
	qm.server.Shutdown()
}

// Close closes all connections
func (qm *QueueManager) Close() error {
	if err := qm.client.Close(); err != nil {
		return fmt.Errorf("failed to close Asynq client: %w", err)
	}
	if err := qm.redis.Close(); err != nil {
		return fmt.Errorf("failed to close Redis client: %w", err)
	}
	return nil
}

// HealthCheck performs health checks on Redis connection and queue system
func (qm *QueueManager) HealthCheck(ctx context.Context) error {
	startTime := time.Now()
	
	logger := qm.logger.With().
		Str("operation", "health_check").
		Logger()

	logger.Debug().Msg("Starting queue health check")

	// Check Redis connection
	if err := qm.redis.HealthCheck(ctx); err != nil {
		logger.Error().
			Err(err).
			Dur("duration", time.Since(startTime)).
			Msg("Queue health check failed - Redis connection error")
		return err
	}

	// Log queue metrics
	if err := qm.LogQueueMetrics(ctx); err != nil {
		logger.Warn().
			Err(err).
			Msg("Failed to log queue metrics during health check")
	}

	logger.Debug().
		Dur("duration", time.Since(startTime)).
		Msg("Queue health check completed successfully")
	
	return nil
}

// LogQueueMetrics logs current queue metrics and backlog sizes
func (qm *QueueManager) LogQueueMetrics(ctx context.Context) error {
	logger := qm.logger.With().
		Str("operation", "queue_metrics").
		Logger()

	// Get queue information from Redis
	redisClient := qm.redis.Client()
	
	// Check email queue size
	emailQueueSize, err := redisClient.LLen(ctx, "asynq:queues:email").Result()
	if err != nil {
		logger.Warn().
			Err(err).
			Msg("Failed to get email queue size")
		emailQueueSize = -1
	}

	// Check default queue size
	defaultQueueSize, err := redisClient.LLen(ctx, "asynq:queues:default").Result()
	if err != nil {
		logger.Warn().
			Err(err).
			Msg("Failed to get default queue size")
		defaultQueueSize = -1
	}

	// Check low priority queue size
	lowQueueSize, err := redisClient.LLen(ctx, "asynq:queues:low").Result()
	if err != nil {
		logger.Warn().
			Err(err).
			Msg("Failed to get low priority queue size")
		lowQueueSize = -1
	}

	// Check active tasks
	activeTasks, err := redisClient.ZCard(ctx, "asynq:active").Result()
	if err != nil {
		logger.Warn().
			Err(err).
			Msg("Failed to get active tasks count")
		activeTasks = -1
	}

	// Check scheduled tasks
	scheduledTasks, err := redisClient.ZCard(ctx, "asynq:scheduled").Result()
	if err != nil {
		logger.Warn().
			Err(err).
			Msg("Failed to get scheduled tasks count")
		scheduledTasks = -1
	}

	// Check retry tasks
	retryTasks, err := redisClient.ZCard(ctx, "asynq:retry").Result()
	if err != nil {
		logger.Warn().
			Err(err).
			Msg("Failed to get retry tasks count")
		retryTasks = -1
	}

	// Check dead tasks
	deadTasks, err := redisClient.ZCard(ctx, "asynq:dead").Result()
	if err != nil {
		logger.Warn().
			Err(err).
			Msg("Failed to get dead tasks count")
		deadTasks = -1
	}

	totalQueueSize := emailQueueSize + defaultQueueSize + lowQueueSize

	// Calculate processing rates (simplified - in production you'd track this over time)
	emailProcessingRate := qm.calculateProcessingRate("email", emailQueueSize)
	defaultProcessingRate := qm.calculateProcessingRate("default", defaultQueueSize)
	lowProcessingRate := qm.calculateProcessingRate("low", lowQueueSize)

	// Log individual queue metrics with performance data
	qm.performanceLogger.LogQueueDepth("email", emailQueueSize, emailProcessingRate)
	qm.performanceLogger.LogQueueDepth("default", defaultQueueSize, defaultProcessingRate)
	qm.performanceLogger.LogQueueDepth("low", lowQueueSize, lowProcessingRate)

	// Create comprehensive metrics
	metrics := JobQueueMetrics{
		TotalPending:        totalQueueSize,
		TotalActive:         activeTasks,
		TotalScheduled:      scheduledTasks,
		TotalRetry:          retryTasks,
		TotalDead:           deadTasks,
		ProcessedLastMinute: qm.getProcessedCount(ctx),
		FailedLastMinute:    qm.getFailedCount(ctx),
		SuccessRatePercent:  qm.calculateSuccessRate(ctx),
	}

	// Log comprehensive metrics
	qm.performanceLogger.LogJobQueueMetrics(metrics)

	logger.Info().
		Int64("email_queue_size", emailQueueSize).
		Int64("default_queue_size", defaultQueueSize).
		Int64("low_queue_size", lowQueueSize).
		Int64("total_queue_size", totalQueueSize).
		Int64("active_tasks", activeTasks).
		Int64("scheduled_tasks", scheduledTasks).
		Int64("retry_tasks", retryTasks).
		Int64("dead_tasks", deadTasks).
		Float64("email_processing_rate", emailProcessingRate).
		Float64("default_processing_rate", defaultProcessingRate).
		Float64("low_processing_rate", lowProcessingRate).
		Msg("Queue metrics")

	// Log warning if queues are backing up
	if totalQueueSize > 100 {
		logger.Warn().
			Int64("total_queue_size", totalQueueSize).
			Msg("Queue backlog detected - high number of pending tasks")
	}

	if retryTasks > 10 {
		logger.Warn().
			Int64("retry_tasks", retryTasks).
			Msg("High number of retry tasks detected")
	}

	if deadTasks > 0 {
		logger.Error().
			Int64("dead_tasks", deadTasks).
			Msg("Dead tasks detected - manual intervention may be required")
	}

	return nil
}

// LogWorkerHealth logs worker health status and resource usage
func (qm *QueueManager) LogWorkerHealth(ctx context.Context) {
	logger := qm.logger.With().
		Str("operation", "worker_health").
		Logger()

	// Get Redis connection info
	redisClient := qm.redis.Client()
	
	// Check Redis connection
	pingResult := redisClient.Ping(ctx)
	redisHealthy := pingResult.Err() == nil
	
	// Get Redis info
	infoResult := redisClient.Info(ctx, "memory", "clients", "stats")
	redisInfo := ""
	if infoResult.Err() == nil {
		redisInfo = infoResult.Val()
	}

	logger.Info().
		Bool("redis_healthy", redisHealthy).
		Str("redis_info", redisInfo).
		Msg("Worker health status")

	if !redisHealthy {
		logger.Error().
			Err(pingResult.Err()).
			Msg("Redis connection unhealthy")
	}

	// Log worker resource usage
	qm.LogWorkerResourceUsage(ctx)
}

// StartPeriodicMetricsLogging starts a goroutine that periodically logs queue metrics
func (qm *QueueManager) StartPeriodicMetricsLogging(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	
	go func() {
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				qm.logger.Info().Msg("Stopping periodic metrics logging")
				return
			case <-ticker.C:
				if err := qm.LogQueueMetrics(ctx); err != nil {
					qm.logger.Error().
						Err(err).
						Msg("Failed to log periodic queue metrics")
				}
				qm.LogWorkerHealth(ctx)
			}
		}
	}()
	
	qm.logger.Info().
		Dur("interval", interval).
		Msg("Started periodic queue metrics logging")
}

// calculateProcessingRate calculates the processing rate for a queue (simplified implementation)
func (qm *QueueManager) calculateProcessingRate(queueName string, currentDepth int64) float64 {
	// In a real implementation, you would track queue depth over time
	// For now, we'll use a simplified calculation based on current depth
	// Lower depth generally indicates higher processing rate
	if currentDepth == 0 {
		return 10.0 // Assume 10 jobs/second when queue is empty
	} else if currentDepth < 10 {
		return 5.0 // Medium processing rate
	} else if currentDepth < 50 {
		return 2.0 // Lower processing rate
	} else {
		return 0.5 // Very low processing rate when queue is backed up
	}
}

// getProcessedCount gets the number of processed jobs in the last minute (simplified)
func (qm *QueueManager) getProcessedCount(ctx context.Context) int64 {
	// In a real implementation, you would track this with Redis counters or time series
	// For now, return a placeholder value
	return 0
}

// getFailedCount gets the number of failed jobs in the last minute (simplified)
func (qm *QueueManager) getFailedCount(ctx context.Context) int64 {
	// In a real implementation, you would track this with Redis counters or time series
	// For now, return a placeholder value
	return 0
}

// calculateSuccessRate calculates the success rate percentage (simplified)
func (qm *QueueManager) calculateSuccessRate(ctx context.Context) float64 {
	processed := qm.getProcessedCount(ctx)
	failed := qm.getFailedCount(ctx)
	
	if processed+failed == 0 {
		return 100.0 // No jobs processed, assume 100% success rate
	}
	
	return float64(processed) / float64(processed+failed) * 100.0
}

// LogWorkerResourceUsage logs worker resource usage metrics
func (qm *QueueManager) LogWorkerResourceUsage(ctx context.Context) {
	// Get basic worker information
	// In a real implementation, you would collect actual resource metrics
	activeWorkers := 10 // This would come from actual worker pool
	totalWorkers := 10
	memoryUsageMB := int64(50) // This would come from runtime.MemStats
	cpuPercent := 25.0         // This would come from system monitoring
	
	qm.performanceLogger.LogWorkerHealth(activeWorkers, totalWorkers, memoryUsageMB, cpuPercent)
}

// GetJobCorrelationID extracts or generates a correlation ID for job tracing
func (qm *QueueManager) GetJobCorrelationID(ctx context.Context, jobType string) string {
	// Try to get existing correlation ID from context
	if id := ctx.Value("correlation_id"); id != nil {
		if correlationID, ok := id.(string); ok {
			return correlationID
		}
	}
	
	// Generate new correlation ID with job type prefix
	return fmt.Sprintf("%s_%d_%d", jobType, time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

// EmailProcessor interface for processing email tasks
type EmailProcessor interface {
	ProcessWelcomeEmail(ctx context.Context, payload WelcomeEmailPayload) error
}

// getCorrelationID extracts correlation ID from context, generates one if not present
func getCorrelationID(ctx context.Context) string {
	if id := ctx.Value("correlation_id"); id != nil {
		if correlationID, ok := id.(string); ok {
			return correlationID
		}
	}
	return generateCorrelationID()
}

// generateCorrelationID generates a new correlation ID for job tracing
func generateCorrelationID() string {
	return fmt.Sprintf("job_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

// PerformanceLogger provides performance logging for queue operations
type PerformanceLogger struct {
	logger zerolog.Logger
}

// NewPerformanceLogger creates a new performance logger for queue operations
func NewPerformanceLogger(logger zerolog.Logger) *PerformanceLogger {
	return &PerformanceLogger{
		logger: logger.With().Str("log_type", "performance").Str("component", "queue").Logger(),
	}
}

// LogJobExecution logs job execution performance metrics
func (pl *PerformanceLogger) LogJobExecution(jobType, correlationID string, duration time.Duration, success bool, retryCount int) {
	level := pl.logger.Info()
	if !success {
		level = pl.logger.Warn()
	}
	
	level.
		Str("metric_type", "job_execution").
		Str("job_type", jobType).
		Str("correlation_id", correlationID).
		Int64("duration_ms", duration.Milliseconds()).
		Float64("duration_seconds", duration.Seconds()).
		Bool("success", success).
		Int("retry_count", retryCount).
		Time("timestamp", time.Now()).
		Msg("Job execution performance")
}

// LogQueueDepth logs queue depth and processing rate metrics
func (pl *PerformanceLogger) LogQueueDepth(queueName string, depth int64, processingRate float64) {
	pl.logger.Info().
		Str("metric_type", "queue_depth").
		Str("queue_name", queueName).
		Int64("queue_depth", depth).
		Float64("processing_rate_per_second", processingRate).
		Time("timestamp", time.Now()).
		Msg("Queue depth metrics")
}

// LogWorkerHealth logs worker health and resource usage
func (pl *PerformanceLogger) LogWorkerHealth(activeWorkers, totalWorkers int, memoryUsageMB int64, cpuPercent float64) {
	pl.logger.Info().
		Str("metric_type", "worker_health").
		Int("active_workers", activeWorkers).
		Int("total_workers", totalWorkers).
		Float64("worker_utilization_percent", float64(activeWorkers)/float64(totalWorkers)*100).
		Int64("memory_usage_mb", memoryUsageMB).
		Float64("cpu_percent", cpuPercent).
		Time("timestamp", time.Now()).
		Msg("Worker health metrics")
}

// LogJobQueueMetrics logs comprehensive job queue metrics
func (pl *PerformanceLogger) LogJobQueueMetrics(metrics JobQueueMetrics) {
	pl.logger.Info().
		Str("metric_type", "job_queue_metrics").
		Int64("total_pending", metrics.TotalPending).
		Int64("total_active", metrics.TotalActive).
		Int64("total_scheduled", metrics.TotalScheduled).
		Int64("total_retry", metrics.TotalRetry).
		Int64("total_dead", metrics.TotalDead).
		Int64("processed_last_minute", metrics.ProcessedLastMinute).
		Int64("failed_last_minute", metrics.FailedLastMinute).
		Float64("success_rate_percent", metrics.SuccessRatePercent).
		Time("timestamp", time.Now()).
		Msg("Job queue comprehensive metrics")
}

// JobQueueMetrics holds comprehensive queue metrics
type JobQueueMetrics struct {
	TotalPending         int64
	TotalActive          int64
	TotalScheduled       int64
	TotalRetry           int64
	TotalDead            int64
	ProcessedLastMinute  int64
	FailedLastMinute     int64
	SuccessRatePercent   float64
}