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

// Mock transfer service for testing
type MockTransferService struct {
	mock.Mock
}

func (m *MockTransferService) TransferMoney(ctx context.Context, req services.TransferMoneyRequest) (*models.Transfer, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Transfer), args.Error(1)
}

func (m *MockTransferService) GetTransferHistory(ctx context.Context, req services.GetTransferHistoryRequest) (*services.TransferHistoryResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TransferHistoryResponse), args.Error(1)
}

func (m *MockTransferService) GetTransfer(ctx context.Context, transferID int32) (*models.Transfer, error) {
	args := m.Called(ctx, transferID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Transfer), args.Error(1)
}

func (m *MockTransferService) UpdateTransferStatus(ctx context.Context, transferID int32, status string) (*models.Transfer, error) {
	args := m.Called(ctx, transferID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Transfer), args.Error(1)
}

func (m *MockTransferService) GetTransfersByStatus(ctx context.Context, status string, limit, offset int32) (*services.TransferHistoryResponse, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TransferHistoryResponse), args.Error(1)
}

func (m *MockTransferService) GetTransfersByUser(ctx context.Context, userID int32, limit, offset int32) (*services.TransferHistoryResponse, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.TransferHistoryResponse), args.Error(1)
}

// Test setup helper for transfer handlers
func setupTransferHandlersTest() (*TransferHandlers, *MockTransferService, *MockAccountService) {
	gin.SetMode(gin.TestMode)
	
	mockTransferService := &MockTransferService{}
	mockAccountService := &MockAccountService{}
	handlers := NewTransferHandlers(mockTransferService, mockAccountService)
	
	return handlers, mockTransferService, mockAccountService
}

func TestTransferHandlers_CreateTransfer(t *testing.T) {
	handlers, mockTransferService, mockAccountService := setupTransferHandlersTest()

	tests := []struct {
		name           string
		userID         int
		requestBody    TransferRequest
		mockSetup      func(*MockTransferService, *MockAccountService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "successful transfer",
			userID: 1,
			requestBody: TransferRequest{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(100.50),
				Description:   "Test transfer",
			},
			mockSetup: func(mt *MockTransferService, ma *MockAccountService) {
				// Mock account ownership verification
				account := &models.Account{
					ID:       1,
					UserID:   1,
					Currency: "USD",
					Balance:  decimal.NewFromFloat(200.00),
				}
				ma.On("GetAccount", mock.Anything, int32(1), int32(1)).Return(account, nil)

				// Mock successful transfer
				transfer := &models.Transfer{
					ID:            1,
					FromAccountID: 1,
					ToAccountID:   2,
					Amount:        decimal.NewFromFloat(100.50),
					Description:   "Test transfer",
					Status:        "completed",
					CreatedAt:     time.Now(),
				}
				transferReq := services.TransferMoneyRequest{
					FromAccountID: 1,
					ToAccountID:   2,
					Amount:        decimal.NewFromFloat(100.50),
					Description:   "Test transfer",
				}
				mt.On("TransferMoney", mock.Anything, transferReq).Return(transfer, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "invalid amount - zero",
			userID: 1,
			requestBody: TransferRequest{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.Zero,
				Description:   "Test transfer",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_amount",
		},
		{
			name:   "invalid amount - negative",
			userID: 1,
			requestBody: TransferRequest{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(-50.00),
				Description:   "Test transfer",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_amount",
		},
		{
			name:   "source account not found",
			userID: 1,
			requestBody: TransferRequest{
				FromAccountID: 999,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(100.50),
				Description:   "Test transfer",
			},
			mockSetup: func(mt *MockTransferService, ma *MockAccountService) {
				ma.On("GetAccount", mock.Anything, int32(999), int32(1)).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "access denied - not user's account",
			userID: 1,
			requestBody: TransferRequest{
				FromAccountID: 2,
				ToAccountID:   3,
				Amount:        decimal.NewFromFloat(100.50),
				Description:   "Test transfer",
			},
			mockSetup: func(mt *MockTransferService, ma *MockAccountService) {
				ma.On("GetAccount", mock.Anything, int32(2), int32(1)).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "missing required fields",
			userID: 1,
			requestBody: TransferRequest{
				FromAccountID: 1,
				// Missing ToAccountID and Amount
				Description: "Test transfer",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockTransferService.ExpectedCalls = nil
			mockTransferService.Calls = nil
			mockAccountService.ExpectedCalls = nil
			mockAccountService.Calls = nil

			if tt.mockSetup != nil {
				tt.mockSetup(mockTransferService, mockAccountService)
			}

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create authenticated context
			c, w := createAuthenticatedContext(tt.userID)
			c.Request = req

			// Call handler
			handlers.CreateTransfer(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			} else if tt.expectedStatus == http.StatusCreated {
				var response models.Transfer
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.requestBody.FromAccountID, response.FromAccountID)
				assert.Equal(t, tt.requestBody.ToAccountID, response.ToAccountID)
			}

			mockTransferService.AssertExpectations(t)
			mockAccountService.AssertExpectations(t)
		})
	}
}

func TestTransferHandlers_GetTransferHistory(t *testing.T) {
	handlers, mockTransferService, mockAccountService := setupTransferHandlersTest()

	tests := []struct {
		name           string
		userID         int
		queryParams    map[string]string
		mockSetup      func(*MockTransferService, *MockAccountService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "get user transfers - no account specified",
			userID: 1,
			queryParams: map[string]string{
				"limit":  "10",
				"offset": "0",
			},
			mockSetup: func(mt *MockTransferService, ma *MockAccountService) {
				transfers := []models.Transfer{
					{
						ID:            1,
						FromAccountID: 1,
						ToAccountID:   2,
						Amount:        decimal.NewFromFloat(100.50),
						Status:        "completed",
						CreatedAt:     time.Now(),
					},
				}
				response := &services.TransferHistoryResponse{
					Transfers: transfers,
					Total:     1,
					Limit:     10,
					Offset:    0,
				}
				mt.On("GetTransfersByUser", mock.Anything, int32(1), int32(10), int32(0)).Return(response, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "get account transfers - specific account",
			userID: 1,
			queryParams: map[string]string{
				"account_id": "1",
				"limit":      "20",
				"offset":     "0",
			},
			mockSetup: func(mt *MockTransferService, ma *MockAccountService) {
				// Mock account ownership verification
				account := &models.Account{
					ID:       1,
					UserID:   1,
					Currency: "USD",
					Balance:  decimal.NewFromFloat(200.00),
				}
				ma.On("GetAccount", mock.Anything, int32(1), int32(1)).Return(account, nil)

				// Mock transfer history
				transfers := []models.Transfer{
					{
						ID:            1,
						FromAccountID: 1,
						ToAccountID:   2,
						Amount:        decimal.NewFromFloat(100.50),
						Status:        "completed",
						CreatedAt:     time.Now(),
					},
				}
				response := &services.TransferHistoryResponse{
					Transfers: transfers,
					Total:     1,
					Limit:     20,
					Offset:    0,
				}
				historyReq := services.GetTransferHistoryRequest{
					AccountID: 1,
					Limit:     20,
					Offset:    0,
				}
				mt.On("GetTransferHistory", mock.Anything, historyReq).Return(response, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "invalid account ID parameter",
			userID: 1,
			queryParams: map[string]string{
				"account_id": "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_account_id",
		},
		{
			name:   "account not found",
			userID: 1,
			queryParams: map[string]string{
				"account_id": "999",
			},
			mockSetup: func(mt *MockTransferService, ma *MockAccountService) {
				ma.On("GetAccount", mock.Anything, int32(999), int32(1)).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "default pagination values",
			userID: 1,
			queryParams: map[string]string{
				// No limit/offset specified, should use defaults
			},
			mockSetup: func(mt *MockTransferService, ma *MockAccountService) {
				response := &services.TransferHistoryResponse{
					Transfers: []models.Transfer{},
					Total:     0,
					Limit:     20, // Default limit
					Offset:    0,  // Default offset
				}
				mt.On("GetTransfersByUser", mock.Anything, int32(1), int32(20), int32(0)).Return(response, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockTransferService.ExpectedCalls = nil
			mockTransferService.Calls = nil
			mockAccountService.ExpectedCalls = nil
			mockAccountService.Calls = nil

			if tt.mockSetup != nil {
				tt.mockSetup(mockTransferService, mockAccountService)
			}

			// Build URL with query parameters
			url := "/transfers"
			if len(tt.queryParams) > 0 {
				url += "?"
				first := true
				for key, value := range tt.queryParams {
					if !first {
						url += "&"
					}
					url += key + "=" + value
					first = false
				}
			}

			// Create authenticated context
			c, w := createAuthenticatedContext(tt.userID)
			c.Request = httptest.NewRequest(http.MethodGet, url, nil)

			// Call handler
			handlers.GetTransferHistory(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			} else if tt.expectedStatus == http.StatusOK {
				var response services.TransferHistoryResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, response.Limit, int32(0))
				assert.GreaterOrEqual(t, response.Offset, int32(0))
			}

			mockTransferService.AssertExpectations(t)
			mockAccountService.AssertExpectations(t)
		})
	}
}

func TestTransferHandlers_GetTransfer(t *testing.T) {
	handlers, mockTransferService, mockAccountService := setupTransferHandlersTest()

	tests := []struct {
		name           string
		userID         int
		transferID     string
		mockSetup      func(*MockTransferService, *MockAccountService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:       "successful retrieval - user owns source account",
			userID:     1,
			transferID: "1",
			mockSetup: func(mt *MockTransferService, ma *MockAccountService) {
				// Mock transfer retrieval
				transfer := &models.Transfer{
					ID:            1,
					FromAccountID: 1,
					ToAccountID:   2,
					Amount:        decimal.NewFromFloat(100.50),
					Status:        "completed",
					CreatedAt:     time.Now(),
				}
				mt.On("GetTransfer", mock.Anything, int32(1)).Return(transfer, nil)

				// Mock account ownership verification (user owns source account)
				account := &models.Account{
					ID:       1,
					UserID:   1,
					Currency: "USD",
					Balance:  decimal.NewFromFloat(200.00),
				}
				ma.On("GetAccount", mock.Anything, int32(1), int32(1)).Return(account, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "successful retrieval - user owns destination account",
			userID:     1,
			transferID: "1",
			mockSetup: func(mt *MockTransferService, ma *MockAccountService) {
				// Mock transfer retrieval
				transfer := &models.Transfer{
					ID:            1,
					FromAccountID: 2,
					ToAccountID:   1,
					Amount:        decimal.NewFromFloat(100.50),
					Status:        "completed",
					CreatedAt:     time.Now(),
				}
				mt.On("GetTransfer", mock.Anything, int32(1)).Return(transfer, nil)

				// Mock account ownership verification (user doesn't own source, but owns destination)
				ma.On("GetAccount", mock.Anything, int32(2), int32(1)).Return(nil, assert.AnError)
				account := &models.Account{
					ID:       1,
					UserID:   1,
					Currency: "USD",
					Balance:  decimal.NewFromFloat(200.00),
				}
				ma.On("GetAccount", mock.Anything, int32(1), int32(1)).Return(account, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "access denied - user owns neither account",
			userID:     1,
			transferID: "1",
			mockSetup: func(mt *MockTransferService, ma *MockAccountService) {
				// Mock transfer retrieval
				transfer := &models.Transfer{
					ID:            1,
					FromAccountID: 2,
					ToAccountID:   3,
					Amount:        decimal.NewFromFloat(100.50),
					Status:        "completed",
					CreatedAt:     time.Now(),
				}
				mt.On("GetTransfer", mock.Anything, int32(1)).Return(transfer, nil)

				// Mock account ownership verification (user owns neither account)
				ma.On("GetAccount", mock.Anything, int32(2), int32(1)).Return(nil, assert.AnError)
				ma.On("GetAccount", mock.Anything, int32(3), int32(1)).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "access_denied",
		},
		{
			name:       "transfer not found",
			userID:     1,
			transferID: "999",
			mockSetup: func(mt *MockTransferService, ma *MockAccountService) {
				mt.On("GetTransfer", mock.Anything, int32(999)).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "invalid transfer ID",
			userID:         1,
			transferID:     "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_transfer_id",
		},
		{
			name:           "zero transfer ID",
			userID:         1,
			transferID:     "0",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_transfer_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockTransferService.ExpectedCalls = nil
			mockTransferService.Calls = nil
			mockAccountService.ExpectedCalls = nil
			mockAccountService.Calls = nil

			if tt.mockSetup != nil {
				tt.mockSetup(mockTransferService, mockAccountService)
			}

			// Create authenticated context
			c, w := createAuthenticatedContext(tt.userID)
			c.Request = httptest.NewRequest(http.MethodGet, "/transfers/"+tt.transferID, nil)
			
			// Set URL parameter
			c.Params = gin.Params{
				{Key: "id", Value: tt.transferID},
			}

			// Call handler
			handlers.GetTransfer(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			} else if tt.expectedStatus == http.StatusOK {
				var response models.Transfer
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Greater(t, response.ID, 0)
			}

			mockTransferService.AssertExpectations(t)
			mockAccountService.AssertExpectations(t)
		})
	}
}

func TestTransferHandlers_UnauthenticatedRequests(t *testing.T) {
	handlers, _, _ := setupTransferHandlersTest()

	tests := []struct {
		name    string
		handler func(*gin.Context)
		method  string
		path    string
	}{
		{
			name:    "CreateTransfer without auth",
			handler: handlers.CreateTransfer,
			method:  http.MethodPost,
			path:    "/transfers",
		},
		{
			name:    "GetTransferHistory without auth",
			handler: handlers.GetTransferHistory,
			method:  http.MethodGet,
			path:    "/transfers",
		},
		{
			name:    "GetTransfer without auth",
			handler: handlers.GetTransfer,
			method:  http.MethodGet,
			path:    "/transfers/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(tt.method, tt.path, nil)
			
			// Set URL parameter for handlers that need it
			if tt.path == "/transfers/1" {
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