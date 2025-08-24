package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/gin-gonic/gin"
)

// TransactionHandler handles transaction management HTTP requests
type TransactionHandler struct {
	transactionService interfaces.TransactionService
}

// NewTransactionHandler creates a new transaction handler
func NewTransactionHandler(transactionService interfaces.TransactionService) interfaces.TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

// RegisterRoutes registers transaction management routes
func (h *TransactionHandler) RegisterRoutes(router gin.IRouter) {
	transactions := router.Group("/transactions")
	{
		transactions.GET("", h.SearchTransactions)
		transactions.GET("/:id", h.GetTransactionDetail)
		transactions.POST("/:id/reverse", h.ReverseTransaction)
	}

	accounts := router.Group("/accounts")
	{
		accounts.GET("/:id/transactions", h.GetAccountTransactions)
	}
}

// SearchTransactions handles transaction search requests
// @Summary Search transactions with advanced filtering
// @Description Search and filter transactions based on various criteria
// @Tags transactions
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param user_id query string false "Filter by user ID"
// @Param account_id query string false "Filter by account ID"
// @Param currency query string false "Filter by currency"
// @Param status query string false "Filter by status"
// @Param amount_min query string false "Minimum amount filter"
// @Param amount_max query string false "Maximum amount filter"
// @Param date_from query string false "Start date filter (RFC3339)"
// @Param date_to query string false "End date filter (RFC3339)"
// @Param description query string false "Description search"
// @Success 200 {object} interfaces.PaginatedTransactions
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/admin/transactions [get]
func (h *TransactionHandler) SearchTransactions(c *gin.Context) {
	var params interfaces.SearchTransactionParams

	// Parse pagination parameters
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			params.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			params.PageSize = ps
		}
	}

	// Parse filter parameters
	params.UserID = c.Query("user_id")
	params.AccountID = c.Query("account_id")
	params.Currency = c.Query("currency")
	params.Status = c.Query("status")
	params.Description = c.Query("description")

	// Parse amount filters
	if amountMin := c.Query("amount_min"); amountMin != "" {
		params.AmountMin = &amountMin
	}
	if amountMax := c.Query("amount_max"); amountMax != "" {
		params.AmountMax = &amountMax
	}

	// Parse date filters
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			params.DateFrom = &t
		} else {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: "Invalid date_from format. Use RFC3339 format.",
				Code:    http.StatusBadRequest,
			})
			return
		}
	}

	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse(time.RFC3339, dateTo); err == nil {
			params.DateTo = &t
		} else {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: "Invalid date_to format. Use RFC3339 format.",
				Code:    http.StatusBadRequest,
			})
			return
		}
	}

	// Search transactions
	result, err := h.transactionService.SearchTransactions(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to search transactions: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTransactionDetail handles transaction detail requests
// @Summary Get detailed transaction information
// @Description Get complete transaction details including audit trail
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Success 200 {object} interfaces.TransactionDetail
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/admin/transactions/{id} [get]
func (h *TransactionHandler) GetTransactionDetail(c *gin.Context) {
	transactionID := c.Param("id")
	if transactionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Transaction ID is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	detail, err := h.transactionService.GetTransactionDetail(c.Request.Context(), transactionID)
	if err != nil {
		if err.Error() == "transaction not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Transaction not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get transaction detail: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// ReverseTransaction handles transaction reversal requests
// @Summary Reverse a transaction
// @Description Reverse a completed transaction with proper authorization
// @Tags transactions
// @Accept json
// @Produce json
// @Param id path string true "Transaction ID"
// @Param request body ReverseTransactionRequest true "Reversal request"
// @Success 200 {object} interfaces.TransactionDetail
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/admin/transactions/{id}/reverse [post]
func (h *TransactionHandler) ReverseTransaction(c *gin.Context) {
	transactionID := c.Param("id")
	if transactionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Transaction ID is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var req ReverseTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request body: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Validate reason is provided
	if req.Reason == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Reversal reason is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	detail, err := h.transactionService.ReverseTransaction(c.Request.Context(), transactionID, req.Reason)
	if err != nil {
		if err.Error() == "transaction not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Transaction not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		if err.Error() == "transaction already reversed" || err.Error() == "only completed transactions can be reversed" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "business_rule_violation",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to reverse transaction: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// GetAccountTransactions handles account transaction requests
// @Summary Get transactions for a specific account
// @Description Get paginated list of transactions for an account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} interfaces.PaginatedTransactions
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/admin/accounts/{id}/transactions [get]
func (h *TransactionHandler) GetAccountTransactions(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Account ID is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var params interfaces.PaginationParams

	// Parse pagination parameters
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			params.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			params.PageSize = ps
		}
	}

	result, err := h.transactionService.GetAccountTransactions(c.Request.Context(), accountID, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get account transactions: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Request/Response types

// ReverseTransactionRequest represents a transaction reversal request
type ReverseTransactionRequest struct {
	Reason string `json:"reason" binding:"required" example:"Fraudulent transaction"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"validation_error"`
	Message string `json:"message" example:"Invalid request parameters"`
	Code    int    `json:"code" example:"400"`
}