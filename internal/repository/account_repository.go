package repository

import (
	"context"
	"time"

	"github.com/phantom-sage/bankgo/internal/database/queries"
)

// AccountRepository defines the interface for account database operations
type AccountRepository interface {
	CreateAccount(ctx context.Context, arg queries.CreateAccountParams) (queries.Account, error)
	GetAccount(ctx context.Context, id int32) (queries.Account, error)
	GetAccountForUpdate(ctx context.Context, id int32) (queries.Account, error)
	GetUserAccounts(ctx context.Context, userID int32) ([]queries.Account, error)
	GetAccountByUserAndCurrency(ctx context.Context, arg queries.GetAccountByUserAndCurrencyParams) (queries.Account, error)
	UpdateAccountBalance(ctx context.Context, arg queries.UpdateAccountBalanceParams) (queries.Account, error)
	UpdateAccount(ctx context.Context, id int32) (queries.Account, error)
	DeleteAccount(ctx context.Context, id int32) error
	ListAccounts(ctx context.Context, arg queries.ListAccountsParams) ([]queries.ListAccountsRow, error)
	GetAccountsWithBalance(ctx context.Context) ([]queries.Account, error)
	AddToBalance(ctx context.Context, arg queries.AddToBalanceParams) (queries.Account, error)
	SubtractFromBalance(ctx context.Context, arg queries.SubtractFromBalanceParams) (queries.Account, error)
}

// AccountRepositoryImpl implements AccountRepository
type AccountRepositoryImpl struct {
	*Repository
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(repo *Repository) AccountRepository {
	return &AccountRepositoryImpl{Repository: repo}
}

func (r *AccountRepositoryImpl) CreateAccount(ctx context.Context, arg queries.CreateAccountParams) (queries.Account, error) {
	startTime := time.Now()
	account, err := r.Queries.CreateAccount(ctx, arg)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "INSERT", "accounts", startTime, 1, err)
	
	return account, err
}

func (r *AccountRepositoryImpl) GetAccount(ctx context.Context, id int32) (queries.Account, error) {
	startTime := time.Now()
	account, err := r.Queries.GetAccount(ctx, id)
	
	// Log the database operation
	rowsAffected := int64(0)
	if err == nil {
		rowsAffected = 1
	}
	r.LogDatabaseOperation(ctx, "SELECT", "accounts", startTime, rowsAffected, err)
	
	return account, err
}

func (r *AccountRepositoryImpl) GetAccountForUpdate(ctx context.Context, id int32) (queries.Account, error) {
	startTime := time.Now()
	account, err := r.Queries.GetAccountForUpdate(ctx, id)
	
	// Log the database operation with FOR UPDATE context
	rowsAffected := int64(0)
	if err == nil {
		rowsAffected = 1
	}
	r.LogDatabaseOperation(ctx, "SELECT FOR UPDATE", "accounts", startTime, rowsAffected, err)
	
	return account, err
}

func (r *AccountRepositoryImpl) GetUserAccounts(ctx context.Context, userID int32) ([]queries.Account, error) {
	startTime := time.Now()
	accounts, err := r.Queries.GetUserAccounts(ctx, userID)
	
	// Log the database operation
	rowsAffected := int64(len(accounts))
	r.LogDatabaseOperation(ctx, "SELECT", "accounts", startTime, rowsAffected, err)
	
	return accounts, err
}

func (r *AccountRepositoryImpl) GetAccountByUserAndCurrency(ctx context.Context, arg queries.GetAccountByUserAndCurrencyParams) (queries.Account, error) {
	startTime := time.Now()
	account, err := r.Queries.GetAccountByUserAndCurrency(ctx, arg)
	
	// Log the database operation
	rowsAffected := int64(0)
	if err == nil {
		rowsAffected = 1
	}
	r.LogDatabaseOperation(ctx, "SELECT", "accounts", startTime, rowsAffected, err)
	
	return account, err
}

func (r *AccountRepositoryImpl) UpdateAccountBalance(ctx context.Context, arg queries.UpdateAccountBalanceParams) (queries.Account, error) {
	startTime := time.Now()
	account, err := r.Queries.UpdateAccountBalance(ctx, arg)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "UPDATE", "accounts", startTime, 1, err)
	
	return account, err
}

func (r *AccountRepositoryImpl) UpdateAccount(ctx context.Context, id int32) (queries.Account, error) {
	startTime := time.Now()
	account, err := r.Queries.UpdateAccount(ctx, id)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "UPDATE", "accounts", startTime, 1, err)
	
	return account, err
}

func (r *AccountRepositoryImpl) DeleteAccount(ctx context.Context, id int32) error {
	startTime := time.Now()
	err := r.Queries.DeleteAccount(ctx, id)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "DELETE", "accounts", startTime, 1, err)
	
	return err
}

func (r *AccountRepositoryImpl) ListAccounts(ctx context.Context, arg queries.ListAccountsParams) ([]queries.ListAccountsRow, error) {
	startTime := time.Now()
	accounts, err := r.Queries.ListAccounts(ctx, arg)
	
	// Log the database operation
	rowsAffected := int64(len(accounts))
	r.LogDatabaseOperation(ctx, "SELECT", "accounts", startTime, rowsAffected, err)
	
	return accounts, err
}

func (r *AccountRepositoryImpl) GetAccountsWithBalance(ctx context.Context) ([]queries.Account, error) {
	startTime := time.Now()
	accounts, err := r.Queries.GetAccountsWithBalance(ctx)
	
	// Log the database operation
	rowsAffected := int64(len(accounts))
	r.LogDatabaseOperation(ctx, "SELECT", "accounts", startTime, rowsAffected, err)
	
	return accounts, err
}

func (r *AccountRepositoryImpl) AddToBalance(ctx context.Context, arg queries.AddToBalanceParams) (queries.Account, error) {
	startTime := time.Now()
	account, err := r.Queries.AddToBalance(ctx, arg)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "UPDATE", "accounts", startTime, 1, err)
	
	return account, err
}

func (r *AccountRepositoryImpl) SubtractFromBalance(ctx context.Context, arg queries.SubtractFromBalanceParams) (queries.Account, error) {
	startTime := time.Now()
	account, err := r.Queries.SubtractFromBalance(ctx, arg)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "UPDATE", "accounts", startTime, 1, err)
	
	return account, err
}