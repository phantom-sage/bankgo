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
	"github.com/phantom-sage/bankgo/internal/admin/services"
	"github.com/phantom-sage/bankgo/internal/database/queries"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// UserIntegrationTestSuite provides integration tests for user management
type UserIntegrationTestSuite struct {
	suite.Suite
	router      *gin.Engine
	userHandler *UserHandler
	mockService *MockUserManagementService
}

func (suite *UserIntegrationTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	
	suite.mockService = &MockUserManagementService{}
	suite.userHandler = NewUserHandler(suite.mockService).(*UserHandler)
	
	suite.router = gin.New()
	suite.userHandler.RegisterRoutes(suite.router)
}

func (suite *UserIntegrationTestSuite) TestCompleteUserManagementWorkflow() {
	ctx := context.Background()
	now := time.Now()

	// Test data
	createReq := interfaces.CreateUserRequest{
		Email:     "testuser@example.com",
		FirstName: "Test",
		LastName:  "User",
		Password:  "password123",
	}

	createdUser := &interfaces.UserDetail{
		ID:               "1",
		Email:            "testuser@example.com",
		FirstName:        "Test",
		LastName:         "User",
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
		AccountCount:     0,
		TransferCount:    0,
		WelcomeEmailSent: false,
		Metadata:         make(map[string]interface{}),
	}

	updatedUser := &interfaces.UserDetail{
		ID:               "1",
		Email:            "testuser@example.com",
		FirstName:        "Updated",
		LastName:         "Name",
		IsActive:         false,
		CreatedAt:        now,
		UpdatedAt:        now,
		AccountCount:     0,
		TransferCount:    0,
		WelcomeEmailSent: false,
		Metadata:         make(map[string]interface{}),
	}

	usersList := &interfaces.PaginatedUsers{
		Users: []interfaces.UserDetail{*createdUser},
		Pagination: interfaces.PaginationInfo{
			Page:       1,
			PageSize:   20,
			TotalItems: 1,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}

	// Step 1: Create a new user
	suite.mockService.On("CreateUser", ctx, createReq).Return(createdUser, nil)

	reqBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusCreated, w.Code)
	var createResponse interfaces.UserDetail
	err := json.Unmarshal(w.Body.Bytes(), &createResponse)
	suite.NoError(err)
	suite.Equal("1", createResponse.ID)
	suite.Equal("testuser@example.com", createResponse.Email)
	suite.True(createResponse.IsActive)

	// Step 2: List users to verify creation
	suite.mockService.On("ListUsers", ctx, mock.AnythingOfType("interfaces.ListUsersParams")).Return(usersList, nil)

	req, _ = http.NewRequest("GET", "/users", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var listResponse interfaces.PaginatedUsers
	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	suite.NoError(err)
	suite.Len(listResponse.Users, 1)
	suite.Equal("testuser@example.com", listResponse.Users[0].Email)

	// Step 3: Get specific user details
	suite.mockService.On("GetUser", ctx, "1").Return(createdUser, nil)

	req, _ = http.NewRequest("GET", "/users/1", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var getResponse interfaces.UserDetail
	err = json.Unmarshal(w.Body.Bytes(), &getResponse)
	suite.NoError(err)
	suite.Equal("1", getResponse.ID)
	suite.Equal("testuser@example.com", getResponse.Email)

	// Step 4: Update user information
	firstName := "Updated"
	lastName := "Name"
	isActive := false
	updateReq := interfaces.UpdateUserRequest{
		FirstName: &firstName,
		LastName:  &lastName,
		IsActive:  &isActive,
	}

	suite.mockService.On("UpdateUser", ctx, "1", updateReq).Return(updatedUser, nil)

	reqBody, _ = json.Marshal(updateReq)
	req, _ = http.NewRequest("PUT", "/users/1", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var updateResponse interfaces.UserDetail
	err = json.Unmarshal(w.Body.Bytes(), &updateResponse)
	suite.NoError(err)
	suite.Equal("Updated", updateResponse.FirstName)
	suite.Equal("Name", updateResponse.LastName)
	suite.False(updateResponse.IsActive)

	// Step 5: Disable user
	suite.mockService.On("DisableUser", ctx, "1").Return(nil)

	req, _ = http.NewRequest("POST", "/users/1/disable", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var disableResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &disableResponse)
	suite.NoError(err)
	suite.Equal("User disabled successfully", disableResponse["message"])

	// Step 6: Enable user
	suite.mockService.On("EnableUser", ctx, "1").Return(nil)

	req, _ = http.NewRequest("POST", "/users/1/enable", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var enableResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &enableResponse)
	suite.NoError(err)
	suite.Equal("User enabled successfully", enableResponse["message"])

	// Step 7: Delete user (should succeed since no accounts/transfers)
	suite.mockService.On("DeleteUser", ctx, "1").Return(nil)

	req, _ = http.NewRequest("DELETE", "/users/1", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var deleteResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &deleteResponse)
	suite.NoError(err)
	suite.Equal("User deleted successfully", deleteResponse["message"])

	// Verify all expectations were met
	suite.mockService.AssertExpectations(suite.T())
}

func (suite *UserIntegrationTestSuite) TestUserManagementErrorScenarios() {
	ctx := context.Background()

	// Test duplicate email creation
	createReq := interfaces.CreateUserRequest{
		Email:     "duplicate@example.com",
		FirstName: "Duplicate",
		LastName:  "User",
		Password:  "password123",
	}

	suite.mockService.On("CreateUser", ctx, createReq).Return((*interfaces.UserDetail)(nil), fmt.Errorf("duplicate key constraint"))

	reqBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusConflict, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("email_already_exists", response["error"])

	// Test delete user with dependencies
	suite.mockService.On("DeleteUser", ctx, "2").Return(fmt.Errorf("cannot delete user with existing accounts"))

	req, _ = http.NewRequest("DELETE", "/users/2", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusConflict, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("user_has_dependencies", response["error"])

	// Test operations on non-existent user
	suite.mockService.On("GetUser", ctx, "999").Return((*interfaces.UserDetail)(nil), fmt.Errorf("user not found"))

	req, _ = http.NewRequest("GET", "/users/999", nil)
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusNotFound, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("user_not_found", response["error"])

	suite.mockService.AssertExpectations(suite.T())
}

func (suite *UserIntegrationTestSuite) TestUserListingWithFiltersAndPagination() {
	ctx := context.Background()
	now := time.Now()

	// Test data for filtered results
	filteredUsers := &interfaces.PaginatedUsers{
		Users: []interfaces.UserDetail{
			{
				ID:               "1",
				Email:            "active@example.com",
				FirstName:        "Active",
				LastName:         "User",
				IsActive:         true,
				CreatedAt:        now,
				UpdatedAt:        now,
				AccountCount:     1,
				TransferCount:    3,
				WelcomeEmailSent: true,
				Metadata:         make(map[string]interface{}),
			},
		},
		Pagination: interfaces.PaginationInfo{
			Page:       2,
			PageSize:   5,
			TotalItems: 10,
			TotalPages: 2,
			HasNext:    false,
			HasPrev:    true,
		},
	}

	// Test with various query parameters
	suite.mockService.On("ListUsers", ctx, mock.AnythingOfType("interfaces.ListUsersParams")).Return(filteredUsers, nil)

	req, _ := http.NewRequest("GET", "/users?page=2&page_size=5&search=active&is_active=true&sort_by=email&sort_desc=false", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var response interfaces.PaginatedUsers
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Len(response.Users, 1)
	suite.Equal("active@example.com", response.Users[0].Email)
	suite.True(response.Users[0].IsActive)

	// Check pagination info
	suite.Equal(2, response.Pagination.Page)
	suite.Equal(5, response.Pagination.PageSize)
	suite.Equal(10, response.Pagination.TotalItems)
	suite.Equal(2, response.Pagination.TotalPages)
	suite.False(response.Pagination.HasNext)
	suite.True(response.Pagination.HasPrev)

	suite.mockService.AssertExpectations(suite.T())
}

func TestUserIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UserIntegrationTestSuite))
}

// TestUserManagementServiceIntegration tests the service layer integration
func TestUserManagementServiceIntegration(t *testing.T) {
	// This would typically use a test database, but for now we'll test the conversion logic
	service := &services.UserManagementService{}

	now := time.Now()
	dbUser := queries.User{
		ID:               1,
		Email:            "test@example.com",
		FirstName:        "Test",
		LastName:         "User",
		IsActive:         pgtype.Bool{Bool: true, Valid: true},
		WelcomeEmailSent: pgtype.Bool{Bool: false, Valid: true},
		CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
	}

	// Test the conversion function
	userDetail := service.ConvertToUserDetail(dbUser, 2, 5)

	assert.Equal(t, "1", userDetail.ID)
	assert.Equal(t, "test@example.com", userDetail.Email)
	assert.Equal(t, "Test", userDetail.FirstName)
	assert.Equal(t, "User", userDetail.LastName)
	assert.True(t, userDetail.IsActive)
	assert.False(t, userDetail.WelcomeEmailSent)
	assert.Equal(t, 2, userDetail.AccountCount)
	assert.Equal(t, 5, userDetail.TransferCount)
	assert.Equal(t, now, userDetail.CreatedAt)
	assert.Equal(t, now, userDetail.UpdatedAt)
	assert.Nil(t, userDetail.LastLogin) // Not implemented yet
	assert.NotNil(t, userDetail.Metadata)
}