package services

import (
	"context"
	"fmt"
	"time"

	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/phantom-sage/bankgo/internal/logging"
	"github.com/phantom-sage/bankgo/internal/models"
	"github.com/phantom-sage/bankgo/internal/repository"
	"github.com/phantom-sage/bankgo/internal/utils"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
)

// TransferService defines the interface for transfer business logic
type TransferService interface {
	TransferMoney(ctx context.Context, req TransferMoneyRequest) (*models.Transfer, error)
	GetTransferHistory(ctx context.Context, req GetTransferHistoryRequest) (*TransferHistoryResponse, error)
	GetTransfer(ctx context.Context, transferID int32) (*models.Transfer, error)
	UpdateTransferStatus(ctx context.Context, transferID int32, status string) (*models.Transfer, error)
	GetTransfersByStatus(ctx context.Context, status string, limit, offset int32) (*TransferHistoryResponse, error)
	GetTransfersByUser(ctx context.Context, userID int32, limit, offset int32) (*TransferHistoryResponse, error)
}

// TransferMoneyRequest represents the request to transfer money between accounts
type TransferMoneyRequest struct {
	FromAccountID int32           `json:"from_account_id" binding:"required"`
	ToAccountID   int32           `json:"to_account_id" binding:"required"`
	Amount        decimal.Decimal `json:"amount" binding:"required"`
	Description   string          `json:"description"`
}

// GetTransferHistoryRequest represents the request to get transfer history
type GetTransferHistoryRequest struct {
	AccountID int32 `json:"account_id" binding:"required"`
	Limit     int32 `json:"limit"`
	Offset    int32 `json:"offset"`
}

// TransferHistoryResponse represents the response for transfer history
type TransferHistoryResponse struct {
	Transfers []models.Transfer `json:"transfers"`
	Total     int64             `json:"total"`
	Limit     int32             `json:"limit"`
	Offset    int32             `json:"offset"`
}

// TransferServiceImpl implements TransferService
type TransferServiceImpl struct {
	repo              *repository.Repository
	accountRepo       repository.AccountRepository
	transferRepo      repository.TransferRepository
	logger            zerolog.Logger
	auditLogger       *logging.AuditLogger
	performanceLogger *logging.PerformanceLogger
}

// NewTransferService creates a new transfer service
func NewTransferService(repo *repository.Repository, accountRepo repository.AccountRepository, transferRepo repository.TransferRepository, logger zerolog.Logger) TransferService {
	auditLogger := logging.NewAuditLogger(logger)
	performanceLogger := logging.NewPerformanceLogger(logger)
	return &TransferServiceImpl{
		repo:              repo,
		accountRepo:       accountRepo,
		transferRepo:      transferRepo,
		logger:            logger.With().Str("component", "transfer_service").Logger(),
		auditLogger:       auditLogger,
		performanceLogger: performanceLogger,
	}
}

// TransferMoney executes a money transfer between accounts with database transaction
func (s *TransferServiceImpl) TransferMoney(ctx context.Context, req TransferMoneyRequest) (*models.Transfer, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).WithOperation("transfer_money")
	
	contextLogger.Info().
		Int32("from_account_id", req.FromAccountID).
		Int32("to_account_id", req.ToAccountID).
		Str("amount", req.Amount.StringFixed(2)).
		Str("description", req.Description).
		Msg("Starting money transfer")

	// Create transfer model for validation
	transfer := &models.Transfer{
		FromAccountID: int(req.FromAccountID),
		ToAccountID:   int(req.ToAccountID),
		Amount:        req.Amount,
		Description:   req.Description,
		Status:        "completed",
	}

	// Basic field validation
	if err := transfer.ValidateFields(); err != nil {
		contextLogger.Error().
			Err(err).
			Int32("from_account_id", req.FromAccountID).
			Int32("to_account_id", req.ToAccountID).
			Str("amount", req.Amount.StringFixed(2)).
			Msg("Transfer validation failed")
		s.auditLogger.LogTransfer(int64(req.FromAccountID), int64(req.ToAccountID), req.Amount, "failed_validation")
		return nil, fmt.Errorf("transfer validation failed: %w", err)
	}

	var result *models.Transfer
	var txDuration time.Duration
	
	// Execute transfer within database transaction
	txStart := time.Now()
	err := s.repo.WithTx(ctx, func(qtx *queries.Queries) error {
		contextLogger.Debug().Msg("Starting database transaction for transfer")

		// 1. Get and lock both accounts for update to prevent race conditions
		lockStart := time.Now()
		fromAccount, err := qtx.GetAccountForUpdate(ctx, req.FromAccountID)
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int32("from_account_id", req.FromAccountID).
				Msg("Failed to lock from account")
			return fmt.Errorf("failed to get from account: %w", err)
		}

		toAccount, err := qtx.GetAccountForUpdate(ctx, req.ToAccountID)
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int32("to_account_id", req.ToAccountID).
				Msg("Failed to lock to account")
			return fmt.Errorf("failed to get to account: %w", err)
		}
		
		lockDuration := time.Since(lockStart)
		contextLogger.Debug().
			Int64("lock_duration_ms", lockDuration.Milliseconds()).
			Msg("Account locks acquired")

		// 2. Convert database models to business models for validation
		fromAccountModel, err := convertDBAccountToModel(fromAccount)
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int32("from_account_id", req.FromAccountID).
				Msg("Failed to convert from account model")
			return fmt.Errorf("failed to convert from account: %w", err)
		}

		toAccountModel, err := convertDBAccountToModel(toAccount)
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int32("to_account_id", req.ToAccountID).
				Msg("Failed to convert to account model")
			return fmt.Errorf("failed to convert to account: %w", err)
		}

		// 3. Validate currency matching and sufficient balance
		if err := transfer.ValidateTransfer(fromAccountModel, toAccountModel); err != nil {
			contextLogger.Error().
				Err(err).
				Int32("from_account_id", req.FromAccountID).
				Int32("to_account_id", req.ToAccountID).
				Str("from_currency", fromAccountModel.Currency).
				Str("to_currency", toAccountModel.Currency).
				Str("from_balance", fromAccountModel.Balance.StringFixed(2)).
				Str("transfer_amount", req.Amount.StringFixed(2)).
				Msg("Transfer business validation failed")
			return fmt.Errorf("transfer validation failed: %w", err)
		}

		// 4. Subtract amount from source account
		subtractStart := time.Now()
		_, err = qtx.SubtractFromBalance(ctx, queries.SubtractFromBalanceParams{
			ID:      req.FromAccountID,
			Balance: utils.ConvertDecimalToPgNumeric(req.Amount),
		})
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int32("from_account_id", req.FromAccountID).
				Str("amount", req.Amount.StringFixed(2)).
				Msg("Failed to subtract from source account")
			return fmt.Errorf("failed to subtract from source account: %w", err)
		}
		s.performanceLogger.LogDatabaseQuery("UPDATE subtract balance", time.Since(subtractStart), 1)

		// 5. Add amount to destination account
		addStart := time.Now()
		_, err = qtx.AddToBalance(ctx, queries.AddToBalanceParams{
			ID:      req.ToAccountID,
			Balance: utils.ConvertDecimalToPgNumeric(req.Amount),
		})
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int32("to_account_id", req.ToAccountID).
				Str("amount", req.Amount.StringFixed(2)).
				Msg("Failed to add to destination account")
			return fmt.Errorf("failed to add to destination account: %w", err)
		}
		s.performanceLogger.LogDatabaseQuery("UPDATE add balance", time.Since(addStart), 1)

		// 6. Create transfer record
		createStart := time.Now()
		dbTransfer, err := qtx.CreateTransfer(ctx, queries.CreateTransferParams{
			FromAccountID: req.FromAccountID,
			ToAccountID:   req.ToAccountID,
			Amount:        utils.ConvertDecimalToPgNumeric(req.Amount),
			Column4:       req.Description,
			Column5:       "completed",
		})
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int32("from_account_id", req.FromAccountID).
				Int32("to_account_id", req.ToAccountID).
				Msg("Failed to create transfer record")
			return fmt.Errorf("failed to create transfer record: %w", err)
		}
		s.performanceLogger.LogDatabaseQuery("INSERT transfer", time.Since(createStart), 1)

		// Convert database transfer to business model
		result, err = convertDBTransferToModel(dbTransfer)
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int32("transfer_id", dbTransfer.ID).
				Msg("Failed to convert transfer result")
			return fmt.Errorf("failed to convert transfer result: %w", err)
		}
		
		contextLogger.Debug().
			Int32("transfer_id", dbTransfer.ID).
			Msg("Transfer record created successfully")
		
		return nil
	})

	txDuration = time.Since(txStart)
	s.performanceLogger.LogDatabaseTransaction(txDuration, 4, err == nil) // 4 operations: 2 locks, 2 updates, 1 insert

	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("from_account_id", req.FromAccountID).
			Int32("to_account_id", req.ToAccountID).
			Str("amount", req.Amount.StringFixed(2)).
			Int64("tx_duration_ms", txDuration.Milliseconds()).
			Msg("Transfer transaction failed")
		s.auditLogger.LogTransfer(int64(req.FromAccountID), int64(req.ToAccountID), req.Amount, "failed_transaction_error")
		return nil, fmt.Errorf("transfer transaction failed: %w", err)
	}

	// Log successful transfer
	duration := time.Since(start)
	contextLogger.Info().
		Int32("transfer_id", int32(result.ID)).
		Int32("from_account_id", req.FromAccountID).
		Int32("to_account_id", req.ToAccountID).
		Str("amount", req.Amount.StringFixed(2)).
		Str("description", req.Description).
		Int64("duration_ms", duration.Milliseconds()).
		Int64("tx_duration_ms", txDuration.Milliseconds()).
		Msg("Money transfer completed successfully")
	
	// Audit log for successful transfer
	s.auditLogger.LogTransferWithDetails(int64(result.ID), int64(req.FromAccountID), int64(req.ToAccountID), 
		req.Amount, "USD", req.Description, "success", 0) // Note: userID would need to be passed from context

	return result, nil
}

// GetTransferHistory retrieves transfer history for an account with pagination
func (s *TransferServiceImpl) GetTransferHistory(ctx context.Context, req GetTransferHistoryRequest) (*TransferHistoryResponse, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).WithOperation("get_transfer_history")
	
	contextLogger.Debug().
		Int32("account_id", req.AccountID).
		Int32("limit", req.Limit).
		Int32("offset", req.Offset).
		Msg("Retrieving transfer history")

	// Set default pagination values
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100 // Maximum limit
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Get transfers for the account
	dbStart := time.Now()
	dbTransfers, err := s.transferRepo.GetTransfersByAccount(ctx, queries.GetTransfersByAccountParams{
		FromAccountID: req.AccountID,
		Limit:         req.Limit,
		Offset:        req.Offset,
	})
	s.performanceLogger.LogDatabaseQuery("SELECT transfers by account", time.Since(dbStart), int64(len(dbTransfers)))
	
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("account_id", req.AccountID).
			Msg("Failed to get transfer history from database")
		return nil, fmt.Errorf("failed to get transfer history: %w", err)
	}

	// Convert database transfers to business models
	transfers := make([]models.Transfer, len(dbTransfers))
	for i, dbTransfer := range dbTransfers {
		transfer, err := convertDBTransferRowToModel(dbTransfer)
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int("index", i).
				Msg("Failed to convert transfer to model")
			return nil, fmt.Errorf("failed to convert transfer at index %d: %w", i, err)
		}
		transfers[i] = transfer
	}

	// Get total count for pagination
	countStart := time.Now()
	total, err := s.transferRepo.CountTransfersByAccount(ctx, req.AccountID)
	s.performanceLogger.LogDatabaseQuery("COUNT transfers by account", time.Since(countStart), total)
	
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("account_id", req.AccountID).
			Msg("Failed to count transfers")
		return nil, fmt.Errorf("failed to count transfers: %w", err)
	}

	duration := time.Since(start)
	contextLogger.Info().
		Int32("account_id", req.AccountID).
		Int("transfer_count", len(transfers)).
		Int64("total_count", total).
		Int32("limit", req.Limit).
		Int32("offset", req.Offset).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("Transfer history retrieved successfully")

	return &TransferHistoryResponse{
		Transfers: transfers,
		Total:     total,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}, nil
}

// GetTransfer retrieves a specific transfer by ID with detailed information
func (s *TransferServiceImpl) GetTransfer(ctx context.Context, transferID int32) (*models.Transfer, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).WithOperation("get_transfer")
	
	contextLogger.Debug().
		Int32("transfer_id", transferID).
		Msg("Retrieving transfer by ID")

	dbStart := time.Now()
	dbTransfer, err := s.transferRepo.GetTransfer(ctx, transferID)
	s.performanceLogger.LogDatabaseQuery("SELECT transfer by ID", time.Since(dbStart), 1)
	
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("transfer_id", transferID).
			Msg("Failed to get transfer from database")
		return nil, fmt.Errorf("failed to get transfer: %w", err)
	}

	// Convert to business model with additional details
	transfer, err := convertDBGetTransferRowToModel(dbTransfer)
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("transfer_id", transferID).
			Msg("Failed to convert transfer to model")
		return nil, fmt.Errorf("failed to convert transfer: %w", err)
	}

	duration := time.Since(start)
	contextLogger.Debug().
		Int32("transfer_id", transferID).
		Int32("from_account_id", int32(transfer.FromAccountID)).
		Int32("to_account_id", int32(transfer.ToAccountID)).
		Str("amount", transfer.Amount.StringFixed(2)).
		Str("status", transfer.Status).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("Transfer retrieved successfully")

	return &transfer, nil
}

// UpdateTransferStatus updates the status of a transfer
func (s *TransferServiceImpl) UpdateTransferStatus(ctx context.Context, transferID int32, status string) (*models.Transfer, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).WithOperation("update_transfer_status")
	
	contextLogger.Info().
		Int32("transfer_id", transferID).
		Str("new_status", status).
		Msg("Updating transfer status")

	// Validate the status
	transfer := &models.Transfer{Status: status}
	if err := transfer.ValidateStatus(); err != nil {
		contextLogger.Error().
			Err(err).
			Int32("transfer_id", transferID).
			Str("status", status).
			Msg("Invalid transfer status provided")
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	// Update the transfer status in database
	dbStart := time.Now()
	dbTransfer, err := s.transferRepo.UpdateTransferStatus(ctx, queries.UpdateTransferStatusParams{
		ID:     transferID,
		Status: utils.ConvertStringToPgText(status),
	})
	s.performanceLogger.LogDatabaseQuery("UPDATE transfer status", time.Since(dbStart), 1)
	
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("transfer_id", transferID).
			Str("status", status).
			Msg("Failed to update transfer status in database")
		return nil, fmt.Errorf("failed to update transfer status: %w", err)
	}

	// Convert to business model
	result, err := convertDBTransferToModel(dbTransfer)
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("transfer_id", transferID).
			Msg("Failed to convert updated transfer to model")
		return nil, fmt.Errorf("failed to convert transfer result: %w", err)
	}

	duration := time.Since(start)
	contextLogger.Info().
		Int32("transfer_id", transferID).
		Str("old_status", result.Status).
		Str("new_status", status).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("Transfer status updated successfully")

	return result, nil
}

// GetTransfersByStatus retrieves transfers by status with pagination
func (s *TransferServiceImpl) GetTransfersByStatus(ctx context.Context, status string, limit, offset int32) (*TransferHistoryResponse, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).WithOperation("get_transfers_by_status")
	
	contextLogger.Debug().
		Str("status", status).
		Int32("limit", limit).
		Int32("offset", offset).
		Msg("Retrieving transfers by status")

	// Validate status
	transfer := &models.Transfer{Status: status}
	if err := transfer.ValidateStatus(); err != nil {
		contextLogger.Error().
			Err(err).
			Str("status", status).
			Msg("Invalid transfer status provided")
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	// Set default pagination values
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Get transfers by status
	dbStart := time.Now()
	dbTransfers, err := s.transferRepo.GetTransfersByStatus(ctx, queries.GetTransfersByStatusParams{
		Status: utils.ConvertStringToPgText(status),
		Limit:  limit,
		Offset: offset,
	})
	s.performanceLogger.LogDatabaseQuery("SELECT transfers by status", time.Since(dbStart), int64(len(dbTransfers)))
	
	if err != nil {
		contextLogger.Error().
			Err(err).
			Str("status", status).
			Msg("Failed to get transfers by status from database")
		return nil, fmt.Errorf("failed to get transfers by status: %w", err)
	}

	// Convert database transfers to business models
	transfers := make([]models.Transfer, len(dbTransfers))
	for i, dbTransfer := range dbTransfers {
		transfer, err := convertDBTransfersByStatusRowToModel(dbTransfer)
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int("index", i).
				Msg("Failed to convert transfer to model")
			return nil, fmt.Errorf("failed to convert transfer at index %d: %w", i, err)
		}
		transfers[i] = transfer
	}

	// For status-based queries, we'll return the count as the length of results
	// In a real implementation, you might want a separate count query
	total := int64(len(transfers))

	duration := time.Since(start)
	contextLogger.Info().
		Str("status", status).
		Int("transfer_count", len(transfers)).
		Int64("total_count", total).
		Int32("limit", limit).
		Int32("offset", offset).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("Transfers by status retrieved successfully")

	return &TransferHistoryResponse{
		Transfers: transfers,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}, nil
}

// GetTransfersByUser retrieves all transfers for a specific user with pagination
func (s *TransferServiceImpl) GetTransfersByUser(ctx context.Context, userID int32, limit, offset int32) (*TransferHistoryResponse, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).
		WithOperation("get_transfers_by_user").
		WithUserID(int64(userID))
	
	contextLogger.Debug().
		Int32("user_id", userID).
		Int32("limit", limit).
		Int32("offset", offset).
		Msg("Retrieving transfers by user")

	// Set default pagination values
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Get transfers for the user
	dbStart := time.Now()
	dbTransfers, err := s.transferRepo.GetTransfersByUser(ctx, queries.GetTransfersByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	s.performanceLogger.LogDatabaseQuery("SELECT transfers by user", time.Since(dbStart), int64(len(dbTransfers)))
	
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int32("user_id", userID).
			Msg("Failed to get transfers by user from database")
		return nil, fmt.Errorf("failed to get transfers by user: %w", err)
	}

	// Convert database transfers to business models
	transfers := make([]models.Transfer, len(dbTransfers))
	for i, dbTransfer := range dbTransfers {
		transfer, err := convertDBTransfersByUserRowToModel(dbTransfer)
		if err != nil {
			contextLogger.Error().
				Err(err).
				Int("index", i).
				Msg("Failed to convert transfer to model")
			return nil, fmt.Errorf("failed to convert transfer at index %d: %w", i, err)
		}
		transfers[i] = transfer
	}

	// For user-based queries, we'll return the count as the length of results
	total := int64(len(transfers))

	duration := time.Since(start)
	contextLogger.Info().
		Int32("user_id", userID).
		Int("transfer_count", len(transfers)).
		Int64("total_count", total).
		Int32("limit", limit).
		Int32("offset", offset).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("Transfers by user retrieved successfully")

	return &TransferHistoryResponse{
		Transfers: transfers,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}, nil
}

// Helper functions for converting between database and business models

// convertDBAccountToModel converts a database account to business model
func convertDBAccountToModel(dbAccount queries.Account) (*models.Account, error) {
	balance, err := utils.ConvertPgNumericToDecimal(dbAccount.Balance)
	if err != nil {
		return nil, fmt.Errorf("failed to convert balance: %w", err)
	}

	return &models.Account{
		ID:        int(dbAccount.ID),
		UserID:    int(dbAccount.UserID),
		Currency:  dbAccount.Currency,
		Balance:   balance,
		CreatedAt: utils.ConvertPgTimestampToTime(dbAccount.CreatedAt),
		UpdatedAt: utils.ConvertPgTimestampToTime(dbAccount.UpdatedAt),
	}, nil
}

// convertDBTransferToModel converts a database transfer to business model
func convertDBTransferToModel(dbTransfer queries.Transfer) (*models.Transfer, error) {
	amount, err := utils.ConvertPgNumericToDecimal(dbTransfer.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to convert amount: %w", err)
	}

	return &models.Transfer{
		ID:            int(dbTransfer.ID),
		FromAccountID: int(dbTransfer.FromAccountID),
		ToAccountID:   int(dbTransfer.ToAccountID),
		Amount:        amount,
		Description:   utils.ConvertPgTextToString(dbTransfer.Description),
		Status:        utils.ConvertPgTextToString(dbTransfer.Status),
		CreatedAt:     utils.ConvertPgTimestampToTime(dbTransfer.CreatedAt),
	}, nil
}

// convertDBTransferRowToModel converts a database transfer row (with joins) to business model
func convertDBTransferRowToModel(dbTransfer queries.GetTransfersByAccountRow) (models.Transfer, error) {
	amount, err := utils.ConvertPgNumericToDecimal(dbTransfer.Amount)
	if err != nil {
		return models.Transfer{}, fmt.Errorf("failed to convert amount: %w", err)
	}

	return models.Transfer{
		ID:            int(dbTransfer.ID),
		FromAccountID: int(dbTransfer.FromAccountID),
		ToAccountID:   int(dbTransfer.ToAccountID),
		Amount:        amount,
		Description:   utils.ConvertPgTextToString(dbTransfer.Description),
		Status:        utils.ConvertPgTextToString(dbTransfer.Status),
		CreatedAt:     utils.ConvertPgTimestampToTime(dbTransfer.CreatedAt),
	}, nil
}

// convertDBGetTransferRowToModel converts a GetTransferRow (with detailed joins) to business model
func convertDBGetTransferRowToModel(dbTransfer queries.GetTransferRow) (models.Transfer, error) {
	amount, err := utils.ConvertPgNumericToDecimal(dbTransfer.Amount)
	if err != nil {
		return models.Transfer{}, fmt.Errorf("failed to convert amount: %w", err)
	}

	return models.Transfer{
		ID:            int(dbTransfer.ID),
		FromAccountID: int(dbTransfer.FromAccountID),
		ToAccountID:   int(dbTransfer.ToAccountID),
		Amount:        amount,
		Description:   utils.ConvertPgTextToString(dbTransfer.Description),
		Status:        utils.ConvertPgTextToString(dbTransfer.Status),
		CreatedAt:     utils.ConvertPgTimestampToTime(dbTransfer.CreatedAt),
	}, nil
}

// convertDBTransfersByStatusRowToModel converts a GetTransfersByStatusRow to business model
func convertDBTransfersByStatusRowToModel(dbTransfer queries.GetTransfersByStatusRow) (models.Transfer, error) {
	amount, err := utils.ConvertPgNumericToDecimal(dbTransfer.Amount)
	if err != nil {
		return models.Transfer{}, fmt.Errorf("failed to convert amount: %w", err)
	}

	return models.Transfer{
		ID:            int(dbTransfer.ID),
		FromAccountID: int(dbTransfer.FromAccountID),
		ToAccountID:   int(dbTransfer.ToAccountID),
		Amount:        amount,
		Description:   utils.ConvertPgTextToString(dbTransfer.Description),
		Status:        utils.ConvertPgTextToString(dbTransfer.Status),
		CreatedAt:     utils.ConvertPgTimestampToTime(dbTransfer.CreatedAt),
	}, nil
}

// convertDBTransfersByUserRowToModel converts a GetTransfersByUserRow to business model
func convertDBTransfersByUserRowToModel(dbTransfer queries.GetTransfersByUserRow) (models.Transfer, error) {
	amount, err := utils.ConvertPgNumericToDecimal(dbTransfer.Amount)
	if err != nil {
		return models.Transfer{}, fmt.Errorf("failed to convert amount: %w", err)
	}

	return models.Transfer{
		ID:            int(dbTransfer.ID),
		FromAccountID: int(dbTransfer.FromAccountID),
		ToAccountID:   int(dbTransfer.ToAccountID),
		Amount:        amount,
		Description:   utils.ConvertPgTextToString(dbTransfer.Description),
		Status:        utils.ConvertPgTextToString(dbTransfer.Status),
		CreatedAt:     utils.ConvertPgTimestampToTime(dbTransfer.CreatedAt),
	}, nil
}