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

// MockTransactionService is a mock implementation of TransactionService
type MockTransactionService struct {
	mock.Mock
}

func (m *MockTransactionService) SearchTransactions(ctx context.Context, params interfaces.SearchTransactionParams) (*interfaces.PaginatedTransactions, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*interfaces.PaginatedTransactions), args.Error(1)
}

func (m *MockTransactionService) GetTransactionDetail(ctx context.Context, transactionID string) (*interfaces.TransactionDetail, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.TransactionDetail), args.Error(1)
}

func (m *MockTransactionService) ReverseTransaction(ctx context.Context, transactionID string, reason string) (*interfaces.TransactionDetail, error) {
	args := m.Called(ctx, transactionID, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.TransactionDetail), args.Error(1)
}

func (m *MockTransactionService) GetAccountTransactions(ctx context.Context, accountID string, params interfaces.PaginationParams) (*interfaces.PaginatedTransactions, error) {
	args := m.Called(ctx, accountID, params)
	return args.Get(0).(*interfaces.PaginatedTransactions), args.Error(1)
}

func TestTransactionHandler_SearchTransactions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	// Test data
	expectedResult := &interfaces.PaginatedTransactions{
		Transactions: []interfaces.TransactionDetail{
			{
				ID:            "1",
				FromAccountID: "1",
				ToAccountID:   "2",
				Amount:        "100.00",
				Currency:      "USD",
				Status:        "completed",
				Description:   "Test transfer",
				CreatedAt:     time.Now(),
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
	mockService.On("SearchTransactions", mock.Anything, mock.AnythingOfType("interfaces.SearchTransactionParams")).Return(expectedResult, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/transactions?currency=USD&status=completed", nil)

	// Execute
	handler.SearchTransactions(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response interfaces.PaginatedTransactions
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(response.Transactions))
	assert.Equal(t, "1", response.Transactions[0].ID)

	mockService.AssertExpectations(t)
}

func TestTransactionHandler_GetTransactionDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	// Test data
	expectedDetail := &interfaces.TransactionDetail{
		ID:            "1",
		FromAccountID: "1",
		ToAccountID:   "2",
		Amount:        "100.00",
		Currency:      "USD",
		Status:        "completed",
		Description:   "Test transfer",
		CreatedAt:     time.Now(),
		FromAccount: &interfaces.AccountSummary{
			ID:       "1",
			UserID:   "1",
			Currency: "USD",
			Balance:  "900.00",
			IsActive: true,
		},
		ToAccount: &interfaces.AccountSummary{
			ID:       "2",
			UserID:   "2",
			Currency: "USD",
			Balance:  "1100.00",
			IsActive: true,
		},
	}

	// Set up mock expectation
	mockService.On("GetTransactionDetail", mock.Anything, "1").Return(expectedDetail, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/transactions/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	// Execute
	handler.GetTransactionDetail(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response interfaces.TransactionDetail
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "1", response.ID)
	assert.Equal(t, "100.00", response.Amount)
	assert.Equal(t, "completed", response.Status)

	mockService.AssertExpectations(t)
}

func TestTransactionHandler_GetTransactionDetail_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	// Set up mock expectation for not found
	mockService.On("GetTransactionDetail", mock.Anything, "999").Return(nil, assert.AnError)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/transactions/999", nil)
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	// Execute
	handler.GetTransactionDetail(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "internal_error", response.Error)

	mockService.AssertExpectations(t)
}

func TestTransactionHandler_ReverseTransaction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	// Test data
	reason := "Fraudulent transaction"
	now := time.Now()
	expectedDetail := &interfaces.TransactionDetail{
		ID:             "1",
		FromAccountID:  "1",
		ToAccountID:    "2",
		Amount:         "100.00",
		Currency:       "USD",
		Status:         "reversed",
		Description:    "Test transfer",
		CreatedAt:      now.Add(-time.Hour),
		ReversedAt:     &now,
		ReversalReason: reason,
	}

	// Set up mock expectation
	mockService.On("ReverseTransaction", mock.Anything, "1", reason).Return(expectedDetail, nil)

	// Create request body
	requestBody := ReverseTransactionRequest{
		Reason: reason,
	}
	jsonBody, _ := json.Marshal(requestBody)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/transactions/1/reverse", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	// Execute
	handler.ReverseTransaction(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response interfaces.TransactionDetail
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "1", response.ID)
	assert.Equal(t, "reversed", response.Status)
	assert.Equal(t, reason, response.ReversalReason)

	mockService.AssertExpectations(t)
}

func TestTransactionHandler_ReverseTransaction_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	// Create request with empty reason
	requestBody := ReverseTransactionRequest{
		Reason: "",
	}
	jsonBody, _ := json.Marshal(requestBody)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/transactions/1/reverse", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	// Execute
	handler.ReverseTransaction(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "validation_error", response.Error)
	assert.Contains(t, response.Message, "Reason")
}

func TestTransactionHandler_GetAccountTransactions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	// Test data
	expectedResult := &interfaces.PaginatedTransactions{
		Transactions: []interfaces.TransactionDetail{
			{
				ID:            "1",
				FromAccountID: "1",
				ToAccountID:   "2",
				Amount:        "100.00",
				Currency:      "USD",
				Status:        "completed",
				CreatedAt:     time.Now(),
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
	mockService.On("GetAccountTransactions", mock.Anything, "1", mock.AnythingOfType("interfaces.PaginationParams")).Return(expectedResult, nil)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/accounts/1/transactions", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	// Execute
	handler.GetAccountTransactions(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response interfaces.PaginatedTransactions
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(response.Transactions))
	assert.Equal(t, "1", response.Transactions[0].ID)

	mockService.AssertExpectations(t)
}

func TestTransactionHandler_SearchTransactions_WithDateFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	// Test data
	expectedResult := &interfaces.PaginatedTransactions{
		Transactions: []interfaces.TransactionDetail{},
		Pagination: interfaces.PaginationInfo{
			Page:       1,
			PageSize:   20,
			TotalItems: 0,
			TotalPages: 0,
			HasNext:    false,
			HasPrev:    false,
		},
	}

	// Set up mock expectation
	mockService.On("SearchTransactions", mock.Anything, mock.AnythingOfType("interfaces.SearchTransactionParams")).Return(expectedResult, nil)

	// Create request with date filters
	dateFrom := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	dateTo := time.Now().Format(time.RFC3339)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/transactions", nil)
	
	// Add query parameters
	q := req.URL.Query()
	q.Add("date_from", dateFrom)
	q.Add("date_to", dateTo)
	req.URL.RawQuery = q.Encode()
	
	c.Request = req

	// Execute
	handler.SearchTransactions(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response interfaces.PaginatedTransactions
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestTransactionHandler_SearchTransactions_InvalidDateFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTransactionService)
	handler := NewTransactionHandler(mockService)

	// Create request with invalid date format
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/transactions?date_from=invalid-date", nil)

	// Execute
	handler.SearchTransactions(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "validation_error", response.Error)
	assert.Contains(t, response.Message, "Invalid date_from format")
}