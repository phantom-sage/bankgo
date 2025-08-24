package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phantom-sage/bankgo/internal/admin/config"
	"github.com/phantom-sage/bankgo/internal/admin/handlers"
	"github.com/phantom-sage/bankgo/internal/admin/middleware"
	"github.com/phantom-sage/bankgo/internal/admin/router"
	"github.com/phantom-sage/bankgo/internal/admin/services"
)

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Session   *AdminSession `json:"session,omitempty"`
	ExpiresIn int64  `json:"expires_in,omitempty"`
}

// AdminSession represents an admin session for testing
type AdminSession struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	PasetoToken string    `json:"paseto_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
	LastActive  time.Time `json:"last_active"`
}

// UpdateCredentialsRequest represents the credential update request
type UpdateCredentialsRequest struct {
	Username    string `json:"username" binding:"required"`
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// setupTestRouter creates a test router with all dependencies
func setupTestRouter(t *testing.T) *gin.Engine {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test configuration
	cfg := &config.Config{
		Port:             8081,
		Environment:      "test",
		PasetoSecretKey:  "test-secret-key-that-is-32-chars-long-for-paseto-v2-encryption",
		SessionTimeout:   time.Hour,
		DefaultAdminUser: "admin",
		DefaultAdminPass: "admin",
		AllowedOrigins:   []string{"http://localhost:3000"},
	}

	// Initialize services
	serviceContainer, err := services.NewContainer(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		serviceContainer.Close()
	})

	// Initialize handlers
	handlerContainer := handlers.NewContainer(serviceContainer)

	// Initialize middleware
	middlewareContainer := middleware.NewContainer(cfg, serviceContainer)

	// Setup router
	return router.Setup(handlerContainer, middlewareContainer)
}

func TestAdminAuth_Login_Success(t *testing.T) {
	router := setupTestRouter(t)

	// Prepare login request
	loginReq := LoginRequest{
		Username: "admin",
		Password: "admin",
	}
	reqBody, err := json.Marshal(loginReq)
	require.NoError(t, err)

	// Create request
	req, err := http.NewRequest("POST", "/api/admin/auth/login", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, "Login successful", response.Message)
	assert.NotNil(t, response.Session)
	assert.NotEmpty(t, response.Session.PasetoToken)
	assert.Equal(t, "admin", response.Session.Username)
	assert.True(t, response.ExpiresIn > 0)

	// Check that cookie is set
	cookies := w.Result().Cookies()
	var adminTokenCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "admin_token" {
			adminTokenCookie = cookie
			break
		}
	}
	assert.NotNil(t, adminTokenCookie)
	assert.Equal(t, response.Session.PasetoToken, adminTokenCookie.Value)
	assert.True(t, adminTokenCookie.HttpOnly)
}

func TestAdminAuth_Login_InvalidCredentials(t *testing.T) {
	router := setupTestRouter(t)

	// Test cases for invalid credentials
	testCases := []struct {
		name     string
		username string
		password string
	}{
		{"wrong username", "wronguser", "admin"},
		{"wrong password", "admin", "wrongpass"},
		{"empty username", "", "admin"},
		{"empty password", "admin", ""},
		{"both empty", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			loginReq := LoginRequest{
				Username: tc.username,
				Password: tc.password,
			}
			reqBody, err := json.Marshal(loginReq)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/admin/auth/login", bytes.NewBuffer(reqBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.username == "" || tc.password == "" {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			} else {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			}

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Contains(t, response, "error")
		})
	}
}

func TestAdminAuth_ValidateSession_Success(t *testing.T) {
	router := setupTestRouter(t)

	// First, login to get a valid token
	token := loginAndGetToken(t, router)

	// Test session validation with Authorization header
	req, err := http.NewRequest("GET", "/api/admin/auth/session", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["valid"].(bool))
	assert.Contains(t, response, "session")
	assert.Contains(t, response, "expires_in")
}

func TestAdminAuth_ValidateSession_WithCookie(t *testing.T) {
	router := setupTestRouter(t)

	// First, login to get a valid token
	token := loginAndGetToken(t, router)

	// Test session validation with cookie
	req, err := http.NewRequest("GET", "/api/admin/auth/session", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "admin_token",
		Value: token,
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["valid"].(bool))
}

func TestAdminAuth_ValidateSession_InvalidToken(t *testing.T) {
	router := setupTestRouter(t)

	// Test with invalid token
	req, err := http.NewRequest("GET", "/api/admin/auth/session", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer invalid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["valid"].(bool))
	assert.Equal(t, "invalid_token", response["error"])
}

func TestAdminAuth_ValidateSession_NoToken(t *testing.T) {
	router := setupTestRouter(t)

	// Test without token
	req, err := http.NewRequest("GET", "/api/admin/auth/session", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["valid"].(bool))
	assert.Equal(t, "no_token", response["error"])
}

func TestAdminAuth_Logout_Success(t *testing.T) {
	router := setupTestRouter(t)

	// First, login to get a valid token
	token := loginAndGetToken(t, router)

	// Test logout
	req, err := http.NewRequest("POST", "/api/admin/auth/logout", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Logout successful", response["message"])

	// Check that cookie is cleared
	cookies := w.Result().Cookies()
	var adminTokenCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "admin_token" {
			adminTokenCookie = cookie
			break
		}
	}
	assert.NotNil(t, adminTokenCookie)
	assert.Empty(t, adminTokenCookie.Value)
	assert.Equal(t, -1, adminTokenCookie.MaxAge)
}

func TestAdminAuth_Logout_NoToken(t *testing.T) {
	router := setupTestRouter(t)

	// Test logout without token
	req, err := http.NewRequest("POST", "/api/admin/auth/logout", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "validation_error", response["error"])
}

func TestAdminAuth_UpdateCredentials_Success(t *testing.T) {
	router := setupTestRouter(t)

	// Update credentials request
	updateReq := UpdateCredentialsRequest{
		Username:    "admin",
		OldPassword: "admin",
		NewPassword: "newpassword123",
	}
	reqBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	req, err := http.NewRequest("PUT", "/api/admin/auth/credentials", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Contains(t, response["message"], "Credentials updated successfully")
}

func TestAdminAuth_UpdateCredentials_InvalidOldPassword(t *testing.T) {
	router := setupTestRouter(t)

	updateReq := UpdateCredentialsRequest{
		Username:    "admin",
		OldPassword: "wrongpassword",
		NewPassword: "newpassword123",
	}
	reqBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	req, err := http.NewRequest("PUT", "/api/admin/auth/credentials", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "credential_update_failed", response["error"])
}

func TestAdminAuth_ProtectedEndpoint_ValidSession(t *testing.T) {
	router := setupTestRouter(t)

	// First, login to get a valid token
	token := loginAndGetToken(t, router)

	// Test protected validate endpoint
	req, err := http.NewRequest("GET", "/api/admin/validate", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["valid"].(bool))
	assert.Contains(t, response, "session")
}

func TestAdminAuth_ProtectedEndpoint_InvalidSession(t *testing.T) {
	router := setupTestRouter(t)

	// Test protected endpoint without token
	req, err := http.NewRequest("GET", "/api/admin/validate", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuth_CORSHeaders(t *testing.T) {
	router := setupTestRouter(t)

	// Test CORS preflight request
	req, err := http.NewRequest("OPTIONS", "/api/admin/auth/login", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestAdminAuth_HealthEndpoint(t *testing.T) {
	router := setupTestRouter(t)

	// Test health endpoint (no auth required)
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "admin-api", response["service"])
}

// Helper function to login and get token for testing
func loginAndGetToken(t *testing.T, router *gin.Engine) string {
	loginReq := LoginRequest{
		Username: "admin",
		Password: "admin",
	}
	reqBody, err := json.Marshal(loginReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/admin/auth/login", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	return response.Session.PasetoToken
}