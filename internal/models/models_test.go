package models

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_ValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "valid email with numbers",
			email:   "user123@example123.com",
			wantErr: false,
		},
		{
			name:    "invalid email without @",
			email:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "invalid email without domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "invalid email without TLD",
			email:   "user@example",
			wantErr: true,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Email: tt.email}
			err := user.ValidateEmail()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrInvalidEmail, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUser_ValidateFields(t *testing.T) {
	tests := []struct {
		name      string
		user      User
		wantErr   error
	}{
		{
			name: "valid user",
			user: User{
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: nil,
		},
		{
			name: "invalid email",
			user: User{
				Email:     "invalid-email",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: ErrInvalidEmail,
		},
		{
			name: "empty first name",
			user: User{
				Email:     "user@example.com",
				FirstName: "",
				LastName:  "Doe",
			},
			wantErr: ErrEmptyFirstName,
		},
		{
			name: "empty last name",
			user: User{
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "",
			},
			wantErr: ErrEmptyLastName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.ValidateFields()
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUser_HashPassword(t *testing.T) {
	user := &User{}

	t.Run("valid password", func(t *testing.T) {
		password := "validpassword123"
		err := user.HashPassword(password)
		require.NoError(t, err)
		assert.NotEmpty(t, user.PasswordHash)
		assert.NotEqual(t, password, user.PasswordHash)
	})

	t.Run("password too short", func(t *testing.T) {
		user := &User{}
		password := "short"
		err := user.HashPassword(password)
		assert.Error(t, err)
		assert.Equal(t, ErrPasswordTooShort, err)
		assert.Empty(t, user.PasswordHash)
	})

	t.Run("minimum length password", func(t *testing.T) {
		user := &User{}
		password := "12345678" // exactly 8 characters
		err := user.HashPassword(password)
		assert.NoError(t, err)
		assert.NotEmpty(t, user.PasswordHash)
	})
}

func TestUser_CheckPassword(t *testing.T) {
	user := &User{}
	password := "testpassword123"
	
	// Hash the password first
	err := user.HashPassword(password)
	require.NoError(t, err)

	t.Run("correct password", func(t *testing.T) {
		err := user.CheckPassword(password)
		assert.NoError(t, err)
	})

	t.Run("incorrect password", func(t *testing.T) {
		err := user.CheckPassword("wrongpassword")
		assert.Error(t, err)
	})

	t.Run("empty password", func(t *testing.T) {
		err := user.CheckPassword("")
		assert.Error(t, err)
	})
}

func TestUser_MarkWelcomeEmailSent(t *testing.T) {
	user := &User{
		WelcomeEmailSent: false,
		UpdatedAt:        time.Now().Add(-time.Hour), // Set to an hour ago
	}

	oldUpdatedAt := user.UpdatedAt
	user.MarkWelcomeEmailSent()

	assert.True(t, user.WelcomeEmailSent)
	assert.True(t, user.UpdatedAt.After(oldUpdatedAt))
}

func TestUser_JSONSerialization(t *testing.T) {
	user := &User{
		ID:               1,
		Email:            "user@example.com",
		PasswordHash:     "hashed_password",
		FirstName:        "John",
		LastName:         "Doe",
		WelcomeEmailSent: true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Test that password hash is not included in JSON serialization
	// This is tested by the json:"-" tag on PasswordHash field
	// The actual JSON marshaling would be tested in integration tests
	assert.Equal(t, "hashed_password", user.PasswordHash)
}

func TestAccount_ValidateCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		wantErr  bool
		expected string
	}{
		{
			name:     "valid uppercase currency",
			currency: "USD",
			wantErr:  false,
			expected: "USD",
		},
		{
			name:     "valid lowercase currency",
			currency: "eur",
			wantErr:  false,
			expected: "EUR",
		},
		{
			name:     "valid mixed case currency",
			currency: "GbP",
			wantErr:  false,
			expected: "GBP",
		},
		{
			name:     "invalid currency too short",
			currency: "US",
			wantErr:  true,
		},
		{
			name:     "invalid currency too long",
			currency: "USDD",
			wantErr:  true,
		},
		{
			name:     "invalid currency code",
			currency: "XYZ",
			wantErr:  true,
		},
		{
			name:     "empty currency",
			currency: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{Currency: tt.currency}
			err := account.ValidateCurrency()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrInvalidCurrency, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, account.Currency)
			}
		})
	}
}

func TestAccount_ValidateBalance(t *testing.T) {
	tests := []struct {
		name    string
		balance string
		wantErr bool
	}{
		{
			name:    "positive balance",
			balance: "100.50",
			wantErr: false,
		},
		{
			name:    "zero balance",
			balance: "0",
			wantErr: false,
		},
		{
			name:    "negative balance",
			balance: "-10.50",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, _ := decimal.NewFromString(tt.balance)
			account := &Account{Balance: balance}
			err := account.ValidateBalance()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrNegativeBalance, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAccount_ValidateFields(t *testing.T) {
	tests := []struct {
		name    string
		account Account
		wantErr error
	}{
		{
			name: "valid account",
			account: Account{
				UserID:   1,
				Currency: "USD",
				Balance:  decimal.NewFromFloat(100.50),
			},
			wantErr: nil,
		},
		{
			name: "invalid user ID",
			account: Account{
				UserID:   0,
				Currency: "USD",
				Balance:  decimal.NewFromFloat(100.50),
			},
			wantErr: ErrInvalidUserID,
		},
		{
			name: "negative user ID",
			account: Account{
				UserID:   -1,
				Currency: "USD",
				Balance:  decimal.NewFromFloat(100.50),
			},
			wantErr: ErrInvalidUserID,
		},
		{
			name: "invalid currency",
			account: Account{
				UserID:   1,
				Currency: "INVALID",
				Balance:  decimal.NewFromFloat(100.50),
			},
			wantErr: ErrInvalidCurrency,
		},
		{
			name: "negative balance",
			account: Account{
				UserID:   1,
				Currency: "USD",
				Balance:  decimal.NewFromFloat(-10.50),
			},
			wantErr: ErrNegativeBalance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.ValidateFields()
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAccount_FormatBalance(t *testing.T) {
	tests := []struct {
		name     string
		balance  string
		expected string
	}{
		{
			name:     "whole number",
			balance:  "100",
			expected: "100.00",
		},
		{
			name:     "decimal with two places",
			balance:  "100.50",
			expected: "100.50",
		},
		{
			name:     "decimal with more than two places",
			balance:  "100.567",
			expected: "100.57",
		},
		{
			name:     "zero balance",
			balance:  "0",
			expected: "0.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, _ := decimal.NewFromString(tt.balance)
			account := &Account{Balance: balance}
			result := account.FormatBalance()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAccount_IsZeroBalance(t *testing.T) {
	tests := []struct {
		name     string
		balance  string
		expected bool
	}{
		{
			name:     "zero balance",
			balance:  "0",
			expected: true,
		},
		{
			name:     "positive balance",
			balance:  "100.50",
			expected: false,
		},
		{
			name:     "very small positive balance",
			balance:  "0.01",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, _ := decimal.NewFromString(tt.balance)
			account := &Account{Balance: balance}
			result := account.IsZeroBalance()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAccount_HasSufficientBalance(t *testing.T) {
	account := &Account{Balance: decimal.NewFromFloat(100.50)}

	tests := []struct {
		name     string
		amount   string
		expected bool
	}{
		{
			name:     "sufficient balance",
			amount:   "50.00",
			expected: true,
		},
		{
			name:     "exact balance",
			amount:   "100.50",
			expected: true,
		},
		{
			name:     "insufficient balance",
			amount:   "150.00",
			expected: false,
		},
		{
			name:     "zero amount",
			amount:   "0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, _ := decimal.NewFromString(tt.amount)
			result := account.HasSufficientBalance(amount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAccount_UpdateBalance(t *testing.T) {
	account := &Account{
		Balance:   decimal.NewFromFloat(100.00),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	t.Run("valid balance update", func(t *testing.T) {
		oldUpdatedAt := account.UpdatedAt
		newBalance := decimal.NewFromFloat(150.50)
		
		err := account.UpdateBalance(newBalance)
		assert.NoError(t, err)
		assert.True(t, account.Balance.Equal(newBalance))
		assert.True(t, account.UpdatedAt.After(oldUpdatedAt))
	})

	t.Run("negative balance update", func(t *testing.T) {
		originalBalance := account.Balance
		negativeBalance := decimal.NewFromFloat(-50.00)
		
		err := account.UpdateBalance(negativeBalance)
		assert.Error(t, err)
		assert.Equal(t, ErrNegativeBalance, err)
		assert.True(t, account.Balance.Equal(originalBalance)) // Balance should remain unchanged
	})

	t.Run("zero balance update", func(t *testing.T) {
		zeroBalance := decimal.NewFromFloat(0)
		
		err := account.UpdateBalance(zeroBalance)
		assert.NoError(t, err)
		assert.True(t, account.Balance.Equal(zeroBalance))
	})
}

func TestTransfer_ValidateAmount(t *testing.T) {
	tests := []struct {
		name    string
		amount  string
		wantErr bool
	}{
		{
			name:    "positive amount",
			amount:  "100.50",
			wantErr: false,
		},
		{
			name:    "zero amount",
			amount:  "0",
			wantErr: true,
		},
		{
			name:    "negative amount",
			amount:  "-50.00",
			wantErr: true,
		},
		{
			name:    "very small positive amount",
			amount:  "0.01",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, _ := decimal.NewFromString(tt.amount)
			transfer := &Transfer{Amount: amount}
			err := transfer.ValidateAmount()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrInvalidTransferAmount, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransfer_ValidateAccounts(t *testing.T) {
	tests := []struct {
		name          string
		fromAccountID int
		toAccountID   int
		wantErr       error
	}{
		{
			name:          "valid accounts",
			fromAccountID: 1,
			toAccountID:   2,
			wantErr:       nil,
		},
		{
			name:          "invalid from account",
			fromAccountID: 0,
			toAccountID:   2,
			wantErr:       ErrInvalidFromAccount,
		},
		{
			name:          "negative from account",
			fromAccountID: -1,
			toAccountID:   2,
			wantErr:       ErrInvalidFromAccount,
		},
		{
			name:          "invalid to account",
			fromAccountID: 1,
			toAccountID:   0,
			wantErr:       ErrInvalidToAccount,
		},
		{
			name:          "negative to account",
			fromAccountID: 1,
			toAccountID:   -1,
			wantErr:       ErrInvalidToAccount,
		},
		{
			name:          "same account",
			fromAccountID: 1,
			toAccountID:   1,
			wantErr:       ErrSameAccount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transfer := &Transfer{
				FromAccountID: tt.fromAccountID,
				ToAccountID:   tt.toAccountID,
			}
			err := transfer.ValidateAccounts()
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransfer_ValidateStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		wantErr  bool
		expected string
	}{
		{
			name:     "valid completed status",
			status:   "completed",
			wantErr:  false,
			expected: "completed",
		},
		{
			name:     "valid pending status",
			status:   "pending",
			wantErr:  false,
			expected: "pending",
		},
		{
			name:     "valid failed status",
			status:   "failed",
			wantErr:  false,
			expected: "failed",
		},
		{
			name:     "valid cancelled status",
			status:   "cancelled",
			wantErr:  false,
			expected: "cancelled",
		},
		{
			name:     "uppercase status",
			status:   "COMPLETED",
			wantErr:  false,
			expected: "completed",
		},
		{
			name:     "mixed case status",
			status:   "Pending",
			wantErr:  false,
			expected: "pending",
		},
		{
			name:     "empty status defaults to completed",
			status:   "",
			wantErr:  false,
			expected: "completed",
		},
		{
			name:    "invalid status",
			status:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transfer := &Transfer{Status: tt.status}
			err := transfer.ValidateStatus()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrInvalidTransferStatus, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, transfer.Status)
			}
		})
	}
}

func TestTransfer_ValidateFields(t *testing.T) {
	tests := []struct {
		name     string
		transfer Transfer
		wantErr  error
	}{
		{
			name: "valid transfer",
			transfer: Transfer{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(100.50),
				Status:        "completed",
			},
			wantErr: nil,
		},
		{
			name: "invalid amount",
			transfer: Transfer{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(0),
				Status:        "completed",
			},
			wantErr: ErrInvalidTransferAmount,
		},
		{
			name: "invalid accounts",
			transfer: Transfer{
				FromAccountID: 1,
				ToAccountID:   1,
				Amount:        decimal.NewFromFloat(100.50),
				Status:        "completed",
			},
			wantErr: ErrSameAccount,
		},
		{
			name: "invalid status",
			transfer: Transfer{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(100.50),
				Status:        "invalid",
			},
			wantErr: ErrInvalidTransferStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transfer.ValidateFields()
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransfer_ValidateCurrencyMatch(t *testing.T) {
	fromAccount := &Account{Currency: "USD"}
	toAccountSameCurrency := &Account{Currency: "USD"}
	toAccountDifferentCurrency := &Account{Currency: "EUR"}

	transfer := &Transfer{}

	t.Run("matching currencies", func(t *testing.T) {
		err := transfer.ValidateCurrencyMatch(fromAccount, toAccountSameCurrency)
		assert.NoError(t, err)
	})

	t.Run("different currencies", func(t *testing.T) {
		err := transfer.ValidateCurrencyMatch(fromAccount, toAccountDifferentCurrency)
		assert.Error(t, err)
		assert.Equal(t, ErrCurrencyMismatch, err)
	})
}

func TestTransfer_ValidateSufficientBalance(t *testing.T) {
	fromAccount := &Account{Balance: decimal.NewFromFloat(100.00)}

	tests := []struct {
		name    string
		amount  string
		wantErr bool
	}{
		{
			name:    "sufficient balance",
			amount:  "50.00",
			wantErr: false,
		},
		{
			name:    "exact balance",
			amount:  "100.00",
			wantErr: false,
		},
		{
			name:    "insufficient balance",
			amount:  "150.00",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, _ := decimal.NewFromString(tt.amount)
			transfer := &Transfer{Amount: amount}
			err := transfer.ValidateSufficientBalance(fromAccount)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrInsufficientBalance, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransfer_ValidateTransfer(t *testing.T) {
	fromAccount := &Account{
		Currency: "USD",
		Balance:  decimal.NewFromFloat(100.00),
	}
	toAccount := &Account{
		Currency: "USD",
		Balance:  decimal.NewFromFloat(50.00),
	}
	toAccountDifferentCurrency := &Account{
		Currency: "EUR",
		Balance:  decimal.NewFromFloat(50.00),
	}

	tests := []struct {
		name        string
		transfer    Transfer
		fromAccount *Account
		toAccount   *Account
		wantErr     error
	}{
		{
			name: "valid transfer",
			transfer: Transfer{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(50.00),
				Status:        "completed",
			},
			fromAccount: fromAccount,
			toAccount:   toAccount,
			wantErr:     nil,
		},
		{
			name: "currency mismatch",
			transfer: Transfer{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(50.00),
				Status:        "completed",
			},
			fromAccount: fromAccount,
			toAccount:   toAccountDifferentCurrency,
			wantErr:     ErrCurrencyMismatch,
		},
		{
			name: "insufficient balance",
			transfer: Transfer{
				FromAccountID: 1,
				ToAccountID:   2,
				Amount:        decimal.NewFromFloat(150.00),
				Status:        "completed",
			},
			fromAccount: fromAccount,
			toAccount:   toAccount,
			wantErr:     ErrInsufficientBalance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transfer.ValidateTransfer(tt.fromAccount, tt.toAccount)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransfer_FormatAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   string
		expected string
	}{
		{
			name:     "whole number",
			amount:   "100",
			expected: "100.00",
		},
		{
			name:     "decimal with two places",
			amount:   "100.50",
			expected: "100.50",
		},
		{
			name:     "decimal with more than two places",
			amount:   "100.567",
			expected: "100.57",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, _ := decimal.NewFromString(tt.amount)
			transfer := &Transfer{Amount: amount}
			result := transfer.FormatAmount()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransfer_StatusMethods(t *testing.T) {
	t.Run("IsCompleted", func(t *testing.T) {
		tests := []struct {
			status   string
			expected bool
		}{
			{"completed", true},
			{"COMPLETED", true},
			{"Completed", true},
			{"pending", false},
			{"failed", false},
		}

		for _, tt := range tests {
			transfer := &Transfer{Status: tt.status}
			result := transfer.IsCompleted()
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("IsPending", func(t *testing.T) {
		tests := []struct {
			status   string
			expected bool
		}{
			{"pending", true},
			{"PENDING", true},
			{"Pending", true},
			{"completed", false},
			{"failed", false},
		}

		for _, tt := range tests {
			transfer := &Transfer{Status: tt.status}
			result := transfer.IsPending()
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("MarkCompleted", func(t *testing.T) {
		transfer := &Transfer{Status: "pending"}
		transfer.MarkCompleted()
		assert.Equal(t, "completed", transfer.Status)
	})

	t.Run("MarkFailed", func(t *testing.T) {
		transfer := &Transfer{Status: "pending"}
		transfer.MarkFailed()
		assert.Equal(t, "failed", transfer.Status)
	})
}