package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
)

// MockAdminAuthService is a mock implementation of AdminAuthService for testing
type MockAdminAuthService struct {
	mock.Mock
}

func (m *MockAdminAuthService) Login(ctx context.Context, username, password string) (*interfaces.AdminSession, error) {
	args := m.Called(ctx, username, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.AdminSession), args.Error(1)
}

func (m *MockAdminAuthService) ValidateSession(ctx context.Context, token string) (*interfaces.AdminSession, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.AdminSession), args.Error(1)
}

func (m *MockAdminAuthService) RefreshSession(ctx context.Context, token string) (*interfaces.AdminSession, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.AdminSession), args.Error(1)
}

func (m *MockAdminAuthService) Logout(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockAdminAuthService) UpdateCredentials(ctx context.Context, username, oldPassword, newPassword string) error {
	args := m.Called(ctx, username, oldPassword, newPassword)
	return args.Error(0)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func createTestSession() *interfaces.AdminSession {
	return &interfaces.AdminSession{
		ID:          "test-session-id",
		Username:    "admin",
		PasetoToken: "test-token",
		ExpiresAt:   time.Now().Add(time.Hour),
		CreatedAt:   time.Now().Add(-time.Minute),
		LastActive:  time.Now(),
	}
}

func TestNewAuthMiddleware(t *testing.T) {
	mockAuthService := &MockAdminAuthService{}
	middleware := NewAuthMiddleware(mockAuthService)

	assert.NotNil(t, middleware)
	assert.IsType(t, &AuthMiddlewareImpl{}, middleware)
}

func TestAuthMiddleware_RequireAuth_Success(t *testing.T) {
	mockAuthService := &MockAdminAuthService{}
	middleware := NewAuthMiddleware(mockAuthService)
	router := setupTestRouter()

	testSession := createTestSession()
	mockAuthService.On("ValidateSession", mock.Anything, "valid-token").Return(testSession, nil)

	// Set up protected route
	router.Use(middleware.RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	t.Run("with Authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockAuthService.AssertCalled(t, "ValidateSession", mock.Anything, "valid-token")
	})

	t.Run("with cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "admin_token", Value: "valid-token"})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockAuthService.AssertCalled(t, "ValidateSession", mock.Anything, "valid-token")
	})

	t.Run("with query parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected?token=valid-token", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockAuthService.AssertCalled(t, "ValidateSession", mock.Anything, "valid-token")
	})
}

func TestAuthMiddleware_RequireAuth_Failures(t *testing.T) {
	mockAuthService := &MockAdminAuthService{}
	middleware := NewAuthMiddleware(mockAuthService)
	router := setupTestRouter()

	// Set up protected route
	router.Use(middleware.RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Missing authentication token")
	})

	t.Run("invalid token format in Authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Missing authentication token")
	})

	t.Run("invalid token", func(t *testing.T) {
		mockAuthService.On("ValidateSession", mock.Anything, "invalid-token").Return(nil, assert.AnError)

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid or expired token")
		mockAuthService.AssertCalled(t, "ValidateSession", mock.Anything, "invalid-token")
	})

	t.Run("expired token", func(t *testing.T) {
		mockAuthService.On("ValidateSession", mock.Anything, "expired-token").Return(nil, assert.AnError)

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer expired-token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid or expired token")
		mockAuthService.AssertCalled(t, "ValidateSession", mock.Anything, "expired-token")
	})
}

func TestAuthMiddleware_OptionalAuth(t *testing.T) {
	mockAuthService := &MockAdminAuthService{}
	middleware := NewAuthMiddleware(mockAuthService)
	router := setupTestRouter()

	testSession := createTestSession()

	// Set up route with optional auth
	router.Use(middleware.OptionalAuth())
	router.GET("/optional", func(c *gin.Context) {
		session, exists := GetAdminSession(c)
		if exists {
			c.JSON(http.StatusOK, gin.H{"authenticated": true, "username": session.Username})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	t.Run("with valid token", func(t *testing.T) {
		mockAuthService.On("ValidateSession", mock.Anything, "valid-token").Return(testSession, nil)

		req := httptest.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"authenticated":true`)
		assert.Contains(t, w.Body.String(), `"username":"admin"`)
	})

	t.Run("without token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/optional", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"authenticated":false`)
	})

	t.Run("with invalid token", func(t *testing.T) {
		mockAuthService.On("ValidateSession", mock.Anything, "invalid-token").Return(nil, assert.AnError)

		req := httptest.NewRequest("GET", "/optional", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"authenticated":false`)
	})
}

func TestAuthMiddleware_ContextHelpers(t *testing.T) {
	mockAuthService := &MockAdminAuthService{}
	middleware := NewAuthMiddleware(mockAuthService)
	router := setupTestRouter()

	testSession := createTestSession()
	mockAuthService.On("ValidateSession", mock.Anything, "valid-token").Return(testSession, nil)

	// Set up protected route that uses context helpers
	router.Use(middleware.RequireAuth())
	router.GET("/context-test", func(c *gin.Context) {
		session, sessionExists := GetAdminSession(c)
		username, usernameExists := GetAdminUsername(c)
		sessionID, sessionIDExists := GetAdminSessionID(c)

		c.JSON(http.StatusOK, gin.H{
			"session_exists":    sessionExists,
			"username_exists":   usernameExists,
			"session_id_exists": sessionIDExists,
			"session_id":        session.ID,
			"username":          username,
			"session_id_value":  sessionID,
		})
	})

	req := httptest.NewRequest("GET", "/context-test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"session_exists":true`)
	assert.Contains(t, w.Body.String(), `"username_exists":true`)
	assert.Contains(t, w.Body.String(), `"session_id_exists":true`)
	assert.Contains(t, w.Body.String(), `"session_id":"test-session-id"`)
	assert.Contains(t, w.Body.String(), `"username":"admin"`)
	assert.Contains(t, w.Body.String(), `"session_id_value":"test-session-id"`)
}

func TestAuthMiddleware_ContextHelpers_NoAuth(t *testing.T) {
	router := setupTestRouter()

	// Set up route without auth middleware
	router.GET("/no-auth", func(c *gin.Context) {
		session, sessionExists := GetAdminSession(c)
		username, usernameExists := GetAdminUsername(c)
		sessionID, sessionIDExists := GetAdminSessionID(c)

		response := gin.H{
			"session_exists":    sessionExists,
			"username_exists":   usernameExists,
			"session_id_exists": sessionIDExists,
		}

		if session != nil {
			response["session_id"] = session.ID
		}
		if usernameExists {
			response["username"] = username
		}
		if sessionIDExists {
			response["session_id_value"] = sessionID
		}

		c.JSON(http.StatusOK, response)
	})

	req := httptest.NewRequest("GET", "/no-auth", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"session_exists":false`)
	assert.Contains(t, w.Body.String(), `"username_exists":false`)
	assert.Contains(t, w.Body.String(), `"session_id_exists":false`)
}

func TestSessionRefreshMiddleware(t *testing.T) {
	mockAuthService := &MockAdminAuthService{}
	router := setupTestRouter()

	// Create session that needs refresh (close to expiring)
	now := time.Now()
	testSession := &interfaces.AdminSession{
		ID:          "test-session-id",
		Username:    "admin",
		PasetoToken: "original-token",
		ExpiresAt:   now.Add(10 * time.Minute),  // Expires in 10 minutes
		CreatedAt:   now.Add(-50 * time.Minute), // Created 50 minutes ago (total 60 min session)
		LastActive:  now.Add(-1 * time.Minute),  // Last active 1 minute ago
	}

	refreshedSession := &interfaces.AdminSession{
		ID:          "test-session-id",
		Username:    "admin",
		PasetoToken: "refreshed-token",
		ExpiresAt:   now.Add(time.Hour),
		CreatedAt:   testSession.CreatedAt,
		LastActive:  now,
	}

	mockAuthService.On("ValidateSession", mock.Anything, "original-token").Return(testSession, nil)
	mockAuthService.On("RefreshSession", mock.Anything, "original-token").Return(refreshedSession, nil)

	// Set up middleware chain
	authMiddleware := NewAuthMiddleware(mockAuthService)
	router.Use(authMiddleware.RequireAuth())
	router.Use(SessionRefreshMiddleware(mockAuthService, 0.25)) // Refresh when <25% time remaining

	router.GET("/test", func(c *gin.Context) {
		session, _ := GetAdminSession(c)
		c.JSON(http.StatusOK, gin.H{
			"token": session.PasetoToken,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer original-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"token":"refreshed-token"`)
	assert.Equal(t, "refreshed-token", w.Header().Get("X-Refreshed-Token"))

	mockAuthService.AssertCalled(t, "ValidateSession", mock.Anything, "original-token")
	mockAuthService.AssertCalled(t, "RefreshSession", mock.Anything, "original-token")
}

func TestSessionRefreshMiddleware_NoRefreshNeeded(t *testing.T) {
	mockAuthService := &MockAdminAuthService{}
	router := setupTestRouter()

	// Create session that doesn't need refresh (plenty of time remaining)
	now := time.Now()
	testSession := &interfaces.AdminSession{
		ID:          "test-session-id",
		Username:    "admin",
		PasetoToken: "original-token",
		ExpiresAt:   now.Add(50 * time.Minute), // Expires in 50 minutes
		CreatedAt:   now.Add(-10 * time.Minute), // Created 10 minutes ago (total 60 min session)
		LastActive:  now.Add(-1 * time.Minute),  // Last active 1 minute ago
	}

	mockAuthService.On("ValidateSession", mock.Anything, "original-token").Return(testSession, nil)

	// Set up middleware chain
	authMiddleware := NewAuthMiddleware(mockAuthService)
	router.Use(authMiddleware.RequireAuth())
	router.Use(SessionRefreshMiddleware(mockAuthService, 0.25)) // Refresh when <25% time remaining

	router.GET("/test", func(c *gin.Context) {
		session, _ := GetAdminSession(c)
		c.JSON(http.StatusOK, gin.H{
			"token": session.PasetoToken,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer original-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"token":"original-token"`)
	assert.Empty(t, w.Header().Get("X-Refreshed-Token"))

	mockAuthService.AssertCalled(t, "ValidateSession", mock.Anything, "original-token")
	mockAuthService.AssertNotCalled(t, "RefreshSession", mock.Anything, mock.Anything)
}

func TestCORSMiddleware(t *testing.T) {
	router := setupTestRouter()
	allowedOrigins := []string{"http://localhost:3000", "https://admin.example.com"}

	router.Use(CORSMiddleware(allowedOrigins))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	t.Run("allowed origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("disallowed origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://malicious.com")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("wildcard origin", func(t *testing.T) {
		wildcardRouter := setupTestRouter()
		wildcardRouter.Use(CORSMiddleware([]string{"*"}))
		wildcardRouter.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://any-origin.com")
		w := httptest.NewRecorder()

		wildcardRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://any-origin.com", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("OPTIONS preflight request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	})
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age=31536000")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'self'")
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
}

func TestRequireAdminAuth_Helper(t *testing.T) {
	mockAuthService := &MockAdminAuthService{}
	testSession := createTestSession()

	t.Run("successful authentication", func(t *testing.T) {
		mockAuthService.On("ValidateSession", mock.Anything, "valid-token").Return(testSession, nil)

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer valid-token")

		session, err := RequireAdminAuth(c, mockAuthService)

		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, testSession.ID, session.ID)
		assert.Equal(t, testSession.Username, session.Username)
	})

	t.Run("missing token", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/test", nil)

		session, err := RequireAdminAuth(c, mockAuthService)

		assert.Error(t, err)
		assert.Nil(t, session)
	})

	t.Run("invalid token", func(t *testing.T) {
		mockAuthService.On("ValidateSession", mock.Anything, "invalid-token").Return(nil, assert.AnError)

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer invalid-token")

		session, err := RequireAdminAuth(c, mockAuthService)

		assert.Error(t, err)
		assert.Nil(t, session)
	})
}