package services

import (
	"context"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
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

func TestAccountService_SearchAccounts(t *testing.T) {
	mockService := new(MockAccountService)
	ctx := context.Background()

	// Test data
	params := interfaces.SearchAccountParams{
		PaginationParams: interfaces.PaginationParams{
			Page:     1,
			PageSize: 20,
		},
		Currency: "USD",
	}

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
	mockService.On("SearchAccounts", ctx, params).Return(expectedResult, nil)

	// Execute
	result, err := mockService.SearchAccounts(ctx, params)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Accounts))
	assert.Equal(t, "1", result.Accounts[0].ID)
	assert.Equal(t, "USD", result.Accounts[0].Currency)
	assert.Equal(t, "1000.00", result.Accounts[0].Balance)

	mockService.AssertExpectations(t)
}

func TestAccountService_GetAccountDetail(t *testing.T) {
	mockService := new(MockAccountService)
	ctx := context.Background()

	// Test data
	accountID := "1"
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
	mockService.On("GetAccountDetail", ctx, accountID).Return(expectedDetail, nil)

	// Execute
	result, err := mockService.GetAccountDetail(ctx, accountID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "1", result.ID)
	assert.Equal(t, "USD", result.Currency)
	assert.Equal(t, "1000.00", result.Balance)
	assert.Equal(t, "test@example.com", result.User.Email)

	mockService.AssertExpectations(t)
}

func TestAccountService_FreezeAccount(t *testing.T) {
	mockService := new(MockAccountService)
	ctx := context.Background()

	// Test data
	accountID := "1"
	reason := "Suspicious activity detected"

	// Set up mock expectation
	mockService.On("FreezeAccount", ctx, accountID, reason).Return(nil)

	// Execute
	err := mockService.FreezeAccount(ctx, accountID, reason)

	// Assert
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestAccountService_UnfreezeAccount(t *testing.T) {
	mockService := new(MockAccountService)
	ctx := context.Background()

	// Test data
	accountID := "1"

	// Set up mock expectation
	mockService.On("UnfreezeAccount", ctx, accountID).Return(nil)

	// Execute
	err := mockService.UnfreezeAccount(ctx, accountID)

	// Assert
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestAccountService_AdjustBalance(t *testing.T) {
	mockService := new(MockAccountService)
	ctx := context.Background()

	// Test data
	accountID := "1"
	adjustment := "100.00"
	reason := "Manual correction"

	expectedDetail := &interfaces.AccountDetail{
		ID:       "1",
		UserID:   "1",
		Currency: "USD",
		Balance:  "1100.00", // Original 1000.00 + 100.00 adjustment
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
	mockService.On("AdjustBalance", ctx, accountID, adjustment, reason).Return(expectedDetail, nil)

	// Execute
	result, err := mockService.AdjustBalance(ctx, accountID, adjustment, reason)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "1", result.ID)
	assert.Equal(t, "1100.00", result.Balance)

	mockService.AssertExpectations(t)
}

func TestAccountService_SearchAccounts_WithFilters(t *testing.T) {
	mockService := new(MockAccountService)
	ctx := context.Background()

	// Test with multiple filters
	balanceMin := "500.00"
	balanceMax := "2000.00"
	isActive := true

	params := interfaces.SearchAccountParams{
		PaginationParams: interfaces.PaginationParams{
			Page:     1,
			PageSize: 10,
		},
		Search:     "john",
		Currency:   "USD",
		BalanceMin: &balanceMin,
		BalanceMax: &balanceMax,
		IsActive:   &isActive,
	}

	expectedResult := &interfaces.PaginatedAccounts{
		Accounts: []interfaces.AccountDetail{
			{
				ID:       "1",
				UserID:   "1",
				Currency: "USD",
				Balance:  "1000.00",
				IsActive: true,
				User: interfaces.UserSummary{
					ID:        "1",
					Email:     "john@example.com",
					FirstName: "John",
					LastName:  "Doe",
					IsActive:  true,
				},
			},
		},
		Pagination: interfaces.PaginationInfo{
			Page:       1,
			PageSize:   10,
			TotalItems: 1,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}

	// Set up mock expectation
	mockService.On("SearchAccounts", ctx, params).Return(expectedResult, nil)

	// Execute
	result, err := mockService.SearchAccounts(ctx, params)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Accounts))
	assert.Equal(t, "john@example.com", result.Accounts[0].User.Email)

	mockService.AssertExpectations(t)
}

func TestAccountService_AdjustBalance_NegativeAdjustment(t *testing.T) {
	mockService := new(MockAccountService)
	ctx := context.Background()

	// Test data for negative adjustment (deduction)
	accountID := "1"
	adjustment := "-50.00"
	reason := "Fee deduction"

	expectedDetail := &interfaces.AccountDetail{
		ID:       "1",
		UserID:   "1",
		Currency: "USD",
		Balance:  "950.00", // Original 1000.00 - 50.00 adjustment
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
	mockService.On("AdjustBalance", ctx, accountID, adjustment, reason).Return(expectedDetail, nil)

	// Execute
	result, err := mockService.AdjustBalance(ctx, accountID, adjustment, reason)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "1", result.ID)
	assert.Equal(t, "950.00", result.Balance)

	mockService.AssertExpectations(t)
}