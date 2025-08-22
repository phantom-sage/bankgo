package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/phantom-sage/bankgo/internal/models"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Code    int               `json:"code"`
	Details map[string]string `json:"details,omitempty"`
}

// AppError represents an application-specific error with HTTP status code
type AppError struct {
	Code    int
	Type    string
	Message string
	Details map[string]string
	Err     error
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// NewAppError creates a new application error
func NewAppError(code int, errorType, message string) *AppError {
	return &AppError{
		Code:    code,
		Type:    errorType,
		Message: message,
	}
}

// NewAppErrorWithDetails creates a new application error with details
func NewAppErrorWithDetails(code int, errorType, message string, details map[string]string) *AppError {
	return &AppError{
		Code:    code,
		Type:    errorType,
		Message: message,
		Details: details,
	}
}

// NewAppErrorFromError creates an application error from a standard error
func NewAppErrorFromError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	// Map known model errors to appropriate HTTP status codes
	switch {
	case errors.Is(err, models.ErrInvalidEmail):
		return NewAppError(http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, models.ErrPasswordTooShort):
		return NewAppError(http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, models.ErrEmptyFirstName):
		return NewAppError(http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, models.ErrEmptyLastName):
		return NewAppError(http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, models.ErrInvalidCurrency):
		return NewAppError(http.StatusBadRequest, "invalid_currency", err.Error())
	case errors.Is(err, models.ErrNegativeBalance):
		return NewAppError(http.StatusBadRequest, "invalid_balance", err.Error())
	case errors.Is(err, models.ErrInvalidUserID):
		return NewAppError(http.StatusBadRequest, "invalid_user_id", err.Error())
	case errors.Is(err, models.ErrInvalidTransferAmount):
		return NewAppError(http.StatusBadRequest, "invalid_amount", err.Error())
	case errors.Is(err, models.ErrInvalidFromAccount):
		return NewAppError(http.StatusBadRequest, "invalid_from_account", err.Error())
	case errors.Is(err, models.ErrInvalidToAccount):
		return NewAppError(http.StatusBadRequest, "invalid_to_account", err.Error())
	case errors.Is(err, models.ErrSameAccount):
		return NewAppError(http.StatusBadRequest, "same_account", err.Error())
	case errors.Is(err, models.ErrInvalidTransferStatus):
		return NewAppError(http.StatusBadRequest, "invalid_status", err.Error())
	case errors.Is(err, models.ErrCurrencyMismatch):
		return NewAppError(http.StatusUnprocessableEntity, "currency_mismatch", err.Error())
	case errors.Is(err, models.ErrInsufficientBalance):
		return NewAppError(http.StatusUnprocessableEntity, "insufficient_balance", err.Error())
	}

	// Check for common error patterns in error messages
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "already exists"):
		return NewAppError(http.StatusConflict, "resource_exists", errMsg)
	case strings.Contains(errMsg, "not found"):
		return NewAppError(http.StatusNotFound, "resource_not_found", errMsg)
	case strings.Contains(errMsg, "access denied"):
		return NewAppError(http.StatusForbidden, "access_denied", errMsg)
	case strings.Contains(errMsg, "unauthorized"):
		return NewAppError(http.StatusUnauthorized, "unauthorized", errMsg)
	case strings.Contains(errMsg, "validation failed"):
		return NewAppError(http.StatusBadRequest, "validation_error", errMsg)
	case strings.Contains(errMsg, "non-zero balance"):
		return NewAppError(http.StatusUnprocessableEntity, "non_zero_balance", errMsg)
	case strings.Contains(errMsg, "transaction history"):
		return NewAppError(http.StatusUnprocessableEntity, "has_transactions", errMsg)
	case strings.Contains(errMsg, "invalid token"):
		return NewAppError(http.StatusUnauthorized, "invalid_token", errMsg)
	case strings.Contains(errMsg, "token expired"):
		return NewAppError(http.StatusUnauthorized, "token_expired", errMsg)
	}

	// Default to internal server error for unknown errors
	return NewAppError(http.StatusInternalServerError, "internal_error", "An internal error occurred")
}

// ErrorHandler middleware handles errors consistently across the application
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			
			var appErr *AppError
			if ae, ok := err.Err.(*AppError); ok {
				appErr = ae
			} else {
				appErr = NewAppErrorFromError(err.Err)
			}

			// Don't override status if it's already set
			if c.Writer.Status() == http.StatusOK {
				c.Status(appErr.Code)
			}

			c.JSON(appErr.Code, ErrorResponse{
				Error:   appErr.Type,
				Message: appErr.Message,
				Code:    appErr.Code,
				Details: appErr.Details,
			})
			return
		}
	}
}

// HandleValidationError processes Gin validation errors and returns detailed error information
func HandleValidationError(err error) *AppError {
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		details := make(map[string]string)
		
		for _, fieldError := range validationErrors {
			field := strings.ToLower(fieldError.Field())
			
			switch fieldError.Tag() {
			case "required":
				details[field] = field + " is required"
			case "email":
				details[field] = "Invalid email format"
			case "min":
				details[field] = field + " must be at least " + fieldError.Param() + " characters long"
			case "max":
				details[field] = field + " must be at most " + fieldError.Param() + " characters long"
			case "len":
				details[field] = field + " must be exactly " + fieldError.Param() + " characters long"
			case "gt":
				details[field] = field + " must be greater than " + fieldError.Param()
			case "gte":
				details[field] = field + " must be greater than or equal to " + fieldError.Param()
			case "lt":
				details[field] = field + " must be less than " + fieldError.Param()
			case "lte":
				details[field] = field + " must be less than or equal to " + fieldError.Param()
			default:
				details[field] = "Invalid " + field
			}
		}
		
		return NewAppErrorWithDetails(
			http.StatusBadRequest,
			"validation_error",
			"Request validation failed",
			details,
		)
	}
	
	return NewAppError(http.StatusBadRequest, "validation_error", err.Error())
}

// HandleError is a helper function to handle errors in handlers
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	var appErr *AppError
	if ae, ok := err.(*AppError); ok {
		appErr = ae
	} else {
		appErr = NewAppErrorFromError(err)
	}

	c.JSON(appErr.Code, ErrorResponse{
		Error:   appErr.Type,
		Message: appErr.Message,
		Code:    appErr.Code,
		Details: appErr.Details,
	})
}

// HandleValidationErrorInContext handles validation errors within a Gin context
func HandleValidationErrorInContext(c *gin.Context, err error) {
	appErr := HandleValidationError(err)
	c.JSON(appErr.Code, ErrorResponse{
		Error:   appErr.Type,
		Message: appErr.Message,
		Code:    appErr.Code,
		Details: appErr.Details,
	})
}

// Business logic error helpers

// NewInsufficientBalanceError creates an insufficient balance error
func NewInsufficientBalanceError(currentBalance, requestedAmount string) *AppError {
	return NewAppErrorWithDetails(
		http.StatusUnprocessableEntity,
		"insufficient_balance",
		"Insufficient balance for this transaction",
		map[string]string{
			"current_balance":    currentBalance,
			"requested_amount":   requestedAmount,
		},
	)
}

// NewCurrencyMismatchError creates a currency mismatch error
func NewCurrencyMismatchError(fromCurrency, toCurrency string) *AppError {
	return NewAppErrorWithDetails(
		http.StatusUnprocessableEntity,
		"currency_mismatch",
		"Cannot transfer between accounts with different currencies",
		map[string]string{
			"from_currency": fromCurrency,
			"to_currency":   toCurrency,
		},
	)
}

// NewDuplicateCurrencyError creates a duplicate currency error
func NewDuplicateCurrencyError(currency string) *AppError {
	return NewAppErrorWithDetails(
		http.StatusConflict,
		"duplicate_currency",
		"User already has an account with this currency",
		map[string]string{
			"currency": currency,
		},
	)
}

// NewAccountNotFoundError creates an account not found error
func NewAccountNotFoundError(accountID string) *AppError {
	return NewAppErrorWithDetails(
		http.StatusNotFound,
		"account_not_found",
		"Account not found",
		map[string]string{
			"account_id": accountID,
		},
	)
}

// NewAccessDeniedError creates an access denied error
func NewAccessDeniedError(resource string) *AppError {
	return NewAppErrorWithDetails(
		http.StatusForbidden,
		"access_denied",
		"Access denied to "+resource,
		map[string]string{
			"resource": resource,
		},
	)
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(reason string) *AppError {
	return NewAppError(http.StatusUnauthorized, "unauthorized", reason)
}

// NewInternalError creates an internal server error (without exposing internal details)
func NewInternalError() *AppError {
	return NewAppError(http.StatusInternalServerError, "internal_error", "An internal error occurred")
}