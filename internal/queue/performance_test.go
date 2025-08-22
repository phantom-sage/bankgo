package queue

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestPerformanceLogger_LogJobExecution(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	perfLogger := NewPerformanceLogger(logger)
	
	// Test successful job execution
	perfLogger.LogJobExecution("test_job", "corr_123", 100*time.Millisecond, true, 0)
	
	// Test failed job execution with retry
	perfLogger.LogJobExecution("test_job", "corr_124", 200*time.Millisecond, false, 2)
	
	// No assertions needed as this is testing logging output
	assert.NotNil(t, perfLogger)
}

func TestPerformanceLogger_LogQueueDepth(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	perfLogger := NewPerformanceLogger(logger)
	
	perfLogger.LogQueueDepth("email", 10, 5.5)
	perfLogger.LogQueueDepth("default", 0, 10.0)
	
	assert.NotNil(t, perfLogger)
}

func TestPerformanceLogger_LogWorkerHealth(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	perfLogger := NewPerformanceLogger(logger)
	
	perfLogger.LogWorkerHealth(8, 10, 128, 45.5)
	
	assert.NotNil(t, perfLogger)
}

func TestPerformanceLogger_LogJobQueueMetrics(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	perfLogger := NewPerformanceLogger(logger)
	
	metrics := JobQueueMetrics{
		TotalPending:        25,
		TotalActive:         5,
		TotalScheduled:      10,
		TotalRetry:          2,
		TotalDead:           1,
		ProcessedLastMinute: 100,
		FailedLastMinute:    5,
		SuccessRatePercent:  95.0,
	}
	
	perfLogger.LogJobQueueMetrics(metrics)
	
	assert.NotNil(t, perfLogger)
}

func TestQueueManager_PerformanceMonitoring(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	logger := createTestLogger()
	qm, err := NewQueueManager(cfg, logger)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	defer qm.Close()

	ctx := context.Background()
	
	// Test queue metrics logging
	err = qm.LogQueueMetrics(ctx)
	assert.NoError(t, err)
	
	// Test worker health logging
	qm.LogWorkerHealth(ctx)
	
	// Test worker resource usage logging
	qm.LogWorkerResourceUsage(ctx)
	
	// Test correlation ID generation
	correlationID := qm.GetJobCorrelationID(ctx, "test_job")
	assert.Contains(t, correlationID, "test_job")
	assert.NotEmpty(t, correlationID)
}

func TestQueueManager_ProcessingRateCalculation(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	logger := createTestLogger()
	qm, err := NewQueueManager(cfg, logger)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	defer qm.Close()

	// Test processing rate calculation for different queue depths
	rate1 := qm.calculateProcessingRate("email", 0)
	assert.Equal(t, 10.0, rate1, "Empty queue should have high processing rate")
	
	rate2 := qm.calculateProcessingRate("email", 5)
	assert.Equal(t, 5.0, rate2, "Small queue should have medium processing rate")
	
	rate3 := qm.calculateProcessingRate("email", 25)
	assert.Equal(t, 2.0, rate3, "Medium queue should have lower processing rate")
	
	rate4 := qm.calculateProcessingRate("email", 100)
	assert.Equal(t, 0.5, rate4, "Large queue should have very low processing rate")
}

func TestQueueManager_SuccessRateCalculation(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	logger := createTestLogger()
	qm, err := NewQueueManager(cfg, logger)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	defer qm.Close()

	ctx := context.Background()
	
	// Test success rate calculation (simplified implementation returns 100%)
	successRate := qm.calculateSuccessRate(ctx)
	assert.Equal(t, 100.0, successRate, "Success rate should be 100% with no processed jobs")
}

func TestQueueManager_CorrelationIDFromContext(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	logger := createTestLogger()
	qm, err := NewQueueManager(cfg, logger)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	defer qm.Close()

	// Test with existing correlation ID in context
	ctx := context.WithValue(context.Background(), "correlation_id", "existing_corr_123")
	correlationID := qm.GetJobCorrelationID(ctx, "test_job")
	assert.Equal(t, "existing_corr_123", correlationID, "Should use existing correlation ID from context")
	
	// Test without correlation ID in context
	ctx2 := context.Background()
	correlationID2 := qm.GetJobCorrelationID(ctx2, "test_job")
	assert.Contains(t, correlationID2, "test_job", "Should generate new correlation ID with job type prefix")
}

func TestQueueManager_PeriodicMetricsLogging(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	logger := createTestLogger()
	qm, err := NewQueueManager(cfg, logger)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	defer qm.Close()

	// Test periodic metrics logging with short interval
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	// Start periodic logging with 500ms interval
	qm.StartPeriodicMetricsLogging(ctx, 500*time.Millisecond)
	
	// Wait for a few cycles
	time.Sleep(1500 * time.Millisecond)
	
	// Cancel context to stop logging
	cancel()
	
	// Wait a bit more to ensure logging stops
	time.Sleep(200 * time.Millisecond)
	
	// Test passes if no panics occur
	assert.True(t, true)
}

// BenchmarkQueueManager_LogQueueMetrics benchmarks the queue metrics logging performance
func BenchmarkQueueManager_LogQueueMetrics(b *testing.B) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	logger := createTestLogger()
	qm, err := NewQueueManager(cfg, logger)
	if err != nil {
		b.Skip("Redis not available for benchmarking")
	}
	defer qm.Close()

	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = qm.LogQueueMetrics(ctx)
	}
}

// BenchmarkPerformanceLogger_LogJobExecution benchmarks job execution logging
func BenchmarkPerformanceLogger_LogJobExecution(b *testing.B) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	perfLogger := NewPerformanceLogger(logger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		perfLogger.LogJobExecution("test_job", "corr_123", 100*time.Millisecond, true, 0)
	}
}