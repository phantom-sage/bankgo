package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// transactionService implements the TransactionService interface
type transactionService struct {
	db      *pgxpool.Pool
	queries *queries.Queries
}

// NewTransactionService creates a new transaction service
func NewTransactionService(db *pgxpool.Pool) interfaces.TransactionService {
	return &transactionService{
		db:      db,
		queries: queries.New(db),
	}
}

// SearchTransactions returns transactions based on search criteria
func (s *transactionService) SearchTransactions(ctx context.Context, params interfaces.SearchTransactionParams) (*interfaces.PaginatedTransactions, error) {
	// Build dynamic query based on search parameters
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT t.id, t.from_account_id, t.to_account_id, t.amount, t.description, t.status, t.created_at,
		       fa.currency as from_currency, ta.currency as to_currency,
		       fu.email as from_user_email, tu.email as to_user_email,
		       fu.id as from_user_id, tu.id as to_user_id
		FROM transfers t
		JOIN accounts fa ON t.from_account_id = fa.id
		JOIN accounts ta ON t.to_account_id = ta.id
		JOIN users fu ON fa.user_id = fu.id
		JOIN users tu ON ta.user_id = tu.id`

	// Add search conditions
	if params.UserID != "" {
		userID, err := strconv.Atoi(params.UserID)
		if err != nil {
			return nil, fmt.Errorf("invalid user_id: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("(fa.user_id = $%d OR ta.user_id = $%d)", argIndex, argIndex))
		args = append(args, userID)
		argIndex++
	}

	if params.AccountID != "" {
		accountID, err := strconv.Atoi(params.AccountID)
		if err != nil {
			return nil, fmt.Errorf("invalid account_id: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("(t.from_account_id = $%d OR t.to_account_id = $%d)", argIndex, argIndex))
		args = append(args, accountID)
		argIndex++
	}

	if params.Currency != "" {
		conditions = append(conditions, fmt.Sprintf("fa.currency = $%d", argIndex))
		args = append(args, params.Currency)
		argIndex++
	}

	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("t.status = $%d", argIndex))
		args = append(args, params.Status)
		argIndex++
	}

	if params.AmountMin != nil {
		conditions = append(conditions, fmt.Sprintf("t.amount >= $%d", argIndex))
		args = append(args, *params.AmountMin)
		argIndex++
	}

	if params.AmountMax != nil {
		conditions = append(conditions, fmt.Sprintf("t.amount <= $%d", argIndex))
		args = append(args, *params.AmountMax)
		argIndex++
	}

	if params.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("t.created_at >= $%d", argIndex))
		args = append(args, *params.DateFrom)
		argIndex++
	}

	if params.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("t.created_at <= $%d", argIndex))
		args = append(args, *params.DateTo)
		argIndex++
	}

	if params.Description != "" {
		conditions = append(conditions, fmt.Sprintf("t.description ILIKE $%d", argIndex))
		args = append(args, "%"+params.Description+"%")
		argIndex++
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total records
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM transfers t
		JOIN accounts fa ON t.from_account_id = fa.id
		JOIN accounts ta ON t.to_account_id = ta.id
		JOIN users fu ON fa.user_id = fu.id
		JOIN users tu ON ta.user_id = tu.id
		%s`, whereClause)

	var totalCount int64
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count transactions: %w", err)
	}

	// Calculate pagination
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize

	// Build main query with pagination
	mainQuery := fmt.Sprintf(`
		%s
		%s
		ORDER BY t.created_at DESC
		LIMIT $%d OFFSET $%d`,
		baseQuery, whereClause, argIndex, argIndex+1)

	args = append(args, params.PageSize, offset)

	// Execute query
	rows, err := s.db.Query(ctx, mainQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search transactions: %w", err)
	}
	defer rows.Close()

	var transactions []interfaces.TransactionDetail
	for rows.Next() {
		var t interfaces.TransactionDetail
		var fromUserID, toUserID int32
		var amount pgtype.Numeric
		var description, status pgtype.Text
		var createdAt pgtype.Timestamp

		err := rows.Scan(
			&t.ID, &t.FromAccountID, &t.ToAccountID, &amount, &description, &status, &createdAt,
			&t.Currency, &t.Currency, // from_currency, to_currency (should be same)
			&t.FromAccount.UserID, &t.ToAccount.UserID, // from_user_email, to_user_email (we'll use as UserID for now)
			&fromUserID, &toUserID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		// Convert pgtype values
		if amount.Valid {
			t.Amount = amount.Int.String()
		}
		if description.Valid {
			t.Description = description.String
		}
		if status.Valid {
			t.Status = status.String
		}
		if createdAt.Valid {
			t.CreatedAt = createdAt.Time
		}

		// Set account summaries
		t.FromAccount = &interfaces.AccountSummary{
			ID:     t.FromAccountID,
			UserID: strconv.Itoa(int(fromUserID)),
		}
		t.ToAccount = &interfaces.AccountSummary{
			ID:     t.ToAccountID,
			UserID: strconv.Itoa(int(toUserID)),
		}

		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows: %w", err)
	}

	// Calculate pagination info
	totalPages := int((totalCount + int64(params.PageSize) - 1) / int64(params.PageSize))
	
	return &interfaces.PaginatedTransactions{
		Transactions: transactions,
		Pagination: interfaces.PaginationInfo{
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalItems: int(totalCount),
			TotalPages: totalPages,
			HasNext:    params.Page < totalPages,
			HasPrev:    params.Page > 1,
		},
	}, nil
}

// GetTransactionDetail returns detailed transaction information
func (s *transactionService) GetTransactionDetail(ctx context.Context, transactionID string) (*interfaces.TransactionDetail, error) {
	id, err := strconv.Atoi(transactionID)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction ID: %w", err)
	}

	transfer, err := s.queries.GetTransfer(ctx, int32(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Get account details
	fromAccount, err := s.queries.GetAccount(ctx, transfer.FromAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get from account: %w", err)
	}

	toAccount, err := s.queries.GetAccount(ctx, transfer.ToAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get to account: %w", err)
	}

	// Build transaction detail
	detail := &interfaces.TransactionDetail{
		ID:            strconv.Itoa(int(transfer.ID)),
		FromAccountID: strconv.Itoa(int(transfer.FromAccountID)),
		ToAccountID:   strconv.Itoa(int(transfer.ToAccountID)),
		Currency:      transfer.FromCurrency,
	}

	if transfer.Amount.Valid {
		detail.Amount = transfer.Amount.Int.String()
	}
	if transfer.Description.Valid {
		detail.Description = transfer.Description.String
	}
	if transfer.Status.Valid {
		detail.Status = transfer.Status.String
	}
	if transfer.CreatedAt.Valid {
		detail.CreatedAt = transfer.CreatedAt.Time
	}

	// Set account summaries
	detail.FromAccount = &interfaces.AccountSummary{
		ID:       strconv.Itoa(int(fromAccount.ID)),
		UserID:   strconv.Itoa(int(fromAccount.UserID)),
		Currency: fromAccount.Currency,
		Balance:  fromAccount.Balance.Int.String(),
		IsActive: true, // Assume active for now
	}

	detail.ToAccount = &interfaces.AccountSummary{
		ID:       strconv.Itoa(int(toAccount.ID)),
		UserID:   strconv.Itoa(int(toAccount.UserID)),
		Currency: toAccount.Currency,
		Balance:  toAccount.Balance.Int.String(),
		IsActive: true, // Assume active for now
	}

	// Create basic audit trail
	detail.AuditTrail = []interfaces.AuditEntry{
		{
			ID:        "1",
			Action:    "transfer_created",
			Actor:     "system",
			ActorType: "system",
			Timestamp: detail.CreatedAt,
			Details: map[string]interface{}{
				"from_account": detail.FromAccountID,
				"to_account":   detail.ToAccountID,
				"amount":       detail.Amount,
				"currency":     detail.Currency,
			},
		},
	}

	if detail.Status == "completed" {
		detail.ProcessedAt = &detail.CreatedAt
		detail.AuditTrail = append(detail.AuditTrail, interfaces.AuditEntry{
			ID:        "2",
			Action:    "transfer_completed",
			Actor:     "system",
			ActorType: "system",
			Timestamp: detail.CreatedAt,
			Details: map[string]interface{}{
				"status": "completed",
			},
		})
	}

	return detail, nil
}

// ReverseTransaction reverses a transaction
func (s *transactionService) ReverseTransaction(ctx context.Context, transactionID string, reason string) (*interfaces.TransactionDetail, error) {
	id, err := strconv.Atoi(transactionID)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction ID: %w", err)
	}

	// Start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// Get original transfer
	transfer, err := qtx.GetTransfer(ctx, int32(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Check if already reversed
	if transfer.Status.Valid && transfer.Status.String == "reversed" {
		return nil, fmt.Errorf("transaction already reversed")
	}

	// Check if transaction can be reversed (only completed transactions)
	if !transfer.Status.Valid || transfer.Status.String != "completed" {
		return nil, fmt.Errorf("only completed transactions can be reversed")
	}

	// Get accounts for update
	fromAccount, err := qtx.GetAccountForUpdate(ctx, transfer.FromAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get from account: %w", err)
	}

	toAccount, err := qtx.GetAccountForUpdate(ctx, transfer.ToAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get to account: %w", err)
	}

	// Reverse the balances
	if transfer.Amount.Valid {
		// Add amount back to from account
		_, err = qtx.AddToBalance(ctx, queries.AddToBalanceParams{
			ID:      fromAccount.ID,
			Balance: transfer.Amount,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add balance to from account: %w", err)
		}

		// Subtract amount from to account
		_, err = qtx.SubtractFromBalance(ctx, queries.SubtractFromBalanceParams{
			ID:      toAccount.ID,
			Balance: transfer.Amount,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to subtract balance from to account: %w", err)
		}
	}

	// Update transfer status
	_, err = qtx.UpdateTransferStatus(ctx, queries.UpdateTransferStatusParams{
		ID: transfer.ID,
		Status: pgtype.Text{
			String: "reversed",
			Valid:  true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update transfer status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit reversal transaction: %w", err)
	}

	// Get updated transaction detail
	detail, err := s.GetTransactionDetail(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated transaction detail: %w", err)
	}

	// Update reversal information
	now := time.Now()
	detail.ReversedAt = &now
	detail.ReversalReason = reason
	detail.Status = "reversed"

	// Add reversal audit entry
	detail.AuditTrail = append(detail.AuditTrail, interfaces.AuditEntry{
		ID:        strconv.Itoa(len(detail.AuditTrail) + 1),
		Action:    "transfer_reversed",
		Actor:     "admin", // TODO: Get actual admin from context
		ActorType: "admin",
		Timestamp: now,
		Details: map[string]interface{}{
			"reason":           reason,
			"reversed_amount":  detail.Amount,
			"reversal_method":  "admin_action",
		},
	})

	return detail, nil
}

// GetAccountTransactions returns transactions for a specific account
func (s *transactionService) GetAccountTransactions(ctx context.Context, accountID string, params interfaces.PaginationParams) (*interfaces.PaginatedTransactions, error) {
	id, err := strconv.Atoi(accountID)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize

	// Get transactions for account
	transfers, err := s.queries.GetTransfersByAccount(ctx, queries.GetTransfersByAccountParams{
		FromAccountID: int32(id),
		Limit:         int32(params.PageSize),
		Offset:        int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get account transactions: %w", err)
	}

	// Count total transactions for account
	totalCount, err := s.queries.CountTransfersByAccount(ctx, int32(id))
	if err != nil {
		return nil, fmt.Errorf("failed to count account transactions: %w", err)
	}

	// Convert to transaction details
	var transactions []interfaces.TransactionDetail
	for _, transfer := range transfers {
		detail := interfaces.TransactionDetail{
			ID:            strconv.Itoa(int(transfer.ID)),
			FromAccountID: strconv.Itoa(int(transfer.FromAccountID)),
			ToAccountID:   strconv.Itoa(int(transfer.ToAccountID)),
			Currency:      transfer.FromCurrency,
		}

		if transfer.Amount.Valid {
			detail.Amount = transfer.Amount.Int.String()
		}
		if transfer.Description.Valid {
			detail.Description = transfer.Description.String
		}
		if transfer.Status.Valid {
			detail.Status = transfer.Status.String
		}
		if transfer.CreatedAt.Valid {
			detail.CreatedAt = transfer.CreatedAt.Time
		}

		transactions = append(transactions, detail)
	}

	// Calculate pagination info
	totalPages := int((totalCount + int64(params.PageSize) - 1) / int64(params.PageSize))

	return &interfaces.PaginatedTransactions{
		Transactions: transactions,
		Pagination: interfaces.PaginationInfo{
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalItems: int(totalCount),
			TotalPages: totalPages,
			HasNext:    params.Page < totalPages,
			HasPrev:    params.Page > 1,
		},
	}, nil
}