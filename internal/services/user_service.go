package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/phantom-sage/bankgo/internal/logging"
	"github.com/phantom-sage/bankgo/internal/models"
	"github.com/phantom-sage/bankgo/internal/repository"
	"github.com/rs/zerolog"
)

// UserService defines the interface for user business logic operations
type UserService interface {
	CreateUser(ctx context.Context, email, password, firstName, lastName string) (*models.User, error)
	GetUser(ctx context.Context, userID int) (*models.User, error)
	AuthenticateUser(ctx context.Context, email, password string) (*models.User, error)
	MarkWelcomeEmailSent(ctx context.Context, userID int) error
}

// UserServiceImpl implements UserService
type UserServiceImpl struct {
	userRepo    repository.UserRepository
	logger      zerolog.Logger
	auditLogger *logging.AuditLogger
}

// NewUserService creates a new user service
func NewUserService(userRepo repository.UserRepository, logger zerolog.Logger) UserService {
	auditLogger := logging.NewAuditLogger(logger)
	return &UserServiceImpl{
		userRepo:    userRepo,
		logger:      logger.With().Str("component", "user_service").Logger(),
		auditLogger: auditLogger,
	}
}

// CreateUser creates a new user with password hashing and validation
func (s *UserServiceImpl) CreateUser(ctx context.Context, email, password, firstName, lastName string) (*models.User, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).WithOperation("create_user")
	
	contextLogger.Info().
		Str("user_email", email).
		Str("first_name", firstName).
		Str("last_name", lastName).
		Msg("Starting user creation")

	// Create user model for validation
	user := &models.User{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	}

	// Validate user fields
	if err := user.ValidateFields(); err != nil {
		contextLogger.Error().
			Err(err).
			Str("user_email", email).
			Msg("User validation failed")
		s.auditLogger.LogUserRegistration(0, email, "failed_validation")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Hash password
	if err := user.HashPassword(password); err != nil {
		contextLogger.Error().
			Err(err).
			Str("user_email", email).
			Msg("Password hashing failed")
		s.auditLogger.LogUserRegistration(0, email, "failed_password_hash")
		return nil, fmt.Errorf("password hashing failed: %w", err)
	}

	// Check if user already exists
	_, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil {
		contextLogger.Warn().
			Str("user_email", email).
			Msg("User with email already exists")
		s.auditLogger.LogUserRegistration(0, email, "failed_duplicate_email")
		return nil, errors.New("user with this email already exists")
	}

	// Create user in database
	dbUser, err := s.userRepo.CreateUser(ctx, queries.CreateUserParams{
		Email:        email,
		PasswordHash: user.PasswordHash,
		FirstName:    firstName,
		LastName:     lastName,
	})
	if err != nil {
		contextLogger.Error().
			Err(err).
			Str("user_email", email).
			Msg("Failed to create user in database")
		s.auditLogger.LogUserRegistration(0, email, "failed_database_error")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Convert database user to model
	result := s.dbUserToModel(dbUser)
	
	// Log successful creation
	duration := time.Since(start)
	contextLogger.Info().
		Int64("user_id", int64(result.ID)).
		Str("user_email", email).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("User created successfully")
	
	// Audit log for successful user registration
	s.auditLogger.LogUserRegistration(int64(result.ID), email, "success")
	
	return result, nil
}

// GetUser retrieves a user by ID
func (s *UserServiceImpl) GetUser(ctx context.Context, userID int) (*models.User, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).
		WithOperation("get_user").
		WithUserID(int64(userID))
	
	contextLogger.Debug().
		Int("user_id", userID).
		Msg("Retrieving user by ID")

	if userID <= 0 {
		contextLogger.Error().
			Int("user_id", userID).
			Msg("Invalid user ID provided")
		return nil, errors.New("invalid user ID")
	}

	dbUser, err := s.userRepo.GetUser(ctx, int32(userID))
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int("user_id", userID).
			Msg("User not found in database")
		return nil, fmt.Errorf("user not found: %w", err)
	}

	result := s.dbUserToModel(dbUser)
	duration := time.Since(start)
	
	contextLogger.Debug().
		Int("user_id", userID).
		Str("user_email", result.Email).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("User retrieved successfully")

	return result, nil
}

// AuthenticateUser authenticates a user with email and password
func (s *UserServiceImpl) AuthenticateUser(ctx context.Context, email, password string) (*models.User, error) {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).
		WithOperation("authenticate_user").
		WithUserEmail(email)
	
	contextLogger.Info().
		Str("user_email", email).
		Msg("Starting user authentication")

	// Validate email format
	user := &models.User{Email: email}
	if err := user.ValidateEmail(); err != nil {
		contextLogger.Error().
			Err(err).
			Str("user_email", email).
			Msg("Invalid email format provided")
		s.auditLogger.LogFailedAuthentication(email, "invalid_email_format", "")
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// Get user by email
	dbUser, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		contextLogger.Warn().
			Err(err).
			Str("user_email", email).
			Msg("User not found during authentication")
		s.auditLogger.LogFailedAuthentication(email, "user_not_found", "")
		return nil, errors.New("invalid credentials")
	}

	// Convert to model for password checking
	userModel := s.dbUserToModel(dbUser)

	// Check password
	if err := userModel.CheckPassword(password); err != nil {
		contextLogger.Warn().
			Str("user_email", email).
			Int64("user_id", int64(userModel.ID)).
			Msg("Invalid password provided during authentication")
		s.auditLogger.LogAuthentication(int64(userModel.ID), email, "login", "failed_invalid_password")
		return nil, errors.New("invalid credentials")
	}

	// Log successful authentication
	duration := time.Since(start)
	contextLogger.Info().
		Int64("user_id", int64(userModel.ID)).
		Str("user_email", email).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("User authenticated successfully")
	
	// Audit log for successful authentication
	s.auditLogger.LogAuthentication(int64(userModel.ID), email, "login", "success")

	return userModel, nil
}

// MarkWelcomeEmailSent marks the user as having received the welcome email
func (s *UserServiceImpl) MarkWelcomeEmailSent(ctx context.Context, userID int) error {
	start := time.Now()
	contextLogger := logging.NewContextLogger(s.logger, ctx).
		WithOperation("mark_welcome_email_sent").
		WithUserID(int64(userID))
	
	contextLogger.Debug().
		Int("user_id", userID).
		Msg("Marking welcome email as sent")

	if userID <= 0 {
		contextLogger.Error().
			Int("user_id", userID).
			Msg("Invalid user ID provided for welcome email marking")
		return errors.New("invalid user ID")
	}

	err := s.userRepo.MarkWelcomeEmailSent(ctx, int32(userID))
	if err != nil {
		contextLogger.Error().
			Err(err).
			Int("user_id", userID).
			Msg("Failed to mark welcome email as sent in database")
		return fmt.Errorf("failed to mark welcome email sent: %w", err)
	}

	duration := time.Since(start)
	contextLogger.Info().
		Int("user_id", userID).
		Int64("duration_ms", duration.Milliseconds()).
		Msg("Welcome email marked as sent successfully")

	return nil
}

// dbUserToModel converts a database user to a model user
func (s *UserServiceImpl) dbUserToModel(dbUser queries.User) *models.User {
	return &models.User{
		ID:               int(dbUser.ID),
		Email:            dbUser.Email,
		PasswordHash:     dbUser.PasswordHash,
		FirstName:        dbUser.FirstName,
		LastName:         dbUser.LastName,
		WelcomeEmailSent: dbUser.WelcomeEmailSent.Bool,
		CreatedAt:        dbUser.CreatedAt.Time,
		UpdatedAt:        dbUser.UpdatedAt.Time,
	}
}