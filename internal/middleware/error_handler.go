package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/phantom-sage/bankgo/internal/logging"
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
	// Error logging context
	ErrorContext logging.ErrorContext
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
		ErrorContext: logging.ErrorContext{
			Category: logging.ClassifyError(errors.New(message)),
			Severity: logging.DetermineSeverity(errors.New(message), logging.ClassifyError(errors.New(message))),
		},
	}
}

// NewAppErrorWithDetails creates a new application error with details
func NewAppErrorWithDetails(code int, errorType, message string, details map[string]string) *AppError {
	return &AppError{
		Code:    code,
		Type:    errorType,
		Message: message,
		Details: details,
		ErrorContext: logging.ErrorContext{
			Category: logging.ClassifyError(errors.New(message)),
			Severity: logging.DetermineSeverity(errors.New(message), logging.ClassifyError(errors.New(message))),
		},
	}
}

// NewAppErrorFromError creates an application error from a standard error
func NewAppErrorFromError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	// Create error context
	category := logging.ClassifyError(err)
	severity := logging.DetermineSeverity(err, category)
	
	errorCtx := logging.ErrorContext{
		Category: category,
		Severity: severity,
	}

	// Map known model errors to appropriate HTTP status codes
	switch {
	case errors.Is(err, models.ErrInvalidEmail):
		return &AppError{
			Code: http.StatusBadRequest, Type: "validation_error", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case errors.Is(err, models.ErrPasswordTooShort):
		return &AppError{
			Code: http.StatusBadRequest, Type: "validation_error", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case errors.Is(err, models.ErrEmptyFirstName):
		return &AppError{
			Code: http.StatusBadRequest, Type: "validation_error", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case errors.Is(err, models.ErrEmptyLastName):
		return &AppError{
			Code: http.StatusBadRequest, Type: "validation_error", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case errors.Is(err, models.ErrInvalidCurrency):
		return &AppError{
			Code: http.StatusBadRequest, Type: "invalid_currency", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case errors.Is(err, models.ErrNegativeBalance):
		return &AppError{
			Code: http.StatusBadRequest, Type: "invalid_balance", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case errors.Is(err, models.ErrInvalidUserID):
		return &AppError{
			Code: http.StatusBadRequest, Type: "invalid_user_id", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case errors.Is(err, models.ErrInvalidTransferAmount):
		return &AppError{
			Code: http.StatusBadRequest, Type: "invalid_amount", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case errors.Is(err, models.ErrInvalidFromAccount):
		return &AppError{
			Code: http.StatusBadRequest, Type: "invalid_from_account", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case errors.Is(err, models.ErrInvalidToAccount):
		return &AppError{
			Code: http.StatusBadRequest, Type: "invalid_to_account", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case errors.Is(err, models.ErrSameAccount):
		return &AppError{
			Code: http.StatusBadRequest, Type: "same_account", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.BusinessLogicError),
		}
	case errors.Is(err, models.ErrInvalidTransferStatus):
		return &AppError{
			Code: http.StatusBadRequest, Type: "invalid_status", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case errors.Is(err, models.ErrCurrencyMismatch):
		return &AppError{
			Code: http.StatusUnprocessableEntity, Type: "currency_mismatch", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.BusinessLogicError),
		}
	case errors.Is(err, models.ErrInsufficientBalance):
		return &AppError{
			Code: http.StatusUnprocessableEntity, Type: "insufficient_balance", Message: err.Error(), Err: err,
			ErrorContext: errorCtx.WithCategory(logging.BusinessLogicError),
		}
	}

	// Check for common error patterns in error messages
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "already exists"):
		return &AppError{
			Code: http.StatusConflict, Type: "resource_exists", Message: errMsg, Err: err,
			ErrorContext: errorCtx.WithCategory(logging.BusinessLogicError),
		}
	case strings.Contains(errMsg, "not found"):
		return &AppError{
			Code: http.StatusNotFound, Type: "resource_not_found", Message: errMsg, Err: err,
			ErrorContext: errorCtx.WithCategory(logging.BusinessLogicError),
		}
	case strings.Contains(errMsg, "access denied"):
		return &AppError{
			Code: http.StatusForbidden, Type: "access_denied", Message: errMsg, Err: err,
			ErrorContext: errorCtx.WithCategory(logging.AuthenticationError),
		}
	case strings.Contains(errMsg, "unauthorized"):
		return &AppError{
			Code: http.StatusUnauthorized, Type: "unauthorized", Message: errMsg, Err: err,
			ErrorContext: errorCtx.WithCategory(logging.AuthenticationError),
		}
	case strings.Contains(errMsg, "validation failed"):
		return &AppError{
			Code: http.StatusBadRequest, Type: "validation_error", Message: errMsg, Err: err,
			ErrorContext: errorCtx.WithCategory(logging.ValidationError),
		}
	case strings.Contains(errMsg, "non-zero balance"):
		return &AppError{
			Code: http.StatusUnprocessableEntity, Type: "non_zero_balance", Message: errMsg, Err: err,
			ErrorContext: errorCtx.WithCategory(logging.BusinessLogicError),
		}
	case strings.Contains(errMsg, "transaction history"):
		return &AppError{
			Code: http.StatusUnprocessableEntity, Type: "has_transactions", Message: errMsg, Err: err,
			ErrorContext: errorCtx.WithCategory(logging.BusinessLogicError),
		}
	case strings.Contains(errMsg, "invalid token"):
		return &AppError{
			Code: http.StatusUnauthorized, Type: "invalid_token", Message: errMsg, Err: err,
			ErrorContext: errorCtx.WithCategory(logging.AuthenticationError),
		}
	case strings.Contains(errMsg, "token expired"):
		return &AppError{
			Code: http.StatusUnauthorized, Type: "token_expired", Message: errMsg, Err: err,
			ErrorContext: errorCtx.WithCategory(logging.AuthenticationError),
		}
	}

	// Default to internal server error for unknown errors
	return &AppError{
		Code: http.StatusInternalServerError, Type: "internal_error", Message: "An internal error occurred", Err: err,
		ErrorContext: errorCtx.WithCategory(logging.SystemError).WithSeverity(logging.HighSeverity),
	}
}

// ErrorHandler middleware handles errors consistently across the application
func ErrorHandler(errorLogger *logging.ErrorLogger) gin.HandlerFunc {
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

			// Create error context from Gin context
			errorCtx := NewErrorContextFromGinContext(c)
			
			// Add HTTP-specific context
			errorCtx.Method = c.Request.Method
			errorCtx.HTTPStatus = appErr.Code
			errorCtx.Operation = c.FullPath()
			errorCtx.Component = "http_handler"
			
			// Use existing error context if available
			if appErr.ErrorContext.Category != "" {
				errorCtx.Category = appErr.ErrorContext.Category
			} else {
				errorCtx.Category = logging.ClassifyError(appErr.Err)
			}
			
			if appErr.ErrorContext.Severity != "" {
				errorCtx.Severity = appErr.ErrorContext.Severity
			} else {
				errorCtx.Severity = logging.DetermineSeverity(appErr.Err, errorCtx.Category)
			}
			
			// Add error details
			if appErr.Details != nil {
				errorCtx.Details = make(map[string]interface{})
				for k, v := range appErr.Details {
					errorCtx.Details[k] = v
				}
			}
			
			// Log the error with structured context
			if errorLogger != nil {
				errorLogger.LogError(appErr.Err, errorCtx)
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

// HandleErrorWithLogging is a helper function to handle errors with structured logging
func HandleErrorWithLogging(c *gin.Context, err error, errorLogger *logging.ErrorLogger) {
	if err == nil {
		return
	}

	var appErr *AppError
	if ae, ok := err.(*AppError); ok {
		appErr = ae
	} else {
		appErr = NewAppErrorFromError(err)
	}

	// Create error context from Gin context
	errorCtx := NewErrorContextFromGinContext(c)
	errorCtx.Method = c.Request.Method
	errorCtx.HTTPStatus = appErr.Code
	errorCtx.Operation = c.FullPath()
	errorCtx.Component = "http_handler"
	
	// Use existing error context if available
	if appErr.ErrorContext.Category != "" {
		errorCtx.Category = appErr.ErrorContext.Category
	}
	if appErr.ErrorContext.Severity != "" {
		errorCtx.Severity = appErr.ErrorContext.Severity
	}
	
	// Add error details
	if appErr.Details != nil {
		errorCtx.Details = make(map[string]interface{})
		for k, v := range appErr.Details {
			errorCtx.Details[k] = v
		}
	}
	
	// Log the error with structured context
	if errorLogger != nil {
		errorLogger.LogError(appErr.Err, errorCtx)
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

// NewAppErrorWithContext creates a new application error with structured error context
func NewAppErrorWithContext(code int, errorType, message string, errorCtx logging.ErrorContext) *AppError {
	return &AppError{
		Code:         code,
		Type:         errorType,
		Message:      message,
		ErrorContext: errorCtx,
	}
}

// NewValidationError creates a validation error with structured context
func NewValidationError(message string, details map[string]string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Type:    "validation_error",
		Message: message,
		Details: details,
		ErrorContext: logging.ErrorContext{
			Category: logging.ValidationError,
			Severity: logging.LowSeverity,
		},
	}
}

// NewBusinessLogicError creates a business logic error with structured context
func NewBusinessLogicError(message string, details map[string]string) *AppError {
	return &AppError{
		Code:    http.StatusUnprocessableEntity,
		Type:    "business_logic_error",
		Message: message,
		Details: details,
		ErrorContext: logging.ErrorContext{
			Category: logging.BusinessLogicError,
			Severity: logging.MediumSeverity,
		},
	}
}

// NewAuthenticationErrorWithContext creates an authentication error with structured context
func NewAuthenticationErrorWithContext(message string) *AppError {
	return &AppError{
		Code:    http.StatusUnauthorized,
		Type:    "authentication_error",
		Message: message,
		ErrorContext: logging.ErrorContext{
			Category: logging.AuthenticationError,
			Severity: logging.MediumSeverity,
		},
	}
}

// NewSystemErrorWithContext creates a system error with structured context
func NewSystemErrorWithContext(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Type:    "system_error",
		Message: message,
		Err:     err,
		ErrorContext: logging.ErrorContext{
			Category: logging.SystemError,
			Severity: logging.HighSeverity,
		},
	}
}

// NewErrorContextFromGinContext creates an ErrorContext from a Gin context
func NewErrorContextFromGinContext(c *gin.Context) logging.ErrorContext {
	ctx := logging.ErrorContext{}

	// Extract request ID if available
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			ctx.RequestID = id
		}
	}

	// Extract user ID if available
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(int64); ok {
			ctx.UserID = id
		} else if id, ok := userID.(int); ok {
			ctx.UserID = int64(id)
		}
	}

	// Extract user email if available
	if userEmail, exists := c.Get("user_email"); exists {
		if email, ok := userEmail.(string); ok {
			ctx.UserEmail = email
		}
	}

	return ctx
}