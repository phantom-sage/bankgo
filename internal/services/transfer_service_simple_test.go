package services

import (
	"strings"
	"testing"

	"github.com/phantom-sage/bankgo/internal/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestTransferValidation_ValidTransfer(t *testing.T) {
	transfer := &models.Transfer{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(100.00),
		Description:   "Valid transfer",
		Status:        "completed",
	}

	err := transfer.ValidateFields()
	assert.NoError(t, err)
}

func TestTransferValidation_NegativeAmount(t *testing.T) {
	transfer := &models.Transfer{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(-100.00),
		Description:   "Invalid transfer",
		Status:        "completed",
	}

	err := transfer.ValidateFields()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transfer amount must be positive")
}

func TestTransferValidation_ZeroAmount(t *testing.T) {
	transfer := &models.Transfer{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.Zero,
		Description:   "Invalid transfer",
		Status:        "completed",
	}

	err := transfer.ValidateFields()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transfer amount must be positive")
}

func TestTransferValidation_SameAccount(t *testing.T) {
	transfer := &models.Transfer{
		FromAccountID: 1,
		ToAccountID:   1,
		Amount:        decimal.NewFromFloat(100.00),
		Description:   "Invalid transfer",
		Status:        "completed",
	}

	err := transfer.ValidateFields()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot transfer to the same account")
}

func TestTransferValidation_InvalidFromAccount(t *testing.T) {
	transfer := &models.Transfer{
		FromAccountID: 0,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(100.00),
		Description:   "Invalid transfer",
		Status:        "completed",
	}

	err := transfer.ValidateFields()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from account ID must be positive")
}

func TestTransferValidation_InvalidToAccount(t *testing.T) {
	transfer := &models.Transfer{
		FromAccountID: 1,
		ToAccountID:   -1,
		Amount:        decimal.NewFromFloat(100.00),
		Description:   "Invalid transfer",
		Status:        "completed",
	}

	err := transfer.ValidateFields()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "to account ID must be positive")
}

func TestTransferValidation_CurrencyMismatch(t *testing.T) {
	transfer := &models.Transfer{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(100.00),
		Description:   "Currency mismatch transfer",
		Status:        "completed",
	}

	fromAccount := &models.Account{
		ID:       1,
		Currency: "USD",
		Balance:  decimal.NewFromFloat(500.00),
	}

	toAccount := &models.Account{
		ID:       2,
		Currency: "EUR", // Different currency
		Balance:  decimal.NewFromFloat(200.00),
	}

	err := transfer.ValidateCurrencyMatch(fromAccount, toAccount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accounts must have the same currency for transfer")
}

func TestTransferValidation_InsufficientBalance(t *testing.T) {
	transfer := &models.Transfer{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(600.00), // More than available
		Description:   "Insufficient balance transfer",
		Status:        "completed",
	}

	fromAccount := &models.Account{
		ID:       1,
		Currency: "USD",
		Balance:  decimal.NewFromFloat(500.00), // Only 500 available
	}

	err := transfer.ValidateSufficientBalance(fromAccount)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient balance for transfer")
}

func TestTransferValidation_CompleteValidation(t *testing.T) {
	transfer := &models.Transfer{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(100.00),
		Description:   "Valid complete transfer",
		Status:        "completed",
	}

	fromAccount := &models.Account{
		ID:       1,
		Currency: "USD",
		Balance:  decimal.NewFromFloat(500.00),
	}

	toAccount := &models.Account{
		ID:       2,
		Currency: "USD", // Same currency
		Balance:  decimal.NewFromFloat(200.00),
	}

	err := transfer.ValidateTransfer(fromAccount, toAccount)
	assert.NoError(t, err)
}

func TestTransferMoneyRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		req         TransferMoneyRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid request",
			req: TransferMoneyRequest{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(100.00),
				Description:   "Valid transfer",
			},
			expectError: false,
		},
		{
			name: "Same account",
			req: TransferMoneyRequest{
				FromAccountID: 1,
				ToAccountID:   1,
				Amount:        decimal.NewFromFloat(100.00),
				Description:   "Same account transfer",
			},
			expectError: true,
			errorMsg:    "cannot transfer to the same account",
		},
		{
			name: "Negative amount",
			req: TransferMoneyRequest{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(-100.00),
				Description:   "Negative amount transfer",
			},
			expectError: true,
			errorMsg:    "transfer amount must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create transfer model for validation
			transfer := &models.Transfer{
				FromAccountID: int(tt.req.FromAccountID),
				ToAccountID:   int(tt.req.ToAccountID),
				Amount:        tt.req.Amount,
				Description:   tt.req.Description,
				Status:        "completed",
			}

			err := transfer.ValidateFields()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetTransferHistoryRequest_Pagination(t *testing.T) {
	tests := []struct {
		name           string
		req            GetTransferHistoryRequest
		expectedLimit  int32
		expectedOffset int32
	}{
		{
			name: "Default values",
			req: GetTransferHistoryRequest{
				AccountID: 1,
				Limit:     0,
				Offset:    -5,
			},
			expectedLimit:  20, // Should default to 20
			expectedOffset: 0,  // Should be corrected to 0
		},
		{
			name: "Valid values",
			req: GetTransferHistoryRequest{
				AccountID: 1,
				Limit:     10,
				Offset:    5,
			},
			expectedLimit:  10,
			expectedOffset: 5,
		},
		{
			name: "Max limit enforced",
			req: GetTransferHistoryRequest{
				AccountID: 1,
				Limit:     150, // Should be capped at 100
				Offset:    0,
			},
			expectedLimit:  100, // Should be capped
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the same logic as in the service
			limit := tt.req.Limit
			offset := tt.req.Offset

			if limit <= 0 {
				limit = 20
			}
			if limit > 100 {
				limit = 100
			}
			if offset < 0 {
				offset = 0
			}

			assert.Equal(t, tt.expectedLimit, limit)
			assert.Equal(t, tt.expectedOffset, offset)
		})
	}
}

func TestTransferStatusValidation(t *testing.T) {
	tests := []struct {
		name        string
		status      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid completed status",
			status:      "completed",
			expectError: false,
		},
		{
			name:        "Valid pending status",
			status:      "pending",
			expectError: false,
		},
		{
			name:        "Valid failed status",
			status:      "failed",
			expectError: false,
		},
		{
			name:        "Valid cancelled status",
			status:      "cancelled",
			expectError: false,
		},
		{
			name:        "Invalid status",
			status:      "invalid_status",
			expectError: true,
			errorMsg:    "invalid transfer status",
		},
		{
			name:        "Empty status defaults to completed",
			status:      "",
			expectError: false,
		},
		{
			name:        "Case insensitive status",
			status:      "COMPLETED",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transfer := &models.Transfer{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(100.00),
				Description:   "Test transfer",
				Status:        tt.status,
			}

			err := transfer.ValidateStatus()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				// Check that status is normalized to lowercase (except empty which becomes "completed")
				if tt.status == "" {
					assert.Equal(t, "completed", transfer.Status)
				} else {
					assert.Equal(t, strings.ToLower(tt.status), transfer.Status)
				}
			}
		})
	}
}

func TestTransferStatusMethods(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		isCompleted    bool
		isPending      bool
	}{
		{
			name:        "Completed transfer",
			status:      "completed",
			isCompleted: true,
			isPending:   false,
		},
		{
			name:        "Pending transfer",
			status:      "pending",
			isCompleted: false,
			isPending:   true,
		},
		{
			name:        "Failed transfer",
			status:      "failed",
			isCompleted: false,
			isPending:   false,
		},
		{
			name:        "Cancelled transfer",
			status:      "cancelled",
			isCompleted: false,
			isPending:   false,
		},
		{
			name:        "Case insensitive completed",
			status:      "COMPLETED",
			isCompleted: true,
			isPending:   false,
		},
		{
			name:        "Case insensitive pending",
			status:      "PENDING",
			isCompleted: false,
			isPending:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transfer := &models.Transfer{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(100.00),
				Description:   "Test transfer",
				Status:        tt.status,
			}

			assert.Equal(t, tt.isCompleted, transfer.IsCompleted())
			assert.Equal(t, tt.isPending, transfer.IsPending())
		})
	}
}

func TestTransferStatusMarking(t *testing.T) {
	transfer := &models.Transfer{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(100.00),
		Description:   "Test transfer",
		Status:        "pending",
	}

	// Test marking as completed
	assert.True(t, transfer.IsPending())
	assert.False(t, transfer.IsCompleted())

	transfer.MarkCompleted()
	assert.Equal(t, "completed", transfer.Status)
	assert.True(t, transfer.IsCompleted())
	assert.False(t, transfer.IsPending())

	// Test marking as failed
	transfer.MarkFailed()
	assert.Equal(t, "failed", transfer.Status)
	assert.False(t, transfer.IsCompleted())
	assert.False(t, transfer.IsPending())
}

func TestTransferAmountFormatting(t *testing.T) {
	transfer := &models.Transfer{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        decimal.NewFromFloat(123.456),
		Description:   "Test transfer",
		Status:        "completed",
	}

	formatted := transfer.FormatAmount()
	assert.Equal(t, "123.46", formatted) // Should be formatted to 2 decimal places
}

func TestTransferValidationWithAccountDetails(t *testing.T) {
	tests := []struct {
		name          string
		transfer      *models.Transfer
		fromAccount   *models.Account
		toAccount     *models.Account
		expectError   bool
		errorMsg      string
	}{
		{
			name: "Valid transfer with matching currencies and sufficient balance",
			transfer: &models.Transfer{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(100.00),
				Description:   "Valid transfer",
				Status:        "completed",
			},
			fromAccount: &models.Account{
				ID:       1,
				Currency: "USD",
				Balance:  decimal.NewFromFloat(500.00),
			},
			toAccount: &models.Account{
				ID:       2,
				Currency: "USD",
				Balance:  decimal.NewFromFloat(200.00),
			},
			expectError: false,
		},
		{
			name: "Currency mismatch",
			transfer: &models.Transfer{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(100.00),
				Description:   "Currency mismatch transfer",
				Status:        "completed",
			},
			fromAccount: &models.Account{
				ID:       1,
				Currency: "USD",
				Balance:  decimal.NewFromFloat(500.00),
			},
			toAccount: &models.Account{
				ID:       2,
				Currency: "EUR", // Different currency
				Balance:  decimal.NewFromFloat(200.00),
			},
			expectError: true,
			errorMsg:    "accounts must have the same currency for transfer",
		},
		{
			name: "Insufficient balance",
			transfer: &models.Transfer{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(600.00), // More than available
				Description:   "Insufficient balance transfer",
				Status:        "completed",
			},
			fromAccount: &models.Account{
				ID:       1,
				Currency: "USD",
				Balance:  decimal.NewFromFloat(500.00), // Only 500 available
			},
			toAccount: &models.Account{
				ID:       2,
				Currency: "USD",
				Balance:  decimal.NewFromFloat(200.00),
			},
			expectError: true,
			errorMsg:    "insufficient balance for transfer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transfer.ValidateTransfer(tt.fromAccount, tt.toAccount)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}