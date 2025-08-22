package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAccountRepository is a mock implementation of AccountRepository
type MockAccountRepository struct {
	mock.Mock
}

func (m *MockAccountRepository) CreateAccount(ctx context.Context, arg queries.CreateAccountParams) (queries.Account, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Account), args.Error(1)
}

func (m *MockAccountRepository) GetAccount(ctx context.Context, id int32) (queries.Account, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Account), args.Error(1)
}

func (m *MockAccountRepository) GetAccountForUpdate(ctx context.Context, id int32) (queries.Account, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Account), args.Error(1)
}

func (m *MockAccountRepository) GetUserAccounts(ctx context.Context, userID int32) ([]queries.Account, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]queries.Account), args.Error(1)
}

func (m *MockAccountRepository) GetAccountByUserAndCurrency(ctx context.Context, arg queries.GetAccountByUserAndCurrencyParams) (queries.Account, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Account), args.Error(1)
}

func (m *MockAccountRepository) UpdateAccountBalance(ctx context.Context, arg queries.UpdateAccountBalanceParams) (queries.Account, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Account), args.Error(1)
}

func (m *MockAccountRepository) UpdateAccount(ctx context.Context, id int32) (queries.Account, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.Account), args.Error(1)
}

func (m *MockAccountRepository) DeleteAccount(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAccountRepository) ListAccounts(ctx context.Context, arg queries.ListAccountsParams) ([]queries.ListAccountsRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.ListAccountsRow), args.Error(1)
}

func (m *MockAccountRepository) GetAccountsWithBalance(ctx context.Context) ([]queries.Account, error) {
	args := m.Called(ctx)
	return args.Get(0).([]queries.Account), args.Error(1)
}

func (m *MockAccountRepository) AddToBalance(ctx context.Context, arg queries.AddToBalanceParams) (queries.Account, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Account), args.Error(1)
}

func (m *MockAccountRepository) SubtractFromBalance(ctx context.Context, arg queries.SubtractFromBalanceParams) (queries.Account, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Account), args.Error(1)
}

// MockTransferRepository is a mock implementation of TransferRepository
type MockTransferRepository struct {
	mock.Mock
}

func (m *MockTransferRepository) CreateTransfer(ctx context.Context, arg queries.CreateTransferParams) (queries.Transfer, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Transfer), args.Error(1)
}

func (m *MockTransferRepository) GetTransfer(ctx context.Context, id int32) (queries.GetTransferRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.GetTransferRow), args.Error(1)
}

func (m *MockTransferRepository) GetTransfersByAccount(ctx context.Context, arg queries.GetTransfersByAccountParams) ([]queries.GetTransfersByAccountRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetTransfersByAccountRow), args.Error(1)
}

func (m *MockTransferRepository) GetTransfersByUser(ctx context.Context, arg queries.GetTransfersByUserParams) ([]queries.GetTransfersByUserRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetTransfersByUserRow), args.Error(1)
}

func (m *MockTransferRepository) UpdateTransferStatus(ctx context.Context, arg queries.UpdateTransferStatusParams) (queries.Transfer, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.Transfer), args.Error(1)
}

func (m *MockTransferRepository) ListTransfers(ctx context.Context, arg queries.ListTransfersParams) ([]queries.ListTransfersRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.ListTransfersRow), args.Error(1)
}

func (m *MockTransferRepository) GetTransfersByStatus(ctx context.Context, arg queries.GetTransfersByStatusParams) ([]queries.GetTransfersByStatusRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetTransfersByStatusRow), args.Error(1)
}

func (m *MockTransferRepository) GetTransfersByDateRange(ctx context.Context, arg queries.GetTransfersByDateRangeParams) ([]queries.GetTransfersByDateRangeRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.GetTransfersByDateRangeRow), args.Error(1)
}

func (m *MockTransferRepository) CountTransfersByAccount(ctx context.Context, fromAccountID int32) (int64, error) {
	args := m.Called(ctx, fromAccountID)
	return args.Get(0).(int64), args.Error(1)
}

// Helper function to create a valid pgtype.Numeric
func createPgNumeric(value string) pgtype.Numeric {
	dec, _ := decimal.NewFromString(value)
	return pgtype.Numeric{
		Int:   dec.Coefficient(),
		Exp:   dec.Exponent(),
		Valid: true,
	}
}

// Helper function to create a valid pgtype.Timestamp
func createPgTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{
		Time:  t,
		Valid: true,
	}
}

func TestAccountService_CreateAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("successful account creation", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		userID := int32(1)
		currency := "USD"
		now := time.Now()

		// Mock: Check for existing account (should not exist)
		mockAccountRepo.On("GetAccountByUserAndCurrency", ctx, queries.GetAccountByUserAndCurrencyParams{
			UserID:   userID,
			Currency: currency,
		}).Return(queries.Account{}, sql.ErrNoRows)

		// Mock: Create account
		expectedDBAccount := queries.Account{
			ID:        1,
			UserID:    userID,
			Currency:  currency,
			Balance:   createPgNumeric("0.00"),
			CreatedAt: createPgTimestamp(now),
			UpdatedAt: createPgTimestamp(now),
		}
		mockAccountRepo.On("CreateAccount", ctx, queries.CreateAccountParams{
			UserID:   userID,
			Currency: currency,
			Column3:  nil,
		}).Return(expectedDBAccount, nil)

		account, err := service.CreateAccount(ctx, userID, currency)

		assert.NoError(t, err)
		assert.NotNil(t, account)
		assert.Equal(t, 1, account.ID)
		assert.Equal(t, 1, account.UserID)
		assert.Equal(t, "USD", account.Currency)
		assert.True(t, account.Balance.IsZero())
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("invalid currency format", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		userID := int32(1)
		currency := "INVALID"

		account, err := service.CreateAccount(ctx, userID, currency)

		assert.Error(t, err)
		assert.Nil(t, account)
		assert.Contains(t, err.Error(), "invalid currency")
	})

	t.Run("duplicate currency for user", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		userID := int32(1)
		currency := "USD"

		// Mock: Check for existing account (should exist)
		existingAccount := queries.Account{
			ID:       1,
			UserID:   userID,
			Currency: currency,
		}
		mockAccountRepo.On("GetAccountByUserAndCurrency", ctx, queries.GetAccountByUserAndCurrencyParams{
			UserID:   userID,
			Currency: currency,
		}).Return(existingAccount, nil)

		account, err := service.CreateAccount(ctx, userID, currency)

		assert.Error(t, err)
		assert.Nil(t, account)
		assert.Contains(t, err.Error(), "already has an account with currency")
		mockAccountRepo.AssertExpectations(t)
	})
}

func TestAccountService_GetAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("successful account retrieval", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		accountID := int32(1)
		userID := int32(1)
		now := time.Now()

		expectedDBAccount := queries.Account{
			ID:        accountID,
			UserID:    userID,
			Currency:  "USD",
			Balance:   createPgNumeric("100.50"),
			CreatedAt: createPgTimestamp(now),
			UpdatedAt: createPgTimestamp(now),
		}
		mockAccountRepo.On("GetAccount", ctx, accountID).Return(expectedDBAccount, nil)

		account, err := service.GetAccount(ctx, accountID, userID)

		assert.NoError(t, err)
		assert.NotNil(t, account)
		assert.Equal(t, 1, account.ID)
		assert.Equal(t, 1, account.UserID)
		assert.Equal(t, "USD", account.Currency)
		assert.Equal(t, "100.5", account.Balance.String())
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("account not found", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		accountID := int32(999)
		userID := int32(1)

		mockAccountRepo.On("GetAccount", ctx, accountID).Return(queries.Account{}, sql.ErrNoRows)

		account, err := service.GetAccount(ctx, accountID, userID)

		assert.Error(t, err)
		assert.Nil(t, account)
		assert.Contains(t, err.Error(), "account not found")
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("access denied - wrong user", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		accountID := int32(1)
		userID := int32(1)
		wrongUserID := int32(2)

		expectedDBAccount := queries.Account{
			ID:     accountID,
			UserID: wrongUserID, // Different user
		}
		mockAccountRepo.On("GetAccount", ctx, accountID).Return(expectedDBAccount, nil)

		account, err := service.GetAccount(ctx, accountID, userID)

		assert.Error(t, err)
		assert.Nil(t, account)
		assert.Contains(t, err.Error(), "access denied")
		mockAccountRepo.AssertExpectations(t)
	})
}

func TestAccountService_GetUserAccounts(t *testing.T) {
	ctx := context.Background()

	t.Run("successful user accounts retrieval", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		userID := int32(1)
		now := time.Now()

		expectedDBAccounts := []queries.Account{
			{
				ID:        1,
				UserID:    userID,
				Currency:  "USD",
				Balance:   createPgNumeric("100.00"),
				CreatedAt: createPgTimestamp(now),
				UpdatedAt: createPgTimestamp(now),
			},
			{
				ID:        2,
				UserID:    userID,
				Currency:  "EUR",
				Balance:   createPgNumeric("50.25"),
				CreatedAt: createPgTimestamp(now),
				UpdatedAt: createPgTimestamp(now),
			},
		}
		mockAccountRepo.On("GetUserAccounts", ctx, userID).Return(expectedDBAccounts, nil)

		accounts, err := service.GetUserAccounts(ctx, userID)

		assert.NoError(t, err)
		assert.Len(t, accounts, 2)
		assert.Equal(t, "USD", accounts[0].Currency)
		assert.Equal(t, "EUR", accounts[1].Currency)
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("no accounts found", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		userID := int32(1)

		mockAccountRepo.On("GetUserAccounts", ctx, userID).Return([]queries.Account{}, nil)

		accounts, err := service.GetUserAccounts(ctx, userID)

		assert.NoError(t, err)
		assert.Len(t, accounts, 0)
		mockAccountRepo.AssertExpectations(t)
	})
}

func TestAccountService_DeleteAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("successful account deletion", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		accountID := int32(1)
		userID := int32(1)
		now := time.Now()

		// Mock: Get account (zero balance)
		expectedDBAccount := queries.Account{
			ID:        accountID,
			UserID:    userID,
			Currency:  "USD",
			Balance:   createPgNumeric("0.00"),
			CreatedAt: createPgTimestamp(now),
			UpdatedAt: createPgTimestamp(now),
		}
		mockAccountRepo.On("GetAccount", ctx, accountID).Return(expectedDBAccount, nil)

		// Mock: Count transfers (no transactions)
		mockTransferRepo.On("CountTransfersByAccount", ctx, accountID).Return(int64(0), nil)

		// Mock: Delete account
		mockAccountRepo.On("DeleteAccount", ctx, accountID).Return(nil)

		err := service.DeleteAccount(ctx, accountID, userID)

		assert.NoError(t, err)
		mockAccountRepo.AssertExpectations(t)
		mockTransferRepo.AssertExpectations(t)
	})

	t.Run("cannot delete account with non-zero balance", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		accountID := int32(1)
		userID := int32(1)
		now := time.Now()

		// Mock: Get account (non-zero balance)
		expectedDBAccount := queries.Account{
			ID:        accountID,
			UserID:    userID,
			Currency:  "USD",
			Balance:   createPgNumeric("100.00"),
			CreatedAt: createPgTimestamp(now),
			UpdatedAt: createPgTimestamp(now),
		}
		mockAccountRepo.On("GetAccount", ctx, accountID).Return(expectedDBAccount, nil)

		err := service.DeleteAccount(ctx, accountID, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete account with non-zero balance")
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("cannot delete account with transaction history", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		accountID := int32(1)
		userID := int32(1)
		now := time.Now()

		// Mock: Get account (zero balance)
		expectedDBAccount := queries.Account{
			ID:        accountID,
			UserID:    userID,
			Currency:  "USD",
			Balance:   createPgNumeric("0.00"),
			CreatedAt: createPgTimestamp(now),
			UpdatedAt: createPgTimestamp(now),
		}
		mockAccountRepo.On("GetAccount", ctx, accountID).Return(expectedDBAccount, nil)

		// Mock: Count transfers (has transactions)
		mockTransferRepo.On("CountTransfersByAccount", ctx, accountID).Return(int64(5), nil)

		err := service.DeleteAccount(ctx, accountID, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete account with transaction history")
		mockAccountRepo.AssertExpectations(t)
		mockTransferRepo.AssertExpectations(t)
	})
}

func TestAccountService_UpdateAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("successful account update", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		accountID := int32(1)
		userID := int32(1)
		now := time.Now()

		// Mock: Get account (for ownership verification)
		expectedDBAccount := queries.Account{
			ID:        accountID,
			UserID:    userID,
			Currency:  "USD",
			Balance:   createPgNumeric("100.00"),
			CreatedAt: createPgTimestamp(now),
			UpdatedAt: createPgTimestamp(now),
		}
		mockAccountRepo.On("GetAccount", ctx, accountID).Return(expectedDBAccount, nil)

		// Mock: Update account
		updatedDBAccount := queries.Account{
			ID:        accountID,
			UserID:    userID,
			Currency:  "USD",
			Balance:   createPgNumeric("100.00"),
			CreatedAt: createPgTimestamp(now),
			UpdatedAt: createPgTimestamp(now.Add(time.Minute)),
		}
		mockAccountRepo.On("UpdateAccount", ctx, accountID).Return(updatedDBAccount, nil)

		req := UpdateAccountRequest{}
		account, err := service.UpdateAccount(ctx, accountID, userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, account)
		assert.Equal(t, 1, account.ID)
		assert.Equal(t, 1, account.UserID)
		assert.Equal(t, "USD", account.Currency)
		assert.Equal(t, "100", account.Balance.String())
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("update account - access denied for wrong user", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		accountID := int32(1)
		userID := int32(1)
		wrongUserID := int32(2)

		// Mock: Get account (different user)
		expectedDBAccount := queries.Account{
			ID:     accountID,
			UserID: wrongUserID, // Different user
		}
		mockAccountRepo.On("GetAccount", ctx, accountID).Return(expectedDBAccount, nil)

		req := UpdateAccountRequest{}
		account, err := service.UpdateAccount(ctx, accountID, userID, req)

		assert.Error(t, err)
		assert.Nil(t, account)
		assert.Contains(t, err.Error(), "access denied")
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("update account - account not found", func(t *testing.T) {
		mockAccountRepo := new(MockAccountRepository)
		mockTransferRepo := new(MockTransferRepository)
		service := NewAccountService(mockAccountRepo, mockTransferRepo)

		accountID := int32(999)
		userID := int32(1)

		// Mock: Get account (not found)
		mockAccountRepo.On("GetAccount", ctx, accountID).Return(queries.Account{}, sql.ErrNoRows)

		req := UpdateAccountRequest{}
		account, err := service.UpdateAccount(ctx, accountID, userID, req)

		assert.Error(t, err)
		assert.Nil(t, account)
		assert.Contains(t, err.Error(), "account not found")
		mockAccountRepo.AssertExpectations(t)
	})
}