package repository

import (
	"context"
	"time"

	"github.com/phantom-sage/bankgo/internal/database/queries"
)

// TransferRepository defines the interface for transfer database operations
type TransferRepository interface {
	CreateTransfer(ctx context.Context, arg queries.CreateTransferParams) (queries.Transfer, error)
	GetTransfer(ctx context.Context, id int32) (queries.GetTransferRow, error)
	GetTransfersByAccount(ctx context.Context, arg queries.GetTransfersByAccountParams) ([]queries.GetTransfersByAccountRow, error)
	GetTransfersByUser(ctx context.Context, arg queries.GetTransfersByUserParams) ([]queries.GetTransfersByUserRow, error)
	UpdateTransferStatus(ctx context.Context, arg queries.UpdateTransferStatusParams) (queries.Transfer, error)
	ListTransfers(ctx context.Context, arg queries.ListTransfersParams) ([]queries.ListTransfersRow, error)
	GetTransfersByStatus(ctx context.Context, arg queries.GetTransfersByStatusParams) ([]queries.GetTransfersByStatusRow, error)
	GetTransfersByDateRange(ctx context.Context, arg queries.GetTransfersByDateRangeParams) ([]queries.GetTransfersByDateRangeRow, error)
	CountTransfersByAccount(ctx context.Context, fromAccountID int32) (int64, error)
}

// TransferRepositoryImpl implements TransferRepository
type TransferRepositoryImpl struct {
	*Repository
}

// NewTransferRepository creates a new transfer repository
func NewTransferRepository(repo *Repository) TransferRepository {
	return &TransferRepositoryImpl{Repository: repo}
}

func (r *TransferRepositoryImpl) CreateTransfer(ctx context.Context, arg queries.CreateTransferParams) (queries.Transfer, error) {
	startTime := time.Now()
	transfer, err := r.Queries.CreateTransfer(ctx, arg)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "INSERT", "transfers", startTime, 1, err)
	
	return transfer, err
}

func (r *TransferRepositoryImpl) GetTransfer(ctx context.Context, id int32) (queries.GetTransferRow, error) {
	startTime := time.Now()
	transfer, err := r.Queries.GetTransfer(ctx, id)
	
	// Log the database operation
	rowsAffected := int64(0)
	if err == nil {
		rowsAffected = 1
	}
	r.LogDatabaseOperation(ctx, "SELECT", "transfers", startTime, rowsAffected, err)
	
	return transfer, err
}

func (r *TransferRepositoryImpl) GetTransfersByAccount(ctx context.Context, arg queries.GetTransfersByAccountParams) ([]queries.GetTransfersByAccountRow, error) {
	startTime := time.Now()
	transfers, err := r.Queries.GetTransfersByAccount(ctx, arg)
	
	// Log the database operation
	rowsAffected := int64(len(transfers))
	r.LogDatabaseOperation(ctx, "SELECT", "transfers", startTime, rowsAffected, err)
	
	return transfers, err
}

func (r *TransferRepositoryImpl) GetTransfersByUser(ctx context.Context, arg queries.GetTransfersByUserParams) ([]queries.GetTransfersByUserRow, error) {
	startTime := time.Now()
	transfers, err := r.Queries.GetTransfersByUser(ctx, arg)
	
	// Log the database operation
	rowsAffected := int64(len(transfers))
	r.LogDatabaseOperation(ctx, "SELECT", "transfers", startTime, rowsAffected, err)
	
	return transfers, err
}

func (r *TransferRepositoryImpl) UpdateTransferStatus(ctx context.Context, arg queries.UpdateTransferStatusParams) (queries.Transfer, error) {
	startTime := time.Now()
	transfer, err := r.Queries.UpdateTransferStatus(ctx, arg)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "UPDATE", "transfers", startTime, 1, err)
	
	return transfer, err
}

func (r *TransferRepositoryImpl) ListTransfers(ctx context.Context, arg queries.ListTransfersParams) ([]queries.ListTransfersRow, error) {
	startTime := time.Now()
	transfers, err := r.Queries.ListTransfers(ctx, arg)
	
	// Log the database operation
	rowsAffected := int64(len(transfers))
	r.LogDatabaseOperation(ctx, "SELECT", "transfers", startTime, rowsAffected, err)
	
	return transfers, err
}

func (r *TransferRepositoryImpl) GetTransfersByStatus(ctx context.Context, arg queries.GetTransfersByStatusParams) ([]queries.GetTransfersByStatusRow, error) {
	startTime := time.Now()
	transfers, err := r.Queries.GetTransfersByStatus(ctx, arg)
	
	// Log the database operation
	rowsAffected := int64(len(transfers))
	r.LogDatabaseOperation(ctx, "SELECT", "transfers", startTime, rowsAffected, err)
	
	return transfers, err
}

func (r *TransferRepositoryImpl) GetTransfersByDateRange(ctx context.Context, arg queries.GetTransfersByDateRangeParams) ([]queries.GetTransfersByDateRangeRow, error) {
	startTime := time.Now()
	transfers, err := r.Queries.GetTransfersByDateRange(ctx, arg)
	
	// Log the database operation
	rowsAffected := int64(len(transfers))
	r.LogDatabaseOperation(ctx, "SELECT", "transfers", startTime, rowsAffected, err)
	
	return transfers, err
}

func (r *TransferRepositoryImpl) CountTransfersByAccount(ctx context.Context, fromAccountID int32) (int64, error) {
	startTime := time.Now()
	count, err := r.Queries.CountTransfersByAccount(ctx, fromAccountID)
	
	// Log the database operation
	rowsAffected := int64(1) // COUNT queries return 1 row
	r.LogDatabaseOperation(ctx, "SELECT COUNT", "transfers", startTime, rowsAffected, err)
	
	return count, err
}