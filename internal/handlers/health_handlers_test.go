package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDB implements a mock database for testing
type MockDB struct {
	shouldFail bool
	stats      *pgxpool.Stat
}

func (m *MockDB) HealthCheck(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("database connection failed")
	}
	return nil
}

func (m *MockDB) Stats() *pgxpool.Stat {
	if m.stats != nil {
		return m.stats
	}
	// Return a mock stat with some connections
	return &pgxpool.Stat{}
}

func (m *MockDB) Close() {}

// MockLoggerManagerHealth implements a mock logger manager for health testing
type MockLoggerManagerHealth struct {
	shouldFail bool
}

func (m *MockLoggerManagerHealth) HealthCheck() error {
	if m.shouldFail {
		return errors.New("logging system failed")
	}
	return nil
}

func (m *MockLoggerManagerHealth) Close() error {
	return nil
}

func TestHealthHandlers_HealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with nil services (unhealthy scenario)
	t.Run("nil_services", func(t *testing.T) {
		handler := NewHealthHandlers(nil, nil, nil, "test-version")

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
		assert.Equal(t, "test-version", response.Version)
		assert.NotZero(t, response.Timestamp)
		assert.Contains(t, response.Services, "database")
		assert.Contains(t, response.Services, "redis")
		assert.Contains(t, response.Services, "logging")

		// All services should be unhealthy
		assert.Equal(t, "unhealthy", response.Services["database"].Status)
		assert.Equal(t, "unhealthy", response.Services["redis"].Status)
		assert.Equal(t, "unhealthy", response.Services["logging"].Status)
	})
}

func TestHealthHandlers_HealthCheck_Timeout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a handler with nil dependencies to simulate timeout/failure
	handler := NewHealthHandlers(nil, nil, nil, "test-version")

	// Create test router
	router := gin.New()
	router.GET("/health", handler.HealthCheck)

	// Create test request
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	// Record response
	w := httptest.NewRecorder()
	
	// Test should complete within reasonable time
	start := time.Now()
	router.ServeHTTP(w, req)
	duration := time.Since(start)

	// Should complete quickly since we're testing nil dependencies
	assert.Less(t, duration, 5*time.Second)

	// Should return service unavailable for failed health checks
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHealthHandlers_HealthCheck_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup with nil services to test response format
	handler := NewHealthHandlers(nil, nil, nil, "v1.0.0")

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

	// Verify response structure
	assert.Equal(t, "unhealthy", response.Status)
	assert.Equal(t, "v1.0.0", response.Version)
	assert.WithinDuration(t, time.Now(), response.Timestamp, time.Second)

	// Verify services structure
	require.Len(t, response.Services, 3)
	
	dbService := response.Services["database"]
	assert.Equal(t, "unhealthy", dbService.Status)
	assert.Equal(t, "database connection not initialized", dbService.Message)

	redisService := response.Services["redis"]
	assert.Equal(t, "unhealthy", redisService.Status)
	assert.Equal(t, "redis connection not initialized", redisService.Message)

	loggingService := response.Services["logging"]
	assert.Equal(t, "unhealthy", loggingService.Status)
	assert.Equal(t, "logger manager not initialized", loggingService.Message)
}

func TestHealthHandlers_HealthCheck_HTTPMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHealthHandlers(nil, nil, nil, "test-version")

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
}