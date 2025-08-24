package services

import (
	"context"
	"fmt"
	"strconv"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// accountService implements account management operations
type accountService struct {
	db      *pgxpool.Pool
	queries *queries.Queries
}

// NewAccountService creates a new account service
func NewAccountService(db *pgxpool.Pool) interfaces.AccountService {
	return &accountService{
		db:      db,
		queries: queries.New(db),
	}
}

// SearchAccounts returns accounts based on search criteria
func (s *accountService) SearchAccounts(ctx context.Context, params interfaces.SearchAccountParams) (*interfaces.PaginatedAccounts, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	offset := (params.Page - 1) * params.PageSize

	// Prepare search parameters
	var search *string
	if params.Search != "" {
		search = &params.Search
	}

	var currency *string
	if params.Currency != "" {
		currency = &params.Currency
	}

	var balanceMin, balanceMax *pgtype.Numeric
	if params.BalanceMin != nil {
		amount, err := decimal.NewFromString(*params.BalanceMin)
		if err != nil {
			return nil, fmt.Errorf("invalid balance_min: %w", err)
		}
		balanceMin = &pgtype.Numeric{}
		balanceMin.Int = amount.BigInt()
		balanceMin.Valid = true
	}

	if params.BalanceMax != nil {
		amount, err := decimal.NewFromString(*params.BalanceMax)
		if err != nil {
			return nil, fmt.Errorf("invalid balance_max: %w", err)
		}
		balanceMax = &pgtype.Numeric{}
		balanceMax.Int = amount.BigInt()
		balanceMax.Valid = true
	}

	// For now, use a simple count query - we'll implement proper counting later
	var totalCount int64 = 0

	// Prepare parameters for SearchAccounts
	searchParam := ""
	if search != nil {
		searchParam = *search
	}
	currencyParam := ""
	if currency != nil {
		currencyParam = *currency
	}
	
	var balanceMinParam, balanceMaxParam pgtype.Numeric
	if balanceMin != nil {
		balanceMinParam = *balanceMin
	}
	if balanceMax != nil {
		balanceMaxParam = *balanceMax
	}
	
	var isActiveParam bool
	if params.IsActive != nil {
		isActiveParam = *params.IsActive
	}

	// Search accounts
	accountRows, err := s.queries.SearchAccounts(ctx, queries.SearchAccountsParams{
		Column1: searchParam,
		Column2: currencyParam,
		Column3: balanceMinParam,
		Column4: balanceMaxParam,
		Column5: isActiveParam,
		Limit:   int32(params.PageSize),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search accounts: %w", err)
	}

	// Convert to account details
	var accounts []interfaces.AccountDetail
	for _, row := range accountRows {
		account := interfaces.AccountDetail{
			ID:       strconv.Itoa(int(row.ID)),
			UserID:   strconv.Itoa(int(row.UserID)),
			Currency: row.Currency,
			User: interfaces.UserSummary{
				ID:        strconv.Itoa(int(row.UserID)),
				Email:     row.Email,
				FirstName: row.FirstName,
				LastName:  row.LastName,
			},
		}

		if row.Balance.Valid {
			account.Balance = row.Balance.Int.String()
		}
		if row.CreatedAt.Valid {
			account.CreatedAt = row.CreatedAt.Time
		}
		if row.UpdatedAt.Valid {
			account.UpdatedAt = row.UpdatedAt.Time
		}
		if row.IsActive.Valid {
			account.IsActive = row.IsActive.Bool
			account.User.IsActive = row.IsActive.Bool
		}

		accounts = append(accounts, account)
	}

	// Set total count to the number of results for now
	totalCount = int64(len(accounts))

	// Calculate pagination info
	totalPages := int((totalCount + int64(params.PageSize) - 1) / int64(params.PageSize))

	return &interfaces.PaginatedAccounts{
		Accounts: accounts,
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

// GetAccountDetail returns detailed account information
func (s *accountService) GetAccountDetail(ctx context.Context, accountID string) (*interfaces.AccountDetail, error) {
	id, err := strconv.Atoi(accountID)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	account, err := s.queries.GetAccountWithUser(ctx, int32(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("account not found")
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	detail := &interfaces.AccountDetail{
		ID:       strconv.Itoa(int(account.ID)),
		UserID:   strconv.Itoa(int(account.UserID)),
		Currency: account.Currency,
		User: interfaces.UserSummary{
			ID:        strconv.Itoa(int(account.UserID)),
			Email:     account.Email,
			FirstName: account.FirstName,
			LastName:  account.LastName,
		},
	}

	if account.Balance.Valid {
		detail.Balance = account.Balance.Int.String()
	}
	if account.CreatedAt.Valid {
		detail.CreatedAt = account.CreatedAt.Time
	}
	if account.UpdatedAt.Valid {
		detail.UpdatedAt = account.UpdatedAt.Time
	}
	if account.IsActive.Valid {
		detail.IsActive = account.IsActive.Bool
		detail.User.IsActive = account.IsActive.Bool
	}

	return detail, nil
}

// FreezeAccount freezes an account
func (s *accountService) FreezeAccount(ctx context.Context, accountID string, reason string) error {
	id, err := strconv.Atoi(accountID)
	if err != nil {
		return fmt.Errorf("invalid account ID: %w", err)
	}

	// For now, we'll just update the account timestamp to indicate it was modified
	// In a real implementation, you might add a "frozen" status field
	_, err = s.queries.FreezeAccount(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to freeze account: %w", err)
	}

	// TODO: Log the freeze action with reason in audit trail
	return nil
}

// UnfreezeAccount unfreezes an account
func (s *accountService) UnfreezeAccount(ctx context.Context, accountID string) error {
	id, err := strconv.Atoi(accountID)
	if err != nil {
		return fmt.Errorf("invalid account ID: %w", err)
	}

	_, err = s.queries.UnfreezeAccount(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to unfreeze account: %w", err)
	}

	// TODO: Log the unfreeze action in audit trail
	return nil
}

// AdjustBalance adjusts an account balance
func (s *accountService) AdjustBalance(ctx context.Context, accountID string, adjustment string, reason string) (*interfaces.AccountDetail, error) {
	id, err := strconv.Atoi(accountID)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	amount, err := decimal.NewFromString(adjustment)
	if err != nil {
		return nil, fmt.Errorf("invalid adjustment amount: %w", err)
	}

	// Convert to pgtype.Numeric
	pgAmount := pgtype.Numeric{
		Int:   amount.BigInt(),
		Valid: true,
	}

	// Start transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// Get account for update
	account, err := qtx.GetAccountForUpdate(ctx, int32(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("account not found")
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Apply adjustment
	if amount.IsPositive() {
		// Add to balance
		_, err = qtx.AddToBalance(ctx, queries.AddToBalanceParams{
			ID:      account.ID,
			Balance: pgAmount,
		})
	} else {
		// Subtract from balance (make amount positive for subtraction)
		absAmount := amount.Abs()
		pgAbsAmount := pgtype.Numeric{
			Int:   absAmount.BigInt(),
			Valid: true,
		}
		_, err = qtx.SubtractFromBalance(ctx, queries.SubtractFromBalanceParams{
			ID:      account.ID,
			Balance: pgAbsAmount,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to adjust balance: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit balance adjustment: %w", err)
	}

	// Get updated account detail
	detail, err := s.GetAccountDetail(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated account detail: %w", err)
	}

	// TODO: Log the balance adjustment in audit trail with reason

	return detail, nil
}