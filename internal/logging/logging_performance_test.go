package logging

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BenchmarkLoggerManager_BasicLogging benchmarks basic logging operations
func BenchmarkLoggerManager_BasicLogging(b *testing.B) {
	tempDir := b.TempDir()
	
	config := LogConfig{
		Level:     "info",
		Format:    "json",
		Output:    "file",
		Directory: tempDir,
	}
	
	lm, err := NewLoggerManager(config)
	require.NoError(b, err)
	defer lm.Close()
	
	logger := lm.GetLogger()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logger.Info().
				Int("iteration", i).
				Str("request_id", fmt.Sprintf("req-%d", i)).
				Msg("benchmark log message")
			i++
		}
	})
}

// BenchmarkContextLogger_WithFieldsIntegration benchmarks context logger with fields integration
func BenchmarkContextLogger_WithFieldsIntegration(b *testing.B) {
	logger := zerolog.New(zerolog.NewTestWriter(b))
	
	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "req-123")
	ctx = context.WithValue(ctx, "user_id", int64(456))
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cl := NewContextLogger(logger, ctx)
			cl.WithComponent("test").
				WithOperation("benchmark").
				WithField("iteration", 1).
				Info().Msg("benchmark message")
		}
	})
}

// BenchmarkAuditLogger_LogTransfer benchmarks audit logging
func BenchmarkAuditLogger_LogTransfer(b *testing.B) {
	logger := zerolog.New(zerolog.NewTestWriter(b))
	auditLogger := NewAuditLogger(logger)
	
	amount := decimal.NewFromFloat(100.50)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			auditLogger.LogTransfer(123, 456, amount, "success")
		}
	})
}

// BenchmarkPerformanceLogger_LogHTTPRequest benchmarks performance logging
func BenchmarkPerformanceLogger_LogHTTPRequest(b *testing.B) {
	logger := zerolog.New(zerolog.NewTestWriter(b))
	perfLogger := NewPerformanceLogger(logger)
	
	duration := 150 * time.Millisecond
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			perfLogger.LogHTTPRequest("GET", "/api/users", duration, 200)
		}
	})
}

// BenchmarkErrorLogger_LogError benchmarks error logging
func BenchmarkErrorLogger_LogError(b *testing.B) {
	logger := zerolog.New(zerolog.NewTestWriter(b))
	errorLogger := NewErrorLogger(logger)
	
	err := errors.New("benchmark error")
	ctx := ErrorContext{
		RequestID: "req-123",
		UserID:    456,
		Operation: "benchmark_operation",
		Component: "benchmark_component",
		Category:  ValidationError,
		Severity:  MediumSeverity,
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			errorLogger.LogError(err, ctx)
		}
	})
}

// TestLoggingThroughput tests logging throughput under various conditions
func TestLoggingThroughput(t *testing.T) {
	tests := []struct {
		name           string
		numGoroutines  int
		logsPerRoutine int
		expectedMinTPS float64 // Minimum transactions per second
	}{
		{
			name:           "single threaded",
			numGoroutines:  1,
			logsPerRoutine: 10000,
			expectedMinTPS: 5000,
		},
		{
			name:           "multi threaded",
			numGoroutines:  4,
			logsPerRoutine: 2500,
			expectedMinTPS: 8000,
		},
		{
			name:           "high concurrency",
			numGoroutines:  10,
			logsPerRoutine: 1000,
			expectedMinTPS: 10000,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			
			config := LogConfig{
				Level:     "info",
				Format:    "json",
				Output:    "file",
				Directory: tempDir,
			}
			
			lm, err := NewLoggerManager(config)
			require.NoError(t, err)
			defer lm.Close()
			
			logger := lm.GetLogger()
			
			startTime := time.Now()
			
			var wg sync.WaitGroup
			wg.Add(tt.numGoroutines)
			
			for i := 0; i < tt.numGoroutines; i++ {
				go func(routineID int) {
					defer wg.Done()
					
					for j := 0; j < tt.logsPerRoutine; j++ {
						logger.Info().
							Int("routine", routineID).
							Int("iteration", j).
							Str("data", fmt.Sprintf("data-%d-%d", routineID, j)).
							Msg("throughput test message")
					}
				}(i)
			}
			
			wg.Wait()
			
			// Sync to ensure all logs are written
			err = lm.Sync()
			require.NoError(t, err)
			
			duration := time.Since(startTime)
			totalLogs := tt.numGoroutines * tt.logsPerRoutine
			tps := float64(totalLogs) / duration.Seconds()
			
			t.Logf("Throughput test results:")
			t.Logf("  Goroutines: %d", tt.numGoroutines)
			t.Logf("  Logs per routine: %d", tt.logsPerRoutine)
			t.Logf("  Total logs: %d", totalLogs)
			t.Logf("  Duration: %v", duration)
			t.Logf("  Throughput: %.2f logs/sec", tps)
			
			assert.Greater(t, tps, tt.expectedMinTPS, "Throughput should meet minimum requirement")
		})
	}
}

// TestMemoryUsageStability tests memory usage stability over time
func TestMemoryUsageStability(t *testing.T) {
	tempDir := t.TempDir()
	
	config := LogConfig{
		Level:     "info",
		Format:    "json",
		Output:    "file",
		Directory: tempDir,
	}
	
	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()
	
	logger := lm.GetLogger()
	
	// Measure memory usage over multiple cycles
	const cycles = 5
	const logsPerCycle = 2000
	
	var memStats []runtime.MemStats
	
	for cycle := 0; cycle < cycles; cycle++ {
		// Force GC before measurement
		runtime.GC()
		runtime.GC() // Call twice to ensure cleanup
		
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		memStats = append(memStats, m)
		
		// Generate logs
		for i := 0; i < logsPerCycle; i++ {
			logger.Info().
				Int("cycle", cycle).
				Int("iteration", i).
				Str("data", fmt.Sprintf("cycle-%d-data-%d", cycle, i)).
				Msg("memory stability test")
		}
		
		// Sync logs
		err = lm.Sync()
		require.NoError(t, err)
		
		t.Logf("Cycle %d completed, current alloc: %d bytes", cycle, m.Alloc)
	}
	
	// Analyze memory growth
	initialAlloc := memStats[0].Alloc
	finalAlloc := memStats[len(memStats)-1].Alloc
	
	// Calculate growth rate
	totalLogs := cycles * logsPerCycle
	memoryGrowth := finalAlloc - initialAlloc
	memoryPerLog := float64(memoryGrowth) / float64(totalLogs)
	
	t.Logf("Memory stability analysis:")
	t.Logf("  Initial alloc: %d bytes", initialAlloc)
	t.Logf("  Final alloc: %d bytes", finalAlloc)
	t.Logf("  Memory growth: %d bytes", memoryGrowth)
	t.Logf("  Memory per log: %.2f bytes", memoryPerLog)
	
	// Memory growth should be reasonable
	assert.Less(t, memoryPerLog, 50.0, "Memory growth per log should be less than 50 bytes")
	
	// Check for memory leaks by comparing first and last cycles
	firstCycleAlloc := memStats[1].Alloc - memStats[0].Alloc
	lastCycleAlloc := memStats[len(memStats)-1].Alloc - memStats[len(memStats)-2].Alloc
	
	// Last cycle shouldn't use significantly more memory than first cycle
	growthRatio := float64(lastCycleAlloc) / float64(firstCycleAlloc)
	assert.Less(t, growthRatio, 2.0, "Memory usage shouldn't grow significantly between cycles")
}

// TestConcurrentRotationPerformance tests performance during log rotation
func TestConcurrentRotationPerformance(t *testing.T) {
	tempDir := t.TempDir()
	
	config := LogConfig{
		Level:     "info",
		Format:    "json",
		Output:    "file",
		Directory: tempDir,
	}
	
	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()
	
	logger := lm.GetLogger()
	
	const testDuration = 2 * time.Second
	const rotationInterval = 200 * time.Millisecond
	
	var totalLogs int64
	var mu sync.Mutex
	
	stopChan := make(chan bool)
	var wg sync.WaitGroup
	
	// Start logging goroutines
	numLoggers := 4
	wg.Add(numLoggers)
	
	for i := 0; i < numLoggers; i++ {
		go func(loggerID int) {
			defer wg.Done()
			
			logCount := 0
			for {
				select {
				case <-stopChan:
					mu.Lock()
					totalLogs += int64(logCount)
					mu.Unlock()
					return
				default:
					logger.Info().
						Int("logger_id", loggerID).
						Int("count", logCount).
						Msg("concurrent rotation test")
					logCount++
					
					// Small delay to make rotation more likely
					time.Sleep(time.Microsecond * 100)
				}
			}
		}(i)
	}
	
	// Start rotation goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		ticker := time.NewTicker(rotationInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				err := lm.RotateLogFile()
				if err != nil {
					t.Logf("Rotation error: %v", err)
				}
			}
		}
	}()
	
	// Run test for specified duration
	time.Sleep(testDuration)
	close(stopChan)
	wg.Wait()
	
	// Final sync
	err = lm.Sync()
	require.NoError(t, err)
	
	// Calculate performance metrics
	logsPerSecond := float64(totalLogs) / testDuration.Seconds()
	
	t.Logf("Concurrent rotation performance:")
	t.Logf("  Test duration: %v", testDuration)
	t.Logf("  Total logs: %d", totalLogs)
	t.Logf("  Logs per second: %.2f", logsPerSecond)
	t.Logf("  Rotation interval: %v", rotationInterval)
	
	// Should maintain reasonable performance even with frequent rotations
	assert.Greater(t, logsPerSecond, 1000.0, "Should maintain at least 1000 logs/sec during rotation")
	assert.Greater(t, totalLogs, int64(2000), "Should log at least 2000 entries during test")
}

// TestLargeLogEntryPerformance tests performance with large log entries
func TestLargeLogEntryPerformance(t *testing.T) {
	tempDir := t.TempDir()
	
	config := LogConfig{
		Level:     "info",
		Format:    "json",
		Output:    "file",
		Directory: tempDir,
	}
	
	lm, err := NewLoggerManager(config)
	require.NoError(t, err)
	defer lm.Close()
	
	logger := lm.GetLogger()
	
	// Test different log entry sizes
	testSizes := []struct {
		name     string
		dataSize int
		numLogs  int
	}{
		{"small entries", 100, 5000},
		{"medium entries", 1024, 2000},
		{"large entries", 10240, 500},
		{"very large entries", 102400, 100},
	}
	
	for _, ts := range testSizes {
		t.Run(ts.name, func(t *testing.T) {
			data := make([]byte, ts.dataSize)
			for i := range data {
				data[i] = byte('A' + (i % 26))
			}
			dataStr := string(data)
			
			startTime := time.Now()
			
			for i := 0; i < ts.numLogs; i++ {
				logger.Info().
					Int("iteration", i).
					Str("large_data", dataStr).
					Int("data_size", ts.dataSize).
					Msg("large entry performance test")
			}
			
			err = lm.Sync()
			require.NoError(t, err)
			
			duration := time.Since(startTime)
			logsPerSecond := float64(ts.numLogs) / duration.Seconds()
			bytesPerSecond := float64(ts.numLogs*ts.dataSize) / duration.Seconds()
			
			t.Logf("Large entry performance (%s):", ts.name)
			t.Logf("  Entry size: %d bytes", ts.dataSize)
			t.Logf("  Number of logs: %d", ts.numLogs)
			t.Logf("  Duration: %v", duration)
			t.Logf("  Logs per second: %.2f", logsPerSecond)
			t.Logf("  Bytes per second: %.2f", bytesPerSecond)
			
			// Performance should degrade gracefully with larger entries
			minLogsPerSecond := 100.0 // Even very large entries should achieve 100/sec
			if ts.dataSize <= 1024 {
				minLogsPerSecond = 1000.0
			} else if ts.dataSize <= 10240 {
				minLogsPerSecond = 500.0
			}
			
			assert.Greater(t, logsPerSecond, minLogsPerSecond, 
				"Should maintain minimum performance for %s", ts.name)
		})
	}
}