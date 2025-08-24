package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseService implements the DatabaseService interface
type DatabaseService struct {
	db *pgxpool.Pool
}

// NewDatabaseService creates a new database service
func NewDatabaseService(db *pgxpool.Pool) interfaces.DatabaseService {
	return &DatabaseService{
		db: db,
	}
}

// ListTables returns available database tables
func (s *DatabaseService) ListTables(ctx context.Context) ([]interfaces.TableInfo, error) {
	query := `
		SELECT 
			t.table_name,
			t.table_schema,
			COALESCE(c.reltuples::bigint, 0) as estimated_count,
			obj_description(c.oid) as description
		FROM information_schema.tables t
		LEFT JOIN pg_class c ON c.relname = t.table_name
		LEFT JOIN pg_namespace n ON n.nspname = t.table_schema AND c.relnamespace = n.oid
		WHERE t.table_schema = 'public' 
		AND t.table_type = 'BASE TABLE'
		ORDER BY t.table_name`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []interfaces.TableInfo
	for rows.Next() {
		var table interfaces.TableInfo
		var description sql.NullString
		
		err := rows.Scan(&table.Name, &table.Schema, &table.RecordCount, &description)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table info: %w", err)
		}
		
		if description.Valid {
			table.Description = description.String
		}
		
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	return tables, nil
}

// GetTableSchema returns table structure information
func (s *DatabaseService) GetTableSchema(ctx context.Context, tableName string) (*interfaces.TableSchema, error) {
	// Get columns information
	columnsQuery := `
		SELECT 
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as nullable,
			c.column_default,
			CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_primary_key,
			CASE WHEN fk.column_name IS NOT NULL THEN true ELSE false END as is_foreign_key,
			c.character_maximum_length
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku ON tc.constraint_name = ku.constraint_name
			WHERE tc.table_name = $1 AND tc.constraint_type = 'PRIMARY KEY'
		) pk ON c.column_name = pk.column_name
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku ON tc.constraint_name = ku.constraint_name
			WHERE tc.table_name = $1 AND tc.constraint_type = 'FOREIGN KEY'
		) fk ON c.column_name = fk.column_name
		WHERE c.table_name = $1 AND c.table_schema = 'public'
		ORDER BY c.ordinal_position`

	rows, err := s.db.Query(ctx, columnsQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query table columns: %w", err)
	}
	defer rows.Close()

	var columns []interfaces.Column
	var primaryKeys []string

	for rows.Next() {
		var col interfaces.Column
		var defaultValue sql.NullString
		var maxLength sql.NullInt32

		err := rows.Scan(
			&col.Name,
			&col.Type,
			&col.Nullable,
			&defaultValue,
			&col.IsPrimaryKey,
			&col.IsForeignKey,
			&maxLength,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}

		if defaultValue.Valid {
			col.DefaultValue = defaultValue.String
		}

		if maxLength.Valid {
			length := int(maxLength.Int32)
			col.MaxLength = &length
		}

		if col.IsPrimaryKey {
			primaryKeys = append(primaryKeys, col.Name)
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating column rows: %w", err)
	}

	// Get foreign keys information
	foreignKeysQuery := `
		SELECT 
			kcu.column_name,
			ccu.table_name AS referenced_table,
			ccu.column_name AS referenced_column,
			tc.constraint_name,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON tc.constraint_name = ccu.constraint_name
		JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
		WHERE tc.table_name = $1 AND tc.constraint_type = 'FOREIGN KEY'`

	fkRows, err := s.db.Query(ctx, foreignKeysQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query foreign keys: %w", err)
	}
	defer fkRows.Close()

	var foreignKeys []interfaces.ForeignKey
	for fkRows.Next() {
		var fk interfaces.ForeignKey
		err := fkRows.Scan(
			&fk.ColumnName,
			&fk.ReferencedTable,
			&fk.ReferencedColumn,
			&fk.ConstraintName,
			&fk.OnDelete,
			&fk.OnUpdate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan foreign key info: %w", err)
		}
		foreignKeys = append(foreignKeys, fk)
	}

	// Get indexes information
	indexesQuery := `
		SELECT 
			i.relname as index_name,
			array_agg(a.attname ORDER BY c.ordinality) as column_names,
			ix.indisunique as is_unique,
			am.amname as index_type
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_am am ON i.relam = am.oid
		JOIN unnest(ix.indkey) WITH ORDINALITY AS c(attnum, ordinality) ON true
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = c.attnum
		WHERE t.relname = $1 AND t.relkind = 'r'
		GROUP BY i.relname, ix.indisunique, am.amname
		ORDER BY i.relname`

	idxRows, err := s.db.Query(ctx, indexesQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer idxRows.Close()

	var indexes []interfaces.Index
	for idxRows.Next() {
		var idx interfaces.Index
		var columnNames []string
		err := idxRows.Scan(&idx.Name, &columnNames, &idx.IsUnique, &idx.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index info: %w", err)
		}
		idx.Columns = columnNames
		indexes = append(indexes, idx)
	}

	schema := &interfaces.TableSchema{
		Name:        tableName,
		Schema:      "public",
		Columns:     columns,
		PrimaryKeys: primaryKeys,
		ForeignKeys: foreignKeys,
		Indexes:     indexes,
	}

	return schema, nil
}

// ListRecords returns paginated records from a table
func (s *DatabaseService) ListRecords(ctx context.Context, tableName string, params interfaces.ListRecordsParams) (*interfaces.PaginatedRecords, error) {
	// Validate table name to prevent SQL injection
	if !s.isValidTableName(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	// Set default pagination
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}

	// Build WHERE clause for search and filters
	whereClause, args := s.buildWhereClause(params.Search, params.Filters)
	
	// Build ORDER BY clause
	orderClause := s.buildOrderClause(params.SortBy, params.SortDesc)

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", tableName, whereClause)
	var totalCount int
	err := s.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count records: %w", err)
	}

	// Calculate pagination
	offset := (params.Page - 1) * params.PageSize
	totalPages := (totalCount + params.PageSize - 1) / params.PageSize

	// Query records
	query := fmt.Sprintf(
		"SELECT * FROM %s %s %s LIMIT $%d OFFSET $%d",
		tableName, whereClause, orderClause, len(args)+1, len(args)+2,
	)
	args = append(args, params.PageSize, offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	// Get column descriptions
	fieldDescriptions := rows.FieldDescriptions()
	
	var records []interfaces.TableRecord
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to scan row values: %w", err)
		}

		// Convert values to map
		data := make(map[string]interface{})
		primaryKey := make(map[string]interface{})
		
		for i, value := range values {
			columnName := fieldDescriptions[i].Name
			data[columnName] = s.convertValue(value)
			
			// Assume first column is primary key for now
			// In a real implementation, we'd get this from schema
			if i == 0 {
				primaryKey[columnName] = s.convertValue(value)
			}
		}

		record := interfaces.TableRecord{
			TableName: tableName,
			Data:      data,
			Metadata: interfaces.RecordMetadata{
				PrimaryKey: primaryKey,
			},
		}

		// Set timestamps if they exist
		if createdAt, ok := data["created_at"]; ok {
			if t, ok := createdAt.(time.Time); ok {
				record.Metadata.CreatedAt = &t
			}
		}
		if updatedAt, ok := data["updated_at"]; ok {
			if t, ok := updatedAt.(time.Time); ok {
				record.Metadata.UpdatedAt = &t
			}
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating record rows: %w", err)
	}

	// Get table schema for response
	schema, err := s.GetTableSchema(ctx, tableName)
	if err != nil {
		// Don't fail the request if schema fetch fails
		schema = nil
	}

	pagination := interfaces.PaginationInfo{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: totalCount,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}

	return &interfaces.PaginatedRecords{
		Records:    records,
		Pagination: pagination,
		Schema:     schema,
	}, nil
}

// GetRecord returns a specific record by ID
func (s *DatabaseService) GetRecord(ctx context.Context, tableName string, recordID interface{}) (*interfaces.TableRecord, error) {
	if !s.isValidTableName(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	// Get table schema to find primary key
	schema, err := s.GetTableSchema(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table schema: %w", err)
	}

	if len(schema.PrimaryKeys) == 0 {
		return nil, fmt.Errorf("table %s has no primary key", tableName)
	}

	// For simplicity, assume single column primary key
	primaryKeyColumn := schema.PrimaryKeys[0]
	
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", tableName, primaryKeyColumn)
	row := s.db.QueryRow(ctx, query, recordID)

	// Get column names from schema
	columnNames := make([]string, len(schema.Columns))
	for i, col := range schema.Columns {
		columnNames[i] = col.Name
	}

	// Scan into interface slice
	values := make([]interface{}, len(columnNames))
	valuePtrs := make([]interface{}, len(columnNames))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	err = row.Scan(valuePtrs...)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("record not found")
		}
		return nil, fmt.Errorf("failed to scan record: %w", err)
	}

	// Convert to map
	data := make(map[string]interface{})
	primaryKey := make(map[string]interface{})
	
	for i, value := range values {
		columnName := columnNames[i]
		convertedValue := s.convertValue(value)
		data[columnName] = convertedValue
		
		if columnName == primaryKeyColumn {
			primaryKey[columnName] = convertedValue
		}
	}

	record := &interfaces.TableRecord{
		TableName: tableName,
		Data:      data,
		Metadata: interfaces.RecordMetadata{
			PrimaryKey: primaryKey,
		},
	}

	// Set timestamps if they exist
	if createdAt, ok := data["created_at"]; ok {
		if t, ok := createdAt.(time.Time); ok {
			record.Metadata.CreatedAt = &t
		}
	}
	if updatedAt, ok := data["updated_at"]; ok {
		if t, ok := updatedAt.(time.Time); ok {
			record.Metadata.UpdatedAt = &t
		}
	}

	return record, nil
}

// CreateRecord creates a new record in a table
func (s *DatabaseService) CreateRecord(ctx context.Context, tableName string, data map[string]interface{}) (*interfaces.TableRecord, error) {
	if !s.isValidTableName(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	// Get table schema for validation
	schema, err := s.GetTableSchema(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table schema: %w", err)
	}

	// Validate and prepare data
	validatedData, err := s.validateRecordData(schema, data, false)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Build INSERT query
	columns := make([]string, 0, len(validatedData))
	placeholders := make([]string, 0, len(validatedData))
	values := make([]interface{}, 0, len(validatedData))
	
	i := 1
	for column, value := range validatedData {
		columns = append(columns, column)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, value)
		i++
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	row := s.db.QueryRow(ctx, query, values...)

	// Scan the returned record
	columnNames := make([]string, len(schema.Columns))
	for i, col := range schema.Columns {
		columnNames[i] = col.Name
	}

	scanValues := make([]interface{}, len(columnNames))
	scanValuePtrs := make([]interface{}, len(columnNames))
	for i := range scanValues {
		scanValuePtrs[i] = &scanValues[i]
	}

	err = row.Scan(scanValuePtrs...)
	if err != nil {
		return nil, fmt.Errorf("failed to scan created record: %w", err)
	}

	// Convert to record
	recordData := make(map[string]interface{})
	primaryKey := make(map[string]interface{})
	
	for i, value := range scanValues {
		columnName := columnNames[i]
		convertedValue := s.convertValue(value)
		recordData[columnName] = convertedValue
		
		// Check if this is a primary key column
		for _, pkCol := range schema.PrimaryKeys {
			if columnName == pkCol {
				primaryKey[columnName] = convertedValue
				break
			}
		}
	}

	record := &interfaces.TableRecord{
		TableName: tableName,
		Data:      recordData,
		Metadata: interfaces.RecordMetadata{
			PrimaryKey: primaryKey,
		},
	}

	// Set timestamps
	if createdAt, ok := recordData["created_at"]; ok {
		if t, ok := createdAt.(time.Time); ok {
			record.Metadata.CreatedAt = &t
		}
	}
	if updatedAt, ok := recordData["updated_at"]; ok {
		if t, ok := updatedAt.(time.Time); ok {
			record.Metadata.UpdatedAt = &t
		}
	}

	return record, nil
}

// UpdateRecord updates an existing record
func (s *DatabaseService) UpdateRecord(ctx context.Context, tableName string, recordID interface{}, data map[string]interface{}) (*interfaces.TableRecord, error) {
	if !s.isValidTableName(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	// Get table schema
	schema, err := s.GetTableSchema(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table schema: %w", err)
	}

	if len(schema.PrimaryKeys) == 0 {
		return nil, fmt.Errorf("table %s has no primary key", tableName)
	}

	// Validate data
	validatedData, err := s.validateRecordData(schema, data, true)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Add updated_at if column exists
	if s.hasColumn(schema, "updated_at") {
		validatedData["updated_at"] = time.Now()
	}

	// Build UPDATE query
	setParts := make([]string, 0, len(validatedData))
	values := make([]interface{}, 0, len(validatedData)+1)
	
	i := 1
	for column, value := range validatedData {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", column, i))
		values = append(values, value)
		i++
	}

	primaryKeyColumn := schema.PrimaryKeys[0] // Assume single column PK
	values = append(values, recordID)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d RETURNING *",
		tableName,
		strings.Join(setParts, ", "),
		primaryKeyColumn,
		i,
	)

	row := s.db.QueryRow(ctx, query, values...)

	// Scan the updated record
	columnNames := make([]string, len(schema.Columns))
	for i, col := range schema.Columns {
		columnNames[i] = col.Name
	}

	scanValues := make([]interface{}, len(columnNames))
	scanValuePtrs := make([]interface{}, len(columnNames))
	for i := range scanValues {
		scanValuePtrs[i] = &scanValues[i]
	}

	err = row.Scan(scanValuePtrs...)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("record not found")
		}
		return nil, fmt.Errorf("failed to scan updated record: %w", err)
	}

	// Convert to record
	recordData := make(map[string]interface{})
	primaryKey := make(map[string]interface{})
	
	for i, value := range scanValues {
		columnName := columnNames[i]
		convertedValue := s.convertValue(value)
		recordData[columnName] = convertedValue
		
		for _, pkCol := range schema.PrimaryKeys {
			if columnName == pkCol {
				primaryKey[columnName] = convertedValue
				break
			}
		}
	}

	record := &interfaces.TableRecord{
		TableName: tableName,
		Data:      recordData,
		Metadata: interfaces.RecordMetadata{
			PrimaryKey: primaryKey,
		},
	}

	// Set timestamps
	if createdAt, ok := recordData["created_at"]; ok {
		if t, ok := createdAt.(time.Time); ok {
			record.Metadata.CreatedAt = &t
		}
	}
	if updatedAt, ok := recordData["updated_at"]; ok {
		if t, ok := updatedAt.(time.Time); ok {
			record.Metadata.UpdatedAt = &t
		}
	}

	return record, nil
}

// DeleteRecord deletes a record from a table
func (s *DatabaseService) DeleteRecord(ctx context.Context, tableName string, recordID interface{}) error {
	if !s.isValidTableName(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	// Get table schema
	schema, err := s.GetTableSchema(ctx, tableName)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	if len(schema.PrimaryKeys) == 0 {
		return fmt.Errorf("table %s has no primary key", tableName)
	}

	primaryKeyColumn := schema.PrimaryKeys[0] // Assume single column PK
	
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", tableName, primaryKeyColumn)
	result, err := s.db.Exec(ctx, query, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("record not found")
	}

	return nil
}

// BulkOperation performs bulk operations on records
func (s *DatabaseService) BulkOperation(ctx context.Context, tableName string, operation interfaces.BulkOperation) (*interfaces.BulkOperationResult, error) {
	if !s.isValidTableName(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	result := &interfaces.BulkOperationResult{
		Operation: operation.Operation,
	}

	// Start transaction for bulk operation
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	switch operation.Operation {
	case "update":
		err = s.performBulkUpdate(ctx, tx, tableName, operation, result)
	case "delete":
		err = s.performBulkDelete(ctx, tx, tableName, operation, result)
	default:
		return nil, fmt.Errorf("unsupported bulk operation: %s", operation.Operation)
	}

	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	result.SuccessCount = result.AffectedRows
	result.TotalRecords = len(operation.RecordIDs)

	return result, nil
}

// Helper methods

func (s *DatabaseService) isValidTableName(tableName string) bool {
	// Simple validation - in production, you'd want more robust validation
	validTables := map[string]bool{
		"users":     true,
		"accounts":  true,
		"transfers": true,
	}
	return validTables[tableName]
}

func (s *DatabaseService) buildWhereClause(search string, filters map[string]interface{}) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add search condition (simple text search across all text columns)
	if search != "" {
		// This is a simplified search - in production you'd want more sophisticated search
		searchCondition := fmt.Sprintf("(CAST(id AS TEXT) ILIKE $%d OR email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)", 
			argIndex, argIndex, argIndex, argIndex)
		conditions = append(conditions, searchCondition)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	// Add filter conditions
	for column, value := range filters {
		if value != nil {
			conditions = append(conditions, fmt.Sprintf("%s = $%d", column, argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	if len(conditions) == 0 {
		return "", args
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

func (s *DatabaseService) buildOrderClause(sortBy string, sortDesc bool) string {
	if sortBy == "" {
		sortBy = "id" // Default sort by id
	}

	direction := "ASC"
	if sortDesc {
		direction = "DESC"
	}

	return fmt.Sprintf("ORDER BY %s %s", sortBy, direction)
}

func (s *DatabaseService) convertValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// Handle different types that might come from the database
	switch v := value.(type) {
	case []byte:
		return string(v)
	case time.Time:
		return v
	default:
		return v
	}
}

func (s *DatabaseService) validateRecordData(schema *interfaces.TableSchema, data map[string]interface{}, isUpdate bool) (map[string]interface{}, error) {
	validatedData := make(map[string]interface{})

	for column, value := range data {
		// Find column in schema
		var columnInfo *interfaces.Column
		for _, col := range schema.Columns {
			if col.Name == column {
				columnInfo = &col
				break
			}
		}

		if columnInfo == nil {
			return nil, fmt.Errorf("unknown column: %s", column)
		}

		// Skip primary key columns in updates
		if isUpdate && columnInfo.IsPrimaryKey {
			continue
		}

		// Skip auto-generated columns
		if column == "created_at" || column == "updated_at" {
			continue
		}

		// Validate nullable
		if !columnInfo.Nullable && value == nil {
			return nil, fmt.Errorf("column %s cannot be null", column)
		}

		// Basic type validation (simplified)
		if value != nil {
			switch columnInfo.Type {
			case "integer", "bigint":
				if _, ok := value.(int); !ok {
					if str, ok := value.(string); ok {
						if intVal, err := strconv.Atoi(str); err == nil {
							value = intVal
						} else {
							return nil, fmt.Errorf("column %s must be an integer", column)
						}
					} else {
						return nil, fmt.Errorf("column %s must be an integer", column)
					}
				}
			case "character varying", "text":
				if _, ok := value.(string); !ok {
					return nil, fmt.Errorf("column %s must be a string", column)
				}
			case "boolean":
				if _, ok := value.(bool); !ok {
					if str, ok := value.(string); ok {
						if boolVal, err := strconv.ParseBool(str); err == nil {
							value = boolVal
						} else {
							return nil, fmt.Errorf("column %s must be a boolean", column)
						}
					} else {
						return nil, fmt.Errorf("column %s must be a boolean", column)
					}
				}
			}
		}

		validatedData[column] = value
	}

	return validatedData, nil
}

func (s *DatabaseService) hasColumn(schema *interfaces.TableSchema, columnName string) bool {
	for _, col := range schema.Columns {
		if col.Name == columnName {
			return true
		}
	}
	return false
}

func (s *DatabaseService) performBulkUpdate(ctx context.Context, tx pgx.Tx, tableName string, operation interfaces.BulkOperation, result *interfaces.BulkOperationResult) error {
	if len(operation.Data) == 0 {
		return fmt.Errorf("no data provided for bulk update")
	}

	// Get table schema
	schema, err := s.GetTableSchema(ctx, tableName)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	if len(schema.PrimaryKeys) == 0 {
		return fmt.Errorf("table %s has no primary key", tableName)
	}

	primaryKeyColumn := schema.PrimaryKeys[0]

	// Validate update data
	validatedData, err := s.validateRecordData(schema, operation.Data, true)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Add updated_at if column exists
	if s.hasColumn(schema, "updated_at") {
		validatedData["updated_at"] = time.Now()
	}

	// Build SET clause
	setParts := make([]string, 0, len(validatedData))
	setValues := make([]interface{}, 0, len(validatedData))
	
	i := 1
	for column, value := range validatedData {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", column, i))
		setValues = append(setValues, value)
		i++
	}

	// Build WHERE clause for record IDs
	if len(operation.RecordIDs) > 0 {
		// Update specific records
		for _, recordID := range operation.RecordIDs {
			query := fmt.Sprintf(
				"UPDATE %s SET %s WHERE %s = $%d",
				tableName,
				strings.Join(setParts, ", "),
				primaryKeyColumn,
				i,
			)
			
			args := append(setValues, recordID)
			execResult, err := tx.Exec(ctx, query, args...)
			if err != nil {
				result.Errors = append(result.Errors, interfaces.BulkOperationError{
					RecordID: recordID,
					Error:    err.Error(),
				})
				result.ErrorCount++
			} else {
				result.AffectedRows += int(execResult.RowsAffected())
			}
		}
	} else if len(operation.Filters) > 0 {
		// Update records matching filters
		whereClause, whereArgs := s.buildWhereClause("", operation.Filters)
		
		query := fmt.Sprintf(
			"UPDATE %s SET %s %s",
			tableName,
			strings.Join(setParts, ", "),
			whereClause,
		)
		
		args := append(setValues, whereArgs...)
		execResult, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to execute bulk update: %w", err)
		}
		result.AffectedRows = int(execResult.RowsAffected())
	}

	return nil
}

func (s *DatabaseService) performBulkDelete(ctx context.Context, tx pgx.Tx, tableName string, operation interfaces.BulkOperation, result *interfaces.BulkOperationResult) error {
	// Get table schema
	schema, err := s.GetTableSchema(ctx, tableName)
	if err != nil {
		return fmt.Errorf("failed to get table schema: %w", err)
	}

	if len(schema.PrimaryKeys) == 0 {
		return fmt.Errorf("table %s has no primary key", tableName)
	}

	primaryKeyColumn := schema.PrimaryKeys[0]

	if len(operation.RecordIDs) > 0 {
		// Delete specific records
		for _, recordID := range operation.RecordIDs {
			query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", tableName, primaryKeyColumn)
			
			execResult, err := tx.Exec(ctx, query, recordID)
			if err != nil {
				result.Errors = append(result.Errors, interfaces.BulkOperationError{
					RecordID: recordID,
					Error:    err.Error(),
				})
				result.ErrorCount++
			} else {
				result.AffectedRows += int(execResult.RowsAffected())
			}
		}
	} else if len(operation.Filters) > 0 {
		// Delete records matching filters
		whereClause, args := s.buildWhereClause("", operation.Filters)
		
		query := fmt.Sprintf("DELETE FROM %s %s", tableName, whereClause)
		
		execResult, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to execute bulk delete: %w", err)
		}
		result.AffectedRows = int(execResult.RowsAffected())
	}

	return nil
}