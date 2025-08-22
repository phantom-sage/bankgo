package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/phantom-sage/bankgo/internal/logging"
	"github.com/phantom-sage/bankgo/internal/models"
	"github.com/phantom-sage/bankgo/internal/repository"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
)

// AccountService defines the interface for account business logic
type AccountService interface {
	CreateAccount(ctx context.Context, userID int32, currency string) (*models.Account, error)
	GetAccount(ctx context.Context, accountID int32, userID int32) (*models.Account, error)
	GetUserAccounts(ctx context.Context, userID int32) ([]*models.Account, error)
	UpdateAccount(ctx context.Context, accountID int32, userID int32, req UpdateAccountRequest) (*models.Account, error)
	DeleteAccount(ctx context.Context, accountID int32, userID int32) error
}

// AccountServiceImpl implements AccountService
type AccountServiceImpl struct {
	accountRepo     repository.AccountRepository
	transferRepo    repository.TransferRepository
	logger          zerolog.Logger
	auditLogger     *logging.AuditLogger
	performanceLogger *logging.PerformanceLogger
}

// NewAccountService creates a new account service
func NewAccountService(accountRepo repository.AccountRepository, transferRepo repository.TransferRepository, logger zerolog.Logger) AccountService {
	auditLogger := logging.NewAuditLogger(logger)
	performanceLogger := logging.NewPerformanceLogger(logger)
	return &AccountServiceImpl{
		accountRepo:       accountRepo,
		transferRepo:      transferRepo,
		logger:            logger.With().Str("component", "account_service").Logger(),
		auditLogger:       auditLogger,
		performanceLogger: performanceLogger,
	}
}

// CreateAccount creates a new account with currency validation
func (s *AccountServiceImpl) CreateAccount(ctx context.Context, userID int32, currency string) (*models.Account, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).
		WithOperation("create_account").
		WithUserID(int64(userID))
	
	contextLogger.Info().
		Int32("user_id", userID).
		Str("currency", currency).
		Msg("Starting account creation")

	// Create a temporary account model for validation
	tempAccount := &models.Account{
		UserID:   int(userID),
		Currency: currency,
		Balance:  decimal.Zero,
	}

	// Validate currency format and value
	if err := tempAccount.ValidateCurrency(); err != nil {
		contextLogger.Error().
			Err(err).
			Int32("user_id", userID).
			Str("currency", currency).
			Msg("Currency validation failed")
		s.auditLogger.LogAccountCreation(int64(userID), 0, currency, "failed_validation")
		return nil, fmt.Errorf("invalid currency: %w", err)
	}

	// Check if user already has an account with this currency
	dbStart := time.Now()
	_, err := s.accountRepo.GetAccountByUserAndCurrency(ctx, queries.GetAccountByUserAndCurrencyParams{
		UserID:   userID,
		Currency: tempAccount.Currency, // Use normalized currency
	})
	s.performanceLogger.LogDatabaseQuery("SELECT account by user and currency", time.Since(dbStart), 0)
	
	if err == nil {
		contextLogger.Warn().
			Int32("user_id", userID).
			Str("currency", currency).
			Msg("User already has account with this currency")
		s.auditLogger.LogAccountCreation(int64(userID), 0, currency, "failed_duplicate_currency")
		return nil, fmt.Errorf("user already has an account with currency %s", tempAccount.Currency)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		contextLogger.Error().
			Err(err).
			Int32("user_id", userID).
			Str("currency", currency).
			Msg("Failed to check existing account")
		s.auditLogger.LogAccountCreation(int64(userID), 0, currency, "failed_database_error")
		return nil, fmt.Errorf("failed to check existing account: %w", err)
	}

	// Create the account
	dbStart = time.Now()
	dbAccount, err := s.accountRepo.CreateAccount(ctx, queries.CreateAccountParams{
		UserID:   userID,
		Currency: tempAccount.Currency,
		Column3:  nil, // Use default balance of 0.00
	})
	s.performanceLogger.LogDatabaseQuery("INSERT account", time.Since(dbStart), 1)
	
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("user_id", userID).
			Str("currency", currency).
			Msg("Failed to create account in database")
		s.auditLogger.LogAccountCreation(int64(userID), 0, currency, "failed_database_error")
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	// Convert database account to model
	account, err := s.convertDBAccountToModel(dbAccount)
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("account_id", dbAccount.ID).
			Msg("Failed to convert database account to model")
		return nil, fmt.Errorf("failed to convert account: %w", err)
	}

	// Log successful creation
	duration := time.Since(start)
	contextLogger.Info().
		Int32("user_id", userID).
		Int32("account_id", dbAccount.ID).
		Str("currency", currency).
		Str("balance", account.Balance.StringFixed(2)).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("Account created successfully")
	
	// Audit log for successful account creation
	s.auditLogger.LogAccountCreation(int64(userID), int64(dbAccount.ID), currency, "success")

	return account, nil
}

// GetAccount retrieves an account by ID, ensuring user ownership
func (s *AccountServiceImpl) GetAccount(ctx context.Context, accountID int32, userID int32) (*models.Account, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).
		WithOperation("get_account").
		WithUserID(int64(userID))
	
	contextLogger.Debug().
		Int32("account_id", accountID).
		Int32("user_id", userID).
		Msg("Retrieving account by ID")

	dbStart := time.Now()
	dbAccount, err := s.accountRepo.GetAccount(ctx, accountID)
	s.performanceLogger.LogDatabaseQuery("SELECT account by ID", time.Since(dbStart), 0)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			contextLogger.Warn().
				Int32("account_id", accountID).
				Int32("user_id", userID).
				Msg("Account not found")
			return nil, fmt.Errorf("account not found")
		}
		contextLogger.Error().
			Err(err).
			Int32("account_id", accountID).
			Int32("user_id", userID).
			Msg("Failed to get account from database")
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Ensure user can only access their own accounts
	if dbAccount.UserID != userID {
		contextLogger.Warn().
			Int32("account_id", accountID).
			Int32("requested_user_id", userID).
			Int32("actual_user_id", dbAccount.UserID).
			Msg("Access denied: account does not belong to user")
		s.auditLogger.LogSecurityEvent("unauthorized_account_access", "account_service", 
			fmt.Sprintf("User %d attempted to access account %d belonging to user %d", userID, accountID, dbAccount.UserID))
		return nil, fmt.Errorf("access denied: account does not belong to user")
	}

	// Convert database account to model
	account, err := s.convertDBAccountToModel(dbAccount)
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("account_id", accountID).
			Msg("Failed to convert database account to model")
		return nil, fmt.Errorf("failed to convert account: %w", err)
	}

	duration := time.Since(start)
	contextLogger.Debug().
		Int32("account_id", accountID).
		Int32("user_id", userID).
		Str("currency", account.Currency).
		Str("balance", account.Balance.StringFixed(2)).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("Account retrieved successfully")

	return account, nil
}

// GetUserAccounts retrieves all accounts for a user
func (s *AccountServiceImpl) GetUserAccounts(ctx context.Context, userID int32) ([]*models.Account, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).
		WithOperation("get_user_accounts").
		WithUserID(int64(userID))
	
	contextLogger.Debug().
		Int32("user_id", userID).
		Msg("Retrieving all accounts for user")

	dbStart := time.Now()
	dbAccounts, err := s.accountRepo.GetUserAccounts(ctx, userID)
	s.performanceLogger.LogDatabaseQuery("SELECT accounts by user ID", time.Since(dbStart), int64(len(dbAccounts)))
	
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("user_id", userID).
			Msg("Failed to get user accounts from database")
		return nil, fmt.Errorf("failed to get user accounts: %w", err)
	}

	accounts := make([]*models.Account, len(dbAccounts))
	for i, dbAccount := range dbAccounts {
		account, err := s.convertDBAccountToModel(dbAccount)
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int32("account_id", dbAccount.ID).
				Int("index", i).
				Msg("Failed to convert account to model")
			return nil, fmt.Errorf("failed to convert account %d: %w", dbAccount.ID, err)
		}
		accounts[i] = account
	}

	duration := time.Since(start)
	contextLogger.Info().
		Int32("user_id", userID).
		Int("account_count", len(accounts)).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("User accounts retrieved successfully")

	return accounts, nil
}

// UpdateAccountRequest represents the allowed fields for account updates
type UpdateAccountRequest struct {
	// Note: Balance updates are not allowed through this method for security
	// Currency updates are not allowed to maintain data integrity
	// Only metadata updates are permitted
}

// UpdateAccount updates account metadata (not balance or currency), ensuring user ownership
// This method implements field restrictions as per requirement 4
func (s *AccountServiceImpl) UpdateAccount(ctx context.Context, accountID int32, userID int32, req UpdateAccountRequest) (*models.Account, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).
		WithOperation("update_account").
		WithUserID(int64(userID))
	
	contextLogger.Info().
		Int32("account_id", accountID).
		Int32("user_id", userID).
		Msg("Starting account update")

	// First verify the account exists and belongs to the user
	existingAccount, err := s.GetAccount(ctx, accountID, userID)
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("account_id", accountID).
			Int32("user_id", userID).
			Msg("Failed to verify account ownership for update")
		return nil, err // Error already formatted in GetAccount
	}

	// Update the account (this only updates the updated_at timestamp)
	// Note: Direct balance modification is not allowed through this method
	// Balance changes must go through transfer operations for audit trail
	dbStart := time.Now()
	dbAccount, err := s.accountRepo.UpdateAccount(ctx, accountID)
	s.performanceLogger.LogDatabaseQuery("UPDATE account timestamp", time.Since(dbStart), 1)
	
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("account_id", accountID).
			Int32("user_id", userID).
			Msg("Failed to update account in database")
		s.auditLogger.LogAccountOperation(int64(userID), int64(accountID), "update", "failed_database_error")
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	// Convert database account to model
	account, err := s.convertDBAccountToModel(dbAccount)
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("account_id", accountID).
			Msg("Failed to convert updated account to model")
		return nil, fmt.Errorf("failed to convert account: %w", err)
	}

	duration := time.Since(start)
	contextLogger.Info().
		Int32("account_id", accountID).
		Int32("user_id", userID).
		Str("currency", account.Currency).
		Str("balance", account.Balance.StringFixed(2)).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("Account updated successfully")
	
	// Audit log for account update
	s.auditLogger.LogAccountOperationWithDetails(int64(userID), int64(accountID), "update", "success", 
		existingAccount.Currency, existingAccount.Balance)

	return account, nil
}

// DeleteAccount deletes an account with zero balance validation and transaction history checking
func (s *AccountServiceImpl) DeleteAccount(ctx context.Context, accountID int32, userID int32) error {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).
		WithOperation("delete_account").
		WithUserID(int64(userID))
	
	contextLogger.Info().
		Int32("account_id", accountID).
		Int32("user_id", userID).
		Msg("Starting account deletion")

	// First verify the account exists and belongs to the user
	account, err := s.GetAccount(ctx, accountID, userID)
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("account_id", accountID).
			Int32("user_id", userID).
			Msg("Failed to verify account ownership for deletion")
		return err // Error already formatted in GetAccount
	}

	// Check if account has zero balance
	if !account.IsZeroBalance() {
		contextLogger.Warn().
			Int32("account_id", accountID).
			Int32("user_id", userID).
			Str("current_balance", account.FormatBalance()).
			Msg("Cannot delete account with non-zero balance")
		s.auditLogger.LogAccountDeletion(int64(userID), int64(accountID), "failed_non_zero_balance")
		return fmt.Errorf("cannot delete account with non-zero balance: current balance is %s", account.FormatBalance())
	}

	// Check if account has any transaction history
	dbStart := time.Now()
	transferCount, err := s.transferRepo.CountTransfersByAccount(ctx, accountID)
	s.performanceLogger.LogDatabaseQuery("COUNT transfers by account", time.Since(dbStart), transferCount)
	
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("account_id", accountID).
			Int32("user_id", userID).
			Msg("Failed to check transaction history")
		s.auditLogger.LogAccountDeletion(int64(userID), int64(accountID), "failed_database_error")
		return fmt.Errorf("failed to check transaction history: %w", err)
	}

	if transferCount > 0 {
		contextLogger.Warn().
			Int32("account_id", accountID).
			Int32("user_id", userID).
			Int64("transfer_count", transferCount).
			Msg("Cannot delete account with transaction history")
		s.auditLogger.LogAccountDeletion(int64(userID), int64(accountID), "failed_has_transaction_history")
		return fmt.Errorf("cannot delete account with transaction history: %d transactions found", transferCount)
	}

	// Delete the account (database constraint ensures balance is zero)
	dbStart = time.Now()
	err = s.accountRepo.DeleteAccount(ctx, accountID)
	s.performanceLogger.LogDatabaseQuery("DELETE account", time.Since(dbStart), 1)
	
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("account_id", accountID).
			Int32("user_id", userID).
			Msg("Failed to delete account from database")
		s.auditLogger.LogAccountDeletion(int64(userID), int64(accountID), "failed_database_error")
		return fmt.Errorf("failed to delete account: %w", err)
	}

	duration := time.Since(start)
	contextLogger.Info().
		Int32("account_id", accountID).
		Int32("user_id", userID).
		Str("currency", account.Currency).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("Account deleted successfully")
	
	// Audit log for successful account deletion
	s.auditLogger.LogAccountDeletion(int64(userID), int64(accountID), "success")

	return nil
}

// convertDBAccountToModel converts a database account to a model account
func (s *AccountServiceImpl) convertDBAccountToModel(dbAccount queries.Account) (*models.Account, error) {
	// Convert pgtype.Numeric to decimal.Decimal
	balance, err := s.convertPgNumericToDecimal(dbAccount.Balance)
	if err != nil {
		return nil, fmt.Errorf("failed to convert balance: %w", err)
	}

	// Convert pgtype.Timestamp to time.Time
	createdAt, err := s.convertPgTimestampToTime(dbAccount.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to convert created_at: %w", err)
	}

	updatedAt, err := s.convertPgTimestampToTime(dbAccount.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to convert updated_at: %w", err)
	}

	return &models.Account{
		ID:        int(dbAccount.ID),
		UserID:    int(dbAccount.UserID),
		Currency:  dbAccount.Currency,
		Balance:   balance,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// convertPgNumericToDecimal converts pgtype.Numeric to decimal.Decimal
func (s *AccountServiceImpl) convertPgNumericToDecimal(pgNum pgtype.Numeric) (decimal.Decimal, error) {
	if !pgNum.Valid {
		return decimal.Zero, nil
	}

	// Convert pgtype.Numeric to string and then to decimal.Decimal
	numStr := pgNum.Int.String()
	if pgNum.Exp < 0 {
		// Handle decimal places
		exp := int(-pgNum.Exp)
		if len(numStr) <= exp {
			// Pad with zeros if needed
			numStr = "0." + fmt.Sprintf("%0*s", exp, numStr)
		} else {
			// Insert decimal point
			pos := len(numStr) - exp
			numStr = numStr[:pos] + "." + numStr[pos:]
		}
	}

	dec, err := decimal.NewFromString(numStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to parse numeric value: %w", err)
	}

	return dec, nil
}

// convertPgTimestampToTime converts pgtype.Timestamp to time.Time
func (s *AccountServiceImpl) convertPgTimestampToTime(pgTime pgtype.Timestamp) (time.Time, error) {
	if !pgTime.Valid {
		return time.Time{}, fmt.Errorf("invalid timestamp")
	}
	return pgTime.Time, nil
}