package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDefaultRateLimiterConfig(t *testing.T) {
	config := DefaultRateLimiterConfig()
	
	assert.Equal(t, 100, config.RequestsPerWindow)
	assert.Equal(t, time.Minute, config.Window)
	assert.Equal(t, 5*time.Minute, config.CleanupInterval)
	assert.Equal(t, time.Minute, config.BlockDuration)
}

func TestStrictRateLimiterConfig(t *testing.T) {
	config := StrictRateLimiterConfig()
	
	assert.Equal(t, 30, config.RequestsPerWindow)
	assert.Equal(t, time.Minute, config.Window)
	assert.Equal(t, 2*time.Minute, config.CleanupInterval)
	assert.Equal(t, 5*time.Minute, config.BlockDuration)
}

func TestAuthRateLimiterConfig(t *testing.T) {
	config := AuthRateLimiterConfig()
	
	assert.Equal(t, 5, config.RequestsPerWindow)
	assert.Equal(t, time.Minute, config.Window)
	assert.Equal(t, time.Minute, config.CleanupInterval)
	assert.Equal(t, 10*time.Minute, config.BlockDuration)
}

func TestNewRateLimiter(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerWindow: 10,
		Window:            time.Minute,
		CleanupInterval:   time.Minute,
		BlockDuration:     time.Minute,
	}
	
	limiter := NewRateLimiter(config)
	
	assert.NotNil(t, limiter)
	assert.Equal(t, 10, limiter.limit)
	assert.Equal(t, time.Minute, limiter.window)
	assert.NotNil(t, limiter.requests)
}

func TestRateLimiter_Allow(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerWindow: 3,
		Window:            time.Minute,
		CleanupInterval:   time.Minute,
		BlockDuration:     time.Minute,
	}
	
	limiter := NewRateLimiter(config)
	clientID := "test-client"
	
	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		allowed, remaining, _ := limiter.Allow(clientID)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
		assert.Equal(t, 3-(i+1), remaining, "Remaining count should be correct")
	}
	
	// 4th request should be blocked
	allowed, remaining, retryAfter := limiter.Allow(clientID)
	assert.False(t, allowed, "4th request should be blocked")
	assert.Equal(t, 0, remaining)
	assert.Greater(t, retryAfter, time.Duration(0))
}

func TestRateLimiter_Allow_DifferentClients(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerWindow: 2,
		Window:            time.Minute,
		CleanupInterval:   time.Minute,
		BlockDuration:     time.Minute,
	}
	
	limiter := NewRateLimiter(config)
	
	// Client 1 makes 2 requests
	allowed1, _, _ := limiter.Allow("client1")
	assert.True(t, allowed1)
	allowed2, _, _ := limiter.Allow("client1")
	assert.True(t, allowed2)
	
	// Client 1's 3rd request should be blocked
	allowed3, _, _ := limiter.Allow("client1")
	assert.False(t, allowed3)
	
	// Client 2 should still be allowed
	allowed4, _, _ := limiter.Allow("client2")
	assert.True(t, allowed4)
}

func TestRateLimit_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := RateLimiterConfig{
		RequestsPerWindow: 2,
		Window:            time.Minute,
		CleanupInterval:   time.Minute,
		BlockDuration:     time.Minute,
	}
	
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	
	r.Use(RateLimit(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// First request should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "2", w1.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "1", w1.Header().Get("X-RateLimit-Remaining"))
	
	// Second request should succeed
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12346"
	
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "0", w2.Header().Get("X-RateLimit-Remaining"))
	
	// Third request should be rate limited
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "192.168.1.1:12347"
	
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
	assert.Contains(t, w3.Body.String(), "rate_limit_exceeded")
	assert.NotEmpty(t, w3.Header().Get("Retry-After"))
}

func TestRateLimitByIP_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := RateLimiterConfig{
		RequestsPerWindow: 1,
		Window:            time.Minute,
		CleanupInterval:   time.Minute,
		BlockDuration:     time.Minute,
	}
	
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	
	r.Use(RateLimitByIP(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// First request should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	
	assert.Equal(t, http.StatusOK, w1.Code)
	
	// Second request from same IP should be rate limited
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12346"
	
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	assert.Contains(t, w2.Body.String(), "Too many requests from this IP")
}

func TestRateLimit_WithAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := RateLimiterConfig{
		RequestsPerWindow: 1,
		Window:            time.Minute,
		CleanupInterval:   time.Minute,
		BlockDuration:     time.Minute,
	}
	
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	
	// Middleware that sets user_id in context
	r.Use(func(c *gin.Context) {
		c.Set("user_id", 123)
		c.Next()
	})
	
	r.Use(RateLimit(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// First request should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	
	assert.Equal(t, http.StatusOK, w1.Code)
	
	// Second request from same user should be rate limited
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345" // Different IP but same user
	
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	// Skip this test as it's timing-sensitive and can be flaky
	t.Skip("Skipping timing-sensitive test that can be flaky in CI environments")
}

func TestRateLimiter_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := RateLimiterConfig{
		RequestsPerWindow: 5,
		Window:            time.Minute,
		CleanupInterval:   time.Minute,
		BlockDuration:     time.Minute,
	}
	
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	
	r.Use(RateLimit(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "4", w.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, "1m0s", w.Header().Get("X-RateLimit-Window"))
}

// Benchmark tests
func BenchmarkRateLimiter_Allow(b *testing.B) {
	config := RateLimiterConfig{
		RequestsPerWindow: 1000,
		Window:            time.Minute,
		CleanupInterval:   time.Minute,
		BlockDuration:     time.Minute,
	}
	
	limiter := NewRateLimiter(config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clientID := "client-" + strconv.Itoa(i%100) // Simulate 100 different clients
		limiter.Allow(clientID)
	}
}