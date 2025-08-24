package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/gin-gonic/gin"
)

// DatabaseHandler implements database management HTTP handlers
type DatabaseHandler struct {
	databaseService interfaces.DatabaseService
}

// NewDatabaseHandler creates a new database handler
func NewDatabaseHandler(databaseService interfaces.DatabaseService) interfaces.DatabaseHandler {
	return &DatabaseHandler{
		databaseService: databaseService,
	}
}

// RegisterRoutes registers database management routes
func (h *DatabaseHandler) RegisterRoutes(router gin.IRouter) {
	db := router.Group("/database")
	{
		db.GET("/tables", h.ListTables)
		db.GET("/tables/:table/schema", h.GetTableSchema)
		db.GET("/tables/:table/records", h.ListRecords)
		db.GET("/tables/:table/records/:id", h.GetRecord)
		db.POST("/tables/:table/records", h.CreateRecord)
		db.PUT("/tables/:table/records/:id", h.UpdateRecord)
		db.DELETE("/tables/:table/records/:id", h.DeleteRecord)
		db.POST("/tables/:table/bulk", h.BulkOperation)
	}
}

// ListTables handles GET /api/admin/database/tables
func (h *DatabaseHandler) ListTables(c *gin.Context) {
	tables, err := h.databaseService.ListTables(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_list_tables",
			"message": "Failed to retrieve database tables",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tables": tables,
		"count":  len(tables),
	})
}

// GetTableSchema handles GET /api/admin/database/tables/:table/schema
func (h *DatabaseHandler) GetTableSchema(c *gin.Context) {
	tableName := c.Param("table")
	if tableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_table_name",
			"message": "Table name is required",
		})
		return
	}

	schema, err := h.databaseService.GetTableSchema(c.Request.Context(), tableName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_get_schema",
			"message": "Failed to retrieve table schema",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, schema)
}

// ListRecords handles GET /api/admin/database/tables/:table/records
func (h *DatabaseHandler) ListRecords(c *gin.Context) {
	tableName := c.Param("table")
	if tableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_table_name",
			"message": "Table name is required",
		})
		return
	}

	// Parse pagination parameters
	var params interfaces.ListRecordsParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameters",
			"message": "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Parse filters from query parameters
	params.Filters = make(map[string]interface{})
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 && !isReservedParam(key) {
			// Try to parse as different types
			value := values[0]
			if value != "" {
				// Try boolean
				if boolVal, err := strconv.ParseBool(value); err == nil {
					params.Filters[key] = boolVal
				} else if intVal, err := strconv.Atoi(value); err == nil {
					// Try integer
					params.Filters[key] = intVal
				} else {
					// Default to string
					params.Filters[key] = value
				}
			}
		}
	}

	records, err := h.databaseService.ListRecords(c.Request.Context(), tableName, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_list_records",
			"message": "Failed to retrieve table records",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, records)
}

// GetRecord handles GET /api/admin/database/tables/:table/records/:id
func (h *DatabaseHandler) GetRecord(c *gin.Context) {
	tableName := c.Param("table")
	recordIDStr := c.Param("id")
	
	if tableName == "" || recordIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameters",
			"message": "Table name and record ID are required",
		})
		return
	}

	// Try to parse ID as integer first, then as string
	var recordID interface{}
	if intID, err := strconv.Atoi(recordIDStr); err == nil {
		recordID = intID
	} else {
		recordID = recordIDStr
	}

	record, err := h.databaseService.GetRecord(c.Request.Context(), tableName, recordID)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "record_not_found",
				"message": "Record not found",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_get_record",
			"message": "Failed to retrieve record",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, record)
}

// CreateRecord handles POST /api/admin/database/tables/:table/records
func (h *DatabaseHandler) CreateRecord(c *gin.Context) {
	tableName := c.Param("table")
	if tableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_table_name",
			"message": "Table name is required",
		})
		return
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request_body",
			"message": "Invalid JSON data",
			"details": err.Error(),
		})
		return
	}

	record, err := h.databaseService.CreateRecord(c.Request.Context(), tableName, data)
	if err != nil {
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": "Data validation failed",
				"details": err.Error(),
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_create_record",
			"message": "Failed to create record",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, record)
}

// UpdateRecord handles PUT /api/admin/database/tables/:table/records/:id
func (h *DatabaseHandler) UpdateRecord(c *gin.Context) {
	tableName := c.Param("table")
	recordIDStr := c.Param("id")
	
	if tableName == "" || recordIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameters",
			"message": "Table name and record ID are required",
		})
		return
	}

	// Try to parse ID as integer first, then as string
	var recordID interface{}
	if intID, err := strconv.Atoi(recordIDStr); err == nil {
		recordID = intID
	} else {
		recordID = recordIDStr
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request_body",
			"message": "Invalid JSON data",
			"details": err.Error(),
		})
		return
	}

	record, err := h.databaseService.UpdateRecord(c.Request.Context(), tableName, recordID, data)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "record_not_found",
				"message": "Record not found",
			})
			return
		}
		
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": "Data validation failed",
				"details": err.Error(),
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_update_record",
			"message": "Failed to update record",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, record)
}

// DeleteRecord handles DELETE /api/admin/database/tables/:table/records/:id
func (h *DatabaseHandler) DeleteRecord(c *gin.Context) {
	tableName := c.Param("table")
	recordIDStr := c.Param("id")
	
	if tableName == "" || recordIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameters",
			"message": "Table name and record ID are required",
		})
		return
	}

	// Try to parse ID as integer first, then as string
	var recordID interface{}
	if intID, err := strconv.Atoi(recordIDStr); err == nil {
		recordID = intID
	} else {
		recordID = recordIDStr
	}

	err := h.databaseService.DeleteRecord(c.Request.Context(), tableName, recordID)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "record_not_found",
				"message": "Record not found",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_delete_record",
			"message": "Failed to delete record",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Record deleted successfully",
	})
}

// BulkOperation handles POST /api/admin/database/tables/:table/bulk
func (h *DatabaseHandler) BulkOperation(c *gin.Context) {
	tableName := c.Param("table")
	if tableName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_table_name",
			"message": "Table name is required",
		})
		return
	}

	var operation interfaces.BulkOperation
	if err := c.ShouldBindJSON(&operation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request_body",
			"message": "Invalid bulk operation data",
			"details": err.Error(),
		})
		return
	}

	// Validate operation type
	if operation.Operation != "update" && operation.Operation != "delete" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_operation",
			"message": "Operation must be 'update' or 'delete'",
		})
		return
	}

	// Validate that either RecordIDs or Filters are provided
	if len(operation.RecordIDs) == 0 && len(operation.Filters) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_criteria",
			"message": "Either record_ids or filters must be provided",
		})
		return
	}

	// For update operations, data is required
	if operation.Operation == "update" && len(operation.Data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_data",
			"message": "Data is required for update operations",
		})
		return
	}

	result, err := h.databaseService.BulkOperation(c.Request.Context(), tableName, operation)
	if err != nil {
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": "Bulk operation validation failed",
				"details": err.Error(),
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "bulk_operation_failed",
			"message": "Failed to execute bulk operation",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Helper functions

// isReservedParam checks if a query parameter is reserved for pagination/sorting
func isReservedParam(param string) bool {
	reserved := map[string]bool{
		"page":      true,
		"page_size": true,
		"search":    true,
		"sort_by":   true,
		"sort_desc": true,
	}
	return reserved[param]
}

// isValidationError checks if an error is a validation error
func isValidationError(err error) bool {
	errMsg := err.Error()
	return strings.Contains(errMsg, "validation failed") ||
		strings.Contains(errMsg, "invalid column") ||
		strings.Contains(errMsg, "cannot be null") ||
		strings.Contains(errMsg, "must be")
}