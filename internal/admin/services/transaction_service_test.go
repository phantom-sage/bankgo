package services

import (
	"context"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
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

func TestTransactionService_SearchTransactions(t *testing.T) {
	mockService := new(MockTransactionService)
	ctx := context.Background()

	// Test data
	params := interfaces.SearchTransactionParams{
		PaginationParams: interfaces.PaginationParams{
			Page:     1,
			PageSize: 20,
		},
		Currency: "USD",
		Status:   "completed",
	}

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
	mockService.On("SearchTransactions", ctx, params).Return(expectedResult, nil)

	// Execute
	result, err := mockService.SearchTransactions(ctx, params)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Transactions))
	assert.Equal(t, "1", result.Transactions[0].ID)
	assert.Equal(t, "USD", result.Transactions[0].Currency)
	assert.Equal(t, "completed", result.Transactions[0].Status)

	mockService.AssertExpectations(t)
}

func TestTransactionService_GetTransactionDetail(t *testing.T) {
	mockService := new(MockTransactionService)
	ctx := context.Background()

	// Test data
	transactionID := "1"
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
		AuditTrail: []interfaces.AuditEntry{
			{
				ID:        "1",
				Action:    "transfer_created",
				Actor:     "system",
				ActorType: "system",
				Timestamp: time.Now(),
				Details: map[string]interface{}{
					"from_account": "1",
					"to_account":   "2",
					"amount":       "100.00",
					"currency":     "USD",
				},
			},
		},
	}

	// Set up mock expectation
	mockService.On("GetTransactionDetail", ctx, transactionID).Return(expectedDetail, nil)

	// Execute
	result, err := mockService.GetTransactionDetail(ctx, transactionID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "1", result.ID)
	assert.Equal(t, "100.00", result.Amount)
	assert.Equal(t, "completed", result.Status)
	assert.NotNil(t, result.FromAccount)
	assert.NotNil(t, result.ToAccount)
	assert.Equal(t, 1, len(result.AuditTrail))

	mockService.AssertExpectations(t)
}

func TestTransactionService_ReverseTransaction(t *testing.T) {
	mockService := new(MockTransactionService)
	ctx := context.Background()

	// Test data
	transactionID := "1"
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
		AuditTrail: []interfaces.AuditEntry{
			{
				ID:        "1",
				Action:    "transfer_created",
				Actor:     "system",
				ActorType: "system",
				Timestamp: now.Add(-time.Hour),
			},
			{
				ID:        "2",
				Action:    "transfer_reversed",
				Actor:     "admin",
				ActorType: "admin",
				Timestamp: now,
				Details: map[string]interface{}{
					"reason":          reason,
					"reversed_amount": "100.00",
				},
			},
		},
	}

	// Set up mock expectation
	mockService.On("ReverseTransaction", ctx, transactionID, reason).Return(expectedDetail, nil)

	// Execute
	result, err := mockService.ReverseTransaction(ctx, transactionID, reason)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "1", result.ID)
	assert.Equal(t, "reversed", result.Status)
	assert.NotNil(t, result.ReversedAt)
	assert.Equal(t, reason, result.ReversalReason)
	assert.Equal(t, 2, len(result.AuditTrail))

	mockService.AssertExpectations(t)
}

func TestTransactionService_GetAccountTransactions(t *testing.T) {
	mockService := new(MockTransactionService)
	ctx := context.Background()

	// Test data
	accountID := "1"
	params := interfaces.PaginationParams{
		Page:     1,
		PageSize: 10,
	}

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
			{
				ID:            "2",
				FromAccountID: "3",
				ToAccountID:   "1",
				Amount:        "50.00",
				Currency:      "USD",
				Status:        "completed",
				CreatedAt:     time.Now().Add(-time.Hour),
			},
		},
		Pagination: interfaces.PaginationInfo{
			Page:       1,
			PageSize:   10,
			TotalItems: 2,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}

	// Set up mock expectation
	mockService.On("GetAccountTransactions", ctx, accountID, params).Return(expectedResult, nil)

	// Execute
	result, err := mockService.GetAccountTransactions(ctx, accountID, params)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Transactions))
	assert.Equal(t, "1", result.Transactions[0].ID)
	assert.Equal(t, "2", result.Transactions[1].ID)

	mockService.AssertExpectations(t)
}

func TestTransactionService_SearchTransactions_WithFilters(t *testing.T) {
	mockService := new(MockTransactionService)
	ctx := context.Background()

	// Test with multiple filters
	amountMin := "50.00"
	amountMax := "200.00"
	dateFrom := time.Now().Add(-24 * time.Hour)
	dateTo := time.Now()

	params := interfaces.SearchTransactionParams{
		PaginationParams: interfaces.PaginationParams{
			Page:     1,
			PageSize: 20,
		},
		UserID:      "1",
		Currency:    "USD",
		Status:      "completed",
		AmountMin:   &amountMin,
		AmountMax:   &amountMax,
		DateFrom:    &dateFrom,
		DateTo:      &dateTo,
		Description: "test",
	}

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
	mockService.On("SearchTransactions", ctx, params).Return(expectedResult, nil)

	// Execute
	result, err := mockService.SearchTransactions(ctx, params)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.Transactions))

	mockService.AssertExpectations(t)
}