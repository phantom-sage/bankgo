package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupRouter_HealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup router with nil dependencies (simulating service unavailability)
	router := SetupRouter(nil, nil, nil, "test-version")

	t.Run("health_endpoint_registered", func(t *testing.T) {
		// Test that the health endpoint is properly registered at /api/v1/health
		req, err := http.NewRequest("GET", "/api/v1/health", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return service unavailable (503) since dependencies are nil
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		// Verify response structure
		var response handlers.HealthResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "unhealthy", response.Status)
		assert.Equal(t, "test-version", response.Version)
		assert.Contains(t, response.Services, "database")
		assert.Contains(t, response.Services, "redis")
	})

	t.Run("health_endpoint_path_validation", func(t *testing.T) {
		// Test that health endpoint is only available at the correct path
		testCases := []struct {
			path           string
			expectedStatus int
		}{
			{"/api/v1/health", http.StatusServiceUnavailable}, // Correct path
			{"/health", http.StatusNotFound},                  // Wrong path
			{"/api/health", http.StatusNotFound},              // Wrong path
			{"/v1/health", http.StatusNotFound},               // Wrong path
			{"/api/v1/health/", http.StatusMovedPermanently},  // Trailing slash (Gin redirects)
		}

		for _, tc := range testCases {
			t.Run("path_"+tc.path, func(t *testing.T) {
				req, err := http.NewRequest("GET", tc.path, nil)
				require.NoError(t, err)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				assert.Equal(t, tc.expectedStatus, w.Code)
			})
		}
	})

	t.Run("health_endpoint_methods", func(t *testing.T) {
		// Test that only GET method is supported for health endpoint
		methods := []struct {
			method         string
			expectedStatus int
		}{
			{"GET", http.StatusServiceUnavailable}, // Should work
			{"POST", http.StatusNotFound},          // Gin returns 404 for unsupported methods
			{"PUT", http.StatusNotFound},           // Gin returns 404 for unsupported methods
			{"DELETE", http.StatusNotFound},        // Gin returns 404 for unsupported methods
			{"PATCH", http.StatusNotFound},         // Gin returns 404 for unsupported methods
		}

		for _, m := range methods {
			t.Run("method_"+m.method, func(t *testing.T) {
				req, err := http.NewRequest(m.method, "/api/v1/health", nil)
				require.NoError(t, err)

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				assert.Equal(t, m.expectedStatus, w.Code)
			})
		}
	})

	t.Run("middleware_applied", func(t *testing.T) {
		// Test that middleware is properly applied
		req, err := http.NewRequest("GET", "/api/v1/health", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check that CORS headers are present (indicating CORS middleware is applied)
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Origin"))

		// Check that request ID header is present (indicating RequestID middleware is applied)
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	})
}

func TestSetupRouter_Configuration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("router_creation", func(t *testing.T) {
		// Test that router is created successfully
		router := SetupRouter(nil, nil, nil, "v1.0.0")
		assert.NotNil(t, router)
	})

	t.Run("api_v1_group", func(t *testing.T) {
		// Test that API v1 group is properly configured
		router := SetupRouter(nil, nil, nil, "v1.0.0")

		// Test a non-existent endpoint in the v1 group
		req, err := http.NewRequest("GET", "/api/v1/nonexistent", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 404 for non-existent endpoints, not 405
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}