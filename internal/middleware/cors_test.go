package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCORSConfig(t *testing.T) {
	config := DefaultCORSConfig()
	
	assert.Contains(t, config.AllowOrigins, "*")
	assert.Contains(t, config.AllowMethods, http.MethodGet)
	assert.Contains(t, config.AllowMethods, http.MethodPost)
	assert.Contains(t, config.AllowHeaders, "Authorization")
	assert.False(t, config.AllowCredentials)
	assert.Equal(t, 12*60*60, config.MaxAge)
}

func TestRestrictiveCORSConfig(t *testing.T) {
	allowedOrigins := []string{"https://example.com", "https://app.example.com"}
	config := RestrictiveCORSConfig(allowedOrigins)
	
	assert.Equal(t, allowedOrigins, config.AllowOrigins)
	assert.NotContains(t, config.AllowMethods, http.MethodPatch)
	assert.True(t, config.AllowCredentials)
	assert.Equal(t, 1*60*60, config.MaxAge)
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		config         CORSConfig
		origin         string
		method         string
		expectedStatus int
		expectedHeader string
	}{
		{
			name: "wildcard origin allowed",
			config: CORSConfig{
				AllowOrigins: []string{"*"},
				AllowMethods: []string{http.MethodGet, http.MethodPost},
			},
			origin:         "https://example.com",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedHeader: "https://example.com", // When origin is provided, it's returned instead of *
		},
		{
			name: "specific origin allowed",
			config: CORSConfig{
				AllowOrigins: []string{"https://example.com"},
				AllowMethods: []string{http.MethodGet, http.MethodPost},
			},
			origin:         "https://example.com",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedHeader: "https://example.com",
		},
		{
			name: "origin not allowed",
			config: CORSConfig{
				AllowOrigins: []string{"https://allowed.com"},
				AllowMethods: []string{http.MethodGet, http.MethodPost},
			},
			origin:         "https://notallowed.com",
			method:         http.MethodGet,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "preflight request",
			config: CORSConfig{
				AllowOrigins: []string{"*"},
				AllowMethods: []string{http.MethodGet, http.MethodPost},
			},
			origin:         "https://example.com",
			method:         http.MethodOptions,
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			
			r.Use(CORS(tt.config))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})
			r.OPTIONS("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "options"})
			})
			
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			c.Request = req
			
			r.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedHeader, w.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestCORS_WithCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := CORSConfig{
		AllowOrigins:     []string{"https://example.com"},
		AllowCredentials: true,
	}
	
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	
	r.Use(CORS(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	c.Request = req
	
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_WithHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	config := CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
		ExposeHeaders: []string{"X-Total-Count"},
	}
	
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	
	r.Use(CORS(config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	c.Request = req
	
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Equal(t, "X-Total-Count", w.Header().Get("Access-Control-Expose-Headers"))
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}
	
	assert.True(t, contains(slice, "banana"))
	assert.False(t, contains(slice, "grape"))
	assert.False(t, contains([]string{}, "anything"))
}