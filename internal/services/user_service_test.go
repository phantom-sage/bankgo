package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(ctx context.Context, arg queries.CreateUserParams) (queries.User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.User), args.Error(1)
}

func (m *MockUserRepository) GetUser(ctx context.Context, id int32) (queries.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(queries.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (queries.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(queries.User), args.Error(1)
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, arg queries.UpdateUserParams) (queries.User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(queries.User), args.Error(1)
}

func (m *MockUserRepository) MarkWelcomeEmailSent(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteUser(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) ListUsers(ctx context.Context, arg queries.ListUsersParams) ([]queries.User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]queries.User), args.Error(1)
}

func TestUserService_CreateUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo)

	t.Run("successful user creation", func(t *testing.T) {
		email := "test@example.com"
		password := "password123"
		firstName := "John"
		lastName := "Doe"

		// Mock that user doesn't exist
		mockRepo.On("GetUserByEmail", ctx, email).Return(queries.User{}, errors.New("not found")).Once()

		// Mock successful creation
		expectedDBUser := queries.User{
			ID:               1,
			Email:            email,
			PasswordHash:     "hashed_password",
			FirstName:        firstName,
			LastName:         lastName,
			WelcomeEmailSent: pgtype.Bool{Bool: false, Valid: true},
			CreatedAt:        pgtype.Timestamp{Time: time.Now(), Valid: true},
			UpdatedAt:        pgtype.Timestamp{Time: time.Now(), Valid: true},
		}

		mockRepo.On("CreateUser", ctx, mock.MatchedBy(func(params queries.CreateUserParams) bool {
			return params.Email == email &&
				params.FirstName == firstName &&
				params.LastName == lastName &&
				params.PasswordHash != "" // Password should be hashed
		})).Return(expectedDBUser, nil).Once()

		user, err := service.CreateUser(ctx, email, password, firstName, lastName)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, firstName, user.FirstName)
		assert.Equal(t, lastName, user.LastName)
		assert.Equal(t, 1, user.ID)
		assert.False(t, user.WelcomeEmailSent)
		mockRepo.AssertExpectations(t)
	})

	t.Run("user already exists", func(t *testing.T) {
		email := "existing@example.com"
		password := "password123"
		firstName := "Jane"
		lastName := "Doe"

		// Mock that user already exists
		existingUser := queries.User{ID: 1, Email: email}
		mockRepo.On("GetUserByEmail", ctx, email).Return(existingUser, nil).Once()

		user, err := service.CreateUser(ctx, email, password, firstName, lastName)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "user with this email already exists")
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid email format", func(t *testing.T) {
		email := "invalid-email"
		password := "password123"
		firstName := "John"
		lastName := "Doe"

		user, err := service.CreateUser(ctx, email, password, firstName, lastName)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "validation failed")
		assert.Contains(t, err.Error(), "invalid email format")
	})

	t.Run("password too short", func(t *testing.T) {
		email := "test@example.com"
		password := "short"
		firstName := "John"
		lastName := "Doe"

		user, err := service.CreateUser(ctx, email, password, firstName, lastName)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "password hashing failed")
		assert.Contains(t, err.Error(), "password must be at least 8 characters long")
	})

	t.Run("empty first name", func(t *testing.T) {
		email := "test@example.com"
		password := "password123"
		firstName := ""
		lastName := "Doe"

		user, err := service.CreateUser(ctx, email, password, firstName, lastName)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "validation failed")
		assert.Contains(t, err.Error(), "first name cannot be empty")
	})

	t.Run("empty last name", func(t *testing.T) {
		email := "test@example.com"
		password := "password123"
		firstName := "John"
		lastName := ""

		user, err := service.CreateUser(ctx, email, password, firstName, lastName)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "validation failed")
		assert.Contains(t, err.Error(), "last name cannot be empty")
	})

	t.Run("database creation error", func(t *testing.T) {
		email := "test@example.com"
		password := "password123"
		firstName := "John"
		lastName := "Doe"

		// Mock that user doesn't exist
		mockRepo.On("GetUserByEmail", ctx, email).Return(queries.User{}, errors.New("not found")).Once()

		// Mock database error
		mockRepo.On("CreateUser", ctx, mock.AnythingOfType("queries.CreateUserParams")).Return(queries.User{}, errors.New("database error")).Once()

		user, err := service.CreateUser(ctx, email, password, firstName, lastName)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "failed to create user")
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_GetUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo)

	t.Run("successful user retrieval", func(t *testing.T) {
		userID := 1
		expectedDBUser := queries.User{
			ID:               int32(userID),
			Email:            "test@example.com",
			PasswordHash:     "hashed_password",
			FirstName:        "John",
			LastName:         "Doe",
			WelcomeEmailSent: pgtype.Bool{Bool: true, Valid: true},
			CreatedAt:        pgtype.Timestamp{Time: time.Now(), Valid: true},
			UpdatedAt:        pgtype.Timestamp{Time: time.Now(), Valid: true},
		}

		mockRepo.On("GetUser", ctx, int32(userID)).Return(expectedDBUser, nil).Once()

		user, err := service.GetUser(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "John", user.FirstName)
		assert.Equal(t, "Doe", user.LastName)
		assert.True(t, user.WelcomeEmailSent)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid user ID", func(t *testing.T) {
		userID := 0

		user, err := service.GetUser(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("negative user ID", func(t *testing.T) {
		userID := -1

		user, err := service.GetUser(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("user not found", func(t *testing.T) {
		userID := 999

		mockRepo.On("GetUser", ctx, int32(userID)).Return(queries.User{}, errors.New("not found")).Once()

		user, err := service.GetUser(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "user not found")
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_AuthenticateUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo)

	t.Run("successful authentication", func(t *testing.T) {
		email := "test@example.com"
		password := "password123"

		// Hash the password for the mock user
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		expectedDBUser := queries.User{
			ID:               1,
			Email:            email,
			PasswordHash:     string(hashedPassword),
			FirstName:        "John",
			LastName:         "Doe",
			WelcomeEmailSent: pgtype.Bool{Bool: false, Valid: true},
			CreatedAt:        pgtype.Timestamp{Time: time.Now(), Valid: true},
			UpdatedAt:        pgtype.Timestamp{Time: time.Now(), Valid: true},
		}

		mockRepo.On("GetUserByEmail", ctx, email).Return(expectedDBUser, nil).Once()

		user, err := service.AuthenticateUser(ctx, email, password)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, "John", user.FirstName)
		assert.Equal(t, "Doe", user.LastName)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid email format", func(t *testing.T) {
		email := "invalid-email"
		password := "password123"

		user, err := service.AuthenticateUser(ctx, email, password)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid email")
	})

	t.Run("user not found", func(t *testing.T) {
		email := "nonexistent@example.com"
		password := "password123"

		mockRepo.On("GetUserByEmail", ctx, email).Return(queries.User{}, errors.New("not found")).Once()

		user, err := service.AuthenticateUser(ctx, email, password)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid credentials")
		mockRepo.AssertExpectations(t)
	})

	t.Run("wrong password", func(t *testing.T) {
		email := "test@example.com"
		password := "wrongpassword"

		// Hash a different password
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)

		expectedDBUser := queries.User{
			ID:           1,
			Email:        email,
			PasswordHash: string(hashedPassword),
			FirstName:    "John",
			LastName:     "Doe",
		}

		mockRepo.On("GetUserByEmail", ctx, email).Return(expectedDBUser, nil).Once()

		user, err := service.AuthenticateUser(ctx, email, password)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid credentials")
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_MarkWelcomeEmailSent(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo)

	t.Run("successful welcome email marking", func(t *testing.T) {
		userID := 1

		mockRepo.On("MarkWelcomeEmailSent", ctx, int32(userID)).Return(nil).Once()

		err := service.MarkWelcomeEmailSent(ctx, userID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid user ID", func(t *testing.T) {
		userID := 0

		err := service.MarkWelcomeEmailSent(ctx, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("negative user ID", func(t *testing.T) {
		userID := -1

		err := service.MarkWelcomeEmailSent(ctx, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("database error", func(t *testing.T) {
		userID := 1

		mockRepo.On("MarkWelcomeEmailSent", ctx, int32(userID)).Return(errors.New("database error")).Once()

		err := service.MarkWelcomeEmailSent(ctx, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to mark welcome email sent")
		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_dbUserToModel(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := &UserServiceImpl{userRepo: mockRepo}

	t.Run("successful conversion", func(t *testing.T) {
		now := time.Now()
		dbUser := queries.User{
			ID:               123,
			Email:            "test@example.com",
			PasswordHash:     "hashed_password",
			FirstName:        "John",
			LastName:         "Doe",
			WelcomeEmailSent: pgtype.Bool{Bool: true, Valid: true},
			CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
			UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
		}

		user := service.dbUserToModel(dbUser)

		assert.Equal(t, 123, user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "hashed_password", user.PasswordHash)
		assert.Equal(t, "John", user.FirstName)
		assert.Equal(t, "Doe", user.LastName)
		assert.True(t, user.WelcomeEmailSent)
		assert.Equal(t, now, user.CreatedAt)
		assert.Equal(t, now, user.UpdatedAt)
	})
}