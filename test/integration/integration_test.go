package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/phantom-sage/bankgo/internal/config"
	"github.com/phantom-sage/bankgo/internal/database"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/phantom-sage/bankgo/internal/handlers"
	"github.com/phantom-sage/bankgo/internal/middleware"
	"github.com/phantom-sage/bankgo/internal/models"
	"github.com/phantom-sage/bankgo/internal/queue"
	"github.com/phantom-sage/bankgo/internal/repository"
	"github.com/phantom-sage/bankgo/internal/services"
	"	"github.com/phantom-sage/bankgo/pkg/auth"
	"github.com/rs/zerolog""
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite provides a test suite for integration tests
type IntegrationTestSuite struct {
	suite.Suite
	db                *database.DB
	queueManager      *QueueManagerTestExtensions
	router            *gin.Engine
	tokenManager      *auth.PASETOManager
	
	// Services
	userService     services.UserService
	accountService  services.AccountService
	transferService services.TransferService
	
	// Test data
	testUsers    []*models.User
	testAccounts []*models.Account
	testTokens   []string
}

// SetupSuite runs once before all tests in the suite
func (suite *IntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	// Load test configuration
	cfg := suite.loadTestConfig()
	
	// Initialize database connection
	var err error
	suite.db, err = database.New(cfg.Database)
	require.NoError(suite.T(), err, "Failed to connect to test database")
	
	// Run migrations
	err = suite.db.Migrate()
	require.NoError(suite.T(), err, "Failed to run database migrations")
	
	// Initialize queue manager
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	baseQueueManager, err := queue.NewQueueManager(cfg.Redis, logger)
	require.NoError(suite.T(), err, "Failed to connect to Redis")
	
	// Wrap with test extensions
	suite.queueManager = NewQueueManagerTestExtensions(baseQueueManager)
	
	// Initialize token manager
	suite.tokenManager, err = auth.NewPASETOManager(cfg.PASETO.SecretKey, cfg.PASETO.Expiration)
	require.NoError(suite.T(), err, "Failed to initialize token manager")
	
	// Initialize repositories
	userRepo := repository.NewUserRepository(suite.db.GetQueries())
	accountRepo := repository.NewAccountRepository(suite.db.GetQueries())
	transferRepo := repository.NewTransferRepository(suite.db.GetQueries())
	
	// Initialize services
	suite.userService = services.NewUserService(userRepo)
	suite.accountService = services.NewAccountService(accountRepo, transferRepo)
	suite.transferService = services.NewTransferService(accountRepo, transferRepo, suite.db.GetDB())
	
	// Setup router
	suite.setupRouter()
}

// TearDownSuite runs once after all tests in the suite
func (suite *IntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
	if suite.queueManager != nil {
		suite.queueManager.Close()
	}
}

// SetupTest runs before each test
func (suite *IntegrationTestSuite) SetupTest() {
	// Clean up database
	suite.cleanupDatabase()
	
	// Reset test data
	suite.testUsers = nil
	suite.testAccounts = nil
	suite.testTokens = nil
}

// TearDownTest runs after each test
func (suite *IntegrationTestSuite) TearDownTest() {
	// Clean up database
	suite.cleanupDatabase()
}

// loadTestConfig loads configuration for testing
func (suite *IntegrationTestSuite) loadTestConfig() *config.Config {
	// Set test environment variables
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_NAME", "bankapi_test")
	os.Setenv("DB_USER", "bankuser")
	os.Setenv("DB_PASSWORD", "testpassword")
	os.Setenv("DB_SSL_MODE", "disable")
	
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("REDIS_DB", "1") // Use different DB for tests
	
	os.Setenv("PASETO_SECRET_KEY", "test-secret-key-32-characters-long")
	os.Setenv("PASETO_EXPIRATION", "1h")
	
	os.Setenv("SMTP_HOST", "localhost")
	os.Setenv("SMTP_PORT", "1025")
	os.Setenv("SMTP_USERNAME", "test@example.com")
	os.Setenv("SMTP_PASSWORD", "testpassword")
	
	cfg, err := config.LoadConfig()
	require.NoError(suite.T(), err, "Failed to load test configuration")
	
	return cfg
}

// setupRouter configures the test router
func (suite *IntegrationTestSuite) setupRouter() {
	suite.router = gin.New()
	
	// Add middleware
	suite.router.Use(gin.Recovery())
	suite.router.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	suite.router.Use(middleware.RequestID())
	
	// Create handlers
	authHandlers := handlers.NewAuthHandlers(suite.userService, suite.tokenManager, suite.queueManager.QueueManager)
	accountHandlers := handlers.NewAccountHandlers(suite.accountService)
	transferHandlers := handlers.NewTransferHandlers(suite.transferService, suite.accountService)
	healthHandlers := handlers.NewHealthHandlers(suite.db, suite.queueManager.QueueManager, "test-v1.0.0")
	
	// API v1 routes
	v1 := suite.router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", healthHandlers.HealthCheck)
		
		// Authentication routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandlers.Register)
			auth.POST("/login", authHandlers.Login)
			auth.POST("/logout", authHandlers.Logout)
		}
		
		// Protected routes
		protected := v1.Group("")
		protected.Use(authHandlers.AuthMiddleware())
		{
			// Account routes
			accounts := protected.Group("/accounts")
			{
				accounts.GET("", accountHandlers.GetUserAccounts)
				accounts.POST("", accountHandlers.CreateAccount)
				accounts.GET("/:id", accountHandlers.GetAccount)
				accounts.PUT("/:id", accountHandlers.UpdateAccount)
				accounts.DELETE("/:id", accountHandlers.DeleteAccount)
			}
			
			// Transfer routes
			transfers := protected.Group("/transfers")
			{
				transfers.POST("", transferHandlers.CreateTransfer)
				transfers.GET("", transferHandlers.GetTransferHistory)
				transfers.GET("/:id", transferHandlers.GetTransfer)
			}
		}
	}
}

// cleanupDatabase removes all test data
func (suite *IntegrationTestSuite) cleanupDatabase() {
	ctx := context.Background()
	queries := suite.db.GetQueries()
	
	// Delete in reverse order of dependencies
	_, err := suite.db.GetDB().Exec(ctx, "DELETE FROM transfers")
	require.NoError(suite.T(), err)
	
	_, err = suite.db.GetDB().Exec(ctx, "DELETE FROM accounts")
	require.NoError(suite.T(), err)
	
	_, err = suite.db.GetDB().Exec(ctx, "DELETE FROM users")
	require.NoError(suite.T(), err)
	
	// Clear Redis queues
	suite.queueManager.ClearQueues(ctx)
}

// createTestUser creates a test user and returns the user and token
func (suite *IntegrationTestSuite) createTestUser(email, firstName, lastName string) (*models.User, string) {
	ctx := context.Background()
	
	user, err := suite.userService.CreateUser(ctx, email, "password123", firstName, lastName)
	require.NoError(suite.T(), err)
	
	token, err := suite.tokenManager.GenerateToken(user.ID, user.Email)
	require.NoError(suite.T(), err)
	
	suite.testUsers = append(suite.testUsers, user)
	suite.testTokens = append(suite.testTokens, token)
	
	return user, token
}

// createTestAccount creates a test account for a user
func (suite *IntegrationTestSuite) createTestAccount(userID int32, currency string) *models.Account {
	ctx := context.Background()
	
	account, err := suite.accountService.CreateAccount(ctx, userID, currency)
	require.NoError(suite.T(), err)
	
	suite.testAccounts = append(suite.testAccounts, account)
	
	return account
}

// makeRequest makes an HTTP request to the test server
func (suite *IntegrationTestSuite) makeRequest(method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody []byte
	var err error
	
	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(suite.T(), err)
	}
	
	req, err := http.NewRequest(method, path, bytes.NewBuffer(reqBody))
	require.NoError(suite.T(), err)
	
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	
	return w
}

// parseResponse parses JSON response into the provided interface
func (suite *IntegrationTestSuite) parseResponse(w *httptest.ResponseRecorder, v interface{}) {
	err := json.Unmarshal(w.Body.Bytes(), v)
	require.NoError(suite.T(), err)
}

// TestIntegrationSuite runs the integration test suite
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}// T
estUserRegistrationAndLoginWorkflow tests the complete user registration and login workflow
// Requirements: 8 (welcome email queuing and processing)
func (suite *IntegrationTestSuite) TestUserRegistrationAndLoginWorkflow() {
	// Test user registration
	suite.Run("user_registration", func() {
		registerReq := handlers.RegisterRequest{
			Email:     "test@example.com",
			Password:  "password123",
			FirstName: "John",
			LastName:  "Doe",
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, "")
		
		assert.Equal(suite.T(), http.StatusCreated, w.Code)
		
		var response handlers.AuthResponse
		suite.parseResponse(w, &response)
		
		assert.NotEmpty(suite.T(), response.Token)
		assert.NotNil(suite.T(), response.User)
		assert.Equal(suite.T(), "test@example.com", response.User.Email)
		assert.Equal(suite.T(), "John", response.User.FirstName)
		assert.Equal(suite.T(), "Doe", response.User.LastName)
		assert.False(suite.T(), response.User.WelcomeEmailSent)
	})
	
	// Test first-time login (should queue welcome email)
	suite.Run("first_time_login_with_welcome_email", func() {
		// First register a user
		registerReq := handlers.RegisterRequest{
			Email:     "firsttime@example.com",
			Password:  "password123",
			FirstName: "Jane",
			LastName:  "Smith",
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, "")
		assert.Equal(suite.T(), http.StatusCreated, w.Code)
		
		// Now login for the first time
		loginReq := handlers.LoginRequest{
			Email:    "firsttime@example.com",
			Password: "password123",
		}
		
		w = suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
		assert.Equal(suite.T(), http.StatusOK, w.Code)
		
		var response handlers.AuthResponse
		suite.parseResponse(w, &response)
		
		assert.NotEmpty(suite.T(), response.Token)
		assert.NotNil(suite.T(), response.User)
		assert.Equal(suite.T(), "firsttime@example.com", response.User.Email)
		assert.False(suite.T(), response.User.WelcomeEmailSent) // Should still be false until email is processed
		
		// Verify welcome email was queued
		ctx := context.Background()
		queuedTasks, err := suite.queueManager.GetQueuedTasks(ctx, "welcome_email")
		assert.NoError(suite.T(), err)
		assert.Greater(suite.T(), len(queuedTasks), 0, "Welcome email should be queued")
	})
	
	// Test subsequent login (should not queue welcome email again)
	suite.Run("subsequent_login_no_welcome_email", func() {
		// Create user and mark welcome email as sent
		user, _ := suite.createTestUser("existing@example.com", "Existing", "User")
		ctx := context.Background()
		err := suite.userService.MarkWelcomeEmailSent(ctx, user.ID)
		require.NoError(suite.T(), err)
		
		// Clear any existing queued tasks
		suite.queueManager.ClearQueues(ctx)
		
		// Login
		loginReq := handlers.LoginRequest{
			Email:    "existing@example.com",
			Password: "password123",
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
		assert.Equal(suite.T(), http.StatusOK, w.Code)
		
		// Verify no welcome email was queued
		queuedTasks, err := suite.queueManager.GetQueuedTasks(ctx, "welcome_email")
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 0, len(queuedTasks), "No welcome email should be queued for existing users")
	})
	
	// Test invalid credentials
	suite.Run("invalid_credentials", func() {
		loginReq := handlers.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "wrongpassword",
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
		assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
		
		var response handlers.ErrorResponse
		suite.parseResponse(w, &response)
		
		assert.Equal(suite.T(), "authentication_failed", response.Error)
		assert.Contains(suite.T(), response.Message, "Invalid email or password")
	})
	
	// Test duplicate registration
	suite.Run("duplicate_registration", func() {
		// First registration
		registerReq := handlers.RegisterRequest{
			Email:     "duplicate@example.com",
			Password:  "password123",
			FirstName: "First",
			LastName:  "User",
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, "")
		assert.Equal(suite.T(), http.StatusCreated, w.Code)
		
		// Attempt duplicate registration
		w = suite.makeRequest("POST", "/api/v1/auth/register", registerReq, "")
		assert.Equal(suite.T(), http.StatusConflict, w.Code)
		
		var response handlers.ErrorResponse
		suite.parseResponse(w, &response)
		
		assert.Equal(suite.T(), "user_exists", response.Error)
		assert.Contains(suite.T(), response.Message, "already exists")
	})
}

// TestAccountCreationAndManagementWorkflow tests the complete account creation and management flow
// Requirements: 1, 2, 4 (account management with multi-currency support)
func (suite *IntegrationTestSuite) TestAccountCreationAndManagementWorkflow() {
	// Create test user
	user, token := suite.createTestUser("account@example.com", "Account", "User")
	
	// Test account creation
	suite.Run("create_account", func() {
		createReq := handlers.CreateAccountRequest{
			Currency: "USD",
		}
		
		w := suite.makeRequest("POST", "/api/v1/accounts", createReq, token)
		assert.Equal(suite.T(), http.StatusCreated, w.Code)
		
		var account models.Account
		suite.parseResponse(w, &account)
		
		assert.Equal(suite.T(), user.ID, account.UserID)
		assert.Equal(suite.T(), "USD", account.Currency)
		assert.True(suite.T(), account.Balance.IsZero())
		assert.NotZero(suite.T(), account.ID)
	})
	
	// Test multiple accounts with different currencies
	suite.Run("create_multiple_currency_accounts", func() {
		currencies := []string{"EUR", "GBP", "JPY"}
		
		for _, currency := range currencies {
			createReq := handlers.CreateAccountRequest{
				Currency: currency,
			}
			
			w := suite.makeRequest("POST", "/api/v1/accounts", createReq, token)
			assert.Equal(suite.T(), http.StatusCreated, w.Code)
			
			var account models.Account
			suite.parseResponse(w, &account)
			
			assert.Equal(suite.T(), currency, account.Currency)
			assert.Equal(suite.T(), user.ID, account.UserID)
		}
	})
	
	// Test duplicate currency prevention
	suite.Run("prevent_duplicate_currency", func() {
		// Create first USD account
		createReq := handlers.CreateAccountRequest{
			Currency: "USD",
		}
		
		w := suite.makeRequest("POST", "/api/v1/accounts", createReq, token)
		assert.Equal(suite.T(), http.StatusCreated, w.Code)
		
		// Attempt to create another USD account
		w = suite.makeRequest("POST", "/api/v1/accounts", createReq, token)
		assert.Equal(suite.T(), http.StatusConflict, w.Code)
		
		var response handlers.ErrorResponse
		suite.parseResponse(w, &response)
		
		assert.Equal(suite.T(), "duplicate_currency", response.Error)
		assert.Contains(suite.T(), response.Message, "already has an account")
	})
	
	// Test invalid currency
	suite.Run("invalid_currency", func() {
		createReq := handlers.CreateAccountRequest{
			Currency: "INVALID",
		}
		
		w := suite.makeRequest("POST", "/api/v1/accounts", createReq, token)
		assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
		
		var response handlers.ErrorResponse
		suite.parseResponse(w, &response)
		
		assert.Equal(suite.T(), "invalid_currency", response.Error)
	})
	
	// Test get user accounts
	suite.Run("get_user_accounts", func() {
		// Create multiple accounts
		currencies := []string{"USD", "EUR", "GBP"}
		for _, currency := range currencies {
			suite.createTestAccount(int32(user.ID), currency)
		}
		
		w := suite.makeRequest("GET", "/api/v1/accounts", nil, token)
		assert.Equal(suite.T(), http.StatusOK, w.Code)
		
		var response struct {
			Accounts []models.Account `json:"accounts"`
			Count    int              `json:"count"`
		}
		suite.parseResponse(w, &response)
		
		assert.Equal(suite.T(), 3, response.Count)
		assert.Len(suite.T(), response.Accounts, 3)
		
		// Verify all accounts belong to the user
		for _, account := range response.Accounts {
			assert.Equal(suite.T(), user.ID, account.UserID)
		}
	})
	
	// Test get specific account
	suite.Run("get_specific_account", func() {
		account := suite.createTestAccount(int32(user.ID), "USD")
		
		w := suite.makeRequest("GET", fmt.Sprintf("/api/v1/accounts/%d", account.ID), nil, token)
		assert.Equal(suite.T(), http.StatusOK, w.Code)
		
		var retrievedAccount models.Account
		suite.parseResponse(w, &retrievedAccount)
		
		assert.Equal(suite.T(), account.ID, retrievedAccount.ID)
		assert.Equal(suite.T(), account.Currency, retrievedAccount.Currency)
		assert.Equal(suite.T(), account.UserID, retrievedAccount.UserID)
	})
	
	// Test access control - user cannot access other user's accounts
	suite.Run("access_control", func() {
		// Create another user and account
		otherUser, _ := suite.createTestUser("other@example.com", "Other", "User")
		otherAccount := suite.createTestAccount(int32(otherUser.ID), "USD")
		
		// Try to access other user's account
		w := suite.makeRequest("GET", fmt.Sprintf("/api/v1/accounts/%d", otherAccount.ID), nil, token)
		assert.Equal(suite.T(), http.StatusForbidden, w.Code)
		
		var response handlers.ErrorResponse
		suite.parseResponse(w, &response)
		
		assert.Equal(suite.T(), "access_denied", response.Error)
	})
	
	// Test account deletion with zero balance
	suite.Run("delete_account_zero_balance", func() {
		account := suite.createTestAccount(int32(user.ID), "CAD")
		
		w := suite.makeRequest("DELETE", fmt.Sprintf("/api/v1/accounts/%d", account.ID), nil, token)
		assert.Equal(suite.T(), http.StatusOK, w.Code)
		
		// Verify account is deleted
		w = suite.makeRequest("GET", fmt.Sprintf("/api/v1/accounts/%d", account.ID), nil, token)
		assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	})
}

// TestMoneyTransferOperationsWithRollback tests money transfer operations with rollback scenarios
// Requirements: 3 (money transfers with database transactions and rollback)
func (suite *IntegrationTestSuite) TestMoneyTransferOperationsWithRollback() {
	// Create test users and accounts
	user1, token1 := suite.createTestUser("user1@example.com", "User", "One")
	user2, _ := suite.createTestUser("user2@example.com", "User", "Two")
	
	account1 := suite.createTestAccount(int32(user1.ID), "USD")
	account2 := suite.createTestAccount(int32(user2.ID), "USD")
	
	// Add initial balance to account1
	ctx := context.Background()
	initialBalance := decimal.NewFromFloat(1000.00)
	_, err := suite.db.GetQueries().AddToBalance(ctx, queries.AddToBalanceParams{
		ID:     int32(account1.ID),
		Amount: pgtype.Numeric{Int: initialBalance.Coefficient(), Exp: initialBalance.Exponent(), Valid: true},
	})
	require.NoError(suite.T(), err)
	
	// Test successful transfer
	suite.Run("successful_transfer", func() {
		transferReq := handlers.TransferRequest{
			FromAccountID: account1.ID,
			ToAccountID:   account2.ID,
			Amount:        decimal.NewFromFloat(100.00),
			Description:   "Test transfer",
		}
		
		w := suite.makeRequest("POST", "/api/v1/transfers", transferReq, token1)
		assert.Equal(suite.T(), http.StatusCreated, w.Code)
		
		var transfer models.Transfer
		suite.parseResponse(w, &transfer)
		
		assert.Equal(suite.T(), account1.ID, transfer.FromAccountID)
		assert.Equal(suite.T(), account2.ID, transfer.ToAccountID)
		assert.Equal(suite.T(), "100", transfer.Amount.String())
		assert.Equal(suite.T(), "Test transfer", transfer.Description)
		assert.Equal(suite.T(), "completed", transfer.Status)
		
		// Verify balances were updated
		fromAccount, err := suite.accountService.GetAccount(ctx, int32(account1.ID), int32(user1.ID))
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "900", fromAccount.Balance.String())
		
		toAccount, err := suite.accountService.GetAccount(ctx, int32(account2.ID), int32(user2.ID))
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "100", toAccount.Balance.String())
	})
	
	// Test insufficient balance (should rollback)
	suite.Run("insufficient_balance_rollback", func() {
		// Get current balances
		fromAccountBefore, err := suite.accountService.GetAccount(ctx, int32(account1.ID), int32(user1.ID))
		require.NoError(suite.T(), err)
		
		toAccountBefore, err := suite.accountService.GetAccount(ctx, int32(account2.ID), int32(user2.ID))
		require.NoError(suite.T(), err)
		
		// Attempt transfer with insufficient balance
		transferReq := handlers.TransferRequest{
			FromAccountID: account1.ID,
			ToAccountID:   account2.ID,
			Amount:        decimal.NewFromFloat(2000.00), // More than available balance
			Description:   "Insufficient balance test",
		}
		
		w := suite.makeRequest("POST", "/api/v1/transfers", transferReq, token1)
		assert.Equal(suite.T(), http.StatusUnprocessableEntity, w.Code)
		
		var response handlers.ErrorResponse
		suite.parseResponse(w, &response)
		
		assert.Equal(suite.T(), "insufficient_balance", response.Error)
		
		// Verify balances remain unchanged (rollback successful)
		fromAccountAfter, err := suite.accountService.GetAccount(ctx, int32(account1.ID), int32(user1.ID))
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), fromAccountBefore.Balance.String(), fromAccountAfter.Balance.String())
		
		toAccountAfter, err := suite.accountService.GetAccount(ctx, int32(account2.ID), int32(user2.ID))
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), toAccountBefore.Balance.String(), toAccountAfter.Balance.String())
	})
	
	// Test currency mismatch (should rollback)
	suite.Run("currency_mismatch_rollback", func() {
		// Create EUR account for user2
		eurAccount := suite.createTestAccount(int32(user2.ID), "EUR")
		
		// Get current balances
		fromAccountBefore, err := suite.accountService.GetAccount(ctx, int32(account1.ID), int32(user1.ID))
		require.NoError(suite.T(), err)
		
		toAccountBefore, err := suite.accountService.GetAccount(ctx, int32(eurAccount.ID), int32(user2.ID))
		require.NoError(suite.T(), err)
		
		// Attempt transfer between different currencies
		transferReq := handlers.TransferRequest{
			FromAccountID: account1.ID,    // USD
			ToAccountID:   eurAccount.ID,  // EUR
			Amount:        decimal.NewFromFloat(50.00),
			Description:   "Currency mismatch test",
		}
		
		w := suite.makeRequest("POST", "/api/v1/transfers", transferReq, token1)
		assert.Equal(suite.T(), http.StatusUnprocessableEntity, w.Code)
		
		var response handlers.ErrorResponse
		suite.parseResponse(w, &response)
		
		assert.Equal(suite.T(), "currency_mismatch", response.Error)
		
		// Verify balances remain unchanged (rollback successful)
		fromAccountAfter, err := suite.accountService.GetAccount(ctx, int32(account1.ID), int32(user1.ID))
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), fromAccountBefore.Balance.String(), fromAccountAfter.Balance.String())
		
		toAccountAfter, err := suite.accountService.GetAccount(ctx, int32(eurAccount.ID), int32(user2.ID))
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), toAccountBefore.Balance.String(), toAccountAfter.Balance.String())
	})
	
	// Test transfer to same account (should be rejected)
	suite.Run("same_account_transfer", func() {
		transferReq := handlers.TransferRequest{
			FromAccountID: account1.ID,
			ToAccountID:   account1.ID, // Same account
			Amount:        decimal.NewFromFloat(50.00),
			Description:   "Same account test",
		}
		
		w := suite.makeRequest("POST", "/api/v1/transfers", transferReq, token1)
		assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
		
		var response handlers.ErrorResponse
		suite.parseResponse(w, &response)
		
		assert.Equal(suite.T(), "same_account", response.Error)
	})
	
	// Test unauthorized transfer (user trying to transfer from account they don't own)
	suite.Run("unauthorized_transfer", func() {
		transferReq := handlers.TransferRequest{
			FromAccountID: account2.ID, // User1 trying to transfer from User2's account
			ToAccountID:   account1.ID,
			Amount:        decimal.NewFromFloat(50.00),
			Description:   "Unauthorized test",
		}
		
		w := suite.makeRequest("POST", "/api/v1/transfers", transferReq, token1)
		assert.Equal(suite.T(), http.StatusForbidden, w.Code)
		
		var response handlers.ErrorResponse
		suite.parseResponse(w, &response)
		
		assert.Equal(suite.T(), "access_denied", response.Error)
	})
	
	// Test transfer history
	suite.Run("transfer_history", func() {
		// Make a few transfers first
		for i := 0; i < 3; i++ {
			transferReq := handlers.TransferRequest{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        decimal.NewFromFloat(10.00),
				Description:   fmt.Sprintf("History test %d", i+1),
			}
			
			w := suite.makeRequest("POST", "/api/v1/transfers", transferReq, token1)
			assert.Equal(suite.T(), http.StatusCreated, w.Code)
		}
		
		// Get transfer history
		w := suite.makeRequest("GET", fmt.Sprintf("/api/v1/transfers?account_id=%d", account1.ID), nil, token1)
		assert.Equal(suite.T(), http.StatusOK, w.Code)
		
		var history []models.Transfer
		suite.parseResponse(w, &history)
		
		assert.GreaterOrEqual(suite.T(), len(history), 3)
		
		// Verify transfers are ordered by creation time (most recent first)
		for i := 1; i < len(history); i++ {
			assert.True(suite.T(), history[i-1].CreatedAt.After(history[i].CreatedAt) || 
				history[i-1].CreatedAt.Equal(history[i].CreatedAt))
		}
	})
}

// TestWelcomeEmailQueueingAndProcessing tests welcome email queuing and processing workflow
// Requirements: 8 (welcome email queuing and processing with Redis and Asyncq)
func (suite *IntegrationTestSuite) TestWelcomeEmailQueueingAndProcessing() {
	ctx := context.Background()
	
	// Test welcome email queuing on first login
	suite.Run("welcome_email_queuing", func() {
		// Register user
		registerReq := handlers.RegisterRequest{
			Email:     "welcome@example.com",
			Password:  "password123",
			FirstName: "Welcome",
			LastName:  "User",
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, "")
		assert.Equal(suite.T(), http.StatusCreated, w.Code)
		
		// Clear any existing queued tasks
		suite.queueManager.ClearQueues(ctx)
		
		// Login for the first time
		loginReq := handlers.LoginRequest{
			Email:    "welcome@example.com",
			Password: "password123",
		}
		
		w = suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
		assert.Equal(suite.T(), http.StatusOK, w.Code)
		
		// Verify welcome email was queued
		queuedTasks, err := suite.queueManager.GetQueuedTasks(ctx, "welcome_email")
		assert.NoError(suite.T(), err)
		assert.Greater(suite.T(), len(queuedTasks), 0, "Welcome email should be queued")
		
		// Verify task payload
		if len(queuedTasks) > 0 {
			payload := queuedTasks[0]
			assert.Contains(suite.T(), payload, "welcome@example.com")
			assert.Contains(suite.T(), payload, "Welcome")
			assert.Contains(suite.T(), payload, "User")
		}
	})
	
	// Test welcome email processing
	suite.Run("welcome_email_processing", func() {
		// Create user
		user, _ := suite.createTestUser("process@example.com", "Process", "User")
		
		// Queue welcome email manually
		payload := queue.WelcomeEmailPayload{
			UserID:    user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		}
		
		err := suite.queueManager.QueueWelcomeEmail(ctx, payload)
		assert.NoError(suite.T(), err)
		
		// Process the queued email
		err = suite.queueManager.ProcessWelcomeEmail(ctx, payload)
		assert.NoError(suite.T(), err)
		
		// Verify user's welcome_email_sent flag was updated
		updatedUser, err := suite.userService.GetUser(ctx, user.ID)
		assert.NoError(suite.T(), err)
		assert.True(suite.T(), updatedUser.WelcomeEmailSent)
	})
	
	// Test email retry logic
	suite.Run("email_retry_logic", func() {
		// Create user
		user, _ := suite.createTestUser("retry@example.com", "Retry", "User")
		
		// Queue welcome email
		payload := queue.WelcomeEmailPayload{
			UserID:    user.ID,
			Email:     "invalid-email-address", // Invalid email to trigger failure
			FirstName: user.FirstName,
			LastName:  user.LastName,
		}
		
		err := suite.queueManager.QueueWelcomeEmail(ctx, payload)
		assert.NoError(suite.T(), err)
		
		// Attempt to process (should fail and be retried)
		err = suite.queueManager.ProcessWelcomeEmail(ctx, payload)
		assert.Error(suite.T(), err, "Processing should fail with invalid email")
		
		// Verify task is still in retry queue
		retryTasks, err := suite.queueManager.GetRetryTasks(ctx, "welcome_email")
		assert.NoError(suite.T(), err)
		assert.Greater(suite.T(), len(retryTasks), 0, "Failed task should be in retry queue")
	})
	
	// Test idempotency - no duplicate emails
	suite.Run("email_idempotency", func() {
		// Create user and mark welcome email as already sent
		user, _ := suite.createTestUser("idempotent@example.com", "Idempotent", "User")
		err := suite.userService.MarkWelcomeEmailSent(ctx, user.ID)
		require.NoError(suite.T(), err)
		
		// Clear queues
		suite.queueManager.ClearQueues(ctx)
		
		// Login (should not queue welcome email)
		loginReq := handlers.LoginRequest{
			Email:    "idempotent@example.com",
			Password: "password123",
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
		assert.Equal(suite.T(), http.StatusOK, w.Code)
		
		// Verify no welcome email was queued
		queuedTasks, err := suite.queueManager.GetQueuedTasks(ctx, "welcome_email")
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 0, len(queuedTasks), "No welcome email should be queued for users who already received it")
	})
	
	// Test queue failure handling
	suite.Run("queue_failure_handling", func() {
		// Simulate Redis connection failure by closing the queue manager
		suite.queueManager.Close()
		
		// Register and login should still work even if email queuing fails
		registerReq := handlers.RegisterRequest{
			Email:     "queuefail@example.com",
			Password:  "password123",
			FirstName: "Queue",
			LastName:  "Fail",
		}
		
		w := suite.makeRequest("POST", "/api/v1/auth/register", registerReq, "")
		assert.Equal(suite.T(), http.StatusCreated, w.Code)
		
		loginReq := handlers.LoginRequest{
			Email:    "queuefail@example.com",
			Password: "password123",
		}
		
		w = suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
		assert.Equal(suite.T(), http.StatusOK, w.Code, "Login should succeed even if email queuing fails")
		
		// Reconnect queue manager for cleanup
		cfg := suite.loadTestConfig()
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		baseQueueManager, _ := queue.NewQueueManager(cfg.Redis, logger)
		suite.queueManager = NewQueueManagerTestExtensions(baseQueueManager)
	})
}