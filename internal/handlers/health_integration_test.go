package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthHandlers_Integration tests the health check endpoint with real integration scenarios
func TestHealthHandlers_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("nil_dependencies", func(t *testing.T) {
		// Test with nil dependencies to simulate service unavailability
		handler := NewHealthHandlers(nil, nil, "v1.0.0")

		// Create test router
		router := gin.New()
		router.GET("/health", handler.HealthCheck)

		// Create test request
		req, err := http.NewRequest("GET", "/health", nil)
		require.NoError(t, err)

		// Record response
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response HealthResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "unhealthy", response.Status)
		assert.Equal(t, "v1.0.0", response.Version)
		assert.WithinDuration(t, time.Now(), response.Timestamp, time.Second)

		// Verify services structure
		require.Len(t, response.Services, 2)

		dbService := response.Services["database"]
		assert.Equal(t, "unhealthy", dbService.Status)
		assert.Equal(t, "database connection not initialized", dbService.Message)

		redisService := response.Services["redis"]
		assert.Equal(t, "unhealthy", redisService.Status)
		assert.Equal(t, "redis connection not initialized", redisService.Message)
	})

	t.Run("response_format_validation", func(t *testing.T) {
		handler := NewHealthHandlers(nil, nil, "test-version")

		// Create test router
		router := gin.New()
		router.GET("/health", handler.HealthCheck)

		// Create test request
		req, err := http.NewRequest("GET", "/health", nil)
		require.NoError(t, err)

		// Record response
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Parse response
		var response HealthResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response structure matches expected format
		assert.NotEmpty(t, response.Status)
		assert.NotEmpty(t, response.Version)
		assert.NotZero(t, response.Timestamp)
		assert.NotNil(t, response.Services)

		// Verify required services are present
		_, hasDatabase := response.Services["database"]
		_, hasRedis := response.Services["redis"]
		assert.True(t, hasDatabase, "Response should include database service status")
		assert.True(t, hasRedis, "Response should include redis service status")

		// Verify service status structure
		for serviceName, serviceStatus := range response.Services {
			assert.NotEmpty(t, serviceStatus.Status, "Service %s should have a status", serviceName)
			assert.Contains(t, []string{"healthy", "unhealthy"}, serviceStatus.Status, 
				"Service %s status should be either 'healthy' or 'unhealthy'", serviceName)
		}
	})

	t.Run("http_methods", func(t *testing.T) {
		handler := NewHealthHandlers(nil, nil, "test-version")

		// Create test router
		router := gin.New()
		router.GET("/health", handler.HealthCheck)

		// Test that only GET method is supported
		methods := []string{"POST", "PUT", "DELETE", "PATCH"}

		for _, method := range methods {
			t.Run("method_"+method, func(t *testing.T) {
				req, err := http.NewRequest(method, "/health", nil)
				require.NoError(t, err)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				// Should return 404 for unsupported methods
				assert.Equal(t, http.StatusNotFound, w.Code)
			})
		}

		// Test that GET method works
		t.Run("method_GET", func(t *testing.T) {
			req, err := http.NewRequest("GET", "/health", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should return either 200 or 503, but not 404
			assert.NotEqual(t, http.StatusNotFound, w.Code)
			assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, w.Code)
		})
	})

	t.Run("timeout_handling", func(t *testing.T) {
		// Test that health check completes within reasonable time
		handler := NewHealthHandlers(nil, nil, "test-version")

		// Create test router
		router := gin.New()
		router.GET("/health", handler.HealthCheck)

		// Create test request
		req, err := http.NewRequest("GET", "/health", nil)
		require.NoError(t, err)

		// Record response with timing
		start := time.Now()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		duration := time.Since(start)

		// Should complete quickly since we're testing nil dependencies
		assert.Less(t, duration, 5*time.Second, "Health check should complete within 5 seconds")

		// Should return service unavailable for failed health checks
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("concurrent_requests", func(t *testing.T) {
		// Test that health check can handle concurrent requests
		handler := NewHealthHandlers(nil, nil, "test-version")

		// Create test router
		router := gin.New()
		router.GET("/health", handler.HealthCheck)

		// Number of concurrent requests
		numRequests := 10
		responses := make(chan *httptest.ResponseRecorder, numRequests)

		// Launch concurrent requests
		for i := 0; i < numRequests; i++ {
			go func() {
				req, _ := http.NewRequest("GET", "/health", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				responses <- w
			}()
		}

		// Collect all responses
		for i := 0; i < numRequests; i++ {
			select {
			case w := <-responses:
				// All requests should return the same status
				assert.Equal(t, http.StatusServiceUnavailable, w.Code)

				var response HealthResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "unhealthy", response.Status)

			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent health check responses")
			}
		}
	})
}

// TestHealthHandlers_DatabaseConnectivity tests database connectivity validation
func TestHealthHandlers_DatabaseConnectivity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("database_nil_check", func(t *testing.T) {
		handler := NewHealthHandlers(nil, nil, "test-version")

		// Create context for health check
		ctx := context.Background()

		// Test database health check directly
		status := handler.checkDatabaseHealth(ctx)

		assert.Equal(t, "unhealthy", status.Status)
		assert.Equal(t, "database connection not initialized", status.Message)
	})
}

// TestHealthHandlers_RedisConnectivity tests Redis connectivity validation
func TestHealthHandlers_RedisConnectivity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("redis_nil_check", func(t *testing.T) {
		handler := NewHealthHandlers(nil, nil, "test-version")

		// Create context for health check
		ctx := context.Background()

		// Test Redis health check directly
		status := handler.checkRedisHealth(ctx)

		assert.Equal(t, "unhealthy", status.Status)
		assert.Equal(t, "redis connection not initialized", status.Message)
	})
}