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
	"github.com/phantom-sage/bankgo/internal/database"
	"github.com/phantom-sage/bankgo/internal/queue"
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

// MockQueueManager implements a mock queue manager for testing
type MockQueueManager struct {
	shouldFail bool
}

func (m *MockQueueManager) HealthCheck(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("redis connection failed")
	}
	return nil
}

func (m *MockQueueManager) Close() error {
	return nil
}

func TestHealthHandlers_HealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupDB        func() *database.DB
		setupQueue     func() *queue.QueueManager
		expectedStatus int
		expectedHealth string
	}{
		{
			name: "healthy_services",
			setupDB: func() *database.DB {
				// Return a mock healthy database
				mockDB := &MockDB{shouldFail: false}
				return (*database.DB)(mockDB)
			},
			setupQueue: func() *queue.QueueManager {
				// Return a mock healthy queue manager
				mockQueue := &MockQueueManager{shouldFail: false}
				return (*queue.QueueManager)(mockQueue)
			},
			expectedStatus: http.StatusOK,
			expectedHealth: "healthy",
		},
		{
			name: "unhealthy_database_nil",
			setupDB: func() *database.DB {
				// Return nil to simulate database connection failure
				return nil
			},
			setupQueue: func() *queue.QueueManager {
				// Return a mock healthy queue manager
				mockQueue := &MockQueueManager{shouldFail: false}
				return (*queue.QueueManager)(mockQueue)
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: "unhealthy",
		},
		{
			name: "unhealthy_database_failed",
			setupDB: func() *database.DB {
				// Return a mock failing database
				mockDB := &MockDB{shouldFail: true}
				return (*database.DB)(mockDB)
			},
			setupQueue: func() *queue.QueueManager {
				// Return a mock healthy queue manager
				mockQueue := &MockQueueManager{shouldFail: false}
				return (*queue.QueueManager)(mockQueue)
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: "unhealthy",
		},
		{
			name: "unhealthy_redis_nil",
			setupDB: func() *database.DB {
				// Return a mock healthy database
				mockDB := &MockDB{shouldFail: false}
				return (*database.DB)(mockDB)
			},
			setupQueue: func() *queue.QueueManager {
				// Return nil to simulate Redis connection failure
				return nil
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: "unhealthy",
		},
		{
			name: "unhealthy_redis_failed",
			setupDB: func() *database.DB {
				// Return a mock healthy database
				mockDB := &MockDB{shouldFail: false}
				return (*database.DB)(mockDB)
			},
			setupQueue: func() *queue.QueueManager {
				// Return a mock failing queue manager
				mockQueue := &MockQueueManager{shouldFail: true}
				return (*queue.QueueManager)(mockQueue)
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: "unhealthy",
		},
		{
			name: "both_services_unhealthy",
			setupDB: func() *database.DB {
				return nil
			},
			setupQueue: func() *queue.QueueManager {
				return nil
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := tt.setupDB()
			queueManager := tt.setupQueue()

			handler := NewHealthHandlers(db, queueManager, "test-version")

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
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response HealthResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedHealth, response.Status)
			assert.Equal(t, "test-version", response.Version)
			assert.NotZero(t, response.Timestamp)
			assert.Contains(t, response.Services, "database")
			assert.Contains(t, response.Services, "redis")

			// Verify that unhealthy services are properly reported
			if tt.expectedHealth == "unhealthy" {
				// At least one service should be unhealthy
				dbHealthy := response.Services["database"].Status == "healthy"
				redisHealthy := response.Services["redis"].Status == "healthy"
				assert.False(t, dbHealthy && redisHealthy, "At least one service should be unhealthy")
			}
		})
	}
}

func TestHealthHandlers_HealthCheck_Timeout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a handler with nil dependencies to simulate timeout/failure
	handler := NewHealthHandlers(nil, nil, "test-version")

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

	// Parse response
	var response HealthResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response structure
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
}

func TestHealthHandlers_HealthCheck_HTTPMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
}