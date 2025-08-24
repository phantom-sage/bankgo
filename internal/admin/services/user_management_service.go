package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// UserQueriesInterface defines the interface for user-related database queries
type UserQueriesInterface interface {
	AdminListUsers(ctx context.Context, arg queries.AdminListUsersParams) ([]queries.AdminListUsersRow, error)
	AdminCountUsers(ctx context.Context, arg queries.AdminCountUsersParams) (int64, error)
	AdminGetUserDetail(ctx context.Context, id int32) (queries.AdminGetUserDetailRow, error)
	AdminCreateUser(ctx context.Context, arg queries.AdminCreateUserParams) (queries.User, error)
	AdminUpdateUser(ctx context.Context, arg queries.AdminUpdateUserParams) (queries.User, error)
	AdminDisableUser(ctx context.Context, id int32) error
	AdminEnableUser(ctx context.Context, id int32) error
	AdminDeleteUser(ctx context.Context, id int32) error
}

// UserManagementService implements the UserManagementService interface
type UserManagementService struct {
	db      *pgxpool.Pool
	queries UserQueriesInterface
}

// NewUserManagementService creates a new user management service
func NewUserManagementService(db *pgxpool.Pool) interfaces.UserManagementService {
	return &UserManagementService{
		db:      db,
		queries: queries.New(db),
	}
}

// ListUsers returns paginated list of users with optional filtering
func (s *UserManagementService) ListUsers(ctx context.Context, params interfaces.ListUsersParams) (*interfaces.PaginatedUsers, error) {
	// Set default pagination values
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	// Calculate offset
	offset := (params.Page - 1) * params.PageSize

	// Set default sort
	if params.SortBy == "" {
		params.SortBy = "created_at"
	}

	// Convert search parameter (empty string if nil)
	search := ""
	if params.Search != "" {
		search = params.Search
	}

	// Get total count
	countParams := queries.AdminCountUsersParams{
		Column1: search,
		Column2: params.IsActive != nil && *params.IsActive,
	}
	totalCount, err := s.queries.AdminCountUsers(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users
	listParams := queries.AdminListUsersParams{
		Limit:   int32(params.PageSize),
		Offset:  int32(offset),
		Column3: search,
		Column4: params.IsActive != nil && *params.IsActive,
		Column5: params.SortBy,
		Column6: params.SortDesc,
	}

	dbUsers, err := s.queries.AdminListUsers(ctx, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Convert to UserDetail objects
	users := make([]interfaces.UserDetail, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = s.convertRowToUserDetail(dbUser)
	}

	// Calculate pagination info
	totalPages := int((totalCount + int64(params.PageSize) - 1) / int64(params.PageSize))
	
	pagination := interfaces.PaginationInfo{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: int(totalCount),
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}

	return &interfaces.PaginatedUsers{
		Users:      users,
		Pagination: pagination,
	}, nil
}

// GetUser returns detailed user information
func (s *UserManagementService) GetUser(ctx context.Context, userID string) (*interfaces.UserDetail, error) {
	id, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	dbUser, err := s.queries.AdminGetUserDetail(ctx, int32(id))
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	userDetail := s.convertRowToUserDetail(dbUser)
	return &userDetail, nil
}

// CreateUser creates a new user account
func (s *UserManagementService) CreateUser(ctx context.Context, req interfaces.CreateUserRequest) (*interfaces.UserDetail, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	createParams := queries.AdminCreateUserParams{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Column5:      true, // is_active = true by default
	}

	dbUser, err := s.queries.AdminCreateUser(ctx, createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	userDetail := s.ConvertToUserDetail(dbUser, 0, 0) // New user has no accounts/transfers
	return &userDetail, nil
}

// UpdateUser updates user information
func (s *UserManagementService) UpdateUser(ctx context.Context, userID string, req interfaces.UpdateUserRequest) (*interfaces.UserDetail, error) {
	id, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Prepare update parameters
	updateParams := queries.AdminUpdateUserParams{
		ID: int32(id),
	}

	if req.FirstName != nil {
		updateParams.FirstName = pgtype.Text{String: *req.FirstName, Valid: true}
	}
	if req.LastName != nil {
		updateParams.LastName = pgtype.Text{String: *req.LastName, Valid: true}
	}
	if req.IsActive != nil {
		updateParams.IsActive = pgtype.Bool{Bool: *req.IsActive, Valid: true}
	}

	_, err = s.queries.AdminUpdateUser(ctx, updateParams)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Get updated user details with counts
	updatedUser, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated user details: %w", err)
	}

	return updatedUser, nil
}

// DisableUser disables a user account
func (s *UserManagementService) DisableUser(ctx context.Context, userID string) error {
	id, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	err = s.queries.AdminDisableUser(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to disable user: %w", err)
	}

	return nil
}

// EnableUser enables a user account
func (s *UserManagementService) EnableUser(ctx context.Context, userID string) error {
	id, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	err = s.queries.AdminEnableUser(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to enable user: %w", err)
	}

	return nil
}

// DeleteUser deletes a user account
func (s *UserManagementService) DeleteUser(ctx context.Context, userID string) error {
	id, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user has accounts or transfers
	userDetail, err := s.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check user details: %w", err)
	}

	if userDetail.AccountCount > 0 {
		return fmt.Errorf("cannot delete user with existing accounts")
	}

	if userDetail.TransferCount > 0 {
		return fmt.Errorf("cannot delete user with transfer history")
	}

	err = s.queries.AdminDeleteUser(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// ConvertToUserDetail converts a database User to UserDetail interface type
func (s *UserManagementService) ConvertToUserDetail(dbUser queries.User, accountCount, transferCount int64) interfaces.UserDetail {
	var lastLogin *time.Time
	// Note: We don't have last_login in the current schema, so it's nil for now
	// This could be added in a future migration if needed

	return interfaces.UserDetail{
		ID:               strconv.Itoa(int(dbUser.ID)),
		Email:            dbUser.Email,
		FirstName:        dbUser.FirstName,
		LastName:         dbUser.LastName,
		IsActive:         dbUser.IsActive.Bool,
		CreatedAt:        dbUser.CreatedAt.Time,
		UpdatedAt:        dbUser.UpdatedAt.Time,
		LastLogin:        lastLogin,
		AccountCount:     int(accountCount),
		TransferCount:    int(transferCount),
		WelcomeEmailSent: dbUser.WelcomeEmailSent.Bool,
		Metadata:         make(map[string]interface{}), // Empty for now
	}
}

// convertRowToUserDetail converts AdminListUsersRow or AdminGetUserDetailRow to UserDetail
func (s *UserManagementService) convertRowToUserDetail(row interface{}) interfaces.UserDetail {
	var lastLogin *time.Time

	switch r := row.(type) {
	case queries.AdminListUsersRow:
		return interfaces.UserDetail{
			ID:               strconv.Itoa(int(r.ID)),
			Email:            r.Email,
			FirstName:        r.FirstName,
			LastName:         r.LastName,
			IsActive:         r.IsActive.Bool,
			CreatedAt:        r.CreatedAt.Time,
			UpdatedAt:        r.UpdatedAt.Time,
			LastLogin:        lastLogin,
			AccountCount:     int(r.AccountCount),
			TransferCount:    int(r.TransferCount),
			WelcomeEmailSent: r.WelcomeEmailSent.Bool,
			Metadata:         make(map[string]interface{}),
		}
	case queries.AdminGetUserDetailRow:
		return interfaces.UserDetail{
			ID:               strconv.Itoa(int(r.ID)),
			Email:            r.Email,
			FirstName:        r.FirstName,
			LastName:         r.LastName,
			IsActive:         r.IsActive.Bool,
			CreatedAt:        r.CreatedAt.Time,
			UpdatedAt:        r.UpdatedAt.Time,
			LastLogin:        lastLogin,
			AccountCount:     int(r.AccountCount),
			TransferCount:    int(r.TransferCount),
			WelcomeEmailSent: r.WelcomeEmailSent.Bool,
			Metadata:         make(map[string]interface{}),
		}
	default:
		// Fallback - should not happen
		return interfaces.UserDetail{
			Metadata: make(map[string]interface{}),
		}
	}
}