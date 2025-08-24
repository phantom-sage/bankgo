package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAccountService is a mock implementation of AccountService
type MockAccountService struct {
	mock.Mock
}

func (m *MockAccountService) SearchAccounts(ctx context.Context, params interfaces.SearchAccountParams) (*interfaces.PaginatedAccounts, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*interfaces.PaginatedAccounts), args.Error(1)
}

func (m *MockAccountService) GetAccountDetail(ctx context.Context, accountID string) (*interfaces.AccountDetail, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.AccountDetail), args.Error(1)
}

func (m *MockAccountService) FreezeAccount(ctx context.Context, accountID string, reason string) error {
	args := m.Called(ctx, accountID, reason)
	return args.Error(0)
}

func (m *MockAccountService) UnfreezeAccount(ctx context.Context, accountID string) error {
	args := m.Called(ctx, accountID)
	return args.Error(0)
}

func (m *MockAccountService) AdjustBalance(ctx context.Context, accountID string, adjustment string, reason string) (*interfaces.AccountDetail, error) {
	args := m.Called(ctx, accountID, adjustment, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.AccountDetail), args.Error(1)
}

func TestAccountHandler_SearchAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAccountService)
	handler := NewAccountHandler(mockService)

	// Test data
	expectedResult := &interfaces.PaginatedAccounts{
		Accounts: []interfaces.AccountDetail{
			{
				ID:       "1",
				UserID:   "1",
				Currency: "USD",
				Balance:  "1000.00",
				IsActive: true,
				CreatedAt: time.Now(),
				User: interfaces.UserSummary{
					ID:        "1",
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					IsActive:  true,
				},
			},
		},
		Pagination: interfaces.PaginationInfo{
			Page:       1,
			PageSize:   20,
			TotalItems: 1,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}

	// Set up mock expectation
	mockService.On("SearchAccounts", mock.Anything, mock.AnythingOfType("interfaces.SearchAccountParams")).Return(expectedResult, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/accounts?currency=USD", nil)

	// Execute
	handler.SearchAccounts(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response interfaces.PaginatedAccounts
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(response.Accounts))
	assert.Equal(t, "1", response.Accounts[0].ID)

	mockService.AssertExpectations(t)
}

func TestAccountHandler_GetAccountDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAccountService)
	handler := NewAccountHandler(mockService)

	// Test data
	expectedDetail := &interfaces.AccountDetail{
		ID:       "1",
		UserID:   "1",
		Currency: "USD",
		Balance:  "1000.00",
		IsActive: true,
		IsFrozen: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		User: interfaces.UserSummary{
			ID:        "1",
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
			IsActive:  true,
		},
	}

	// Set up mock expectation
	mockService.On("GetAccountDetail", mock.Anything, "1").Return(expectedDetail, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/accounts/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	// Execute
	handler.GetAccountDetail(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response interfaces.AccountDetail
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "1", response.ID)
	assert.Equal(t, "1000.00", response.Balance)
	assert.Equal(t, "test@example.com", response.User.Email)

	mockService.AssertExpectations(t)
}

func TestAccountHandler_FreezeAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAccountService)
	handler := NewAccountHandler(mockService)

	// Test data
	reason := "Suspicious activity detected"

	// Set up mock expectation
	mockService.On("FreezeAccount", mock.Anything, "1", reason).Return(nil)

	// Create request body
	requestBody := FreezeAccountRequest{
		Reason: reason,
	}
	jsonBody, _ := json.Marshal(requestBody)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/accounts/1/freeze", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	// Execute
	handler.FreezeAccount(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "frozen successfully")

	mockService.AssertExpectations(t)
}

func TestAccountHandler_FreezeAccount_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAccountService)
	handler := NewAccountHandler(mockService)

	// Create request with empty reason
	requestBody := FreezeAccountRequest{
		Reason: "",
	}
	jsonBody, _ := json.Marshal(requestBody)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/accounts/1/freeze", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	// Execute
	handler.FreezeAccount(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "validation_error", response.Error)
	assert.Contains(t, response.Message, "Reason")
}

func TestAccountHandler_UnfreezeAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAccountService)
	handler := NewAccountHandler(mockService)

	// Set up mock expectation
	mockService.On("UnfreezeAccount", mock.Anything, "1").Return(nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/accounts/1/unfreeze", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	// Execute
	handler.UnfreezeAccount(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Contains(t, response.Message, "unfrozen successfully")

	mockService.AssertExpectations(t)
}

func TestAccountHandler_AdjustBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAccountService)
	handler := NewAccountHandler(mockService)

	// Test data
	amount := "100.00"
	reason := "Manual correction"

	expectedDetail := &interfaces.AccountDetail{
		ID:       "1",
		UserID:   "1",
		Currency: "USD",
		Balance:  "1100.00",
		IsActive: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		User: interfaces.UserSummary{
			ID:        "1",
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
			IsActive:  true,
		},
	}

	// Set up mock expectation
	mockService.On("AdjustBalance", mock.Anything, "1", amount, reason).Return(expectedDetail, nil)

	// Create request body
	requestBody := AdjustBalanceRequest{
		Amount: amount,
		Reason: reason,
	}
	jsonBody, _ := json.Marshal(requestBody)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/accounts/1/adjust-balance", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	// Execute
	handler.AdjustBalance(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response interfaces.AccountDetail
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "1", response.ID)
	assert.Equal(t, "1100.00", response.Balance)

	mockService.AssertExpectations(t)
}

func TestAccountHandler_AdjustBalance_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAccountService)
	handler := NewAccountHandler(mockService)

	// Create request with missing amount
	requestBody := AdjustBalanceRequest{
		Amount: "",
		Reason: "Test reason",
	}
	jsonBody, _ := json.Marshal(requestBody)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/accounts/1/adjust-balance", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	// Execute
	handler.AdjustBalance(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "validation_error", response.Error)
	assert.Contains(t, response.Message, "Amount")
}

func TestAccountHandler_SearchAccounts_WithFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAccountService)
	handler := NewAccountHandler(mockService)

	// Test data
	expectedResult := &interfaces.PaginatedAccounts{
		Accounts: []interfaces.AccountDetail{},
		Pagination: interfaces.PaginationInfo{
			Page:       1,
			PageSize:   10,
			TotalItems: 0,
			TotalPages: 0,
			HasNext:    false,
			HasPrev:    false,
		},
	}

	// Set up mock expectation
	mockService.On("SearchAccounts", mock.Anything, mock.AnythingOfType("interfaces.SearchAccountParams")).Return(expectedResult, nil)

	// Create request with multiple filters
	url := "/accounts?search=john&currency=USD&balance_min=500.00&balance_max=2000.00&is_active=true&page=1&page_size=10"
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", url, nil)

	// Execute
	handler.SearchAccounts(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response interfaces.PaginatedAccounts
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}