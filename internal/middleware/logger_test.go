package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultLoggerConfig(t *testing.T) {
	// Create a logger manager for testing
	logConfig := logging.LogConfig{
		Level:  "debug",
		Format: "json",
		Output: "console",
	}
	loggerManager, err := logging.NewLoggerManager(logConfig)
	require.NoError(t, err)
	defer loggerManager.Close()

	config := DefaultLoggerConfig(loggerManager)
	
	assert.NotNil(t, config.LoggerManager)
	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/metrics")
	assert.Contains(t, config.SensitiveHeaders, "authorization")
	assert.Contains(t, config.SensitiveFields, "password")
	assert.True(t, config.EnablePerformanceMetrics)
	assert.Equal(t, 500*time.Millisecond, config.SlowRequestThreshold)
}

func TestRequestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a logger manager for testing
	logConfig := logging.LogConfig{
		Level:  "debug",
		Format: "json",
		Output: "console",
	}
	loggerManager, err := logging.NewLoggerManager(logConfig)
	require.NoError(t, err)
	defer loggerManager.Close()
	
	config := LoggerConfig{
		LoggerManager:            loggerManager,
		SkipPaths:               []string{"/health"},
		SensitiveHeaders:        []string{"authorization"},
		SensitiveFields:         []string{"password"},
		EnablePerformanceMetrics: true,
		SlowRequestThreshold:     500 * time.Millisecond,
	}
	
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	
	r.Use(RequestLogger(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test?param=value", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Authorization", "Bearer secret-token")
	req.RemoteAddr = "192.168.1.1:12345"
	
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Note: With zerolog, we can't easily capture log output in tests
	// without setting up a custom writer. The important thing is that
	// the middleware doesn't crash and processes the request correctly.
}

func TestRequestLogger_SkipPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a logger manager for testing
	logConfig := logging.LogConfig{
		Level:  "debug",
		Format: "json",
		Output: "console",
	}
	loggerManager, err := logging.NewLoggerManager(logConfig)
	require.NoError(t, err)
	defer loggerManager.Close()
	
	config := LoggerConfig{
		LoggerManager:            loggerManager,
		SkipPaths:               []string{"/health"},
		EnablePerformanceMetrics: false,
	}
	
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	
	r.Use(RequestLogger(config))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	// For skipped paths, the middleware should return early without logging
}

func TestRequestLogger_WithError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a logger manager for testing
	logConfig := logging.LogConfig{
		Level:  "debug",
		Format: "json",
		Output: "console",
	}
	loggerManager, err := logging.NewLoggerManager(logConfig)
	require.NoError(t, err)
	defer loggerManager.Close()
	
	config := DefaultLoggerConfig(loggerManager)
	
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	
	r.Use(RequestLogger(config))
	r.GET("/test", func(c *gin.Context) {
		c.Error(gin.Error{Err: assert.AnError, Type: gin.ErrorTypePublic})
		c.JSON(http.StatusBadRequest, gin.H{"error": "test error"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	// The middleware should handle errors correctly and log them
}

func TestRequestLogger_WithUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a logger manager for testing
	logConfig := logging.LogConfig{
		Level:  "debug",
		Format: "json",
		Output: "console",
	}
	loggerManager, err := logging.NewLoggerManager(logConfig)
	require.NoError(t, err)
	defer loggerManager.Close()
	
	config := DefaultLoggerConfig(loggerManager)
	
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(123))
		c.Set("user_email", "test@example.com")
		c.Next()
	})
	r.Use(RequestLogger(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	// The middleware should handle user context correctly
}

func TestIsSensitiveHeader(t *testing.T) {
	sensitiveHeaders := []string{"authorization", "cookie"}
	
	assert.True(t, isSensitiveHeader("Authorization", sensitiveHeaders))
	assert.True(t, isSensitiveHeader("COOKIE", sensitiveHeaders))
	assert.False(t, isSensitiveHeader("Content-Type", sensitiveHeaders))
}

func TestIsJSONContent(t *testing.T) {
	assert.True(t, isJSONContent("application/json"))
	assert.True(t, isJSONContent("application/json; charset=utf-8"))
	assert.False(t, isJSONContent("text/html"))
	assert.False(t, isJSONContent("application/xml"))
}

func TestSanitizeRequestBody(t *testing.T) {
	sensitiveFields := []string{"password", "token"}
	
	tests := []struct {
		name     string
		body     string
		expected map[string]interface{}
	}{
		{
			name: "sanitize password field",
			body: `{"username": "john", "password": "secret123"}`,
			expected: map[string]interface{}{
				"username": "john",
				"password": "[REDACTED]",
			},
		},
		{
			name: "sanitize nested sensitive field",
			body: `{"user": {"name": "john", "auth_token": "abc123"}}`,
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"name":       "john",
					"auth_token": "[REDACTED]",
				},
			},
		},
		{
			name: "no sensitive fields",
			body: `{"username": "john", "email": "john@example.com"}`,
			expected: map[string]interface{}{
				"username": "john",
				"email":    "john@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeRequestBody([]byte(tt.body), sensitiveFields)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeData(t *testing.T) {
	sensitiveFields := []string{"password", "secret"}
	
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name: "map with sensitive field",
			input: map[string]interface{}{
				"username": "john",
				"password": "secret123",
			},
			expected: map[string]interface{}{
				"username": "john",
				"password": "[REDACTED]",
			},
		},
		{
			name: "array of maps",
			input: []interface{}{
				map[string]interface{}{
					"id":     1,
					"secret": "hidden",
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"id":     1,
					"secret": "[REDACTED]",
				},
			},
		},
		{
			name:     "primitive value",
			input:    "simple string",
			expected: "simple string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeData(tt.input, sensitiveFields)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSensitiveField(t *testing.T) {
	sensitiveFields := []string{"password", "token", "secret"}
	
	assert.True(t, isSensitiveField("password", sensitiveFields))
	assert.True(t, isSensitiveField("auth_token", sensitiveFields))
	assert.True(t, isSensitiveField("client_secret", sensitiveFields))
	assert.False(t, isSensitiveField("username", sensitiveFields))
	assert.False(t, isSensitiveField("email", sensitiveFields))
}

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		requestID, exists := c.Get("request_id")
		assert.True(t, exists)
		assert.NotEmpty(t, requestID)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	
	// Check response body contains request ID
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["request_id"])
}

func TestRequestID_WithExistingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "existing-request-id")
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "existing-request-id", w.Header().Get("X-Request-ID"))
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()
	
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2) // Should generate unique IDs
	assert.Contains(t, id1, "-")  // Should contain separator
}

func TestRandomString(t *testing.T) {
	str1 := randomString(8)
	str2 := randomString(16)
	
	assert.Len(t, str1, 8)
	assert.Len(t, str2, 16)
	assert.NotEmpty(t, str1)
	assert.NotEmpty(t, str2)
}

func TestRequestLogger_WithJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a logger manager for testing
	logConfig := logging.LogConfig{
		Level:  "debug",
		Format: "json",
		Output: "console",
	}
	loggerManager, err := logging.NewLoggerManager(logConfig)
	require.NoError(t, err)
	defer loggerManager.Close()
	
	config := LoggerConfig{
		LoggerManager:            loggerManager,
		SensitiveFields:         []string{"password"},
		EnablePerformanceMetrics: true,
		SlowRequestThreshold:     500 * time.Millisecond,
	}
	
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	
	r.Use(RequestLogger(config))
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	body := `{"username": "john", "password": "secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	// The middleware should handle JSON body correctly and sanitize sensitive fields
}