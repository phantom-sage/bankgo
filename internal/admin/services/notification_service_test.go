package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationService_Subscribe(t *testing.T) {
	service := NewNotificationService()
	
	// Create a test WebSocket connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		
		// Keep connection alive for test
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	// Connect to test server
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	ctx := context.Background()
	adminID := "test-admin-1"

	// Test successful subscription
	err = service.Subscribe(ctx, conn, adminID)
	assert.NoError(t, err)
	assert.Equal(t, 1, service.GetConnectionCount())

	// Test nil connection
	err = service.Subscribe(ctx, nil, adminID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "websocket connection cannot be nil")

	// Test empty admin ID
	err = service.Subscribe(ctx, conn, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "admin ID cannot be empty")
}

func TestNotificationService_Unsubscribe(t *testing.T) {
	service := NewNotificationService()
	
	// Create test WebSocket connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	ctx := context.Background()
	adminID := "test-admin-1"

	// Subscribe first
	err = service.Subscribe(ctx, conn, adminID)
	require.NoError(t, err)
	assert.Equal(t, 1, service.GetConnectionCount())

	// Test successful unsubscribe
	err = service.Unsubscribe(adminID, conn)
	assert.NoError(t, err)
	assert.Equal(t, 0, service.GetConnectionCount())

	// Test unsubscribe non-existent admin
	err = service.Unsubscribe("non-existent", conn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has no active connections")

	// Test nil connection
	err = service.Unsubscribe(adminID, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "websocket connection cannot be nil")
}

func TestNotificationService_Broadcast(t *testing.T) {
	service := NewNotificationService()
	ctx := context.Background()

	// Test broadcast with no connections
	notification := &interfaces.Notification{
		ID:        "test-1",
		Type:      "test",
		Title:     "Test Notification",
		Message:   "This is a test",
		Severity:  "info",
		Timestamp: time.Now(),
	}

	err := service.Broadcast(ctx, notification)
	assert.NoError(t, err) // Should not error with no connections

	// Test nil notification
	err = service.Broadcast(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "notification cannot be nil")
}

func TestNotificationService_SendToAdmin(t *testing.T) {
	service := NewNotificationService()
	ctx := context.Background()

	notification := &interfaces.Notification{
		ID:        "test-1",
		Type:      "test",
		Title:     "Test Notification",
		Message:   "This is a test",
		Severity:  "info",
		Timestamp: time.Now(),
	}

	// Test send to non-existent admin
	err := service.SendToAdmin(ctx, "non-existent", notification)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has no active connections")

	// Test nil notification
	err = service.SendToAdmin(ctx, "admin-1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "notification cannot be nil")

	// Test empty admin ID
	err = service.SendToAdmin(ctx, "", notification)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "admin ID cannot be empty")
}

func TestNotificationService_MultipleConnections(t *testing.T) {
	service := NewNotificationService()
	ctx := context.Background()

	// Create multiple test connections
	var connections []*websocket.Conn
	var servers []*httptest.Server

	for i := 0; i < 3; i++ {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{}
			conn, err := upgrader.Upgrade(w, r, nil)
			require.NoError(t, err)
			defer conn.Close()
			time.Sleep(200 * time.Millisecond)
		}))
		servers = append(servers, server)

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		connections = append(connections, conn)
	}

	// Clean up
	defer func() {
		for _, conn := range connections {
			conn.Close()
		}
		for _, server := range servers {
			server.Close()
		}
	}()

	// Subscribe multiple connections for same admin
	adminID := "test-admin-1"
	for _, conn := range connections {
		err := service.Subscribe(ctx, conn, adminID)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, service.GetConnectionCount())
	assert.Equal(t, 3, service.(*NotificationServiceImpl).GetAdminConnectionCount(adminID))

	// Test unsubscribe one connection
	err := service.Unsubscribe(adminID, connections[0])
	assert.NoError(t, err)
	assert.Equal(t, 2, service.GetConnectionCount())
	assert.Equal(t, 2, service.(*NotificationServiceImpl).GetAdminConnectionCount(adminID))
}

func TestNotificationService_CreateSystemAlert(t *testing.T) {
	service := NewNotificationService()
	ctx := context.Background()

	err := service.(*NotificationServiceImpl).CreateSystemAlert(
		ctx,
		"critical",
		"System Error",
		"Database connection failed",
		"database",
		map[string]interface{}{
			"error_code": "DB_CONN_FAILED",
			"retry_count": 3,
		},
	)

	// Should not error even with no connections
	assert.NoError(t, err)
}

func TestNotificationService_CreateUserActivity(t *testing.T) {
	service := NewNotificationService()
	ctx := context.Background()

	err := service.(*NotificationServiceImpl).CreateUserActivity(
		ctx,
		"user-123",
		"login",
		"User logged in successfully",
	)

	// Should not error even with no connections
	assert.NoError(t, err)
}

func TestNotificationService_CreateTransactionAlert(t *testing.T) {
	service := NewNotificationService()
	ctx := context.Background()

	err := service.(*NotificationServiceImpl).CreateTransactionAlert(
		ctx,
		"txn-456",
		"reversal",
		"warning",
		map[string]interface{}{
			"amount": "1000.00",
			"reason": "fraud_detected",
		},
	)

	// Should not error even with no connections
	assert.NoError(t, err)
}

func TestNotificationService_GetConnectedAdmins(t *testing.T) {
	service := NewNotificationService()
	ctx := context.Background()

	// Initially no admins
	admins := service.(*NotificationServiceImpl).GetConnectedAdmins()
	assert.Empty(t, admins)

	// Create test connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Subscribe admin
	adminID := "test-admin-1"
	err = service.Subscribe(ctx, conn, adminID)
	require.NoError(t, err)

	// Check connected admins
	admins = service.(*NotificationServiceImpl).GetConnectedAdmins()
	assert.Len(t, admins, 1)
	assert.Contains(t, admins, adminID)
}

func TestNotificationService_ConnectionManagement(t *testing.T) {
	service := NewNotificationService()
	ctx := context.Background()

	// Test initial state
	assert.Equal(t, 0, service.GetConnectionCount())

	// Create test connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	adminID := "test-admin-1"

	// Subscribe
	err = service.Subscribe(ctx, conn, adminID)
	require.NoError(t, err)
	assert.Equal(t, 1, service.GetConnectionCount())

	// Unsubscribe
	err = service.Unsubscribe(adminID, conn)
	require.NoError(t, err)
	assert.Equal(t, 0, service.GetConnectionCount())

	// Verify admin is removed from connected list
	admins := service.(*NotificationServiceImpl).GetConnectedAdmins()
	assert.Empty(t, admins)
}

// Benchmark tests
func BenchmarkNotificationService_Broadcast(b *testing.B) {
	service := NewNotificationService()
	ctx := context.Background()

	notification := &interfaces.Notification{
		ID:        "bench-test",
		Type:      "benchmark",
		Title:     "Benchmark Test",
		Message:   "Performance test notification",
		Severity:  "info",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.Broadcast(ctx, notification)
	}
}

func BenchmarkNotificationService_Subscribe(b *testing.B) {
	service := NewNotificationService()
	ctx := context.Background()

	// Create a mock connection for benchmarking
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		time.Sleep(10 * time.Millisecond)
	}))
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			b.Fatal(err)
		}

		adminID := fmt.Sprintf("admin-%d", i)
		service.Subscribe(ctx, conn, adminID)
		conn.Close()
	}
}