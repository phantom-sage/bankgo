package repository

import (
	"context"
	"time"

	"github.com/phantom-sage/bankgo/internal/database/queries"
)

// UserRepository defines the interface for user database operations
type UserRepository interface {
	CreateUser(ctx context.Context, arg queries.CreateUserParams) (queries.User, error)
	GetUser(ctx context.Context, id int32) (queries.User, error)
	GetUserByEmail(ctx context.Context, email string) (queries.User, error)
	UpdateUser(ctx context.Context, arg queries.UpdateUserParams) (queries.User, error)
	MarkWelcomeEmailSent(ctx context.Context, id int32) error
	DeleteUser(ctx context.Context, id int32) error
	ListUsers(ctx context.Context, arg queries.ListUsersParams) ([]queries.User, error)
}

// UserRepositoryImpl implements UserRepository
type UserRepositoryImpl struct {
	*Repository
}

// NewUserRepository creates a new user repository
func NewUserRepository(repo *Repository) UserRepository {
	return &UserRepositoryImpl{Repository: repo}
}

func (r *UserRepositoryImpl) CreateUser(ctx context.Context, arg queries.CreateUserParams) (queries.User, error) {
	startTime := time.Now()
	user, err := r.Queries.CreateUser(ctx, arg)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "INSERT", "users", startTime, 1, err)
	
	return user, err
}

func (r *UserRepositoryImpl) GetUser(ctx context.Context, id int32) (queries.User, error) {
	startTime := time.Now()
	user, err := r.Queries.GetUser(ctx, id)
	
	// Log the database operation
	rowsAffected := int64(0)
	if err == nil {
		rowsAffected = 1
	}
	r.LogDatabaseOperation(ctx, "SELECT", "users", startTime, rowsAffected, err)
	
	return user, err
}

func (r *UserRepositoryImpl) GetUserByEmail(ctx context.Context, email string) (queries.User, error) {
	startTime := time.Now()
	user, err := r.Queries.GetUserByEmail(ctx, email)
	
	// Log the database operation (email is not logged for privacy)
	rowsAffected := int64(0)
	if err == nil {
		rowsAffected = 1
	}
	r.LogDatabaseOperation(ctx, "SELECT", "users", startTime, rowsAffected, err)
	
	return user, err
}

func (r *UserRepositoryImpl) UpdateUser(ctx context.Context, arg queries.UpdateUserParams) (queries.User, error) {
	startTime := time.Now()
	user, err := r.Queries.UpdateUser(ctx, arg)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "UPDATE", "users", startTime, 1, err)
	
	return user, err
}

func (r *UserRepositoryImpl) MarkWelcomeEmailSent(ctx context.Context, id int32) error {
	startTime := time.Now()
	err := r.Queries.MarkWelcomeEmailSent(ctx, id)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "UPDATE", "users", startTime, 1, err)
	
	return err
}

func (r *UserRepositoryImpl) DeleteUser(ctx context.Context, id int32) error {
	startTime := time.Now()
	err := r.Queries.DeleteUser(ctx, id)
	
	// Log the database operation
	r.LogDatabaseOperation(ctx, "DELETE", "users", startTime, 1, err)
	
	return err
}

func (r *UserRepositoryImpl) ListUsers(ctx context.Context, arg queries.ListUsersParams) ([]queries.User, error) {
	startTime := time.Now()
	users, err := r.Queries.ListUsers(ctx, arg)
	
	// Log the database operation
	rowsAffected := int64(len(users))
	r.LogDatabaseOperation(ctx, "SELECT", "users", startTime, rowsAffected, err)
	
	return users, err
}