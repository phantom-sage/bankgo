package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/models"
	"github.com/phantom-sage/bankgo/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock services for testing
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, email, password, firstName, lastName string) (*models.User, error) {
	args := m.Called(ctx, email, password, firstName, lastName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) GetUser(ctx context.Context, userID int) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) AuthenticateUser(ctx context.Context, email, password string) (*models.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) MarkWelcomeEmailSent(ctx context.Context, userID int) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

type MockQueueManager struct {
	mock.Mock
}

func (m *MockQueueManager) QueueWelcomeEmail(ctx context.Context, payload interface{}) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

// Test setup helper
func setupAuthHandlersTest() (*AuthHandlers, *MockUserService, *MockQueueManager, *auth.PASETOManager) {
	gin.SetMode(gin.TestMode)
	
	mockUserService := &MockUserService{}
	mockQueueManager := &MockQueueManager{}
	
	// Create PASETO manager for testing
	tokenManager, _ := auth.NewPASETOManager("test-secret-key-that-is-32-chars", time.Hour)
	
	handlers := NewAuthHandlers(mockUserService, tokenManager, mockQueueManager)
	
	return handlers, mockUserService, mockQueueManager, tokenManager
}

func TestAuthHandlers_Register(t *testing.T) {
	handlers, mockUserService, _, _ := setupAuthHandlersTest()

	tests := []struct {
		name           string
		requestBody    RegisterRequest
		mockSetup      func(*MockUserService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful registration",
			requestBody: RegisterRequest{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			mockSetup: func(m *MockUserService) {
				user := &models.User{
					ID:        1,
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("CreateUser", mock.Anything, "test@example.com", "password123", "John", "Doe").Return(user, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid email format",
			requestBody: RegisterRequest{
				Email:     "invalid-email",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "password too short",
			requestBody: RegisterRequest{
				Email:     "test@example.com",
				Password:  "short",
				FirstName: "John",
				LastName:  "Doe",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "missing required fields",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				// Missing FirstName and LastName
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "user already exists",
			requestBody: RegisterRequest{
				Email:     "existing@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			mockSetup: func(m *MockUserService) {
				m.On("CreateUser", mock.Anything, "existing@example.com", "password123", "John", "Doe").
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockUserService.ExpectedCalls = nil
			mockUserService.Calls = nil

			if tt.mockSetup != nil {
				tt.mockSetup(mockUserService)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			handlers.Register(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			} else if tt.expectedStatus == http.StatusCreated {
				var response AuthResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Token)
				assert.NotNil(t, response.User)
			}

			mockUserService.AssertExpectations(t)
		})
	}
}

func TestAuthHandlers_Login(t *testing.T) {
	handlers, mockUserService, mockQueueManager, _ := setupAuthHandlersTest()

	tests := []struct {
		name           string
		requestBody    LoginRequest
		mockSetup      func(*MockUserService, *MockQueueManager)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful login - first time user",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(m *MockUserService, q *MockQueueManager) {
				user := &models.User{
					ID:               1,
					Email:            "test@example.com",
					FirstName:        "John",
					LastName:         "Doe",
					WelcomeEmailSent: false, // First time login
					CreatedAt:        time.Now(),
					UpdatedAt:        time.Now(),
				}
				m.On("AuthenticateUser", mock.Anything, "test@example.com", "password123").Return(user, nil)
				q.On("QueueWelcomeEmail", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "successful login - returning user",
			requestBody: LoginRequest{
				Email:    "returning@example.com",
				Password: "password123",
			},
			mockSetup: func(m *MockUserService, q *MockQueueManager) {
				user := &models.User{
					ID:               2,
					Email:            "returning@example.com",
					FirstName:        "Jane",
					LastName:         "Smith",
					WelcomeEmailSent: true, // Returning user
					CreatedAt:        time.Now(),
					UpdatedAt:        time.Now(),
				}
				m.On("AuthenticateUser", mock.Anything, "returning@example.com", "password123").Return(user, nil)
				// No queue call expected for returning users
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid credentials",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			mockSetup: func(m *MockUserService, q *MockQueueManager) {
				m.On("AuthenticateUser", mock.Anything, "test@example.com", "wrongpassword").
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "authentication_failed",
		},
		{
			name: "invalid email format",
			requestBody: LoginRequest{
				Email:    "invalid-email",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "missing password",
			requestBody: LoginRequest{
				Email: "test@example.com",
				// Missing password
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockUserService.ExpectedCalls = nil
			mockUserService.Calls = nil
			mockQueueManager.ExpectedCalls = nil
			mockQueueManager.Calls = nil

			if tt.mockSetup != nil {
				tt.mockSetup(mockUserService, mockQueueManager)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			handlers.Login(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			} else if tt.expectedStatus == http.StatusOK {
				var response AuthResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Token)
				assert.NotNil(t, response.User)
			}

			mockUserService.AssertExpectations(t)
			mockQueueManager.AssertExpectations(t)
		})
	}
}

func TestAuthHandlers_Logout(t *testing.T) {
	handlers, _, _, _ := setupAuthHandlersTest()

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handlers.Logout(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Successfully logged out", response["message"])
}

func TestAuthHandlers_AuthMiddleware(t *testing.T) {
	handlers, _, _, tokenManager := setupAuthHandlersTest()

	tests := []struct {
		name           string
		setupAuth      func() string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid token",
			setupAuth: func() string {
				token, _ := tokenManager.GenerateToken(1, "test@example.com")
				return "Bearer " + token
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing authorization header",
			setupAuth: func() string {
				return ""
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing_token",
		},
		{
			name: "invalid token format",
			setupAuth: func() string {
				return "InvalidFormat"
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid_token_format",
		},
		{
			name: "invalid token",
			setupAuth: func() string {
				return "Bearer invalid-token"
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test route with the auth middleware
			router := gin.New()
			router.Use(handlers.AuthMiddleware())
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if authHeader := tt.setupAuth(); authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			}
		})
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		setupCtx    func(*gin.Context)
		expectedID  int
		expectError bool
	}{
		{
			name: "valid user ID",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", 123)
			},
			expectedID:  123,
			expectError: false,
		},
		{
			name: "missing user ID",
			setupCtx: func(c *gin.Context) {
				// Don't set user_id
			},
			expectedID:  0,
			expectError: true,
		},
		{
			name: "invalid user ID type",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", "not-an-int")
			},
			expectedID:  0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			tt.setupCtx(c)
			
			id, err := GetUserIDFromContext(c)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, 0, id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func TestParseIDParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		paramValue  string
		expectedID  int
		expectError bool
	}{
		{
			name:        "valid ID",
			paramValue:  "123",
			expectedID:  123,
			expectError: false,
		},
		{
			name:        "zero ID",
			paramValue:  "0",
			expectedID:  0,
			expectError: true,
		},
		{
			name:        "negative ID",
			paramValue:  "-1",
			expectedID:  0,
			expectError: true,
		},
		{
			name:        "non-numeric ID",
			paramValue:  "abc",
			expectedID:  0,
			expectError: true,
		},
		{
			name:        "empty ID",
			paramValue:  "",
			expectedID:  0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			// Set up URL parameter
			c.Params = gin.Params{
				{Key: "id", Value: tt.paramValue},
			}
			
			id, err := ParseIDParam(c, "id")
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, 0, id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}