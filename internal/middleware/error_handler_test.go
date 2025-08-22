package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewAppError(t *testing.T) {
	err := NewAppError(http.StatusBadRequest, "validation_error", "Invalid input")
	
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "validation_error", err.Type)
	assert.Equal(t, "Invalid input", err.Message)
	assert.Nil(t, err.Details)
}

func TestNewAppErrorWithDetails(t *testing.T) {
	details := map[string]string{
		"field1": "error1",
		"field2": "error2",
	}
	
	err := NewAppErrorWithDetails(http.StatusBadRequest, "validation_error", "Invalid input", details)
	
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "validation_error", err.Type)
	assert.Equal(t, "Invalid input", err.Message)
	assert.Equal(t, details, err.Details)
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		expected string
	}{
		{
			name: "with underlying error",
			appError: &AppError{
				Code:    http.StatusBadRequest,
				Type:    "validation_error",
				Message: "Invalid input",
				Err:     errors.New("underlying error"),
			},
			expected: "underlying error",
		},
		{
			name: "without underlying error",
			appError: &AppError{
				Code:    http.StatusBadRequest,
				Type:    "validation_error",
				Message: "Invalid input",
			},
			expected: "Invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.appError.Error())
		})
	}
}

func TestNewAppErrorFromError(t *testing.T) {
	tests := []struct {
		name         string
		inputError   error
		expectedCode int
		expectedType string
	}{
		{
			name:         "invalid email error",
			inputError:   models.ErrInvalidEmail,
			expectedCode: http.StatusBadRequest,
			expectedType: "validation_error",
		},
		{
			name:         "password too short error",
			inputError:   models.ErrPasswordTooShort,
			expectedCode: http.StatusBadRequest,
			expectedType: "validation_error",
		},
		{
			name:         "invalid currency error",
			inputError:   models.ErrInvalidCurrency,
			expectedCode: http.StatusBadRequest,
			expectedType: "invalid_currency",
		},
		{
			name:         "insufficient balance error",
			inputError:   models.ErrInsufficientBalance,
			expectedCode: http.StatusUnprocessableEntity,
			expectedType: "insufficient_balance",
		},
		{
			name:         "currency mismatch error",
			inputError:   models.ErrCurrencyMismatch,
			expectedCode: http.StatusUnprocessableEntity,
			expectedType: "currency_mismatch",
		},
		{
			name:         "already exists pattern",
			inputError:   errors.New("user already exists"),
			expectedCode: http.StatusConflict,
			expectedType: "resource_exists",
		},
		{
			name:         "not found pattern",
			inputError:   errors.New("account not found"),
			expectedCode: http.StatusNotFound,
			expectedType: "resource_not_found",
		},
		{
			name:         "access denied pattern",
			inputError:   errors.New("access denied to resource"),
			expectedCode: http.StatusForbidden,
			expectedType: "access_denied",
		},
		{
			name:         "unauthorized pattern",
			inputError:   errors.New("unauthorized access"),
			expectedCode: http.StatusUnauthorized,
			expectedType: "unauthorized",
		},
		{
			name:         "unknown error",
			inputError:   errors.New("some unknown error"),
			expectedCode: http.StatusInternalServerError,
			expectedType: "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := NewAppErrorFromError(tt.inputError)
			
			assert.Equal(t, tt.expectedCode, appErr.Code)
			assert.Equal(t, tt.expectedType, appErr.Type)
		})
	}
}

func TestNewAppErrorFromError_WithAppError(t *testing.T) {
	originalErr := NewAppError(http.StatusBadRequest, "custom_error", "Custom message")
	
	result := NewAppErrorFromError(originalErr)
	
	assert.Equal(t, originalErr, result)
}

func TestErrorHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupHandler   func(*gin.Context)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "no errors",
			setupHandler: func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "app error",
			setupHandler: func(c *gin.Context) {
				appErr := NewAppError(http.StatusBadRequest, "validation_error", "Invalid input")
				c.Error(appErr)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "standard error",
			setupHandler: func(c *gin.Context) {
				c.Error(models.ErrInvalidEmail)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)
			
			r.Use(ErrorHandler())
			r.GET("/test", tt.setupHandler)
			
			req := httptest.NewRequest("GET", "/test", nil)
			c.Request = req
			
			r.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		inputError     error
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "nil error",
			inputError:     nil,
			expectedStatus: http.StatusOK, // No response should be written
		},
		{
			name:           "app error",
			inputError:     NewAppError(http.StatusBadRequest, "validation_error", "Invalid input"),
			expectedStatus: http.StatusBadRequest,
			expectedType:   "validation_error",
		},
		{
			name:           "standard error",
			inputError:     models.ErrInvalidEmail,
			expectedStatus: http.StatusBadRequest,
			expectedType:   "validation_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			HandleError(c, tt.inputError)
			
			if tt.inputError == nil {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Empty(t, w.Body.String())
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)
				assert.Contains(t, w.Body.String(), tt.expectedType)
			}
		})
	}
}

func TestBusinessLogicErrorHelpers(t *testing.T) {
	t.Run("NewInsufficientBalanceError", func(t *testing.T) {
		err := NewInsufficientBalanceError("100.00", "150.00")
		
		assert.Equal(t, http.StatusUnprocessableEntity, err.Code)
		assert.Equal(t, "insufficient_balance", err.Type)
		assert.Equal(t, "100.00", err.Details["current_balance"])
		assert.Equal(t, "150.00", err.Details["requested_amount"])
	})

	t.Run("NewCurrencyMismatchError", func(t *testing.T) {
		err := NewCurrencyMismatchError("USD", "EUR")
		
		assert.Equal(t, http.StatusUnprocessableEntity, err.Code)
		assert.Equal(t, "currency_mismatch", err.Type)
		assert.Equal(t, "USD", err.Details["from_currency"])
		assert.Equal(t, "EUR", err.Details["to_currency"])
	})

	t.Run("NewDuplicateCurrencyError", func(t *testing.T) {
		err := NewDuplicateCurrencyError("USD")
		
		assert.Equal(t, http.StatusConflict, err.Code)
		assert.Equal(t, "duplicate_currency", err.Type)
		assert.Equal(t, "USD", err.Details["currency"])
	})

	t.Run("NewAccountNotFoundError", func(t *testing.T) {
		err := NewAccountNotFoundError("123")
		
		assert.Equal(t, http.StatusNotFound, err.Code)
		assert.Equal(t, "account_not_found", err.Type)
		assert.Equal(t, "123", err.Details["account_id"])
	})

	t.Run("NewAccessDeniedError", func(t *testing.T) {
		err := NewAccessDeniedError("account")
		
		assert.Equal(t, http.StatusForbidden, err.Code)
		assert.Equal(t, "access_denied", err.Type)
		assert.Equal(t, "account", err.Details["resource"])
	})

	t.Run("NewUnauthorizedError", func(t *testing.T) {
		err := NewUnauthorizedError("invalid token")
		
		assert.Equal(t, http.StatusUnauthorized, err.Code)
		assert.Equal(t, "unauthorized", err.Type)
		assert.Equal(t, "invalid token", err.Message)
	})

	t.Run("NewInternalError", func(t *testing.T) {
		err := NewInternalError()
		
		assert.Equal(t, http.StatusInternalServerError, err.Code)
		assert.Equal(t, "internal_error", err.Type)
		assert.Equal(t, "An internal error occurred", err.Message)
	})
}

func TestHandleValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Test with a simple error (not validator.ValidationErrors)
	t.Run("simple error", func(t *testing.T) {
		err := errors.New("simple validation error")
		appErr := HandleValidationError(err)
		
		assert.Equal(t, http.StatusBadRequest, appErr.Code)
		assert.Equal(t, "validation_error", appErr.Type)
		assert.Equal(t, "simple validation error", appErr.Message)
	})
}

func TestHandleValidationErrorInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	err := errors.New("validation failed")
	HandleValidationErrorInContext(c, err)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "validation_error")
}

// Helper function to create a test Gin context
func createTestContext() (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return w, c
}

func TestErrorResponse_JSON(t *testing.T) {
	w, c := createTestContext()
	
	errResp := ErrorResponse{
		Error:   "validation_error",
		Message: "Invalid input",
		Code:    http.StatusBadRequest,
		Details: map[string]string{
			"field1": "error1",
		},
	}
	
	c.JSON(http.StatusBadRequest, errResp)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "validation_error")
	assert.Contains(t, body, "Invalid input")
	assert.Contains(t, body, "field1")
	assert.Contains(t, body, "error1")
}