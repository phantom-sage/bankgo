package services

import (
	"context"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockQueries is a mock implementation of the queries interface
type MockQueries struct {
	mock.Mock
}

func (m *MockQueries) AdminListUsers(ctx context.Context, arg queries.AdminListUsersParams) ([]queries.AdminListUsersRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.AdminListUsersRow), args.Error(1)
}

func (m *MockQueries) AdminCountUsers(ctx context.Context, arg queries.AdminCountUsersParams) (int64, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockQueries) AdminGetUserDetail(ctx context.Context, id int32) (queries.AdminGetUserDetailRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.AdminGetUserDetailRow), args.Error(1)
}

func (m *MockQueries) AdminCreateUser(ctx context.Context, arg queries.AdminCreateUserParams) (queries.User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.User), args.Error(1)
}

func (m *MockQueries) AdminUpdateUser(ctx context.Context, arg queries.AdminUpdateUserParams) (queries.User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.User), args.Error(1)
}

func (m *MockQueries) AdminDisableUser(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQueries) AdminEnableUser(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQueries) AdminDeleteUser(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// UserManagementServiceWithMock wraps the service with a mock queries interface
type UserManagementServiceWithMock struct {
	*UserManagementService
	mockQueries *MockQueries
}

func NewUserManagementServiceWithMock() *UserManagementServiceWithMock {
	mockQueries := &MockQueries{}
	service := &UserManagementService{
		db:      nil, // Not used in tests
		queries: mockQueries,
	}
	return &UserManagementServiceWithMock{
		UserManagementService: service,
		mockQueries:           mockQueries,
	}
}

func TestUserManagementService_ListUsers(t *testing.T) {
	service := NewUserManagementServiceWithMock()
	ctx := context.Background()

	// Test data
	now := time.Now()
	mockUsers := []queries.AdminListUsersRow{
		{
			ID:               1,
			Email:            "user1@example.com",
			FirstName:        "John",
			LastName:         "Doe",
			IsActive:         pgtype.Bool{Bool: true, Valid: true},
			WelcomeEmailSent: pgtype.Bool{Bool: true, Valid: true},
			CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
			UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
			AccountCount:     2,
			TransferCount:    5,
		},
		{
			ID:               2,
			Email:            "user2@example.com",
			FirstName:        "Jane",
			LastName:         "Smith",
			IsActive:         pgtype.Bool{Bool: false, Valid: true},
			WelcomeEmailSent: pgtype.Bool{Bool: false, Valid: true},
			CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
			UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
			AccountCount:     1,
			TransferCount:    0,
		},
	}

	t.Run("successful list with default parameters", func(t *testing.T) {
		params := interfaces.ListUsersParams{}

		// Setup mocks
		service.mockQueries.On("AdminCountUsers", ctx, mock.AnythingOfType("queries.AdminCountUsersParams")).Return(int64(2), nil)
		service.mockQueries.On("AdminListUsers", ctx, mock.AnythingOfType("queries.AdminListUsersParams")).Return(mockUsers, nil)

		// Execute
		result, err := service.ListUsers(ctx, params)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Users, 2)
		assert.Equal(t, "1", result.Users[0].ID)
		assert.Equal(t, "user1@example.com", result.Users[0].Email)
		assert.Equal(t, true, result.Users[0].IsActive)
		assert.Equal(t, 2, result.Users[0].AccountCount)
		assert.Equal(t, 5, result.Users[0].TransferCount)

		// Check pagination
		assert.Equal(t, 1, result.Pagination.Page)
		assert.Equal(t, 20, result.Pagination.PageSize)
		assert.Equal(t, 2, result.Pagination.TotalItems)
		assert.Equal(t, 1, result.Pagination.TotalPages)
		assert.False(t, result.Pagination.HasNext)
		assert.False(t, result.Pagination.HasPrev)

		service.mockQueries.AssertExpectations(t)
	})

	t.Run("list with search and filtering", func(t *testing.T) {
		// Create a new service instance for this test to avoid mock conflicts
		service2 := NewUserManagementServiceWithMock()
		
		isActive := true
		params := interfaces.ListUsersParams{
			PaginationParams: interfaces.PaginationParams{
				Page:     2,
				PageSize: 10,
			},
			Search:   "john",
			IsActive: &isActive,
			SortBy:   "email",
			SortDesc: true,
		}

		// Create filtered mock data (only first user matches search)
		filteredUsers := []queries.AdminListUsersRow{mockUsers[0]}

		// Setup mocks
		service2.mockQueries.On("AdminCountUsers", ctx, mock.AnythingOfType("queries.AdminCountUsersParams")).Return(int64(15), nil)
		service2.mockQueries.On("AdminListUsers", ctx, mock.AnythingOfType("queries.AdminListUsersParams")).Return(filteredUsers, nil)

		// Execute
		result, err := service2.ListUsers(ctx, params)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Users, 1)

		// Check pagination
		assert.Equal(t, 2, result.Pagination.Page)
		assert.Equal(t, 10, result.Pagination.PageSize)
		assert.Equal(t, 15, result.Pagination.TotalItems)
		assert.Equal(t, 2, result.Pagination.TotalPages)
		assert.False(t, result.Pagination.HasNext)
		assert.True(t, result.Pagination.HasPrev)

		service2.mockQueries.AssertExpectations(t)
	})
}

func TestUserManagementService_GetUser(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	mockUser := queries.AdminGetUserDetailRow{
		ID:               1,
		Email:            "user@example.com",
		FirstName:        "John",
		LastName:         "Doe",
		IsActive:         pgtype.Bool{Bool: true, Valid: true},
		WelcomeEmailSent: pgtype.Bool{Bool: true, Valid: true},
		CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
		AccountCount:     2,
		TransferCount:    5,
	}

	t.Run("successful get user", func(t *testing.T) {
		service := NewUserManagementServiceWithMock()
		
		// Setup mock
		service.mockQueries.On("AdminGetUserDetail", ctx, int32(1)).Return(mockUser, nil)

		// Execute
		result, err := service.GetUser(ctx, "1")

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "1", result.ID)
		assert.Equal(t, "user@example.com", result.Email)
		assert.Equal(t, "John", result.FirstName)
		assert.Equal(t, "Doe", result.LastName)
		assert.True(t, result.IsActive)
		assert.Equal(t, 2, result.AccountCount)
		assert.Equal(t, 5, result.TransferCount)

		service.mockQueries.AssertExpectations(t)
	})

	t.Run("invalid user ID", func(t *testing.T) {
		service := NewUserManagementServiceWithMock()
		
		// Execute
		result, err := service.GetUser(ctx, "invalid")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}

func TestUserManagementService_CreateUser(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	mockUser := queries.User{
		ID:               1,
		Email:            "newuser@example.com",
		FirstName:        "New",
		LastName:         "User",
		IsActive:         pgtype.Bool{Bool: true, Valid: true},
		WelcomeEmailSent: pgtype.Bool{Bool: false, Valid: true},
		CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
	}

	t.Run("successful create user", func(t *testing.T) {
		service := NewUserManagementServiceWithMock()
		
		req := interfaces.CreateUserRequest{
			Email:     "newuser@example.com",
			FirstName: "New",
			LastName:  "User",
			Password:  "password123",
		}

		// Setup mock
		service.mockQueries.On("AdminCreateUser", ctx, mock.AnythingOfType("queries.AdminCreateUserParams")).Return(mockUser, nil)

		// Execute
		result, err := service.CreateUser(ctx, req)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "1", result.ID)
		assert.Equal(t, "newuser@example.com", result.Email)
		assert.Equal(t, "New", result.FirstName)
		assert.Equal(t, "User", result.LastName)
		assert.True(t, result.IsActive)
		assert.Equal(t, 0, result.AccountCount) // New user has no accounts
		assert.Equal(t, 0, result.TransferCount) // New user has no transfers

		service.mockQueries.AssertExpectations(t)
	})
}

func TestUserManagementService_UpdateUser(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	mockUser := queries.User{
		ID:               1,
		Email:            "user@example.com",
		FirstName:        "Updated",
		LastName:         "Name",
		IsActive:         pgtype.Bool{Bool: false, Valid: true},
		WelcomeEmailSent: pgtype.Bool{Bool: true, Valid: true},
		CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
	}

	mockUserDetail := queries.AdminGetUserDetailRow{
		ID:               1,
		Email:            "user@example.com",
		FirstName:        "Updated",
		LastName:         "Name",
		IsActive:         pgtype.Bool{Bool: false, Valid: true},
		WelcomeEmailSent: pgtype.Bool{Bool: true, Valid: true},
		CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
		AccountCount:     1,
		TransferCount:    3,
	}

	t.Run("successful update user", func(t *testing.T) {
		service := NewUserManagementServiceWithMock()
		
		firstName := "Updated"
		lastName := "Name"
		isActive := false
		req := interfaces.UpdateUserRequest{
			FirstName: &firstName,
			LastName:  &lastName,
			IsActive:  &isActive,
		}

		// Setup mocks
		service.mockQueries.On("AdminUpdateUser", ctx, mock.AnythingOfType("queries.AdminUpdateUserParams")).Return(mockUser, nil)
		service.mockQueries.On("AdminGetUserDetail", ctx, int32(1)).Return(mockUserDetail, nil)

		// Execute
		result, err := service.UpdateUser(ctx, "1", req)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "1", result.ID)
		assert.Equal(t, "Updated", result.FirstName)
		assert.Equal(t, "Name", result.LastName)
		assert.False(t, result.IsActive)

		service.mockQueries.AssertExpectations(t)
	})
}

func TestUserManagementService_DisableUser(t *testing.T) {
	ctx := context.Background()

	t.Run("successful disable user", func(t *testing.T) {
		service := NewUserManagementServiceWithMock()
		
		// Setup mock
		service.mockQueries.On("AdminDisableUser", ctx, int32(1)).Return(nil)

		// Execute
		err := service.DisableUser(ctx, "1")

		// Assert
		assert.NoError(t, err)
		service.mockQueries.AssertExpectations(t)
	})

	t.Run("invalid user ID", func(t *testing.T) {
		service := NewUserManagementServiceWithMock()
		
		// Execute
		err := service.DisableUser(ctx, "invalid")

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}

func TestUserManagementService_EnableUser(t *testing.T) {
	ctx := context.Background()

	t.Run("successful enable user", func(t *testing.T) {
		service := NewUserManagementServiceWithMock()
		
		// Setup mock
		service.mockQueries.On("AdminEnableUser", ctx, int32(1)).Return(nil)

		// Execute
		err := service.EnableUser(ctx, "1")

		// Assert
		assert.NoError(t, err)
		service.mockQueries.AssertExpectations(t)
	})
}

func TestUserManagementService_DeleteUser(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("successful delete user with no dependencies", func(t *testing.T) {
		service := NewUserManagementServiceWithMock()
		
		mockUserDetail := queries.AdminGetUserDetailRow{
			ID:               1,
			Email:            "user@example.com",
			FirstName:        "John",
			LastName:         "Doe",
			IsActive:         pgtype.Bool{Bool: true, Valid: true},
			WelcomeEmailSent: pgtype.Bool{Bool: true, Valid: true},
			CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
			UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
			AccountCount:     0, // No accounts
			TransferCount:    0, // No transfers
		}

		// Setup mocks
		service.mockQueries.On("AdminGetUserDetail", ctx, int32(1)).Return(mockUserDetail, nil)
		service.mockQueries.On("AdminDeleteUser", ctx, int32(1)).Return(nil)

		// Execute
		err := service.DeleteUser(ctx, "1")

		// Assert
		assert.NoError(t, err)
		service.mockQueries.AssertExpectations(t)
	})

	t.Run("cannot delete user with accounts", func(t *testing.T) {
		service := NewUserManagementServiceWithMock()
		
		mockUserDetail := queries.AdminGetUserDetailRow{
			ID:               1,
			Email:            "user@example.com",
			FirstName:        "John",
			LastName:         "Doe",
			IsActive:         pgtype.Bool{Bool: true, Valid: true},
			WelcomeEmailSent: pgtype.Bool{Bool: true, Valid: true},
			CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
			UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
			AccountCount:     2, // Has accounts
			TransferCount:    0,
		}

		// Setup mock
		service.mockQueries.On("AdminGetUserDetail", ctx, int32(1)).Return(mockUserDetail, nil)

		// Execute
		err := service.DeleteUser(ctx, "1")

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete user with existing accounts")
		service.mockQueries.AssertExpectations(t)
	})

	t.Run("cannot delete user with transfer history", func(t *testing.T) {
		service := NewUserManagementServiceWithMock()
		
		mockUserDetail := queries.AdminGetUserDetailRow{
			ID:               1,
			Email:            "user@example.com",
			FirstName:        "John",
			LastName:         "Doe",
			IsActive:         pgtype.Bool{Bool: true, Valid: true},
			WelcomeEmailSent: pgtype.Bool{Bool: true, Valid: true},
			CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
			UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
			AccountCount:     0,
			TransferCount:    5, // Has transfer history
		}

		// Setup mock
		service.mockQueries.On("AdminGetUserDetail", ctx, int32(1)).Return(mockUserDetail, nil)

		// Execute
		err := service.DeleteUser(ctx, "1")

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete user with transfer history")
		service.mockQueries.AssertExpectations(t)
	})
}