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

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// DatabaseIntegrationTestSuite provides integration tests for database operations
type DatabaseIntegrationTestSuite struct {
	suite.Suite
	router  *gin.Engine
	handler interfaces.DatabaseHandler
	service *MockDatabaseService
}

func (suite *DatabaseIntegrationTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.service = new(MockDatabaseService)
	suite.handler = NewDatabaseHandler(suite.service)
	suite.handler.RegisterRoutes(suite.router.Group("/api/admin"))
}

func (suite *DatabaseIntegrationTestSuite) TestCompleteUserManagementWorkflow() {
	// Test basic CRUD operations workflow

	// 1. List tables
	expectedTables := []interfaces.TableInfo{
		{Name: "users", Schema: "public", RecordCount: 0},
		{Name: "accounts", Schema: "public", RecordCount: 0},
		{Name: "transfers", Schema: "public", RecordCount: 0},
	}
	suite.service.On("ListTables", context.Background()).Return(expectedTables, nil)

	req, _ := http.NewRequest("GET", "/api/admin/database/tables", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var tablesResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &tablesResponse)
	suite.NoError(err)
	suite.Equal(float64(3), tablesResponse["count"])

	// 2. Create a new user record
	newUserData := map[string]interface{}{
		"email":      "test@example.com",
		"first_name": "Test",
		"last_name":  "User",
		"is_active":  true,
	}
	createdUser := &interfaces.TableRecord{
		TableName: "users",
		Data: map[string]interface{}{
			"id":         float64(1), // JSON unmarshals numbers as float64
			"email":      "test@example.com",
			"first_name": "Test",
			"last_name":  "User",
			"is_active":  true,
			"created_at": time.Now(),
			"updated_at": time.Now(),
		},
		Metadata: interfaces.RecordMetadata{
			PrimaryKey: map[string]interface{}{"id": float64(1)},
		},
	}
	suite.service.On("CreateRecord", context.Background(), "users", newUserData).Return(createdUser, nil)

	jsonData, _ := json.Marshal(newUserData)
	req, _ = http.NewRequest("POST", "/api/admin/database/tables/users/records", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusCreated, w.Code)
	var createResponse interfaces.TableRecord
	err = json.Unmarshal(w.Body.Bytes(), &createResponse)
	suite.NoError(err)
	suite.Equal("users", createResponse.TableName)
	suite.Equal("test@example.com", createResponse.Data["email"])

	suite.service.AssertExpectations(suite.T())
}

func (suite *DatabaseIntegrationTestSuite) TestBulkOperationsWorkflow() {
	// Test bulk update and delete operations

	// 1. Bulk update multiple records
	bulkUpdateOp := interfaces.BulkOperation{
		Operation: "update",
		RecordIDs: []interface{}{float64(1), float64(2), float64(3)}, // JSON unmarshals numbers as float64
		Data: map[string]interface{}{
			"is_active": false,
		},
	}
	bulkUpdateResult := &interfaces.BulkOperationResult{
		Operation:     "update",
		TotalRecords:  3,
		AffectedRows:  3,
		SuccessCount:  3,
		ErrorCount:    0,
		Errors:        []interfaces.BulkOperationError{},
	}
	suite.service.On("BulkOperation", context.Background(), "users", bulkUpdateOp).Return(bulkUpdateResult, nil)

	jsonData, _ := json.Marshal(bulkUpdateOp)
	req, _ := http.NewRequest("POST", "/api/admin/database/tables/users/bulk", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var updateResult interfaces.BulkOperationResult
	err := json.Unmarshal(w.Body.Bytes(), &updateResult)
	suite.NoError(err)
	suite.Equal("update", updateResult.Operation)
	suite.Equal(3, updateResult.SuccessCount)
	suite.Equal(0, updateResult.ErrorCount)

	// 2. Bulk delete with partial failures
	bulkDeleteOp := interfaces.BulkOperation{
		Operation: "delete",
		RecordIDs: []interface{}{float64(4), float64(5), float64(999)}, // JSON unmarshals numbers as float64, 999 doesn't exist
	}
	bulkDeleteResult := &interfaces.BulkOperationResult{
		Operation:     "delete",
		TotalRecords:  3,
		AffectedRows:  2,
		SuccessCount:  2,
		ErrorCount:    1,
		Errors: []interfaces.BulkOperationError{
			{
				RecordID: 999,
				Error:    "record not found",
			},
		},
	}
	suite.service.On("BulkOperation", context.Background(), "users", bulkDeleteOp).Return(bulkDeleteResult, nil)

	jsonData, _ = json.Marshal(bulkDeleteOp)
	req, _ = http.NewRequest("POST", "/api/admin/database/tables/users/bulk", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var deleteResult interfaces.BulkOperationResult
	err = json.Unmarshal(w.Body.Bytes(), &deleteResult)
	suite.NoError(err)
	suite.Equal("delete", deleteResult.Operation)
	suite.Equal(2, deleteResult.SuccessCount)
	suite.Equal(1, deleteResult.ErrorCount)
	suite.Len(deleteResult.Errors, 1)

	suite.service.AssertExpectations(suite.T())
}

func (suite *DatabaseIntegrationTestSuite) TestSearchAndFilteringWorkflow() {
	// Test advanced search and filtering capabilities

	// 1. Search with text query
	searchResults := &interfaces.PaginatedRecords{
		Records: []interfaces.TableRecord{
			{
				TableName: "users",
				Data: map[string]interface{}{
					"id":         1,
					"email":      "john@example.com",
					"first_name": "John",
					"last_name":  "Doe",
					"is_active":  true,
				},
			},
		},
		Pagination: interfaces.PaginationInfo{
			Page:       1,
			PageSize:   10,
			TotalItems: 1,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}
	suite.service.On("ListRecords", context.Background(), "users", mock.MatchedBy(func(params interfaces.ListRecordsParams) bool {
		return params.Search == "john" && params.SortBy == "created_at" && params.SortDesc == true
	})).Return(searchResults, nil)

	req, _ := http.NewRequest("GET", "/api/admin/database/tables/users/records?search=john&sort_by=created_at&sort_desc=true", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	var response interfaces.PaginatedRecords
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Len(response.Records, 1)
	suite.Contains(response.Records[0].Data["email"], "john")

	// 2. Filter by specific field
	filterResults := &interfaces.PaginatedRecords{
		Records: []interfaces.TableRecord{
			{
				TableName: "users",
				Data: map[string]interface{}{
					"id":         2,
					"email":      "active@example.com",
					"first_name": "Active",
					"last_name":  "User",
					"is_active":  true,
				},
			},
		},
		Pagination: interfaces.PaginationInfo{
			Page:       1,
			PageSize:   10,
			TotalItems: 1,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		},
	}
	suite.service.On("ListRecords", context.Background(), "users", mock.MatchedBy(func(params interfaces.ListRecordsParams) bool {
		return params.Search == "" && params.SortBy == "email" && params.SortDesc == false &&
			len(params.Filters) == 1 && params.Filters["is_active"] == true
	})).Return(filterResults, nil)

	req, _ = http.NewRequest("GET", "/api/admin/database/tables/users/records?is_active=true&sort_by=email", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Len(response.Records, 1)
	suite.Equal(true, response.Records[0].Data["is_active"])

	suite.service.AssertExpectations(suite.T())
}

func (suite *DatabaseIntegrationTestSuite) TestPaginationWorkflow() {
	// Test pagination with large datasets

	// Create mock data for multiple pages
	totalRecords := 25
	pageSize := 10

	for page := 1; page <= 3; page++ {
		startRecord := (page-1)*pageSize + 1
		endRecord := page * pageSize
		if endRecord > totalRecords {
			endRecord = totalRecords
		}

		var records []interfaces.TableRecord
		for i := startRecord; i <= endRecord; i++ {
			records = append(records, interfaces.TableRecord{
				TableName: "users",
				Data: map[string]interface{}{
					"id":         i,
					"email":      fmt.Sprintf("user%d@example.com", i),
					"first_name": fmt.Sprintf("User%d", i),
					"last_name":  "Test",
					"is_active":  true,
				},
			})
		}

		paginatedResult := &interfaces.PaginatedRecords{
			Records: records,
			Pagination: interfaces.PaginationInfo{
				Page:       page,
				PageSize:   pageSize,
				TotalItems: totalRecords,
				TotalPages: 3,
				HasNext:    page < 3,
				HasPrev:    page > 1,
			},
		}

		suite.service.On("ListRecords", context.Background(), "users", mock.MatchedBy(func(params interfaces.ListRecordsParams) bool {
			return params.Page == page && params.PageSize == pageSize
		})).Return(paginatedResult, nil)

		url := fmt.Sprintf("/api/admin/database/tables/users/records?page=%d&page_size=%d", page, pageSize)
		req, _ := http.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		suite.Equal(http.StatusOK, w.Code)
		var response interfaces.PaginatedRecords
		err := json.Unmarshal(w.Body.Bytes(), &response)
		suite.NoError(err)
		suite.Equal(page, response.Pagination.Page)
		suite.Equal(pageSize, response.Pagination.PageSize)
		suite.Equal(totalRecords, response.Pagination.TotalItems)
		suite.Equal(3, response.Pagination.TotalPages)
		suite.Equal(page < 3, response.Pagination.HasNext)
		suite.Equal(page > 1, response.Pagination.HasPrev)
	}

	suite.service.AssertExpectations(suite.T())
}

func (suite *DatabaseIntegrationTestSuite) TestErrorHandlingWorkflow() {
	// Test validation error on create
	invalidData := map[string]interface{}{
		"first_name": "Test", // missing required email
	}
	suite.service.On("CreateRecord", context.Background(), "users", invalidData).Return((*interfaces.TableRecord)(nil), fmt.Errorf("validation failed: email is required"))

	jsonData, _ := json.Marshal(invalidData)
	req, _ := http.NewRequest("POST", "/api/admin/database/tables/users/records", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)
	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	suite.NoError(err)
	suite.Equal("validation_error", errorResponse["error"])

	suite.service.AssertExpectations(suite.T())
}

func TestDatabaseIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseIntegrationTestSuite))
}