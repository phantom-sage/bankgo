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
	"github.com/phantom-sage/bankgo/internal/services"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock account service for testing
type MockAccountService struct {
	mock.Mock
}

func (m *MockAccountService) CreateAccount(ctx context.Context, userID int32, currency string) (*models.Account, error) {
	args := m.Called(ctx, userID, currency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountService) GetAccount(ctx context.Context, accountID int32, userID int32) (*models.Account, error) {
	args := m.Called(ctx, accountID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountService) GetUserAccounts(ctx context.Context, userID int32) ([]*models.Account, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Account), args.Error(1)
}

func (m *MockAccountService) UpdateAccount(ctx context.Context, accountID int32, userID int32, req services.UpdateAccountRequest) (*models.Account, error) {
	args := m.Called(ctx, accountID, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Account), args.Error(1)
}

func (m *MockAccountService) DeleteAccount(ctx context.Context, accountID int32, userID int32) error {
	args := m.Called(ctx, accountID, userID)
	return args.Error(0)
}

// Test setup helper for account handlers
func setupAccountHandlersTest() (*AccountHandlers, *MockAccountService) {
	gin.SetMode(gin.TestMode)
	
	mockAccountService := &MockAccountService{}
	handlers := NewAccountHandlers(mockAccountService)
	
	return handlers, mockAccountService
}

// Helper to create authenticated context
func createAuthenticatedContext(userID int) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	return c, w
}

func TestAccountHandlers_GetUserAccounts(t *testing.T) {
	handlers, mockAccountService := setupAccountHandlersTest()

	tests := []struct {
		name           string
		userID         int
		mockSetup      func(*MockAccountService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "successful retrieval",
			userID: 1,
			mockSetup: func(m *MockAccountService) {
				accounts := []*models.Account{
					{
						ID:        1,
						UserID:    1,
						Currency:  "USD",
						Balance:   decimal.NewFromFloat(100.50),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					{
						ID:        2,
						UserID:    1,
						Currency:  "EUR",
						Balance:   decimal.NewFromFloat(200.75),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				}
				m.On("GetUserAccounts", mock.Anything, int32(1)).Return(accounts, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "empty accounts list",
			userID: 2,
			mockSetup: func(m *MockAccountService) {
				accounts := []*models.Account{}
				m.On("GetUserAccounts", mock.Anything, int32(2)).Return(accounts, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "service error",
			userID: 3,
			mockSetup: func(m *MockAccountService) {
				m.On("GetUserAccounts", mock.Anything, int32(3)).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockAccountService.ExpectedCalls = nil
			mockAccountService.Calls = nil

			if tt.mockSetup != nil {
				tt.mockSetup(mockAccountService)
			}

			// Create authenticated context
			c, w := createAuthenticatedContext(tt.userID)
			c.Request = httptest.NewRequest(http.MethodGet, "/accounts", nil)

			// Call handler
			handlers.GetUserAccounts(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			} else if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "accounts")
				assert.Contains(t, response, "count")
			}

			mockAccountService.AssertExpectations(t)
		})
	}
}

func TestAccountHandlers_CreateAccount(t *testing.T) {
	handlers, mockAccountService := setupAccountHandlersTest()

	tests := []struct {
		name           string
		userID         int
		requestBody    CreateAccountRequest
		mockSetup      func(*MockAccountService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "successful creation",
			userID: 1,
			requestBody: CreateAccountRequest{
				Currency: "USD",
			},
			mockSetup: func(m *MockAccountService) {
				account := &models.Account{
					ID:        1,
					UserID:    1,
					Currency:  "USD",
					Balance:   decimal.Zero,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("CreateAccount", mock.Anything, int32(1), "USD").Return(account, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "invalid currency format",
			userID: 1,
			requestBody: CreateAccountRequest{
				Currency: "US", // Too short
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name:   "invalid currency code",
			userID: 1,
			requestBody: CreateAccountRequest{
				Currency: "XYZ",
			},
			mockSetup: func(m *MockAccountService) {
				m.On("CreateAccount", mock.Anything, int32(1), "XYZ").
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "duplicate currency",
			userID: 1,
			requestBody: CreateAccountRequest{
				Currency: "USD",
			},
			mockSetup: func(m *MockAccountService) {
				m.On("CreateAccount", mock.Anything, int32(1), "USD").
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "missing currency",
			userID: 1,
			requestBody: CreateAccountRequest{
				// Missing currency
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockAccountService.ExpectedCalls = nil
			mockAccountService.Calls = nil

			if tt.mockSetup != nil {
				tt.mockSetup(mockAccountService)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create authenticated context
			c, w := createAuthenticatedContext(tt.userID)
			c.Request = req

			// Call handler
			handlers.CreateAccount(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			} else if tt.expectedStatus == http.StatusCreated {
				var response models.Account
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.userID, response.UserID)
			}

			mockAccountService.AssertExpectations(t)
		})
	}
}

func TestAccountHandlers_GetAccount(t *testing.T) {
	handlers, mockAccountService := setupAccountHandlersTest()

	tests := []struct {
		name           string
		userID         int
		accountID      string
		mockSetup      func(*MockAccountService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:      "successful retrieval",
			userID:    1,
			accountID: "1",
			mockSetup: func(m *MockAccountService) {
				account := &models.Account{
					ID:        1,
					UserID:    1,
					Currency:  "USD",
					Balance:   decimal.NewFromFloat(100.50),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("GetAccount", mock.Anything, int32(1), int32(1)).Return(account, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "account not found",
			userID:    1,
			accountID: "999",
			mockSetup: func(m *MockAccountService) {
				m.On("GetAccount", mock.Anything, int32(999), int32(1)).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "access denied - different user",
			userID:    1,
			accountID: "2",
			mockSetup: func(m *MockAccountService) {
				m.On("GetAccount", mock.Anything, int32(2), int32(1)).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "invalid account ID",
			userID:         1,
			accountID:      "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_account_id",
		},
		{
			name:           "zero account ID",
			userID:         1,
			accountID:      "0",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_account_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockAccountService.ExpectedCalls = nil
			mockAccountService.Calls = nil

			if tt.mockSetup != nil {
				tt.mockSetup(mockAccountService)
			}

			// Create authenticated context
			c, w := createAuthenticatedContext(tt.userID)
			c.Request = httptest.NewRequest(http.MethodGet, "/accounts/"+tt.accountID, nil)
			
			// Set URL parameter
			c.Params = gin.Params{
				{Key: "id", Value: tt.accountID},
			}

			// Call handler
			handlers.GetAccount(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			} else if tt.expectedStatus == http.StatusOK {
				var response models.Account
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.userID, response.UserID)
			}

			mockAccountService.AssertExpectations(t)
		})
	}
}

func TestAccountHandlers_UpdateAccount(t *testing.T) {
	handlers, mockAccountService := setupAccountHandlersTest()

	tests := []struct {
		name           string
		userID         int
		accountID      string
		requestBody    UpdateAccountRequest
		mockSetup      func(*MockAccountService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:      "successful update",
			userID:    1,
			accountID: "1",
			requestBody: UpdateAccountRequest{
				// Currently no fields to update
			},
			mockSetup: func(m *MockAccountService) {
				account := &models.Account{
					ID:        1,
					UserID:    1,
					Currency:  "USD",
					Balance:   decimal.NewFromFloat(100.50),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("UpdateAccount", mock.Anything, int32(1), int32(1), mock.AnythingOfType("services.UpdateAccountRequest")).
					Return(account, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "account not found",
			userID:    1,
			accountID: "999",
			requestBody: UpdateAccountRequest{},
			mockSetup: func(m *MockAccountService) {
				m.On("UpdateAccount", mock.Anything, int32(999), int32(1), mock.AnythingOfType("services.UpdateAccountRequest")).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "invalid account ID",
			userID:         1,
			accountID:      "invalid",
			requestBody:    UpdateAccountRequest{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_account_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockAccountService.ExpectedCalls = nil
			mockAccountService.Calls = nil

			if tt.mockSetup != nil {
				tt.mockSetup(mockAccountService)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/accounts/"+tt.accountID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create authenticated context
			c, w := createAuthenticatedContext(tt.userID)
			c.Request = req
			
			// Set URL parameter
			c.Params = gin.Params{
				{Key: "id", Value: tt.accountID},
			}

			// Call handler
			handlers.UpdateAccount(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			} else if tt.expectedStatus == http.StatusOK {
				var response models.Account
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.userID, response.UserID)
			}

			mockAccountService.AssertExpectations(t)
		})
	}
}

func TestAccountHandlers_DeleteAccount(t *testing.T) {
	handlers, mockAccountService := setupAccountHandlersTest()

	tests := []struct {
		name           string
		userID         int
		accountID      string
		mockSetup      func(*MockAccountService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:      "successful deletion",
			userID:    1,
			accountID: "1",
			mockSetup: func(m *MockAccountService) {
				m.On("DeleteAccount", mock.Anything, int32(1), int32(1)).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "account not found",
			userID:    1,
			accountID: "999",
			mockSetup: func(m *MockAccountService) {
				m.On("DeleteAccount", mock.Anything, int32(999), int32(1)).
					Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "invalid account ID",
			userID:         1,
			accountID:      "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_account_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockAccountService.ExpectedCalls = nil
			mockAccountService.Calls = nil

			if tt.mockSetup != nil {
				tt.mockSetup(mockAccountService)
			}

			// Create authenticated context
			c, w := createAuthenticatedContext(tt.userID)
			c.Request = httptest.NewRequest(http.MethodDelete, "/accounts/"+tt.accountID, nil)
			
			// Set URL parameter
			c.Params = gin.Params{
				{Key: "id", Value: tt.accountID},
			}

			// Call handler
			handlers.DeleteAccount(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			} else if tt.expectedStatus == http.StatusOK {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Account deleted successfully", response["message"])
			}

			mockAccountService.AssertExpectations(t)
		})
	}
}

func TestAccountHandlers_UnauthenticatedRequests(t *testing.T) {
	handlers, _ := setupAccountHandlersTest()

	tests := []struct {
		name    string
		handler func(*gin.Context)
		method  string
		path    string
	}{
		{
			name:    "GetUserAccounts without auth",
			handler: handlers.GetUserAccounts,
			method:  http.MethodGet,
			path:    "/accounts",
		},
		{
			name:    "CreateAccount without auth",
			handler: handlers.CreateAccount,
			method:  http.MethodPost,
			path:    "/accounts",
		},
		{
			name:    "GetAccount without auth",
			handler: handlers.GetAccount,
			method:  http.MethodGet,
			path:    "/accounts/1",
		},
		{
			name:    "UpdateAccount without auth",
			handler: handlers.UpdateAccount,
			method:  http.MethodPut,
			path:    "/accounts/1",
		},
		{
			name:    "DeleteAccount without auth",
			handler: handlers.DeleteAccount,
			method:  http.MethodDelete,
			path:    "/accounts/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(tt.method, tt.path, nil)
			
			// Set URL parameter for handlers that need it
			if tt.path == "/accounts/1" {
				c.Params = gin.Params{
					{Key: "id", Value: "1"},
				}
			}

			// Call handler without setting user_id in context
			tt.handler(c)

			// Should return unauthorized
			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, "unauthorized", response.Error)
		})
	}
}