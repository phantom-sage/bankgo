package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/models"
	"github.com/phantom-sage/bankgo/internal/queue"
	"github.com/phantom-sage/bankgo/internal/services"
	"github.com/phantom-sage/bankgo/pkg/auth"
	"github.com/shopspring/decimal"
)

// Request/Response structures for authentication

// RegisterRequest represents the request body for user registration
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

// LoginRequest represents the request body for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents the response for successful authentication
type AuthResponse struct {
	Token string      `json:"token"`
	User  *models.User `json:"user"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Code    int               `json:"code"`
	Details map[string]string `json:"details,omitempty"`
}

// AuthHandlers handles authentication-related HTTP requests
type AuthHandlers struct {
	userService   services.UserService
	tokenManager  *auth.PASETOManager
	queueManager  *queue.QueueManager
}

// NewAuthHandlers creates a new authentication handlers instance
func NewAuthHandlers(userService services.UserService, tokenManager *auth.PASETOManager, queueManager *queue.QueueManager) *AuthHandlers {
	return &AuthHandlers{
		userService:  userService,
		tokenManager: tokenManager,
		queueManager: queueManager,
	}
}

// Register handles user registration
// POST /auth/register
func (h *AuthHandlers) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Code:    http.StatusBadRequest,
			Details: map[string]string{"validation": err.Error()},
		})
		return
	}

	// Create user
	user, err := h.userService.CreateUser(c.Request.Context(), req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		// Check for specific error types
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "user_exists",
				Message: "User with this email already exists",
				Code:    http.StatusConflict,
			})
			return
		}

		if strings.Contains(err.Error(), "validation failed") {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create user",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Generate token
	token, err := h.tokenManager.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "token_error",
			Message: "Failed to generate authentication token",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token: token,
		User:  user,
	})
}

// Login handles user login with welcome email queuing for first-time users
// POST /auth/login
func (h *AuthHandlers) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Code:    http.StatusBadRequest,
			Details: map[string]string{"validation": err.Error()},
		})
		return
	}

	// Authenticate user
	user, err := h.userService.AuthenticateUser(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "authentication_failed",
			Message: "Invalid email or password",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	// Check if this is the first login (welcome email not sent)
	if !user.WelcomeEmailSent {
		// Queue welcome email task
		payload := queue.WelcomeEmailPayload{
			UserID:    user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		}

		if err := h.queueManager.QueueWelcomeEmail(c.Request.Context(), payload); err != nil {
			// Log error but don't fail the login
			// In a production system, you might want to use a proper logger
			// For now, we'll continue with the login process
		}
	}

	// Generate token
	token, err := h.tokenManager.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "token_error",
			Message: "Failed to generate authentication token",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User:  user,
	})
}

// Logout handles user logout
// POST /auth/logout
func (h *AuthHandlers) Logout(c *gin.Context) {
	// For PASETO tokens, logout is typically handled client-side by discarding the token
	// However, we can implement server-side token blacklisting if needed
	// For now, we'll just return a success response
	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully logged out",
	})
}

// AuthMiddleware validates PASETO tokens and sets user context
func (h *AuthHandlers) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "missing_token",
				Message: "Authorization header is required",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Check for Bearer token format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_token_format",
				Message: "Authorization header must be in format 'Bearer <token>'",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		token := tokenParts[1]

		// Validate token
		claims, err := h.tokenManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid or expired token",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)

		c.Next()
	}
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandlers) GetCurrentUser(c *gin.Context) (*models.User, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, models.ErrInvalidUserID
	}

	id, ok := userID.(int)
	if !ok {
		return nil, models.ErrInvalidUserID
	}

	return h.userService.GetUser(c.Request.Context(), id)
}

// Helper function to get user ID from context
func GetUserIDFromContext(c *gin.Context) (int, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, models.ErrInvalidUserID
	}

	id, ok := userID.(int)
	if !ok {
		return 0, models.ErrInvalidUserID
	}

	return id, nil
}

// Helper function to parse ID from URL parameter
func ParseIDParam(c *gin.Context, paramName string) (int, error) {
	idStr := c.Param(paramName)
	if idStr == "" {
		return 0, models.ErrInvalidUserID
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return 0, models.ErrInvalidUserID
	}

	return id, nil
}

// Request/Response structures for account management

// CreateAccountRequest represents the request body for creating an account
type CreateAccountRequest struct {
	Currency string `json:"currency" binding:"required,len=3"`
}

// UpdateAccountRequest represents the request body for updating an account
type UpdateAccountRequest struct {
	// Currently no updatable fields for accounts
	// Balance updates must go through transfer operations
	// Currency updates are not allowed for data integrity
}

// AccountHandlers handles account-related HTTP requests
type AccountHandlers struct {
	accountService services.AccountService
}

// NewAccountHandlers creates a new account handlers instance
func NewAccountHandlers(accountService services.AccountService) *AccountHandlers {
	return &AccountHandlers{
		accountService: accountService,
	}
}

// GetUserAccounts handles listing user accounts
// GET /accounts
func (h *AccountHandlers) GetUserAccounts(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	// Get user accounts
	accounts, err := h.accountService.GetUserAccounts(c.Request.Context(), int32(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve accounts",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": accounts,
		"count":    len(accounts),
	})
}

// CreateAccount handles account creation with currency validation
// POST /accounts
func (h *AccountHandlers) CreateAccount(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Code:    http.StatusBadRequest,
			Details: map[string]string{"validation": err.Error()},
		})
		return
	}

	// Create account
	account, err := h.accountService.CreateAccount(c.Request.Context(), int32(userID), req.Currency)
	if err != nil {
		// Check for specific error types
		if strings.Contains(err.Error(), "invalid currency") {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_currency",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		if strings.Contains(err.Error(), "already has an account") {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "duplicate_currency",
				Message: err.Error(),
				Code:    http.StatusConflict,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create account",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, account)
}

// GetAccount handles retrieving a specific account with ownership validation
// GET /accounts/:id
func (h *AccountHandlers) GetAccount(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	// Parse account ID from URL parameter
	accountID, err := ParseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_account_id",
			Message: "Invalid account ID",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Get account with ownership validation
	account, err := h.accountService.GetAccount(c.Request.Context(), int32(accountID), int32(userID))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "account_not_found",
				Message: "Account not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		if strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "access_denied",
				Message: "You can only access your own accounts",
				Code:    http.StatusForbidden,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve account",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, account)
}

// UpdateAccount handles account updates with proper authorization
// PUT /accounts/:id
func (h *AccountHandlers) UpdateAccount(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	// Parse account ID from URL parameter
	accountID, err := ParseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_account_id",
			Message: "Invalid account ID",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Code:    http.StatusBadRequest,
			Details: map[string]string{"validation": err.Error()},
		})
		return
	}

	// Convert to service request
	serviceReq := services.UpdateAccountRequest{
		// Currently no fields to update
	}

	// Update account
	account, err := h.accountService.UpdateAccount(c.Request.Context(), int32(accountID), int32(userID), serviceReq)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "account_not_found",
				Message: "Account not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		if strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "access_denied",
				Message: "You can only update your own accounts",
				Code:    http.StatusForbidden,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update account",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, account)
}

// DeleteAccount handles account deletion with proper authorization
// DELETE /accounts/:id
func (h *AccountHandlers) DeleteAccount(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	// Parse account ID from URL parameter
	accountID, err := ParseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_account_id",
			Message: "Invalid account ID",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Delete account
	err = h.accountService.DeleteAccount(c.Request.Context(), int32(accountID), int32(userID))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "account_not_found",
				Message: "Account not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		if strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "access_denied",
				Message: "You can only delete your own accounts",
				Code:    http.StatusForbidden,
			})
			return
		}

		if strings.Contains(err.Error(), "non-zero balance") {
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
				Error:   "non_zero_balance",
				Message: err.Error(),
				Code:    http.StatusUnprocessableEntity,
			})
			return
		}

		if strings.Contains(err.Error(), "transaction history") {
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
				Error:   "has_transactions",
				Message: err.Error(),
				Code:    http.StatusUnprocessableEntity,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete account",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Account deleted successfully",
	})
}

// Request/Response structures for transfer management

// TransferRequest represents the request body for creating a transfer
type TransferRequest struct {
	FromAccountID int             `json:"from_account_id" binding:"required"`
	ToAccountID   int             `json:"to_account_id" binding:"required"`
	Amount        decimal.Decimal `json:"amount" binding:"required"`
	Description   string          `json:"description"`
}

// TransferHistoryRequest represents the request for transfer history
type TransferHistoryRequest struct {
	AccountID int `json:"account_id" binding:"required"`
	Limit     int `json:"limit"`
	Offset    int `json:"offset"`
}

// TransferHandlers handles transfer-related HTTP requests
type TransferHandlers struct {
	transferService services.TransferService
	accountService  services.AccountService
}

// NewTransferHandlers creates a new transfer handlers instance
func NewTransferHandlers(transferService services.TransferService, accountService services.AccountService) *TransferHandlers {
	return &TransferHandlers{
		transferService: transferService,
		accountService:  accountService,
	}
}

// CreateTransfer handles money transfer between accounts with transaction processing
// POST /transfers
func (h *TransferHandlers) CreateTransfer(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data",
			Code:    http.StatusBadRequest,
			Details: map[string]string{"validation": err.Error()},
		})
		return
	}

	// Validate that amount is positive
	if req.Amount.IsNegative() || req.Amount.IsZero() {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_amount",
			Message: "Transfer amount must be positive",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Verify that the user owns the source account
	_, err = h.accountService.GetAccount(c.Request.Context(), int32(req.FromAccountID), int32(userID))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "account_not_found",
				Message: "Source account not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		if strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "access_denied",
				Message: "You can only transfer from your own accounts",
				Code:    http.StatusForbidden,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to verify source account",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Create transfer service request
	transferReq := services.TransferMoneyRequest{
		FromAccountID: int32(req.FromAccountID),
		ToAccountID:   int32(req.ToAccountID),
		Amount:        req.Amount,
		Description:   req.Description,
	}

	// Execute transfer
	transfer, err := h.transferService.TransferMoney(c.Request.Context(), transferReq)
	if err != nil {
		// Check for specific error types
		if strings.Contains(err.Error(), "insufficient balance") {
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
				Error:   "insufficient_balance",
				Message: err.Error(),
				Code:    http.StatusUnprocessableEntity,
			})
			return
		}

		if strings.Contains(err.Error(), "currency mismatch") {
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
				Error:   "currency_mismatch",
				Message: err.Error(),
				Code:    http.StatusUnprocessableEntity,
			})
			return
		}

		if strings.Contains(err.Error(), "same account") {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "same_account",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		if strings.Contains(err.Error(), "validation failed") {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "transfer_failed",
			Message: "Failed to process transfer",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, transfer)
}

// GetTransferHistory handles retrieving transfer history for user's accounts
// GET /transfers
func (h *TransferHandlers) GetTransferHistory(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	accountIDStr := c.Query("account_id")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Maximum limit
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// If account_id is specified, get history for that specific account
	if accountIDStr != "" {
		accountID, err := strconv.Atoi(accountIDStr)
		if err != nil || accountID <= 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_account_id",
				Message: "Invalid account ID parameter",
				Code:    http.StatusBadRequest,
			})
			return
		}

		// Verify that the user owns the account
		_, err = h.accountService.GetAccount(c.Request.Context(), int32(accountID), int32(userID))
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				c.JSON(http.StatusNotFound, ErrorResponse{
					Error:   "account_not_found",
					Message: "Account not found",
					Code:    http.StatusNotFound,
				})
				return
			}

			if strings.Contains(err.Error(), "access denied") {
				c.JSON(http.StatusForbidden, ErrorResponse{
					Error:   "access_denied",
					Message: "You can only view transfer history for your own accounts",
					Code:    http.StatusForbidden,
				})
				return
			}

			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to verify account ownership",
				Code:    http.StatusInternalServerError,
			})
			return
		}

		// Get transfer history for the specific account
		historyReq := services.GetTransferHistoryRequest{
			AccountID: int32(accountID),
			Limit:     int32(limit),
			Offset:    int32(offset),
		}

		history, err := h.transferService.GetTransferHistory(c.Request.Context(), historyReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to retrieve transfer history",
				Code:    http.StatusInternalServerError,
			})
			return
		}

		c.JSON(http.StatusOK, history)
		return
	}

	// If no specific account_id, get all transfers for the user
	history, err := h.transferService.GetTransfersByUser(c.Request.Context(), int32(userID), int32(limit), int32(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve transfer history",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetTransfer handles retrieving transfer details
// GET /transfers/:id
func (h *TransferHandlers) GetTransfer(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	// Parse transfer ID from URL parameter
	transferID, err := ParseIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_transfer_id",
			Message: "Invalid transfer ID",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Get transfer details
	transfer, err := h.transferService.GetTransfer(c.Request.Context(), int32(transferID))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "transfer_not_found",
				Message: "Transfer not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve transfer",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Verify that the user has access to this transfer (owns either source or destination account)
	hasAccess := false

	// Check if user owns the source account
	if _, err := h.accountService.GetAccount(c.Request.Context(), int32(transfer.FromAccountID), int32(userID)); err == nil {
		hasAccess = true
	}

	// Check if user owns the destination account (if not already has access)
	if !hasAccess {
		if _, err := h.accountService.GetAccount(c.Request.Context(), int32(transfer.ToAccountID), int32(userID)); err == nil {
			hasAccess = true
		}
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "access_denied",
			Message: "You can only view transfers involving your own accounts",
			Code:    http.StatusForbidden,
		})
		return
	}

	c.JSON(http.StatusOK, transfer)
}