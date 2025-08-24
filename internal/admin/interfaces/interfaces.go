package interfaces

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// AdminAuthService defines the interface for admin authentication
type AdminAuthService interface {
	// Login authenticates admin credentials and returns a session token
	Login(ctx context.Context, username, password string) (*AdminSession, error)
	
	// ValidateSession validates a PASETO token and returns session info
	ValidateSession(ctx context.Context, token string) (*AdminSession, error)
	
	// RefreshSession extends the session expiration
	RefreshSession(ctx context.Context, token string) (*AdminSession, error)
	
	// Logout invalidates a session
	Logout(ctx context.Context, token string) error
	
	// UpdateCredentials changes admin credentials
	UpdateCredentials(ctx context.Context, username, oldPassword, newPassword string) error
}

// UserManagementService defines the interface for user management operations
type UserManagementService interface {
	// ListUsers returns paginated list of users with optional filtering
	ListUsers(ctx context.Context, params ListUsersParams) (*PaginatedUsers, error)
	
	// GetUser returns detailed user information
	GetUser(ctx context.Context, userID string) (*UserDetail, error)
	
	// CreateUser creates a new user account
	CreateUser(ctx context.Context, req CreateUserRequest) (*UserDetail, error)
	
	// UpdateUser updates user information
	UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*UserDetail, error)
	
	// DisableUser disables a user account
	DisableUser(ctx context.Context, userID string) error
	
	// EnableUser enables a user account
	EnableUser(ctx context.Context, userID string) error
	
	// DeleteUser deletes a user account
	DeleteUser(ctx context.Context, userID string) error
}

// SystemMonitoringService defines the interface for system monitoring
type SystemMonitoringService interface {
	// GetSystemHealth returns current system health status
	GetSystemHealth(ctx context.Context) (*SystemHealth, error)
	
	// GetMetrics returns system performance metrics
	GetMetrics(ctx context.Context, timeRange TimeRange) (*SystemMetrics, error)
	
	// GetAlerts returns system alerts
	GetAlerts(ctx context.Context, params AlertParams) (*PaginatedAlerts, error)
	
	// AcknowledgeAlert marks an alert as acknowledged
	AcknowledgeAlert(ctx context.Context, alertID string) error
	
	// ResolveAlert marks an alert as resolved
	ResolveAlert(ctx context.Context, alertID string, notes string) error
}

// DatabaseService defines the interface for database operations
type DatabaseService interface {
	// ListTables returns available database tables
	ListTables(ctx context.Context) ([]TableInfo, error)
	
	// GetTableSchema returns table structure information
	GetTableSchema(ctx context.Context, tableName string) (*TableSchema, error)
	
	// ListRecords returns paginated records from a table
	ListRecords(ctx context.Context, tableName string, params ListRecordsParams) (*PaginatedRecords, error)
	
	// GetRecord returns a specific record by ID
	GetRecord(ctx context.Context, tableName string, recordID interface{}) (*TableRecord, error)
	
	// CreateRecord creates a new record in a table
	CreateRecord(ctx context.Context, tableName string, data map[string]interface{}) (*TableRecord, error)
	
	// UpdateRecord updates an existing record
	UpdateRecord(ctx context.Context, tableName string, recordID interface{}, data map[string]interface{}) (*TableRecord, error)
	
	// DeleteRecord deletes a record from a table
	DeleteRecord(ctx context.Context, tableName string, recordID interface{}) error
	
	// BulkOperation performs bulk operations on records
	BulkOperation(ctx context.Context, tableName string, operation BulkOperation) (*BulkOperationResult, error)
}

// NotificationService defines the interface for real-time notifications
type NotificationService interface {
	// Subscribe adds a WebSocket connection to receive notifications
	Subscribe(ctx context.Context, conn *websocket.Conn, adminID string) error
	
	// Unsubscribe removes a WebSocket connection
	Unsubscribe(adminID string, conn *websocket.Conn) error
	
	// Broadcast sends a notification to all connected admins
	Broadcast(ctx context.Context, notification *Notification) error
	
	// SendToAdmin sends a notification to a specific admin
	SendToAdmin(ctx context.Context, adminID string, notification *Notification) error
	
	// GetConnectionCount returns the number of active connections
	GetConnectionCount() int
}

// AlertService defines the interface for alert management
type AlertService interface {
	// CreateAlert creates a new alert and broadcasts it
	CreateAlert(ctx context.Context, severity, title, message, source string, metadata map[string]interface{}) (*Alert, error)
	
	// GetAlert retrieves a specific alert by ID
	GetAlert(ctx context.Context, alertID string) (*Alert, error)
	
	// ListAlerts returns paginated alerts with filtering
	ListAlerts(ctx context.Context, params AlertParams) (*PaginatedAlerts, error)
	
	// SearchAlerts searches alerts with text search and filtering
	SearchAlerts(ctx context.Context, searchText string, params AlertParams) (*PaginatedAlerts, error)
	
	// AcknowledgeAlert marks an alert as acknowledged
	AcknowledgeAlert(ctx context.Context, alertID, acknowledgedBy string) (*Alert, error)
	
	// ResolveAlert marks an alert as resolved
	ResolveAlert(ctx context.Context, alertID, resolvedBy, notes string) (*Alert, error)
	
	// GetAlertStatistics returns alert statistics for a time range
	GetAlertStatistics(ctx context.Context, timeRange *TimeRange) (*AlertStatistics, error)
	
	// GetUnresolvedAlertsCount returns the count of unresolved alerts
	GetUnresolvedAlertsCount(ctx context.Context) (int, error)
	
	// GetAlertsBySource returns alerts from a specific source
	GetAlertsBySource(ctx context.Context, source string, limit int) ([]Alert, error)
	
	// CleanupOldResolvedAlerts removes old resolved alerts
	CleanupOldResolvedAlerts(ctx context.Context, olderThan time.Time) error
}

// TransactionService defines the interface for transaction management
type TransactionService interface {
	// SearchTransactions returns transactions based on search criteria
	SearchTransactions(ctx context.Context, params SearchTransactionParams) (*PaginatedTransactions, error)
	
	// GetTransactionDetail returns detailed transaction information
	GetTransactionDetail(ctx context.Context, transactionID string) (*TransactionDetail, error)
	
	// ReverseTransaction reverses a transaction
	ReverseTransaction(ctx context.Context, transactionID string, reason string) (*TransactionDetail, error)
	
	// GetAccountTransactions returns transactions for a specific account
	GetAccountTransactions(ctx context.Context, accountID string, params PaginationParams) (*PaginatedTransactions, error)
}

// AccountService defines the interface for account management
type AccountService interface {
	// SearchAccounts returns accounts based on search criteria
	SearchAccounts(ctx context.Context, params SearchAccountParams) (*PaginatedAccounts, error)
	
	// GetAccountDetail returns detailed account information
	GetAccountDetail(ctx context.Context, accountID string) (*AccountDetail, error)
	
	// FreezeAccount freezes an account
	FreezeAccount(ctx context.Context, accountID string, reason string) error
	
	// UnfreezeAccount unfreezes an account
	UnfreezeAccount(ctx context.Context, accountID string) error
	
	// AdjustBalance adjusts an account balance
	AdjustBalance(ctx context.Context, accountID string, adjustment string, reason string) (*AccountDetail, error)
}

// AdminHandler defines the interface for HTTP handlers
type AdminHandler interface {
	// RegisterRoutes registers HTTP routes for this handler
	RegisterRoutes(router gin.IRouter)
}

// AuthHandler defines authentication-related HTTP handlers
type AuthHandler interface {
	AdminHandler
	Login(c *gin.Context)
	Logout(c *gin.Context)
	ValidateSession(c *gin.Context)
	UpdateCredentials(c *gin.Context)
}

// UserHandler defines user management HTTP handlers
type UserHandler interface {
	AdminHandler
	ListUsers(c *gin.Context)
	GetUser(c *gin.Context)
	CreateUser(c *gin.Context)
	UpdateUser(c *gin.Context)
	DisableUser(c *gin.Context)
	EnableUser(c *gin.Context)
	DeleteUser(c *gin.Context)
}

// SystemHandler defines system monitoring HTTP handlers
type SystemHandler interface {
	AdminHandler
	GetHealth(c *gin.Context)
	GetMetrics(c *gin.Context)
	GetAlerts(c *gin.Context)
	AcknowledgeAlert(c *gin.Context)
	ResolveAlert(c *gin.Context)
}

// DatabaseHandler defines database management HTTP handlers
type DatabaseHandler interface {
	AdminHandler
	ListTables(c *gin.Context)
	GetTableSchema(c *gin.Context)
	ListRecords(c *gin.Context)
	GetRecord(c *gin.Context)
	CreateRecord(c *gin.Context)
	UpdateRecord(c *gin.Context)
	DeleteRecord(c *gin.Context)
	BulkOperation(c *gin.Context)
}

// WebSocketHandler defines WebSocket connection handlers
type WebSocketHandler interface {
	AdminHandler
	HandleConnection(c *gin.Context)
	HandleNotifications(c *gin.Context)
}

// TransactionHandler defines transaction management HTTP handlers
type TransactionHandler interface {
	AdminHandler
	SearchTransactions(c *gin.Context)
	GetTransactionDetail(c *gin.Context)
	ReverseTransaction(c *gin.Context)
	GetAccountTransactions(c *gin.Context)
}

// AccountHandler defines account management HTTP handlers
type AccountHandler interface {
	AdminHandler
	SearchAccounts(c *gin.Context)
	GetAccountDetail(c *gin.Context)
	FreezeAccount(c *gin.Context)
	UnfreezeAccount(c *gin.Context)
	AdjustBalance(c *gin.Context)
}

// AlertHandler defines alert management HTTP handlers
type AlertHandler interface {
	AdminHandler
	CreateAlert(c *gin.Context)
	GetAlert(c *gin.Context)
	ListAlerts(c *gin.Context)
	SearchAlerts(c *gin.Context)
	AcknowledgeAlert(c *gin.Context)
	ResolveAlert(c *gin.Context)
	GetAlertStatistics(c *gin.Context)
	GetAlertsBySource(c *gin.Context)
	CleanupOldResolvedAlerts(c *gin.Context)
}

// AdminMiddleware defines the interface for admin-specific middleware
type AdminMiddleware interface {
	// Handler returns the Gin middleware handler function
	Handler() gin.HandlerFunc
}

// AuthMiddleware defines authentication middleware
type AuthMiddleware interface {
	AdminMiddleware
	// RequireAuth ensures the request has valid authentication
	RequireAuth() gin.HandlerFunc
	// OptionalAuth extracts auth info if present but doesn't require it
	OptionalAuth() gin.HandlerFunc
}

// CORSMiddleware defines CORS middleware
type CORSMiddleware interface {
	AdminMiddleware
}

// LoggingMiddleware defines request logging middleware
type LoggingMiddleware interface {
	AdminMiddleware
}

// ErrorMiddleware defines error handling middleware
type ErrorMiddleware interface {
	AdminMiddleware
}

// RateLimitMiddleware defines rate limiting middleware
type RateLimitMiddleware interface {
	AdminMiddleware
}

// Data Transfer Objects and Models

// AdminSession represents an admin authentication session
type AdminSession struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	PasetoToken string    `json:"paseto_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
	LastActive  time.Time `json:"last_active"`
}

// UserDetail represents detailed user information
type UserDetail struct {
	ID              string                 `json:"id"`
	Email           string                 `json:"email"`
	FirstName       string                 `json:"first_name"`
	LastName        string                 `json:"last_name"`
	IsActive        bool                   `json:"is_active"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	LastLogin       *time.Time             `json:"last_login"`
	AccountCount    int                    `json:"account_count"`
	TransferCount   int                    `json:"transfer_count"`
	WelcomeEmailSent bool                  `json:"welcome_email_sent"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// SystemHealth represents system health status
type SystemHealth struct {
	Status      string                 `json:"status"` // healthy, warning, critical
	Timestamp   time.Time              `json:"timestamp"`
	Services    map[string]ServiceHealth `json:"services"`
	Metrics     SystemMetricsSnapshot  `json:"metrics"`
	AlertCount  int                    `json:"alert_count"`
}

// ServiceHealth represents individual service health
type ServiceHealth struct {
	Status      string    `json:"status"`
	LastCheck   time.Time `json:"last_check"`
	ResponseTime time.Duration `json:"response_time"`
	Error       string    `json:"error,omitempty"`
}

// SystemMetricsSnapshot represents current system metrics
type SystemMetricsSnapshot struct {
	CPUUsage        float64 `json:"cpu_usage"`
	MemoryUsage     float64 `json:"memory_usage"`
	DBConnections   int     `json:"db_connections"`
	APIResponseTime float64 `json:"api_response_time"`
	ActiveSessions  int     `json:"active_sessions"`
}

// Pagination and filtering parameters
type PaginationParams struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

type ListUsersParams struct {
	PaginationParams
	Search   string `json:"search" form:"search"`
	IsActive *bool  `json:"is_active" form:"is_active"`
	SortBy   string `json:"sort_by" form:"sort_by"`
	SortDesc bool   `json:"sort_desc" form:"sort_desc"`
}

// Request/Response types
type CreateUserRequest struct {
	Email     string `json:"email" binding:"required,email"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Password  string `json:"password" binding:"required,min=8"`
}

type UpdateUserRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	IsActive  *bool   `json:"is_active"`
}

// Paginated response types
type PaginatedUsers struct {
	Users      []UserDetail `json:"users"`
	Pagination PaginationInfo `json:"pagination"`
}

type PaginationInfo struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// Additional types for other interfaces would be defined here...
// (Alert, Notification, TableInfo, etc.)

// Alert represents a system alert
type Alert struct {
	ID           string                 `json:"id"`
	Severity     string                 `json:"severity"` // critical, warning, info
	Title        string                 `json:"title"`
	Message      string                 `json:"message"`
	Source       string                 `json:"source"`
	Timestamp    time.Time              `json:"timestamp"`
	Acknowledged bool                   `json:"acknowledged"`
	AcknowledgedBy string               `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time           `json:"acknowledged_at,omitempty"`
	Resolved     bool                   `json:"resolved"`
	ResolvedBy   string                 `json:"resolved_by,omitempty"`
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty"`
	ResolvedNotes string                `json:"resolved_notes,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Notification represents a real-time notification
type Notification struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// TableInfo represents database table information
type TableInfo struct {
	Name        string `json:"name"`
	Schema      string `json:"schema"`
	RecordCount int64  `json:"record_count"`
	Description string `json:"description,omitempty"`
}

// TableSchema represents database table structure
type TableSchema struct {
	Name        string       `json:"name"`
	Schema      string       `json:"schema"`
	Columns     []Column     `json:"columns"`
	PrimaryKeys []string     `json:"primary_keys"`
	ForeignKeys []ForeignKey `json:"foreign_keys"`
	Indexes     []Index      `json:"indexes"`
}

// Column represents a database column
type Column struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Nullable     bool        `json:"nullable"`
	DefaultValue interface{} `json:"default_value"`
	IsPrimaryKey bool        `json:"is_primary_key"`
	IsForeignKey bool        `json:"is_foreign_key"`
	MaxLength    *int        `json:"max_length,omitempty"`
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	ColumnName          string `json:"column_name"`
	ReferencedTable     string `json:"referenced_table"`
	ReferencedColumn    string `json:"referenced_column"`
	ConstraintName      string `json:"constraint_name"`
	OnDelete            string `json:"on_delete"`
	OnUpdate            string `json:"on_update"`
}

// Index represents a database index
type Index struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns"`
	IsUnique bool     `json:"is_unique"`
	Type     string   `json:"type"`
}

// TableRecord represents a database record
type TableRecord struct {
	TableName string                 `json:"table_name"`
	Data      map[string]interface{} `json:"data"`
	Metadata  RecordMetadata         `json:"metadata"`
}

// RecordMetadata represents metadata about a database record
type RecordMetadata struct {
	PrimaryKey map[string]interface{} `json:"primary_key"`
	CreatedAt  *time.Time             `json:"created_at,omitempty"`
	UpdatedAt  *time.Time             `json:"updated_at,omitempty"`
	Version    *int                   `json:"version,omitempty"`
}

// TransactionDetail represents detailed transaction information
type TransactionDetail struct {
	ID              string                 `json:"id"`
	FromAccountID   string                 `json:"from_account_id"`
	ToAccountID     string                 `json:"to_account_id"`
	Amount          string                 `json:"amount"` // Decimal as string
	Currency        string                 `json:"currency"`
	Description     string                 `json:"description"`
	Status          string                 `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	ProcessedAt     *time.Time             `json:"processed_at,omitempty"`
	ReversedAt      *time.Time             `json:"reversed_at,omitempty"`
	ReversalReason  string                 `json:"reversal_reason,omitempty"`
	FromAccount     *AccountSummary        `json:"from_account,omitempty"`
	ToAccount       *AccountSummary        `json:"to_account,omitempty"`
	AuditTrail      []AuditEntry           `json:"audit_trail"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// AccountSummary represents basic account information
type AccountSummary struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Currency string `json:"currency"`
	Balance  string `json:"balance"` // Decimal as string
	IsActive bool   `json:"is_active"`
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	Action    string                 `json:"action"`
	Actor     string                 `json:"actor"`
	ActorType string                 `json:"actor_type"` // user, admin, system
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
}

// Search and filter parameters
type AlertParams struct {
	PaginationParams
	Severity     string     `json:"severity" form:"severity"`
	Acknowledged *bool      `json:"acknowledged" form:"acknowledged"`
	Resolved     *bool      `json:"resolved" form:"resolved"`
	Source       string     `json:"source" form:"source"`
	DateFrom     *time.Time `json:"date_from" form:"date_from"`
	DateTo       *time.Time `json:"date_to" form:"date_to"`
}

type ListRecordsParams struct {
	PaginationParams
	Search    string                 `json:"search" form:"search"`
	Filters   map[string]interface{} `json:"filters" form:"filters"`
	SortBy    string                 `json:"sort_by" form:"sort_by"`
	SortDesc  bool                   `json:"sort_desc" form:"sort_desc"`
}

type SearchTransactionParams struct {
	PaginationParams
	UserID        string     `json:"user_id" form:"user_id"`
	AccountID     string     `json:"account_id" form:"account_id"`
	Currency      string     `json:"currency" form:"currency"`
	Status        string     `json:"status" form:"status"`
	AmountMin     *string    `json:"amount_min" form:"amount_min"`
	AmountMax     *string    `json:"amount_max" form:"amount_max"`
	DateFrom      *time.Time `json:"date_from" form:"date_from"`
	DateTo        *time.Time `json:"date_to" form:"date_to"`
	Description   string     `json:"description" form:"description"`
}

// Bulk operations
type BulkOperation struct {
	Operation string                   `json:"operation"` // update, delete
	Filters   map[string]interface{}   `json:"filters"`
	Data      map[string]interface{}   `json:"data,omitempty"`
	RecordIDs []interface{}            `json:"record_ids,omitempty"`
}

type BulkOperationResult struct {
	Operation     string `json:"operation"`
	TotalRecords  int    `json:"total_records"`
	AffectedRows  int    `json:"affected_rows"`
	SuccessCount  int    `json:"success_count"`
	ErrorCount    int    `json:"error_count"`
	Errors        []BulkOperationError `json:"errors,omitempty"`
}

type BulkOperationError struct {
	RecordID interface{} `json:"record_id"`
	Error    string      `json:"error"`
}

// Time range for metrics
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// System metrics over time
type SystemMetrics struct {
	TimeRange TimeRange              `json:"time_range"`
	Interval  time.Duration          `json:"interval"`
	DataPoints []SystemMetricsSnapshot `json:"data_points"`
}

// Paginated response types
type PaginatedAlerts struct {
	Alerts     []Alert        `json:"alerts"`
	Pagination PaginationInfo `json:"pagination"`
}

type PaginatedRecords struct {
	Records    []TableRecord  `json:"records"`
	Pagination PaginationInfo `json:"pagination"`
	Schema     *TableSchema   `json:"schema,omitempty"`
}

type PaginatedTransactions struct {
	Transactions []TransactionDetail `json:"transactions"`
	Pagination   PaginationInfo      `json:"pagination"`
}

// Account management types
type AccountDetail struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	Currency  string      `json:"currency"`
	Balance   string      `json:"balance"` // Decimal as string
	IsActive  bool        `json:"is_active"`
	IsFrozen  bool        `json:"is_frozen"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	User      UserSummary `json:"user"`
}

type UserSummary struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsActive  bool   `json:"is_active"`
}

type SearchAccountParams struct {
	PaginationParams
	Search     string  `json:"search" form:"search"`
	Currency   string  `json:"currency" form:"currency"`
	BalanceMin *string `json:"balance_min" form:"balance_min"`
	BalanceMax *string `json:"balance_max" form:"balance_max"`
	IsActive   *bool   `json:"is_active" form:"is_active"`
}

type PaginatedAccounts struct {
	Accounts   []AccountDetail `json:"accounts"`
	Pagination PaginationInfo  `json:"pagination"`
}

// AlertStatistics represents alert statistics
type AlertStatistics struct {
	TotalAlerts       int `json:"total_alerts"`
	CriticalCount     int `json:"critical_count"`
	WarningCount      int `json:"warning_count"`
	InfoCount         int `json:"info_count"`
	AcknowledgedCount int `json:"acknowledged_count"`
	ResolvedCount     int `json:"resolved_count"`
	UnresolvedCount   int `json:"unresolved_count"`
}