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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDatabaseService for testing handlers
type MockDatabaseService struct {
	mock.Mock
}

func (m *MockDatabaseService) ListTables(ctx context.Context) ([]interfaces.TableInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]interfaces.TableInfo), args.Error(1)
}

func (m *MockDatabaseService) GetTableSchema(ctx context.Context, tableName string) (*interfaces.TableSchema, error) {
	args := m.Called(ctx, tableName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.TableSchema), args.Error(1)
}

func (m *MockDatabaseService) ListRecords(ctx context.Context, tableName string, params interfaces.ListRecordsParams) (*interfaces.PaginatedRecords, error) {
	args := m.Called(ctx, tableName, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.PaginatedRecords), args.Error(1)
}

func (m *MockDatabaseService) GetRecord(ctx context.Context, tableName string, recordID interface{}) (*interfaces.TableRecord, error) {
	args := m.Called(ctx, tableName, recordID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.TableRecord), args.Error(1)
}

func (m *MockDatabaseService) CreateRecord(ctx context.Context, tableName string, data map[string]interface{}) (*interfaces.TableRecord, error) {
	args := m.Called(ctx, tableName, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.TableRecord), args.Error(1)
}

func (m *MockDatabaseService) UpdateRecord(ctx context.Context, tableName string, recordID interface{}, data map[string]interface{}) (*interfaces.TableRecord, error) {
	args := m.Called(ctx, tableName, recordID, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.TableRecord), args.Error(1)
}

func (m *MockDatabaseService) DeleteRecord(ctx context.Context, tableName string, recordID interface{}) error {
	args := m.Called(ctx, tableName, recordID)
	return args.Error(0)
}

func (m *MockDatabaseService) BulkOperation(ctx context.Context, tableName string, operation interfaces.BulkOperation) (*interfaces.BulkOperationResult, error) {
	args := m.Called(ctx, tableName, operation)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.BulkOperationResult), args.Error(1)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestDatabaseHandler_ListTables(t *testing.T) {
	tests := []struct {
		name           string
		mockTables     []interfaces.TableInfo
		mockError      error
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "successful table listing",
			mockTables: []interfaces.TableInfo{
				{Name: "users", Schema: "public", RecordCount: 100},
				{Name: "accounts", Schema: "public", RecordCount: 250},
				{Name: "transfers", Schema: "public", RecordCount: 500},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "database error",
			mockTables:     []interfaces.TableInfo{},
			mockError:      fmt.Errorf("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockDatabaseService)
			mockService.On("ListTables", mock.Anything).Return(tt.mockTables, tt.mockError)

			handler := NewDatabaseHandler(mockService)
			router := setupTestRouter()
			handler.RegisterRoutes(router.Group("/api/admin"))

			req, _ := http.NewRequest("GET", "/api/admin/database/tables", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.mockError == nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, float64(tt.expectedCount), response["count"])
				
				tables, ok := response["tables"].([]interface{})
				assert.True(t, ok)
				assert.Len(t, tables, tt.expectedCount)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestDatabaseHandler_GetTableSchema(t *testing.T) {
	tests := []struct {
		name           string
		tableName      string
		mockSchema     *interfaces.TableSchema
		mockError      error
		expectedStatus int
	}{
		{
			name:      "successful schema retrieval",
			tableName: "users",
			mockSchema: &interfaces.TableSchema{
				Name:   "users",
				Schema: "public",
				Columns: []interfaces.Column{
					{Name: "id", Type: "integer", IsPrimaryKey: true},
					{Name: "email", Type: "character varying", Nullable: false},
				},
				PrimaryKeys: []string{"id"},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "table not found",
			tableName:      "nonexistent",
			mockSchema:     nil,
			mockError:      fmt.Errorf("table not found"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "empty table name",
			tableName:      "",
			mockSchema:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockDatabaseService)
			if tt.tableName != "" {
				mockService.On("GetTableSchema", mock.Anything, tt.tableName).Return(tt.mockSchema, tt.mockError)
			}

			handler := NewDatabaseHandler(mockService)
			router := setupTestRouter()
			handler.RegisterRoutes(router.Group("/api/admin"))

			url := fmt.Sprintf("/api/admin/database/tables/%s/schema", tt.tableName)
			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.mockError == nil && tt.tableName != "" {
				var schema interfaces.TableSchema
				err := json.Unmarshal(w.Body.Bytes(), &schema)
				assert.NoError(t, err)
				assert.Equal(t, tt.tableName, schema.Name)
			}

			if tt.tableName != "" {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestDatabaseHandler_ListRecords(t *testing.T) {
	tests := []struct {
		name           string
		tableName      string
		queryParams    string
		mockResult     *interfaces.PaginatedRecords
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful record listing",
			tableName:   "users",
			queryParams: "?page=1&page_size=10&search=john",
			mockResult: &interfaces.PaginatedRecords{
				Records: []interfaces.TableRecord{
					{
						TableName: "users",
						Data: map[string]interface{}{
							"id":    1,
							"email": "john@example.com",
						},
					},
				},
				Pagination: interfaces.PaginationInfo{
					Page:       1,
					PageSize:   10,
					TotalItems: 1,
					TotalPages: 1,
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "database error",
			tableName:      "users",
			queryParams:    "",
			mockResult:     nil,
			mockError:      fmt.Errorf("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockDatabaseService)
			mockService.On("ListRecords", mock.Anything, tt.tableName, mock.AnythingOfType("interfaces.ListRecordsParams")).Return(tt.mockResult, tt.mockError)

			handler := NewDatabaseHandler(mockService)
			router := setupTestRouter()
			handler.RegisterRoutes(router.Group("/api/admin"))

			url := fmt.Sprintf("/api/admin/database/tables/%s/records%s", tt.tableName, tt.queryParams)
			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.mockError == nil {
				var result interfaces.PaginatedRecords
				err := json.Unmarshal(w.Body.Bytes(), &result)
				assert.NoError(t, err)
				assert.Len(t, result.Records, len(tt.mockResult.Records))
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestDatabaseHandler_GetRecord(t *testing.T) {
	tests := []struct {
		name           string
		tableName      string
		recordID       string
		mockRecord     *interfaces.TableRecord
		mockError      error
		expectedStatus int
	}{
		{
			name:      "successful record retrieval",
			tableName: "users",
			recordID:  "1",
			mockRecord: &interfaces.TableRecord{
				TableName: "users",
				Data: map[string]interface{}{
					"id":    1,
					"email": "john@example.com",
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "record not found",
			tableName:      "users",
			recordID:       "999",
			mockRecord:     nil,
			mockError:      fmt.Errorf("record not found"),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockDatabaseService)
			// Convert string ID to appropriate type for mock
			var recordID interface{}
			if intID := 1; tt.recordID == "1" {
				recordID = intID
			} else if intID := 999; tt.recordID == "999" {
				recordID = intID
			}
			mockService.On("GetRecord", mock.Anything, tt.tableName, recordID).Return(tt.mockRecord, tt.mockError)

			handler := NewDatabaseHandler(mockService)
			router := setupTestRouter()
			handler.RegisterRoutes(router.Group("/api/admin"))

			url := fmt.Sprintf("/api/admin/database/tables/%s/records/%s", tt.tableName, tt.recordID)
			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.mockError == nil {
				var record interfaces.TableRecord
				err := json.Unmarshal(w.Body.Bytes(), &record)
				assert.NoError(t, err)
				assert.Equal(t, tt.tableName, record.TableName)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestDatabaseHandler_CreateRecord(t *testing.T) {
	tests := []struct {
		name           string
		tableName      string
		requestData    map[string]interface{}
		mockRecord     *interfaces.TableRecord
		mockError      error
		expectedStatus int
	}{
		{
			name:      "successful record creation",
			tableName: "users",
			requestData: map[string]interface{}{
				"email":      "newuser@example.com",
				"first_name": "New",
				"last_name":  "User",
			},
			mockRecord: &interfaces.TableRecord{
				TableName: "users",
				Data: map[string]interface{}{
					"id":         2,
					"email":      "newuser@example.com",
					"first_name": "New",
					"last_name":  "User",
					"created_at": time.Now(),
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:        "validation error",
			tableName:   "users",
			requestData: map[string]interface{}{
				"first_name": "New", // missing required email
			},
			mockRecord:     nil,
			mockError:      fmt.Errorf("validation failed: email is required"),
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockDatabaseService)
			mockService.On("CreateRecord", mock.Anything, tt.tableName, tt.requestData).Return(tt.mockRecord, tt.mockError)

			handler := NewDatabaseHandler(mockService)
			router := setupTestRouter()
			handler.RegisterRoutes(router.Group("/api/admin"))

			jsonData, _ := json.Marshal(tt.requestData)
			url := fmt.Sprintf("/api/admin/database/tables/%s/records", tt.tableName)
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.mockError == nil {
				var record interfaces.TableRecord
				err := json.Unmarshal(w.Body.Bytes(), &record)
				assert.NoError(t, err)
				assert.Equal(t, tt.tableName, record.TableName)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestDatabaseHandler_UpdateRecord(t *testing.T) {
	tests := []struct {
		name           string
		tableName      string
		recordID       string
		requestData    map[string]interface{}
		mockRecord     *interfaces.TableRecord
		mockError      error
		expectedStatus int
	}{
		{
			name:      "successful record update",
			tableName: "users",
			recordID:  "1",
			requestData: map[string]interface{}{
				"first_name": "Updated",
				"is_active":  false,
			},
			mockRecord: &interfaces.TableRecord{
				TableName: "users",
				Data: map[string]interface{}{
					"id":         1,
					"email":      "john@example.com",
					"first_name": "Updated",
					"is_active":  false,
					"updated_at": time.Now(),
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:        "record not found",
			tableName:   "users",
			recordID:    "999",
			requestData: map[string]interface{}{"first_name": "Updated"},
			mockRecord:  nil,
			mockError:   fmt.Errorf("record not found"),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockDatabaseService)
			
			// Convert string ID to appropriate type for mock
			var recordID interface{}
			if intID := 1; tt.recordID == "1" {
				recordID = intID
			} else if intID := 999; tt.recordID == "999" {
				recordID = intID
			}
			
			mockService.On("UpdateRecord", mock.Anything, tt.tableName, recordID, tt.requestData).Return(tt.mockRecord, tt.mockError)

			handler := NewDatabaseHandler(mockService)
			router := setupTestRouter()
			handler.RegisterRoutes(router.Group("/api/admin"))

			jsonData, _ := json.Marshal(tt.requestData)
			url := fmt.Sprintf("/api/admin/database/tables/%s/records/%s", tt.tableName, tt.recordID)
			req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.mockError == nil {
				var record interfaces.TableRecord
				err := json.Unmarshal(w.Body.Bytes(), &record)
				assert.NoError(t, err)
				assert.Equal(t, tt.tableName, record.TableName)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestDatabaseHandler_DeleteRecord(t *testing.T) {
	tests := []struct {
		name           string
		tableName      string
		recordID       string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful record deletion",
			tableName:      "users",
			recordID:       "1",
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "record not found",
			tableName:      "users",
			recordID:       "999",
			mockError:      fmt.Errorf("record not found"),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockDatabaseService)
			
			// Convert string ID to appropriate type for mock
			var recordID interface{}
			if intID := 1; tt.recordID == "1" {
				recordID = intID
			} else if intID := 999; tt.recordID == "999" {
				recordID = intID
			}
			
			mockService.On("DeleteRecord", mock.Anything, tt.tableName, recordID).Return(tt.mockError)

			handler := NewDatabaseHandler(mockService)
			router := setupTestRouter()
			handler.RegisterRoutes(router.Group("/api/admin"))

			url := fmt.Sprintf("/api/admin/database/tables/%s/records/%s", tt.tableName, tt.recordID)
			req, _ := http.NewRequest("DELETE", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockService.AssertExpectations(t)
		})
	}
}

func TestDatabaseHandler_BulkOperation(t *testing.T) {
	tests := []struct {
		name           string
		tableName      string
		operation      interfaces.BulkOperation
		mockResult     *interfaces.BulkOperationResult
		mockError      error
		expectedStatus int
	}{
		{
			name:      "successful bulk update",
			tableName: "users",
			operation: interfaces.BulkOperation{
				Operation: "update",
				RecordIDs: []interface{}{float64(1), float64(2), float64(3)}, // JSON unmarshals numbers as float64
				Data: map[string]interface{}{
					"is_active": false,
				},
			},
			mockResult: &interfaces.BulkOperationResult{
				Operation:     "update",
				TotalRecords:  3,
				AffectedRows:  3,
				SuccessCount:  3,
				ErrorCount:    0,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:      "successful bulk delete",
			tableName: "users",
			operation: interfaces.BulkOperation{
				Operation: "delete",
				RecordIDs: []interface{}{float64(4), float64(5)}, // JSON unmarshals numbers as float64
			},
			mockResult: &interfaces.BulkOperationResult{
				Operation:     "delete",
				TotalRecords:  2,
				AffectedRows:  2,
				SuccessCount:  2,
				ErrorCount:    0,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:      "invalid operation type",
			tableName: "users",
			operation: interfaces.BulkOperation{
				Operation: "invalid",
				RecordIDs: []interface{}{float64(1)}, // JSON unmarshals numbers as float64
			},
			mockResult:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "missing criteria",
			tableName: "users",
			operation: interfaces.BulkOperation{
				Operation: "update",
				// No RecordIDs or Filters
			},
			mockResult:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "missing data for update",
			tableName: "users",
			operation: interfaces.BulkOperation{
				Operation: "update",
				RecordIDs: []interface{}{float64(1)}, // JSON unmarshals numbers as float64
				// No Data
			},
			mockResult:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockDatabaseService)
			
			// Only set up mock if the request should reach the service
			if tt.expectedStatus == http.StatusOK {
				mockService.On("BulkOperation", mock.Anything, tt.tableName, tt.operation).Return(tt.mockResult, tt.mockError)
			}

			handler := NewDatabaseHandler(mockService)
			router := setupTestRouter()
			handler.RegisterRoutes(router.Group("/api/admin"))

			jsonData, _ := json.Marshal(tt.operation)
			url := fmt.Sprintf("/api/admin/database/tables/%s/bulk", tt.tableName)
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.mockError == nil && tt.expectedStatus == http.StatusOK {
				var result interfaces.BulkOperationResult
				err := json.Unmarshal(w.Body.Bytes(), &result)
				assert.NoError(t, err)
				assert.Equal(t, tt.operation.Operation, result.Operation)
			}

			if tt.expectedStatus == http.StatusOK {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestDatabaseHandler_ValidationHelpers(t *testing.T) {
	tests := []struct {
		name     string
		param    string
		expected bool
	}{
		{"page parameter", "page", true},
		{"page_size parameter", "page_size", true},
		{"search parameter", "search", true},
		{"sort_by parameter", "sort_by", true},
		{"sort_desc parameter", "sort_desc", true},
		{"custom filter", "is_active", false},
		{"custom filter", "email", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isReservedParam(tt.param)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatabaseHandler_ErrorClassification(t *testing.T) {
	tests := []struct {
		name        string
		error       error
		isValidation bool
	}{
		{
			name:         "validation error",
			error:        fmt.Errorf("validation failed: email is required"),
			isValidation: true,
		},
		{
			name:         "invalid column error",
			error:        fmt.Errorf("invalid column: nonexistent_field"),
			isValidation: true,
		},
		{
			name:         "null constraint error",
			error:        fmt.Errorf("column email cannot be null"),
			isValidation: true,
		},
		{
			name:         "type error",
			error:        fmt.Errorf("column age must be an integer"),
			isValidation: true,
		},
		{
			name:         "database connection error",
			error:        fmt.Errorf("connection refused"),
			isValidation: false,
		},
		{
			name:         "generic error",
			error:        fmt.Errorf("something went wrong"),
			isValidation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidationError(tt.error)
			assert.Equal(t, tt.isValidation, result)
		})
	}
}