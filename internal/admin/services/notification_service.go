package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/rs/zerolog/log"
)

// NotificationServiceImpl implements the NotificationService interface
type NotificationServiceImpl struct {
	// connections maps admin ID to their WebSocket connections
	connections map[string][]*websocket.Conn
	// connectionMutex protects the connections map
	connectionMutex sync.RWMutex
	// connectionCount tracks total active connections
	connectionCount int
}

// NewNotificationService creates a new notification service
func NewNotificationService() interfaces.NotificationService {
	return &NotificationServiceImpl{
		connections: make(map[string][]*websocket.Conn),
	}
}

// Subscribe adds a WebSocket connection to receive notifications
func (s *NotificationServiceImpl) Subscribe(ctx context.Context, conn *websocket.Conn, adminID string) error {
	if conn == nil {
		return fmt.Errorf("websocket connection cannot be nil")
	}
	
	if adminID == "" {
		return fmt.Errorf("admin ID cannot be empty")
	}

	s.connectionMutex.Lock()
	defer s.connectionMutex.Unlock()

	// Initialize admin connections slice if it doesn't exist
	if s.connections[adminID] == nil {
		s.connections[adminID] = make([]*websocket.Conn, 0)
	}

	// Add connection to admin's connection list
	s.connections[adminID] = append(s.connections[adminID], conn)
	s.connectionCount++

	log.Info().
		Str("admin_id", adminID).
		Int("total_connections", s.connectionCount).
		Msg("Admin subscribed to notifications")

	// Send welcome notification
	welcomeNotification := &interfaces.Notification{
		ID:        fmt.Sprintf("welcome_%s_%d", adminID, time.Now().Unix()),
		Type:      "system",
		Title:     "Connected",
		Message:   "Successfully connected to admin notifications",
		Severity:  "info",
		Timestamp: time.Now(),
	}

	// Send welcome message to this specific connection
	if err := s.sendToConnection(conn, welcomeNotification); err != nil {
		log.Warn().
			Err(err).
			Str("admin_id", adminID).
			Msg("Failed to send welcome notification")
	}

	return nil
}

// Unsubscribe removes a WebSocket connection
func (s *NotificationServiceImpl) Unsubscribe(adminID string, conn *websocket.Conn) error {
	if conn == nil {
		return fmt.Errorf("websocket connection cannot be nil")
	}

	s.connectionMutex.Lock()
	defer s.connectionMutex.Unlock()

	connections, exists := s.connections[adminID]
	if !exists {
		return fmt.Errorf("admin %s has no active connections", adminID)
	}

	// Find and remove the specific connection
	for i, c := range connections {
		if c == conn {
			// Remove connection from slice
			s.connections[adminID] = append(connections[:i], connections[i+1:]...)
			s.connectionCount--

			// Clean up empty admin entry
			if len(s.connections[adminID]) == 0 {
				delete(s.connections, adminID)
			}

			log.Info().
				Str("admin_id", adminID).
				Int("remaining_connections", s.connectionCount).
				Msg("Admin unsubscribed from notifications")

			return nil
		}
	}

	return fmt.Errorf("connection not found for admin %s", adminID)
}

// Broadcast sends a notification to all connected admins
func (s *NotificationServiceImpl) Broadcast(ctx context.Context, notification *interfaces.Notification) error {
	if notification == nil {
		return fmt.Errorf("notification cannot be nil")
	}

	s.connectionMutex.RLock()
	defer s.connectionMutex.RUnlock()

	var errors []error
	sentCount := 0

	// Send to all admin connections
	for adminID, connections := range s.connections {
		for _, conn := range connections {
			if err := s.sendToConnection(conn, notification); err != nil {
				errors = append(errors, fmt.Errorf("failed to send to admin %s: %w", adminID, err))
				// Remove failed connection
				go s.removeFailedConnection(adminID, conn)
			} else {
				sentCount++
			}
		}
	}

	log.Info().
		Str("notification_id", notification.ID).
		Str("notification_type", notification.Type).
		Int("sent_count", sentCount).
		Int("error_count", len(errors)).
		Msg("Broadcast notification sent")

	// Return error if all sends failed
	if len(errors) > 0 && sentCount == 0 {
		return fmt.Errorf("failed to send notification to any admin: %v", errors)
	}

	return nil
}

// SendToAdmin sends a notification to a specific admin
func (s *NotificationServiceImpl) SendToAdmin(ctx context.Context, adminID string, notification *interfaces.Notification) error {
	if notification == nil {
		return fmt.Errorf("notification cannot be nil")
	}

	if adminID == "" {
		return fmt.Errorf("admin ID cannot be empty")
	}

	s.connectionMutex.RLock()
	connections, exists := s.connections[adminID]
	s.connectionMutex.RUnlock()

	if !exists || len(connections) == 0 {
		return fmt.Errorf("admin %s has no active connections", adminID)
	}

	var errors []error
	sentCount := 0

	// Send to all connections for this admin
	for _, conn := range connections {
		if err := s.sendToConnection(conn, notification); err != nil {
			errors = append(errors, err)
			// Remove failed connection
			go s.removeFailedConnection(adminID, conn)
		} else {
			sentCount++
		}
	}

	log.Info().
		Str("admin_id", adminID).
		Str("notification_id", notification.ID).
		Int("sent_count", sentCount).
		Int("error_count", len(errors)).
		Msg("Notification sent to admin")

	// Return error if all sends failed
	if len(errors) > 0 && sentCount == 0 {
		return fmt.Errorf("failed to send notification to admin %s: %v", adminID, errors)
	}

	return nil
}

// GetConnectionCount returns the number of active connections
func (s *NotificationServiceImpl) GetConnectionCount() int {
	s.connectionMutex.RLock()
	defer s.connectionMutex.RUnlock()
	return s.connectionCount
}

// sendToConnection sends a notification to a specific WebSocket connection
func (s *NotificationServiceImpl) sendToConnection(conn *websocket.Conn, notification *interfaces.Notification) error {
	// Set write deadline to prevent hanging
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send notification as JSON
	if err := conn.WriteJSON(notification); err != nil {
		return fmt.Errorf("failed to write JSON to websocket: %w", err)
	}

	return nil
}

// removeFailedConnection removes a failed connection from the connections map
func (s *NotificationServiceImpl) removeFailedConnection(adminID string, failedConn *websocket.Conn) {
	s.connectionMutex.Lock()
	defer s.connectionMutex.Unlock()

	connections, exists := s.connections[adminID]
	if !exists {
		return
	}

	// Find and remove the failed connection
	for i, conn := range connections {
		if conn == failedConn {
			s.connections[adminID] = append(connections[:i], connections[i+1:]...)
			s.connectionCount--

			// Clean up empty admin entry
			if len(s.connections[adminID]) == 0 {
				delete(s.connections, adminID)
			}

			log.Warn().
				Str("admin_id", adminID).
				Int("remaining_connections", s.connectionCount).
				Msg("Removed failed WebSocket connection")

			// Close the failed connection
			failedConn.Close()
			break
		}
	}
}

// GetConnectedAdmins returns a list of admin IDs with active connections
func (s *NotificationServiceImpl) GetConnectedAdmins() []string {
	s.connectionMutex.RLock()
	defer s.connectionMutex.RUnlock()

	admins := make([]string, 0, len(s.connections))
	for adminID := range s.connections {
		admins = append(admins, adminID)
	}

	return admins
}

// GetAdminConnectionCount returns the number of connections for a specific admin
func (s *NotificationServiceImpl) GetAdminConnectionCount(adminID string) int {
	s.connectionMutex.RLock()
	defer s.connectionMutex.RUnlock()

	connections, exists := s.connections[adminID]
	if !exists {
		return 0
	}

	return len(connections)
}

// CreateSystemAlert creates and broadcasts a system alert notification
func (s *NotificationServiceImpl) CreateSystemAlert(ctx context.Context, severity, title, message, source string, metadata map[string]interface{}) error {
	notification := &interfaces.Notification{
		ID:        fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		Type:      "alert",
		Title:     title,
		Message:   message,
		Severity:  severity,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"source":   source,
			"metadata": metadata,
		},
	}

	return s.Broadcast(ctx, notification)
}

// CreateUserActivity creates and broadcasts a user activity notification
func (s *NotificationServiceImpl) CreateUserActivity(ctx context.Context, userID, action, details string) error {
	notification := &interfaces.Notification{
		ID:        fmt.Sprintf("user_activity_%d", time.Now().UnixNano()),
		Type:      "user_activity",
		Title:     "User Activity",
		Message:   fmt.Sprintf("User %s: %s", userID, action),
		Severity:  "info",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"user_id": userID,
			"action":  action,
			"details": details,
		},
	}

	return s.Broadcast(ctx, notification)
}

// CreateTransactionAlert creates and broadcasts a transaction-related notification
func (s *NotificationServiceImpl) CreateTransactionAlert(ctx context.Context, transactionID, action, severity string, metadata map[string]interface{}) error {
	notification := &interfaces.Notification{
		ID:        fmt.Sprintf("transaction_%d", time.Now().UnixNano()),
		Type:      "transaction",
		Title:     "Transaction Activity",
		Message:   fmt.Sprintf("Transaction %s: %s", transactionID, action),
		Severity:  severity,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"transaction_id": transactionID,
			"action":         action,
			"metadata":       metadata,
		},
	}

	return s.Broadcast(ctx, notification)
}