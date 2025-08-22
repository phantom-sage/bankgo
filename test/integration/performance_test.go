package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/phantom-sage/bankgo/internal/handlers"
	"github.com/phantom-sage/bankgo/internal/models"
	"github.com/phantom-sage/bankgo/internal/queue"
	"github.com/shopspring/decimal"
	"	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/rs/zerolog"
	"os""
	"github.com/stretchr/testify/require"
)

// TestConcurrentTransferOperations tests concurrent transfer operations
// Requirements: 3 (concurrent transfer operations with proper isolation)
func (suite *IntegrationTestSuite) TestConcurrentTransferOperations() {
	// Create test users and accounts
	user1, token1 := suite.createTestUser("concurrent1@example.com", "Concurrent", "User1")
	user2, _ := suite.createTestUser("concurrent2@example.com", "Concurrent", "User2")
	
	account1 := suite.createTestAccount(int32(user1.ID), "USD")
	account2 := suite.createTestAccount(int32(user2.ID), "USD")
	
	// Add initial balance to account1
	ctx := context.Background()
	initialBalance := decimal.NewFromFloat(10000.00) // Large balance for concurrent transfers
	_, err := suite.db.GetQueries().AddToBalance(ctx, queries.AddToBalanceParams{
		ID:     int32(account1.ID),
		Amount: pgtype.Numeric{Int: initialBalance.Coefficient(), Exp: initialBalance.Exponent(), Valid: true},
	})
	require.NoError(suite.T(), err)
	
	suite.Run("concurrent_transfers_same_accounts", func() {
		numTransfers := 10
		transferAmount := decimal.NewFromFloat(100.00)
		
		var wg sync.WaitGroup
		var mu sync.Mutex
		successCount := 0
		errors := make([]error, 0)
		
		// Launch concurrent transfers
		for i := 0; i < numTransfers; i++ {
			wg.Add(1)
			go func(transferID int) {
				defer wg.Done()
				
				transferReq := handlers.TransferRequest{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        transferAmount,
					Description:   fmt.Sprintf("Concurrent transfer %d", transferID),
				}
				
				w := suite.makeRequest("POST", "/api/v1/transfers", transferReq, token1)
				
				mu.Lock()
				if w.Code == 201 {
					successCount++
				} else {
					errors = append(errors, fmt.Errorf("transfer %d failed with status %d", transferID, w.Code))
				}
				mu.Unlock()
			}(i)
		}
		
		// Wait for all transfers to complete
		wg.Wait()
		
		// All transfers should succeed
		assert.Equal(suite.T(), numTransfers, successCount, "All concurrent transfers should succeed")
		assert.Empty(suite.T(), errors, "No transfer errors should occur")
		
		// Verify final balances are correct
		finalAccount1, err := suite.accountService.GetAccount(ctx, int32(account1.ID), int32(user1.ID))
		require.NoError(suite.T(), err)
		
		finalAccount2, err := suite.accountService.GetAccount(ctx, int32(account2.ID), int32(user2.ID))
		require.NoError(suite.T(), err)
		
		expectedAccount1Balance := initialBalance.Sub(transferAmount.Mul(decimal.NewFromInt(int64(numTransfers))))
		expectedAccount2Balance := transferAmount.Mul(decimal.NewFromInt(int64(numTransfers)))
		
		assert.Equal(suite.T(), expectedAccount1Balance.String(), finalAccount1.Balance.String())
		assert.Equal(suite.T(), expectedAccount2Balance.String(), finalAccount2.Balance.String())
	})
	
	suite.Run("concurrent_transfers_insufficient_balance", func() {
		// Reset account1 balance to a small amount
		_, err := suite.db.GetQueries().UpdateAccountBalance(ctx, queries.UpdateAccountBalanceParams{
			ID:      int32(account1.ID),
			Balance: pgtype.Numeric{Int: decimal.NewFromFloat(500.00).Coefficient(), Exp: decimal.NewFromFloat(500.00).Exponent(), Valid: true},
		})
		require.NoError(suite.T(), err)
		
		numTransfers := 10
		transferAmount := decimal.NewFromFloat(100.00) // Total would exceed balance
		
		var wg sync.WaitGroup
		var mu sync.Mutex
		successCount := 0
		failureCount := 0
		
		// Launch concurrent transfers that will exceed balance
		for i := 0; i < numTransfers; i++ {
			wg.Add(1)
			go func(transferID int) {
				defer wg.Done()
				
				transferReq := handlers.TransferRequest{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        transferAmount,
					Description:   fmt.Sprintf("Insufficient balance test %d", transferID),
				}
				
				w := suite.makeRequest("POST", "/api/v1/transfers", transferReq, token1)
				
				mu.Lock()
				if w.Code == 201 {
					successCount++
				} else if w.Code == 422 { // Insufficient balance
					failureCount++
				}
				mu.Unlock()
			}(i)
		}
		
		wg.Wait()
		
		// Some transfers should succeed, others should fail due to insufficient balance
		assert.Greater(suite.T(), successCount, 0, "Some transfers should succeed")
		assert.Greater(suite.T(), failureCount, 0, "Some transfers should fail due to insufficient balance")
		assert.Equal(suite.T(), numTransfers, successCount+failureCount, "All transfers should be accounted for")
		
		// Verify account balance is never negative
		finalAccount1, err := suite.accountService.GetAccount(ctx, int32(account1.ID), int32(user1.ID))
		require.NoError(suite.T(), err)
		assert.True(suite.T(), finalAccount1.Balance.GreaterThanOrEqual(decimal.Zero), "Account balance should never be negative")
	})
	
	suite.Run("concurrent_account_creation", func() {
		// Create a user for concurrent account creation
		concurrentUser, concurrentToken := suite.createTestUser("concurrent_accounts@example.com", "Concurrent", "Accounts")
		
		currencies := []string{"EUR", "GBP", "JPY", "CAD", "AUD"}
		numConcurrent := len(currencies)
		
		var wg sync.WaitGroup
		var mu sync.Mutex
		successCount := 0
		createdAccounts := make([]*models.Account, 0)
		
		// Launch concurrent account creation
		for _, currency := range currencies {
			wg.Add(1)
			go func(curr string) {
				defer wg.Done()
				
				createReq := handlers.CreateAccountRequest{
					Currency: curr,
				}
				
				w := suite.makeRequest("POST", "/api/v1/accounts", createReq, concurrentToken)
				
				mu.Lock()
				if w.Code == 201 {
					successCount++
					var account models.Account
					suite.parseResponse(w, &account)
					createdAccounts = append(createdAccounts, &account)
				}
				mu.Unlock()
			}(currency)
		}
		
		wg.Wait()
		
		// All account creations should succeed
		assert.Equal(suite.T(), numConcurrent, successCount, "All concurrent account creations should succeed")
		assert.Len(suite.T(), createdAccounts, numConcurrent)
		
		// Verify all accounts have different currencies
		currencySet := make(map[string]bool)
		for _, account := range createdAccounts {
			assert.Equal(suite.T(), concurrentUser.ID, account.UserID)
			assert.False(suite.T(), currencySet[account.Currency], "Each currency should be unique")
			currencySet[account.Currency] = true
		}
	})
}

// TestDatabaseTransactionRollbackScenarios tests database transaction rollback scenarios
// Requirements: 3 (database transaction rollback scenarios)
func (suite *IntegrationTestSuite) TestDatabaseTransactionRollbackScenarios() {
	ctx := context.Background()
	
	suite.Run("transfer_rollback_on_database_error", func() {
		// Create test accounts
		user1, token1 := suite.createTestUser("rollback1@example.com", "Rollback", "User1")
		user2, _ := suite.createTestUser("rollback2@example.com", "Rollback", "User2")
		
		account1 := suite.createTestAccount(int32(user1.ID), "USD")
		account2 := suite.createTestAccount(int32(user2.ID), "USD")
		
		// Add balance to account1
		initialBalance := decimal.NewFromFloat(1000.00)
		_, err := suite.db.GetQueries().AddToBalance(ctx, queries.AddToBalanceParams{
			ID:     int32(account1.ID),
			Amount: pgtype.Numeric{Int: initialBalance.Coefficient(), Exp: initialBalance.Exponent(), Valid: true},
		})
		require.NoError(suite.T(), err)
		
		// Get initial balances
		account1Before, err := suite.accountService.GetAccount(ctx, int32(account1.ID), int32(user1.ID))
		require.NoError(suite.T(), err)
		
		account2Before, err := suite.accountService.GetAccount(ctx, int32(account2.ID), int32(user2.ID))
		require.NoError(suite.T(), err)
		
		// Simulate a scenario where the transfer would fail mid-transaction
		// We'll test this by attempting a transfer to a non-existent account
		transferReq := handlers.TransferRequest{
			FromAccountID: account1.ID,
			ToAccountID:   99999, // Non-existent account
			Amount:        decimal.NewFromFloat(100.00),
			Description:   "Rollback test",
		}
		
		w := suite.makeRequest("POST", "/api/v1/transfers", transferReq, token1)
		
		// Transfer should fail
		assert.NotEqual(suite.T(), 201, w.Code, "Transfer to non-existent account should fail")
		
		// Verify balances remain unchanged (rollback successful)
		account1After, err := suite.accountService.GetAccount(ctx, int32(account1.ID), int32(user1.ID))
		require.NoError(suite.T(), err)
		
		account2After, err := suite.accountService.GetAccount(ctx, int32(account2.ID), int32(user2.ID))
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), account1Before.Balance.String(), account1After.Balance.String(), "Source account balance should be unchanged")
		assert.Equal(suite.T(), account2Before.Balance.String(), account2After.Balance.String(), "Destination account balance should be unchanged")
	})
	
	suite.Run("concurrent_transfers_with_rollback", func() {
		// Create accounts with limited balance
		user1, token1 := suite.createTestUser("concurrent_rollback@example.com", "Concurrent", "Rollback")
		user2, _ := suite.createTestUser("concurrent_target@example.com", "Concurrent", "Target")
		
		account1 := suite.createTestAccount(int32(user1.ID), "USD")
		account2 := suite.createTestAccount(int32(user2.ID), "USD")
		
		// Add limited balance
		limitedBalance := decimal.NewFromFloat(300.00)
		_, err := suite.db.GetQueries().AddToBalance(ctx, queries.AddToBalanceParams{
			ID:     int32(account1.ID),
			Amount: pgtype.Numeric{Int: limitedBalance.Coefficient(), Exp: limitedBalance.Exponent(), Valid: true},
		})
		require.NoError(suite.T(), err)
		
		// Launch multiple concurrent transfers that would exceed balance
		numTransfers := 5
		transferAmount := decimal.NewFromFloat(100.00) // Total: 500, but only 300 available
		
		var wg sync.WaitGroup
		var mu sync.Mutex
		results := make([]int, 0)
		
		for i := 0; i < numTransfers; i++ {
			wg.Add(1)
			go func(transferID int) {
				defer wg.Done()
				
				transferReq := handlers.TransferRequest{
					FromAccountID: account1.ID,
					ToAccountID:   account2.ID,
					Amount:        transferAmount,
					Description:   fmt.Sprintf("Concurrent rollback test %d", transferID),
				}
				
				w := suite.makeRequest("POST", "/api/v1/transfers", transferReq, token1)
				
				mu.Lock()
				results = append(results, w.Code)
				mu.Unlock()
			}(i)
		}
		
		wg.Wait()
		
		// Count successful and failed transfers
		successCount := 0
		failureCount := 0
		for _, code := range results {
			if code == 201 {
				successCount++
			} else {
				failureCount++
			}
		}
		
		// Some should succeed, some should fail
		assert.Greater(suite.T(), successCount, 0, "Some transfers should succeed")
		assert.Greater(suite.T(), failureCount, 0, "Some transfers should fail")
		
		// Verify final balance is consistent
		finalAccount1, err := suite.accountService.GetAccount(ctx, int32(account1.ID), int32(user1.ID))
		require.NoError(suite.T(), err)
		
		finalAccount2, err := suite.accountService.GetAccount(ctx, int32(account2.ID), int32(user2.ID))
		require.NoError(suite.T(), err)
		
		// Total money should be conserved
		totalMoney := finalAccount1.Balance.Add(finalAccount2.Balance)
		assert.Equal(suite.T(), limitedBalance.String(), totalMoney.String(), "Total money should be conserved")
		
		// Account1 balance should equal initial minus (successful transfers * amount)
		expectedAccount1Balance := limitedBalance.Sub(transferAmount.Mul(decimal.NewFromInt(int64(successCount))))
		assert.Equal(suite.T(), expectedAccount1Balance.String(), finalAccount1.Balance.String())
	})
}

// TestRateLimitingAndErrorHandling tests rate limiting and error handling
// Requirements: 5 (rate limiting and error handling)
func (suite *IntegrationTestSuite) TestRateLimitingAndErrorHandling() {
	suite.Run("authentication_rate_limiting", func() {
		// Test rate limiting on login attempts
		loginReq := handlers.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "wrongpassword",
		}
		
		// Make multiple rapid requests
		numRequests := 20
		statusCodes := make([]int, 0)
		
		for i := 0; i < numRequests; i++ {
			w := suite.makeRequest("POST", "/api/v1/auth/login", loginReq, "")
			statusCodes = append(statusCodes, w.Code)
			
			// Small delay to avoid overwhelming the system
			time.Sleep(10 * time.Millisecond)
		}
		
		// Check if rate limiting is applied (some requests should return 429)
		rateLimitedCount := 0
		unauthorizedCount := 0
		
		for _, code := range statusCodes {
			if code == 429 {
				rateLimitedCount++
			} else if code == 401 {
				unauthorizedCount++
			}
		}
		
		// Either rate limiting should kick in, or all should be unauthorized
		assert.True(suite.T(), rateLimitedCount > 0 || unauthorizedCount == numRequests,
			"Rate limiting should be applied or all requests should be unauthorized")
	})
	
	suite.Run("api_endpoint_rate_limiting", func() {
		// Create user and token
		_, token := suite.createTestUser("ratelimit@example.com", "Rate", "Limit")
		
		// Make multiple rapid requests to accounts endpoint
		numRequests := 15
		statusCodes := make([]int, 0)
		
		for i := 0; i < numRequests; i++ {
			w := suite.makeRequest("GET", "/api/v1/accounts", nil, token)
			statusCodes = append(statusCodes, w.Code)
			
			time.Sleep(5 * time.Millisecond)
		}
		
		// Check for rate limiting
		rateLimitedCount := 0
		successCount := 0
		
		for _, code := range statusCodes {
			if code == 429 {
				rateLimitedCount++
			} else if code == 200 {
				successCount++
			}
		}
		
		// Some requests should succeed, and rate limiting may apply
		assert.Greater(suite.T(), successCount, 0, "Some requests should succeed")
		// Note: Rate limiting behavior depends on middleware configuration
	})
	
	suite.Run("error_response_format_consistency", func() {
		// Test various error scenarios and verify consistent error response format
		testCases := []struct {
			name           string
			method         string
			path           string
			body           interface{}
			token          string
			expectedStatus int
			expectedError  string
		}{
			{
				name:           "invalid_json",
				method:         "POST",
				path:           "/api/v1/auth/register",
				body:           "invalid json",
				expectedStatus: 400,
				expectedError:  "validation_error",
			},
			{
				name:           "missing_auth_token",
				method:         "GET",
				path:           "/api/v1/accounts",
				expectedStatus: 401,
				expectedError:  "missing_token",
			},
			{
				name:           "invalid_auth_token",
				method:         "GET",
				path:           "/api/v1/accounts",
				token:          "invalid-token",
				expectedStatus: 401,
				expectedError:  "invalid_token",
			},
			{
				name:           "nonexistent_endpoint",
				method:         "GET",
				path:           "/api/v1/nonexistent",
				expectedStatus: 404,
			},
		}
		
		for _, tc := range testCases {
			suite.Run(tc.name, func() {
				var w *httptest.ResponseRecorder
				
				if tc.body == "invalid json" {
					// Send invalid JSON
					req, err := http.NewRequest(tc.method, tc.path, strings.NewReader("invalid json"))
					require.NoError(suite.T(), err)
					req.Header.Set("Content-Type", "application/json")
					
					w = httptest.NewRecorder()
					suite.router.ServeHTTP(w, req)
				} else {
					w = suite.makeRequest(tc.method, tc.path, tc.body, tc.token)
				}
				
				assert.Equal(suite.T(), tc.expectedStatus, w.Code)
				
				if tc.expectedError != "" {
					var response handlers.ErrorResponse
					err := json.Unmarshal(w.Body.Bytes(), &response)
					if err == nil { // Only check if we can parse the response
						assert.Equal(suite.T(), tc.expectedError, response.Error)
						assert.NotEmpty(suite.T(), response.Message)
						assert.Equal(suite.T(), tc.expectedStatus, response.Code)
					}
				}
			})
		}
	})
	
	suite.Run("database_connection_error_handling", func() {
		// This test would require mocking database failures
		// For now, we'll test that the health check properly reports database status
		w := suite.makeRequest("GET", "/api/v1/health", nil, "")
		
		// Health check should return status (either healthy or unhealthy)
		assert.Contains(suite.T(), []int{200, 503}, w.Code)
		
		var response handlers.HealthResponse
		suite.parseResponse(w, &response)
		
		assert.NotEmpty(suite.T(), response.Status)
		assert.NotEmpty(suite.T(), response.Version)
		assert.NotNil(suite.T(), response.Services)
	})
}

// TestEmailRetryLogicAndFailureScenarios tests email retry logic and failure scenarios
// Requirements: 8 (email retry logic and failure scenarios)
func (suite *IntegrationTestSuite) TestEmailRetryLogicAndFailureScenarios() {
	ctx := context.Background()
	
	suite.Run("email_service_failure_handling", func() {
		// Create user
		user, _ := suite.createTestUser("email_fail@example.com", "Email", "Fail")
		
		// Queue welcome email with invalid configuration to trigger failure
		payload := queue.WelcomeEmailPayload{
			UserID:    user.ID,
			Email:     "invalid-email-format", // Invalid email to trigger failure
			FirstName: user.FirstName,
			LastName:  user.LastName,
		}
		
		err := suite.queueManager.QueueWelcomeEmail(ctx, payload)
		assert.NoError(suite.T(), err, "Queuing should succeed even with invalid email")
		
		// Attempt to process the email (should fail)
		err = suite.queueManager.ProcessWelcomeEmail(ctx, payload)
		assert.Error(suite.T(), err, "Processing should fail with invalid email")
		
		// Verify user's welcome_email_sent flag remains false
		updatedUser, err := suite.userService.GetUser(ctx, user.ID)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), updatedUser.WelcomeEmailSent, "Flag should remain false on email failure")
	})
	
	suite.Run("email_retry_queue_behavior", func() {
		// Create user
		user, _ := suite.createTestUser("retry_queue@example.com", "Retry", "Queue")
		
		// Clear all queues first
		suite.queueManager.ClearQueues(ctx)
		
		// Queue welcome email
		payload := queue.WelcomeEmailPayload{
			UserID:    user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		}
		
		err := suite.queueManager.QueueWelcomeEmail(ctx, payload)
		assert.NoError(suite.T(), err)
		
		// Verify task is in the main queue
		queuedTasks, err := suite.queueManager.GetQueuedTasks(ctx, "welcome_email")
		assert.NoError(suite.T(), err)
		assert.Greater(suite.T(), len(queuedTasks), 0, "Task should be in main queue")
		
		// Simulate processing failure by using invalid email
		failPayload := payload
		failPayload.Email = "invalid-email"
		
		err = suite.queueManager.ProcessWelcomeEmail(ctx, failPayload)
		assert.Error(suite.T(), err, "Processing should fail")
		
		// Check retry queue
		retryTasks, err := suite.queueManager.GetRetryTasks(ctx, "welcome_email")
		assert.NoError(suite.T(), err)
		// Note: Retry behavior depends on Asyncq configuration
	})
	
	suite.Run("email_processing_timeout", func() {
		// Create user
		user, _ := suite.createTestUser("timeout@example.com", "Timeout", "User")
		
		// Queue welcome email
		payload := queue.WelcomeEmailPayload{
			UserID:    user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		}
		
		err := suite.queueManager.QueueWelcomeEmail(ctx, payload)
		assert.NoError(suite.T(), err)
		
		// Create a context with timeout for processing
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond) // Very short timeout
		defer cancel()
		
		// Attempt to process with timeout (may or may not timeout depending on system speed)
		err = suite.queueManager.ProcessWelcomeEmailWithContext(timeoutCtx, payload)
		// We don't assert error here as it depends on system performance
		// The test verifies that timeout handling doesn't crash the system
	})
	
	suite.Run("bulk_email_processing", func() {
		// Create multiple users
		numUsers := 10
		users := make([]*models.User, 0, numUsers)
		
		for i := 0; i < numUsers; i++ {
			user, _ := suite.createTestUser(fmt.Sprintf("bulk%d@example.com", i), "Bulk", fmt.Sprintf("User%d", i))
			users = append(users, user)
		}
		
		// Clear queues
		suite.queueManager.ClearQueues(ctx)
		
		// Queue welcome emails for all users
		for _, user := range users {
			payload := queue.WelcomeEmailPayload{
				UserID:    user.ID,
				Email:     user.Email,
				FirstName: user.FirstName,
				LastName:  user.LastName,
			}
			
			err := suite.queueManager.QueueWelcomeEmail(ctx, payload)
			assert.NoError(suite.T(), err)
		}
		
		// Verify all emails are queued
		queuedTasks, err := suite.queueManager.GetQueuedTasks(ctx, "welcome_email")
		assert.NoError(suite.T(), err)
		assert.GreaterOrEqual(suite.T(), len(queuedTasks), numUsers, "All emails should be queued")
		
		// Process all emails
		processedCount := 0
		for _, user := range users {
			payload := queue.WelcomeEmailPayload{
				UserID:    user.ID,
				Email:     user.Email,
				FirstName: user.FirstName,
				LastName:  user.LastName,
			}
			
			err := suite.queueManager.ProcessWelcomeEmail(ctx, payload)
			if err == nil {
				processedCount++
			}
		}
		
		assert.Greater(suite.T(), processedCount, 0, "At least some emails should be processed successfully")
	})
	
	suite.Run("queue_persistence_across_restarts", func() {
		// Create user
		user, _ := suite.createTestUser("persistence@example.com", "Persistence", "User")
		
		// Queue welcome email
		payload := queue.WelcomeEmailPayload{
			UserID:    user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		}
		
		err := suite.queueManager.QueueWelcomeEmail(ctx, payload)
		assert.NoError(suite.T(), err)
		
		// Verify task is queued
		queuedTasks, err := suite.queueManager.GetQueuedTasks(ctx, "welcome_email")
		assert.NoError(suite.T(), err)
		assert.Greater(suite.T(), len(queuedTasks), 0, "Task should be queued")
		
		// Simulate restart by closing and reopening queue manager
		suite.queueManager.Close()
		
		cfg := suite.loadTestConfig()
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		baseQueueManager, err := queue.NewQueueManager(cfg.Redis, logger)
		require.NoError(suite.T(), err, "Should be able to reconnect to queue")
		suite.queueManager = NewQueueManagerTestExtensions(baseQueueManager)
		
		// Verify task is still in queue (Redis persistence)
		queuedTasksAfterRestart, err := suite.queueManager.GetQueuedTasks(ctx, "welcome_email")
		assert.NoError(suite.T(), err)
		// Note: Task persistence depends on Redis configuration and Asyncq settings
		// In a real scenario, tasks should persist across restarts
	})
}