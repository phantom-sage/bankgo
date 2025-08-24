package handlers

import (
	"net/http"
	"strconv"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/gin-gonic/gin"
)

// AccountHandler handles account management HTTP requests
type AccountHandler struct {
	accountService interfaces.AccountService
}

// NewAccountHandler creates a new account handler
func NewAccountHandler(accountService interfaces.AccountService) interfaces.AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

// RegisterRoutes registers account management routes
func (h *AccountHandler) RegisterRoutes(router gin.IRouter) {
	accounts := router.Group("/accounts")
	{
		accounts.GET("", h.SearchAccounts)
		accounts.GET("/:id", h.GetAccountDetail)
		accounts.POST("/:id/freeze", h.FreezeAccount)
		accounts.POST("/:id/unfreeze", h.UnfreezeAccount)
		accounts.POST("/:id/adjust-balance", h.AdjustBalance)
	}
}

// SearchAccounts handles account search requests
// @Summary Search accounts with filtering
// @Description Search and filter accounts based on various criteria
// @Tags accounts
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param search query string false "Search in user email, first name, or last name"
// @Param currency query string false "Filter by currency"
// @Param balance_min query string false "Minimum balance filter"
// @Param balance_max query string false "Maximum balance filter"
// @Param is_active query bool false "Filter by user active status"
// @Success 200 {object} interfaces.PaginatedAccounts
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/admin/accounts [get]
func (h *AccountHandler) SearchAccounts(c *gin.Context) {
	var params interfaces.SearchAccountParams

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
	params.Search = c.Query("search")
	params.Currency = c.Query("currency")

	// Parse balance filters
	if balanceMin := c.Query("balance_min"); balanceMin != "" {
		params.BalanceMin = &balanceMin
	}
	if balanceMax := c.Query("balance_max"); balanceMax != "" {
		params.BalanceMax = &balanceMax
	}

	// Parse active filter
	if isActive := c.Query("is_active"); isActive != "" {
		if active, err := strconv.ParseBool(isActive); err == nil {
			params.IsActive = &active
		}
	}

	// Search accounts
	result, err := h.accountService.SearchAccounts(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to search accounts: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetAccountDetail handles account detail requests
// @Summary Get detailed account information
// @Description Get complete account details including user information
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} interfaces.AccountDetail
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/admin/accounts/{id} [get]
func (h *AccountHandler) GetAccountDetail(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Account ID is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	detail, err := h.accountService.GetAccountDetail(c.Request.Context(), accountID)
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Account not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get account detail: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// FreezeAccount handles account freeze requests
// @Summary Freeze an account
// @Description Freeze an account to prevent transactions
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param request body FreezeAccountRequest true "Freeze request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/admin/accounts/{id}/freeze [post]
func (h *AccountHandler) FreezeAccount(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Account ID is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var req FreezeAccountRequest
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
			Message: "Freeze reason is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	err := h.accountService.FreezeAccount(c.Request.Context(), accountID, req.Reason)
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Account not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to freeze account: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Account frozen successfully",
	})
}

// UnfreezeAccount handles account unfreeze requests
// @Summary Unfreeze an account
// @Description Unfreeze an account to allow transactions
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/admin/accounts/{id}/unfreeze [post]
func (h *AccountHandler) UnfreezeAccount(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Account ID is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	err := h.accountService.UnfreezeAccount(c.Request.Context(), accountID)
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Account not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to unfreeze account: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Account unfrozen successfully",
	})
}

// AdjustBalance handles balance adjustment requests
// @Summary Adjust account balance
// @Description Adjust an account balance with proper authorization
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param request body AdjustBalanceRequest true "Balance adjustment request"
// @Success 200 {object} interfaces.AccountDetail
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/admin/accounts/{id}/adjust-balance [post]
func (h *AccountHandler) AdjustBalance(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Account ID is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var req AdjustBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request body: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Validate required fields
	if req.Amount == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Adjustment amount is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	if req.Reason == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Adjustment reason is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	detail, err := h.accountService.AdjustBalance(c.Request.Context(), accountID, req.Amount, req.Reason)
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Account not found",
				Code:    http.StatusNotFound,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to adjust balance: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// Request/Response types

// FreezeAccountRequest represents an account freeze request
type FreezeAccountRequest struct {
	Reason string `json:"reason" binding:"required" example:"Suspicious activity detected"`
}

// AdjustBalanceRequest represents a balance adjustment request
type AdjustBalanceRequest struct {
	Amount string `json:"amount" binding:"required" example:"100.50"`
	Reason string `json:"reason" binding:"required" example:"Manual correction for system error"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"Operation completed successfully"`
}