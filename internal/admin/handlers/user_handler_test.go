package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserManagementService is a mock implementation of UserManagementService
type MockUserManagementService struct {
	mock.Mock
}

func (m *MockUserManagementService) ListUsers(ctx context.Context, params interfaces.ListUsersParams) (*interfaces.PaginatedUsers, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*interfaces.PaginatedUsers), args.Error(1)
}

func (m *MockUserManagementService) GetUser(ctx context.Context, userID string) (*interfaces.UserDetail, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.UserDetail), args.Error(1)
}

func (m *MockUserManagementService) CreateUser(ctx context.Context, req interfaces.CreateUserRequest) (*interfaces.UserDetail, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.UserDetail), args.Error(1)
}

func (m *MockUserManagementService) UpdateUser(ctx context.Context, userID string, req interfaces.UpdateUserRequest) (*interfaces.UserDetail, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.UserDetail), args.Error(1)
}

func (m *MockUserManagementService) DisableUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserManagementService) EnableUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserManagementService) DeleteUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func setupUserHandlerTest() (*UserHandler, *MockUserManagementService, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	
	mockService := &MockUserManagementService{}
	handler := NewUserHandler(mockService).(*UserHandler)
	
	router := gin.New()
	handler.RegisterRoutes(router)
	
	return handler, mockService, router
}

func TestUserHandler_ListUsers(t *testing.T) {
	_, mockService, router := setupUserHandlerTest()

	now := time.Now()
	mockUsers := &interfaces.PaginatedUsers{
		Users: []interfaces.UserDetail{
			{
				ID:               "1",
				Email:            "user1@example.com",
				FirstName:        "John",
				LastName:         "Doe",
				IsActive:         true,
				CreatedAt:        now,
				UpdatedAt:        now,
				AccountCount:     2,
				TransferCount:    5,
				WelcomeEmailSent: true,
				Metadata:         make(map[string]interface{}),
			},
			{
				ID:               "2",
				Email:            "user2@example.com",
				FirstName:        "Jane",
				LastName:         "Smith",
				IsActive:         false,
				CreatedAt:        now,
				UpdatedAt:        now,
				AccountCount:     1,
				TransferCount:    0,
				WelcomeEmailSent: false,
				Metadata:         make(map[string]interface{}),
			},
		},
		Pagination: interfaces.PaginationInfo{
			Page:       1,
			PageSize:   20,
			TotalItems: 2,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}

	t.Run("successful list users with default parameters", func(t *testing.T) {
		// Setup mock
		mockService.On("ListUsers", mock.Anything, mock.AnythingOfType("interfaces.ListUsersParams")).Return(mockUsers, nil)

		// Create request
		req, _ := http.NewRequest("GET", "/users", nil)
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response interfaces.PaginatedUsers
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.Users, 2)
		assert.Equal(t, "user1@example.com", response.Users[0].Email)
		assert.Equal(t, true, response.Users[0].IsActive)

		mockService.AssertExpectations(t)
	})

	t.Run("list users with query parameters", func(t *testing.T) {
		// Setup mock
		mockService.On("ListUsers", mock.Anything, mock.AnythingOfType("interfaces.ListUsersParams")).Return(mockUsers, nil)

		// Create request with query parameters
		req, _ := http.NewRequest("GET", "/users?page=2&page_size=10&search=john&is_active=true&sort_by=email&sort_desc=true", nil)
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		// Create a fresh mock and handler for this test
		freshMockService := &MockUserManagementService{}
		freshHandler := NewUserHandler(freshMockService).(*UserHandler)
		freshRouter := gin.New()
		freshHandler.RegisterRoutes(freshRouter)
		
		// Setup mock
		freshMockService.On("ListUsers", mock.Anything, mock.AnythingOfType("interfaces.ListUsersParams")).Return((*interfaces.PaginatedUsers)(nil), fmt.Errorf("database error"))

		// Create request
		req, _ := http.NewRequest("GET", "/users", nil)
		w := httptest.NewRecorder()

		// Execute
		freshRouter.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "failed_to_list_users", response["error"])

		freshMockService.AssertExpectations(t)
	})
}

func TestUserHandler_GetUser(t *testing.T) {
	_, mockService, router := setupUserHandlerTest()

	now := time.Now()
	mockUser := &interfaces.UserDetail{
		ID:               "1",
		Email:            "user@example.com",
		FirstName:        "John",
		LastName:         "Doe",
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
		AccountCount:     2,
		TransferCount:    5,
		WelcomeEmailSent: true,
		Metadata:         make(map[string]interface{}),
	}

	t.Run("successful get user", func(t *testing.T) {
		// Setup mock
		mockService.On("GetUser", mock.Anything, "1").Return(mockUser, nil)

		// Create request
		req, _ := http.NewRequest("GET", "/users/1", nil)
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response interfaces.UserDetail
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "1", response.ID)
		assert.Equal(t, "user@example.com", response.Email)

		mockService.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		// Setup mock
		mockService.On("GetUser", mock.Anything, "999").Return((*interfaces.UserDetail)(nil), fmt.Errorf("user not found"))

		// Create request
		req, _ := http.NewRequest("GET", "/users/999", nil)
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "user_not_found", response["error"])

		mockService.AssertExpectations(t)
	})
}

func TestUserHandler_CreateUser(t *testing.T) {
	_, mockService, router := setupUserHandlerTest()

	now := time.Now()
	mockUser := &interfaces.UserDetail{
		ID:               "1",
		Email:            "newuser@example.com",
		FirstName:        "New",
		LastName:         "User",
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
		AccountCount:     0,
		TransferCount:    0,
		WelcomeEmailSent: false,
		Metadata:         make(map[string]interface{}),
	}

	t.Run("successful create user", func(t *testing.T) {
		createReq := interfaces.CreateUserRequest{
			Email:     "newuser@example.com",
			FirstName: "New",
			LastName:  "User",
			Password:  "password123",
		}

		// Setup mock
		mockService.On("CreateUser", mock.Anything, createReq).Return(mockUser, nil)

		// Create request
		reqBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response interfaces.UserDetail
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "1", response.ID)
		assert.Equal(t, "newuser@example.com", response.Email)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid request data", func(t *testing.T) {
		// Create request with invalid JSON
		req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "invalid_request", response["error"])
	})

	t.Run("duplicate email error", func(t *testing.T) {
		createReq := interfaces.CreateUserRequest{
			Email:     "existing@example.com",
			FirstName: "Existing",
			LastName:  "User",
			Password:  "password123",
		}

		// Setup mock
		mockService.On("CreateUser", mock.Anything, createReq).Return((*interfaces.UserDetail)(nil), fmt.Errorf("duplicate key constraint"))

		// Create request
		reqBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusConflict, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "email_already_exists", response["error"])

		mockService.AssertExpectations(t)
	})
}

func TestUserHandler_UpdateUser(t *testing.T) {
	_, mockService, router := setupUserHandlerTest()

	now := time.Now()
	mockUser := &interfaces.UserDetail{
		ID:               "1",
		Email:            "user@example.com",
		FirstName:        "Updated",
		LastName:         "Name",
		IsActive:         false,
		CreatedAt:        now,
		UpdatedAt:        now,
		AccountCount:     2,
		TransferCount:    5,
		WelcomeEmailSent: true,
		Metadata:         make(map[string]interface{}),
	}

	t.Run("successful update user", func(t *testing.T) {
		firstName := "Updated"
		lastName := "Name"
		isActive := false
		updateReq := interfaces.UpdateUserRequest{
			FirstName: &firstName,
			LastName:  &lastName,
			IsActive:  &isActive,
		}

		// Setup mock
		mockService.On("UpdateUser", mock.Anything, "1", updateReq).Return(mockUser, nil)

		// Create request
		reqBody, _ := json.Marshal(updateReq)
		req, _ := http.NewRequest("PUT", "/users/1", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response interfaces.UserDetail
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Updated", response.FirstName)
		assert.Equal(t, "Name", response.LastName)
		assert.False(t, response.IsActive)

		mockService.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		updateReq := interfaces.UpdateUserRequest{}

		// Setup mock
		mockService.On("UpdateUser", mock.Anything, "999", updateReq).Return((*interfaces.UserDetail)(nil), fmt.Errorf("user not found"))

		// Create request
		reqBody, _ := json.Marshal(updateReq)
		req, _ := http.NewRequest("PUT", "/users/999", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)

		mockService.AssertExpectations(t)
	})
}

func TestUserHandler_DisableUser(t *testing.T) {
	_, mockService, router := setupUserHandlerTest()

	t.Run("successful disable user", func(t *testing.T) {
		// Setup mock
		mockService.On("DisableUser", mock.Anything, "1").Return(nil)

		// Create request
		req, _ := http.NewRequest("POST", "/users/1/disable", nil)
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "User disabled successfully", response["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		// Setup mock
		mockService.On("DisableUser", mock.Anything, "999").Return(fmt.Errorf("user not found"))

		// Create request
		req, _ := http.NewRequest("POST", "/users/999/disable", nil)
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)

		mockService.AssertExpectations(t)
	})
}

func TestUserHandler_EnableUser(t *testing.T) {
	_, mockService, router := setupUserHandlerTest()

	t.Run("successful enable user", func(t *testing.T) {
		// Setup mock
		mockService.On("EnableUser", mock.Anything, "1").Return(nil)

		// Create request
		req, _ := http.NewRequest("POST", "/users/1/enable", nil)
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "User enabled successfully", response["message"])

		mockService.AssertExpectations(t)
	})
}

func TestUserHandler_DeleteUser(t *testing.T) {
	_, mockService, router := setupUserHandlerTest()

	t.Run("successful delete user", func(t *testing.T) {
		// Setup mock
		mockService.On("DeleteUser", mock.Anything, "1").Return(nil)

		// Create request
		req, _ := http.NewRequest("DELETE", "/users/1", nil)
		w := httptest.NewRecorder()

		// Execute
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "User deleted successfully", response["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("user has dependencies", func(t *testing.T) {
		// Create a fresh mock and handler for this test
		freshMockService := &MockUserManagementService{}
		freshHandler := NewUserHandler(freshMockService).(*UserHandler)
		freshRouter := gin.New()
		freshHandler.RegisterRoutes(freshRouter)
		
		// Setup mock
		freshMockService.On("DeleteUser", mock.Anything, "1").Return(fmt.Errorf("cannot delete user with existing accounts"))

		// Create request
		req, _ := http.NewRequest("DELETE", "/users/1", nil)
		w := httptest.NewRecorder()

		// Execute
		freshRouter.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusConflict, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "user_has_dependencies", response["error"])

		freshMockService.AssertExpectations(t)
	})
}