package models

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

// User represents a bank customer with authentication and email tracking
type User struct {
	ID               int       `json:"id" db:"id"`
	Email            string    `json:"email" db:"email"`
	PasswordHash     string    `json:"-" db:"password_hash"`
	FirstName        string    `json:"first_name" db:"first_name"`
	LastName         string    `json:"last_name" db:"last_name"`
	WelcomeEmailSent bool      `json:"welcome_email_sent" db:"welcome_email_sent"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// Email validation regex pattern
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Validation errors
var (
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")
	ErrEmptyFirstName   = errors.New("first name cannot be empty")
	ErrEmptyLastName    = errors.New("last name cannot be empty")
)

// ValidateEmail validates the email format
func (u *User) ValidateEmail() error {
	if !emailRegex.MatchString(u.Email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateFields validates all user fields except password
func (u *User) ValidateFields() error {
	if err := u.ValidateEmail(); err != nil {
		return err
	}
	
	if u.FirstName == "" {
		return ErrEmptyFirstName
	}
	
	if u.LastName == "" {
		return ErrEmptyLastName
	}
	
	return nil
}

// HashPassword hashes the plain text password using bcrypt
func (u *User) HashPassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	u.PasswordHash = string(hashedBytes)
	return nil
}

// CheckPassword verifies if the provided password matches the stored hash
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
}

// MarkWelcomeEmailSent marks the user as having received the welcome email
func (u *User) MarkWelcomeEmailSent() {
	u.WelcomeEmailSent = true
	u.UpdatedAt = time.Now()
}

// Account represents a bank account with multi-currency support
type Account struct {
	ID        int             `json:"id" db:"id"`
	UserID    int             `json:"user_id" db:"user_id"`
	Currency  string          `json:"currency" db:"currency"`
	Balance   decimal.Decimal `json:"balance" db:"balance"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// Account validation errors
var (
	ErrInvalidCurrency = errors.New("currency must be a valid 3-character code")
	ErrNegativeBalance = errors.New("balance cannot be negative")
	ErrInvalidUserID   = errors.New("user ID must be positive")
)

// Common currency codes for validation
var validCurrencies = map[string]bool{
	"USD": true, "EUR": true, "GBP": true, "JPY": true, "CAD": true,
	"AUD": true, "CHF": true, "CNY": true, "SEK": true, "NZD": true,
	"MXN": true, "SGD": true, "HKD": true, "NOK": true, "TRY": true,
	"RUB": true, "INR": true, "BRL": true, "ZAR": true, "KRW": true,
}

// ValidateCurrency validates the currency code format and value
func (a *Account) ValidateCurrency() error {
	if len(a.Currency) != 3 {
		return ErrInvalidCurrency
	}
	
	currency := strings.ToUpper(a.Currency)
	if !validCurrencies[currency] {
		return ErrInvalidCurrency
	}
	
	// Normalize currency to uppercase
	a.Currency = currency
	return nil
}

// ValidateBalance validates that the balance is not negative
func (a *Account) ValidateBalance() error {
	if a.Balance.IsNegative() {
		return ErrNegativeBalance
	}
	return nil
}

// ValidateFields validates all account fields
func (a *Account) ValidateFields() error {
	if a.UserID <= 0 {
		return ErrInvalidUserID
	}
	
	if err := a.ValidateCurrency(); err != nil {
		return err
	}
	
	if err := a.ValidateBalance(); err != nil {
		return err
	}
	
	return nil
}

// FormatBalance returns the balance formatted with appropriate decimal places
func (a *Account) FormatBalance() string {
	return a.Balance.StringFixed(2)
}

// IsZeroBalance returns true if the account balance is zero
func (a *Account) IsZeroBalance() bool {
	return a.Balance.IsZero()
}

// HasSufficientBalance checks if the account has sufficient balance for a given amount
func (a *Account) HasSufficientBalance(amount decimal.Decimal) bool {
	return a.Balance.GreaterThanOrEqual(amount)
}

// UpdateBalance updates the account balance and timestamp
func (a *Account) UpdateBalance(newBalance decimal.Decimal) error {
	if newBalance.IsNegative() {
		return ErrNegativeBalance
	}
	
	a.Balance = newBalance
	a.UpdatedAt = time.Now()
	return nil
}

// Transfer represents a money transfer between accounts
type Transfer struct {
	ID            int             `json:"id" db:"id"`
	FromAccountID int             `json:"from_account_id" db:"from_account_id"`
	ToAccountID   int             `json:"to_account_id" db:"to_account_id"`
	Amount        decimal.Decimal `json:"amount" db:"amount"`
	Description   string          `json:"description" db:"description"`
	Status        string          `json:"status" db:"status"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
}

// Transfer validation errors
var (
	ErrInvalidTransferAmount    = errors.New("transfer amount must be positive")
	ErrInvalidFromAccount       = errors.New("from account ID must be positive")
	ErrInvalidToAccount         = errors.New("to account ID must be positive")
	ErrSameAccount              = errors.New("cannot transfer to the same account")
	ErrInvalidTransferStatus    = errors.New("invalid transfer status")
	ErrCurrencyMismatch         = errors.New("accounts must have the same currency for transfer")
	ErrInsufficientBalance      = errors.New("insufficient balance for transfer")
)

// Valid transfer statuses
var validTransferStatuses = map[string]bool{
	"pending":   true,
	"completed": true,
	"failed":    true,
	"cancelled": true,
}

// ValidateAmount validates that the transfer amount is positive
func (t *Transfer) ValidateAmount() error {
	if t.Amount.IsNegative() || t.Amount.IsZero() {
		return ErrInvalidTransferAmount
	}
	return nil
}

// ValidateAccounts validates the from and to account IDs
func (t *Transfer) ValidateAccounts() error {
	if t.FromAccountID <= 0 {
		return ErrInvalidFromAccount
	}
	
	if t.ToAccountID <= 0 {
		return ErrInvalidToAccount
	}
	
	if t.FromAccountID == t.ToAccountID {
		return ErrSameAccount
	}
	
	return nil
}

// ValidateStatus validates the transfer status
func (t *Transfer) ValidateStatus() error {
	if t.Status == "" {
		t.Status = "completed" // Default status
		return nil
	}
	
	if !validTransferStatuses[strings.ToLower(t.Status)] {
		return ErrInvalidTransferStatus
	}
	
	// Normalize status to lowercase
	t.Status = strings.ToLower(t.Status)
	return nil
}

// ValidateFields validates all transfer fields
func (t *Transfer) ValidateFields() error {
	if err := t.ValidateAmount(); err != nil {
		return err
	}
	
	if err := t.ValidateAccounts(); err != nil {
		return err
	}
	
	if err := t.ValidateStatus(); err != nil {
		return err
	}
	
	return nil
}

// ValidateCurrencyMatch validates that both accounts have the same currency
func (t *Transfer) ValidateCurrencyMatch(fromAccount, toAccount *Account) error {
	if fromAccount.Currency != toAccount.Currency {
		return ErrCurrencyMismatch
	}
	return nil
}

// ValidateSufficientBalance validates that the from account has sufficient balance
func (t *Transfer) ValidateSufficientBalance(fromAccount *Account) error {
	if !fromAccount.HasSufficientBalance(t.Amount) {
		return ErrInsufficientBalance
	}
	return nil
}

// ValidateTransfer performs comprehensive validation including account checks
func (t *Transfer) ValidateTransfer(fromAccount, toAccount *Account) error {
	// Basic field validation
	if err := t.ValidateFields(); err != nil {
		return err
	}
	
	// Currency matching validation
	if err := t.ValidateCurrencyMatch(fromAccount, toAccount); err != nil {
		return err
	}
	
	// Balance validation
	if err := t.ValidateSufficientBalance(fromAccount); err != nil {
		return err
	}
	
	return nil
}

// FormatAmount returns the transfer amount formatted with appropriate decimal places
func (t *Transfer) FormatAmount() string {
	return t.Amount.StringFixed(2)
}

// IsCompleted returns true if the transfer status is completed
func (t *Transfer) IsCompleted() bool {
	return strings.ToLower(t.Status) == "completed"
}

// IsPending returns true if the transfer status is pending
func (t *Transfer) IsPending() bool {
	return strings.ToLower(t.Status) == "pending"
}

// MarkCompleted marks the transfer as completed
func (t *Transfer) MarkCompleted() {
	t.Status = "completed"
}

// MarkFailed marks the transfer as failed
func (t *Transfer) MarkFailed() {
	t.Status = "failed"
}