package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter represents a rate limiter with configurable limits
type RateLimiter struct {
	requests map[string]*ClientInfo
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
	cleanup  time.Duration
}

// ClientInfo stores information about a client's requests
type ClientInfo struct {
	requests  []time.Time
	lastSeen  time.Time
	blocked   bool
	blockTime time.Time
}

// RateLimiterConfig represents rate limiter configuration
type RateLimiterConfig struct {
	RequestsPerWindow int           // Number of requests allowed per window
	Window            time.Duration // Time window for rate limiting
	CleanupInterval   time.Duration // How often to clean up old entries
	BlockDuration     time.Duration // How long to block after exceeding limit
}

// DefaultRateLimiterConfig returns a default rate limiter configuration
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerWindow: 100,                // 100 requests
		Window:            time.Minute,        // per minute
		CleanupInterval:   5 * time.Minute,    // cleanup every 5 minutes
		BlockDuration:     time.Minute,        // block for 1 minute after exceeding
	}
}

// StrictRateLimiterConfig returns a stricter rate limiter configuration
func StrictRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerWindow: 30,                 // 30 requests
		Window:            time.Minute,        // per minute
		CleanupInterval:   2 * time.Minute,    // cleanup every 2 minutes
		BlockDuration:     5 * time.Minute,    // block for 5 minutes after exceeding
	}
}

// AuthRateLimiterConfig returns a rate limiter configuration for authentication endpoints
func AuthRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerWindow: 5,                  // 5 requests
		Window:            time.Minute,        // per minute
		CleanupInterval:   time.Minute,        // cleanup every minute
		BlockDuration:     10 * time.Minute,   // block for 10 minutes after exceeding
	}
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*ClientInfo),
		limit:    config.RequestsPerWindow,
		window:   config.Window,
		cleanup:  config.CleanupInterval,
	}

	// Start cleanup goroutine
	go rl.cleanupRoutine(config.BlockDuration)

	return rl
}

// Allow checks if a request from the given client should be allowed
func (rl *RateLimiter) Allow(clientID string) (bool, int, time.Duration) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	client, exists := rl.requests[clientID]

	if !exists {
		client = &ClientInfo{
			requests: make([]time.Time, 0),
			lastSeen: now,
		}
		rl.requests[clientID] = client
	}

	// Check if client is currently blocked
	if client.blocked && now.Before(client.blockTime) {
		remaining := client.blockTime.Sub(now)
		return false, 0, remaining
	}

	// Unblock client if block time has passed
	if client.blocked && now.After(client.blockTime) {
		client.blocked = false
		client.requests = make([]time.Time, 0) // Reset request history
	}

	// Remove old requests outside the window
	cutoff := now.Add(-rl.window)
	validRequests := make([]time.Time, 0)
	for _, reqTime := range client.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	client.requests = validRequests

	// Check if limit is exceeded
	if len(client.requests) >= rl.limit {
		client.blocked = true
		client.blockTime = now.Add(10 * time.Minute) // Block for 10 minutes
		return false, 0, 10 * time.Minute
	}

	// Allow request and record it
	client.requests = append(client.requests, now)
	client.lastSeen = now
	remaining := rl.limit - len(client.requests)

	return true, remaining, 0
}

// cleanupRoutine periodically removes old client entries
func (rl *RateLimiter) cleanupRoutine(blockDuration time.Duration) {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.cleanup)

		for clientID, client := range rl.requests {
			// Remove clients that haven't been seen for a while and are not blocked
			if client.lastSeen.Before(cutoff) && (!client.blocked || now.After(client.blockTime)) {
				delete(rl.requests, clientID)
			}
		}
		rl.mutex.Unlock()
	}
}

// RateLimit returns a rate limiting middleware
func RateLimit(config RateLimiterConfig) gin.HandlerFunc {
	limiter := NewRateLimiter(config)

	return func(c *gin.Context) {
		// Use IP address as client identifier
		clientID := c.ClientIP()

		// For authenticated users, use user ID if available
		if userID, exists := c.Get("user_id"); exists {
			if id, ok := userID.(int); ok {
				clientID = fmt.Sprintf("user_%d", id)
			}
		}

		allowed, remaining, retryAfter := limiter.Allow(clientID)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerWindow))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Window", config.Window.String())

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "rate_limit_exceeded",
				Message: "Too many requests. Please try again later.",
				Code:    http.StatusTooManyRequests,
				Details: map[string]string{
					"retry_after": retryAfter.String(),
					"limit":       strconv.Itoa(config.RequestsPerWindow),
					"window":      config.Window.String(),
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByIP returns a rate limiting middleware that limits by IP address only
func RateLimitByIP(config RateLimiterConfig) gin.HandlerFunc {
	limiter := NewRateLimiter(config)

	return func(c *gin.Context) {
		clientID := c.ClientIP()

		allowed, remaining, retryAfter := limiter.Allow(clientID)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerWindow))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Window", config.Window.String())

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "rate_limit_exceeded",
				Message: "Too many requests from this IP. Please try again later.",
				Code:    http.StatusTooManyRequests,
				Details: map[string]string{
					"retry_after": retryAfter.String(),
					"limit":       strconv.Itoa(config.RequestsPerWindow),
					"window":      config.Window.String(),
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}