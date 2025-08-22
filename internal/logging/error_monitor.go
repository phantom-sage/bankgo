package logging

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// ErrorFrequency tracks error frequency for monitoring and alerting
type ErrorFrequency struct {
	Category    ErrorCategory `json:"category"`
	Component   string        `json:"component"`
	Operation   string        `json:"operation"`
	Count       int64         `json:"count"`
	LastOccurred time.Time    `json:"last_occurred"`
	FirstOccurred time.Time   `json:"first_occurred"`
}

// AlertThreshold defines thresholds for error alerting
type AlertThreshold struct {
	Category      ErrorCategory `json:"category"`
	Component     string        `json:"component,omitempty"`
	Operation     string        `json:"operation,omitempty"`
	MaxCount      int64         `json:"max_count"`
	TimeWindow    time.Duration `json:"time_window"`
	Severity      ErrorSeverity `json:"severity"`
	AlertInterval time.Duration `json:"alert_interval"`
}

// ErrorMonitorConfig configures the error monitoring system
type ErrorMonitorConfig struct {
	// Enable error frequency tracking
	EnableTracking bool `json:"enable_tracking"`
	
	// Enable alerting
	EnableAlerting bool `json:"enable_alerting"`
	
	// Cleanup interval for old error frequency data
	CleanupInterval time.Duration `json:"cleanup_interval"`
	
	// Maximum age for error frequency data
	MaxAge time.Duration `json:"max_age"`
	
	// Default alert thresholds
	DefaultThresholds []AlertThreshold `json:"default_thresholds"`
}

// ErrorMonitor tracks error frequencies and handles alerting
type ErrorMonitor struct {
	config     ErrorMonitorConfig
	logger     zerolog.Logger
	
	// Error frequency tracking
	frequencies map[string]*ErrorFrequency
	freqMutex   sync.RWMutex
	
	// Alert thresholds
	thresholds map[string]AlertThreshold
	threshMutex sync.RWMutex
	
	// Alert state tracking
	lastAlerts map[string]time.Time
	alertMutex sync.RWMutex
	
	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewErrorMonitor creates a new error monitoring instance
func NewErrorMonitor(config ErrorMonitorConfig, logger zerolog.Logger) *ErrorMonitor {
	em := &ErrorMonitor{
		config:      config,
		logger:      logger.With().Str("component", "error_monitor").Logger(),
		frequencies: make(map[string]*ErrorFrequency),
		thresholds:  make(map[string]AlertThreshold),
		lastAlerts:  make(map[string]time.Time),
		stopCleanup: make(chan struct{}),
	}
	
	// Set up default thresholds
	for _, threshold := range config.DefaultThresholds {
		em.AddThreshold(threshold)
	}
	
	// Start cleanup routine if tracking is enabled
	if config.EnableTracking && config.CleanupInterval > 0 {
		em.startCleanupRoutine()
	}
	
	return em
}

// TrackError records an error occurrence for frequency tracking
func (em *ErrorMonitor) TrackError(ctx ErrorContext) {
	if !em.config.EnableTracking {
		return
	}
	
	key := em.getFrequencyKey(ctx)
	now := time.Now()
	
	em.freqMutex.Lock()
	defer em.freqMutex.Unlock()
	
	freq, exists := em.frequencies[key]
	if !exists {
		freq = &ErrorFrequency{
			Category:      ctx.Category,
			Component:     ctx.Component,
			Operation:     ctx.Operation,
			Count:         0,
			FirstOccurred: now,
		}
		em.frequencies[key] = freq
	}
	
	freq.Count++
	freq.LastOccurred = now
	
	// Check for alert conditions
	if em.config.EnableAlerting {
		go em.checkAlertConditions(key, freq)
	}
}

// AddThreshold adds or updates an alert threshold
func (em *ErrorMonitor) AddThreshold(threshold AlertThreshold) {
	key := em.getThresholdKey(threshold)
	
	em.threshMutex.Lock()
	defer em.threshMutex.Unlock()
	
	em.thresholds[key] = threshold
	
	em.logger.Info().
		Str("threshold_key", key).
		Str("category", string(threshold.Category)).
		Str("component", threshold.Component).
		Str("operation", threshold.Operation).
		Int64("max_count", threshold.MaxCount).
		Dur("time_window", threshold.TimeWindow).
		Msg("Alert threshold added")
}

// RemoveThreshold removes an alert threshold
func (em *ErrorMonitor) RemoveThreshold(category ErrorCategory, component, operation string) {
	threshold := AlertThreshold{
		Category:  category,
		Component: component,
		Operation: operation,
	}
	key := em.getThresholdKey(threshold)
	
	em.threshMutex.Lock()
	defer em.threshMutex.Unlock()
	
	delete(em.thresholds, key)
	
	em.logger.Info().
		Str("threshold_key", key).
		Str("category", string(category)).
		Str("component", component).
		Str("operation", operation).
		Msg("Alert threshold removed")
}

// GetErrorFrequencies returns current error frequency data
func (em *ErrorMonitor) GetErrorFrequencies() map[string]*ErrorFrequency {
	em.freqMutex.RLock()
	defer em.freqMutex.RUnlock()
	
	// Return a copy to avoid race conditions
	result := make(map[string]*ErrorFrequency)
	for key, freq := range em.frequencies {
		freqCopy := *freq
		result[key] = &freqCopy
	}
	
	return result
}

// GetErrorFrequency returns frequency data for a specific error pattern
func (em *ErrorMonitor) GetErrorFrequency(category ErrorCategory, component, operation string) *ErrorFrequency {
	ctx := ErrorContext{
		Category:  category,
		Component: component,
		Operation: operation,
	}
	key := em.getFrequencyKey(ctx)
	
	em.freqMutex.RLock()
	defer em.freqMutex.RUnlock()
	
	if freq, exists := em.frequencies[key]; exists {
		freqCopy := *freq
		return &freqCopy
	}
	
	return nil
}

// GetThresholds returns current alert thresholds
func (em *ErrorMonitor) GetThresholds() map[string]AlertThreshold {
	em.threshMutex.RLock()
	defer em.threshMutex.RUnlock()
	
	// Return a copy to avoid race conditions
	result := make(map[string]AlertThreshold)
	for key, threshold := range em.thresholds {
		result[key] = threshold
	}
	
	return result
}

// ResetErrorFrequency resets frequency data for a specific error pattern
func (em *ErrorMonitor) ResetErrorFrequency(category ErrorCategory, component, operation string) {
	ctx := ErrorContext{
		Category:  category,
		Component: component,
		Operation: operation,
	}
	key := em.getFrequencyKey(ctx)
	
	em.freqMutex.Lock()
	defer em.freqMutex.Unlock()
	
	delete(em.frequencies, key)
	
	em.logger.Info().
		Str("frequency_key", key).
		Str("category", string(category)).
		Str("component", component).
		Str("operation", operation).
		Msg("Error frequency reset")
}

// ClearAllFrequencies clears all error frequency data
func (em *ErrorMonitor) ClearAllFrequencies() {
	em.freqMutex.Lock()
	defer em.freqMutex.Unlock()
	
	count := len(em.frequencies)
	em.frequencies = make(map[string]*ErrorFrequency)
	
	em.logger.Info().
		Int("cleared_count", count).
		Msg("All error frequencies cleared")
}

// Close stops the error monitor and cleanup routines
func (em *ErrorMonitor) Close() error {
	if em.cleanupTicker != nil {
		em.cleanupTicker.Stop()
		close(em.stopCleanup)
	}
	
	em.logger.Info().Msg("Error monitor stopped")
	return nil
}

// getFrequencyKey generates a unique key for error frequency tracking
func (em *ErrorMonitor) getFrequencyKey(ctx ErrorContext) string {
	return fmt.Sprintf("%s:%s:%s", ctx.Category, ctx.Component, ctx.Operation)
}

// getThresholdKey generates a unique key for alert thresholds
func (em *ErrorMonitor) getThresholdKey(threshold AlertThreshold) string {
	return fmt.Sprintf("%s:%s:%s", threshold.Category, threshold.Component, threshold.Operation)
}

// checkAlertConditions checks if any alert thresholds are exceeded
func (em *ErrorMonitor) checkAlertConditions(freqKey string, freq *ErrorFrequency) {
	em.threshMutex.RLock()
	defer em.threshMutex.RUnlock()
	
	now := time.Now()
	
	// Check all thresholds that might apply to this error
	for thresholdKey, threshold := range em.thresholds {
		if em.thresholdMatches(threshold, freq) {
			// Check if threshold is exceeded within time window
			windowStart := now.Add(-threshold.TimeWindow)
			if freq.LastOccurred.After(windowStart) && freq.Count >= threshold.MaxCount {
				// Check if we should send an alert (respect alert interval)
				em.alertMutex.RLock()
				lastAlert, hasLastAlert := em.lastAlerts[thresholdKey]
				em.alertMutex.RUnlock()
				
				shouldAlert := !hasLastAlert || now.Sub(lastAlert) >= threshold.AlertInterval
				
				if shouldAlert {
					em.sendAlert(threshold, freq)
					
					em.alertMutex.Lock()
					em.lastAlerts[thresholdKey] = now
					em.alertMutex.Unlock()
				}
			}
		}
	}
}

// thresholdMatches checks if a threshold applies to an error frequency
func (em *ErrorMonitor) thresholdMatches(threshold AlertThreshold, freq *ErrorFrequency) bool {
	// Category must match
	if threshold.Category != freq.Category {
		return false
	}
	
	// Component must match if specified
	if threshold.Component != "" && threshold.Component != freq.Component {
		return false
	}
	
	// Operation must match if specified
	if threshold.Operation != "" && threshold.Operation != freq.Operation {
		return false
	}
	
	return true
}

// sendAlert sends an alert for a threshold violation
func (em *ErrorMonitor) sendAlert(threshold AlertThreshold, freq *ErrorFrequency) {
	em.logger.Warn().
		Str("alert_type", "error_threshold_exceeded").
		Str("category", string(threshold.Category)).
		Str("component", threshold.Component).
		Str("operation", threshold.Operation).
		Int64("error_count", freq.Count).
		Int64("threshold_max", threshold.MaxCount).
		Dur("time_window", threshold.TimeWindow).
		Str("severity", string(threshold.Severity)).
		Time("first_occurred", freq.FirstOccurred).
		Time("last_occurred", freq.LastOccurred).
		Msg("Error threshold exceeded - alert triggered")
}

// startCleanupRoutine starts the background cleanup routine
func (em *ErrorMonitor) startCleanupRoutine() {
	em.cleanupTicker = time.NewTicker(em.config.CleanupInterval)
	
	go func() {
		for {
			select {
			case <-em.cleanupTicker.C:
				em.cleanupOldFrequencies()
			case <-em.stopCleanup:
				return
			}
		}
	}()
	
	em.logger.Info().
		Dur("cleanup_interval", em.config.CleanupInterval).
		Dur("max_age", em.config.MaxAge).
		Msg("Error monitor cleanup routine started")
}

// cleanupOldFrequencies removes old error frequency data
func (em *ErrorMonitor) cleanupOldFrequencies() {
	if em.config.MaxAge <= 0 {
		return
	}
	
	cutoff := time.Now().Add(-em.config.MaxAge)
	
	em.freqMutex.Lock()
	defer em.freqMutex.Unlock()
	
	var removed []string
	for key, freq := range em.frequencies {
		if freq.LastOccurred.Before(cutoff) {
			delete(em.frequencies, key)
			removed = append(removed, key)
		}
	}
	
	if len(removed) > 0 {
		em.logger.Debug().
			Int("removed_count", len(removed)).
			Strs("removed_keys", removed).
			Time("cutoff", cutoff).
			Msg("Cleaned up old error frequencies")
	}
}

// GetErrorStats returns aggregated error statistics
func (em *ErrorMonitor) GetErrorStats() map[ErrorCategory]int64 {
	em.freqMutex.RLock()
	defer em.freqMutex.RUnlock()
	
	stats := make(map[ErrorCategory]int64)
	
	for _, freq := range em.frequencies {
		stats[freq.Category] += freq.Count
	}
	
	return stats
}

// GetTopErrors returns the most frequent errors
func (em *ErrorMonitor) GetTopErrors(limit int) []*ErrorFrequency {
	em.freqMutex.RLock()
	defer em.freqMutex.RUnlock()
	
	// Convert to slice for sorting
	frequencies := make([]*ErrorFrequency, 0, len(em.frequencies))
	for _, freq := range em.frequencies {
		freqCopy := *freq
		frequencies = append(frequencies, &freqCopy)
	}
	
	// Sort by count (descending)
	for i := 0; i < len(frequencies)-1; i++ {
		for j := i + 1; j < len(frequencies); j++ {
			if frequencies[i].Count < frequencies[j].Count {
				frequencies[i], frequencies[j] = frequencies[j], frequencies[i]
			}
		}
	}
	
	// Return top N
	if limit > 0 && limit < len(frequencies) {
		return frequencies[:limit]
	}
	
	return frequencies
}

// DefaultErrorMonitorConfig returns a default configuration for error monitoring
func DefaultErrorMonitorConfig() ErrorMonitorConfig {
	return ErrorMonitorConfig{
		EnableTracking:  true,
		EnableAlerting:  true,
		CleanupInterval: 1 * time.Hour,
		MaxAge:          24 * time.Hour,
		DefaultThresholds: []AlertThreshold{
			{
				Category:      ValidationError,
				MaxCount:      100,
				TimeWindow:    5 * time.Minute,
				Severity:      LowSeverity,
				AlertInterval: 15 * time.Minute,
			},
			{
				Category:      AuthenticationError,
				MaxCount:      50,
				TimeWindow:    5 * time.Minute,
				Severity:      MediumSeverity,
				AlertInterval: 10 * time.Minute,
			},
			{
				Category:      BusinessLogicError,
				MaxCount:      25,
				TimeWindow:    5 * time.Minute,
				Severity:      MediumSeverity,
				AlertInterval: 10 * time.Minute,
			},
			{
				Category:      SystemError,
				MaxCount:      10,
				TimeWindow:    5 * time.Minute,
				Severity:      HighSeverity,
				AlertInterval: 5 * time.Minute,
			},
			{
				Category:      DatabaseError,
				MaxCount:      10,
				TimeWindow:    5 * time.Minute,
				Severity:      HighSeverity,
				AlertInterval: 5 * time.Minute,
			},
		},
	}
}