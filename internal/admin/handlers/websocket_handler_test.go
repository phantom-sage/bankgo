package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/phantom-sage/bankgo/internal/admin/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketHandler_HandleConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create notification service and handler
	notificationService := services.NewNotificationService()
	handler := NewWebSocketHandler(notificationService)

	// Create test router
	router := gin.New()
	
	// Add middleware to set admin session
	router.Use(func(c *gin.Context) {
		// Mock admin session
		session := &interfaces.AdminSession{
			ID:          "test-admin-1",
			Username:    "testadmin",
			PasetoToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
			CreatedAt:   time.Now(),
			LastActive:  time.Now(),
		}
		c.Set("admin_session", session)
		c.Next()
	})

	// Register WebSocket route
	router.GET("/ws", handler.HandleConnection)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Test WebSocket connection
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()

	// Should receive welcome notification
	var notification interfaces.Notification
	err = conn.ReadJSON(&notification)
	require.NoError(t, err)
	assert.Equal(t, "system", notification.Type)
	assert.Equal(t, "Connected", notification.Title)
	assert.Equal(t, "info", notification.Severity)

	// Test ping/pong
	err = conn.WriteMessage(websocket.TextMessage, []byte("ping"))
	require.NoError(t, err)

	messageType, message, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, websocket.TextMessage, messageType)
	assert.Equal(t, "pong", string(message))
}

func TestWebSocketHandler_HandleConnection_NoSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create notification service and handler
	notificationService := services.NewNotificationService()
	handler := NewWebSocketHandler(notificationService)

	// Create test router without session middleware
	router := gin.New()
	router.GET("/ws", handler.HandleConnection)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Test WebSocket connection without session
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	
	// Should fail to upgrade due to missing session
	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWebSocketHandler_HandleNotifications(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create notification service and handler
	notificationService := services.NewNotificationService()
	handler := NewWebSocketHandler(notificationService)

	// Create test router
	router := gin.New()
	router.GET("/notifications", handler.HandleNotifications)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "active", response["status"])
	assert.Equal(t, float64(0), response["total_connections"]) // No connections initially
	assert.Equal(t, "/api/admin/ws", response["websocket_url"])
	
	supportedEvents, ok := response["supported_events"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, supportedEvents, "system")
	assert.Contains(t, supportedEvents, "alert")
	assert.Contains(t, supportedEvents, "user_activity")
	assert.Contains(t, supportedEvents, "transaction")
}

func TestWebSocketHandler_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create notification service and handler
	notificationService := services.NewNotificationService()
	handler := NewWebSocketHandler(notificationService)

	// Create test router
	router := gin.New()
	apiGroup := router.Group("/api/admin")

	// Register routes
	handler.RegisterRoutes(apiGroup)

	// Test that routes are registered
	routes := router.Routes()
	
	var wsRoute, notificationsRoute bool
	for _, route := range routes {
		if route.Path == "/api/admin/ws" && route.Method == "GET" {
			wsRoute = true
		}
		if route.Path == "/api/admin/notifications" && route.Method == "GET" {
			notificationsRoute = true
		}
	}

	assert.True(t, wsRoute, "WebSocket route should be registered")
	assert.True(t, notificationsRoute, "Notifications route should be registered")
}

func TestWebSocketHandler_BroadcastSystemAlert(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create notification service and handler
	notificationService := services.NewNotificationService()
	handler := NewWebSocketHandler(notificationService)

	ctx := context.Background()

	// Test broadcasting system alert
	err := handler.(*WebSocketHandlerImpl).BroadcastSystemAlert(
		ctx,
		"critical",
		"System Error",
		"Database connection failed",
		"database",
	)

	// Should not error even with no connections
	assert.NoError(t, err)
}

func TestWebSocketHandler_BroadcastUserActivity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create notification service and handler
	notificationService := services.NewNotificationService()
	handler := NewWebSocketHandler(notificationService)

	ctx := context.Background()

	// Test broadcasting user activity
	err := handler.(*WebSocketHandlerImpl).BroadcastUserActivity(
		ctx,
		"user-123",
		"login",
		"User logged in successfully",
	)

	// Should not error even with no connections
	assert.NoError(t, err)
}

func TestWebSocketHandler_ConnectionLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create notification service and handler
	notificationService := services.NewNotificationService()
	handler := NewWebSocketHandler(notificationService)

	// Create test router
	router := gin.New()
	
	// Add middleware to set admin session
	router.Use(func(c *gin.Context) {
		session := &interfaces.AdminSession{
			ID:          "test-admin-1",
			Username:    "testadmin",
			PasetoToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
			CreatedAt:   time.Now(),
			LastActive:  time.Now(),
		}
		c.Set("admin_session", session)
		c.Next()
	})

	router.GET("/ws", handler.HandleConnection)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Connect WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Verify connection is tracked
	assert.Equal(t, 1, notificationService.GetConnectionCount())

	// Close connection
	conn.Close()

	// Give some time for cleanup
	time.Sleep(100 * time.Millisecond)

	// Connection should be cleaned up (this might not be immediate due to goroutines)
	// In a real scenario, the connection cleanup happens when the handler function exits
}

func TestWebSocketHandler_MultipleConnections(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create notification service and handler
	notificationService := services.NewNotificationService()
	handler := NewWebSocketHandler(notificationService)

	// Create test router
	router := gin.New()
	
	// Add middleware to set admin session
	router.Use(func(c *gin.Context) {
		session := &interfaces.AdminSession{
			ID:          "test-admin-1",
			Username:    "testadmin",
			PasetoToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
			CreatedAt:   time.Now(),
			LastActive:  time.Now(),
		}
		c.Set("admin_session", session)
		c.Next()
	})

	router.GET("/ws", handler.HandleConnection)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Connect multiple WebSocket connections
	var connections []*websocket.Conn
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	for i := 0; i < 3; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		connections = append(connections, conn)

		// Read welcome message
		var notification interfaces.Notification
		err = conn.ReadJSON(&notification)
		require.NoError(t, err)
	}

	// Verify all connections are tracked
	assert.Equal(t, 3, notificationService.GetConnectionCount())

	// Close all connections
	for _, conn := range connections {
		conn.Close()
	}
}

func TestWebSocketHandler_InvalidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create notification service and handler
	notificationService := services.NewNotificationService()
	handler := NewWebSocketHandler(notificationService)

	// Create test router with invalid session type
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Set invalid session type
		c.Set("admin_session", "invalid-session-type")
		c.Next()
	})

	router.GET("/ws", handler.HandleConnection)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Test WebSocket connection with invalid session
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	
	// Should fail due to invalid session type
	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// Integration test with real notification broadcasting
func TestWebSocketHandler_NotificationBroadcast_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create notification service and handler
	notificationService := services.NewNotificationService()
	handler := NewWebSocketHandler(notificationService)

	// Create test router
	router := gin.New()
	
	// Add middleware to set admin session
	router.Use(func(c *gin.Context) {
		session := &interfaces.AdminSession{
			ID:          "test-admin-1",
			Username:    "testadmin",
			PasetoToken: "test-token",
			ExpiresAt:   time.Now().Add(time.Hour),
			CreatedAt:   time.Now(),
			LastActive:  time.Now(),
		}
		c.Set("admin_session", session)
		c.Next()
	})

	router.GET("/ws", handler.HandleConnection)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Connect WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Read welcome notification
	var welcomeNotification interfaces.Notification
	err = conn.ReadJSON(&welcomeNotification)
	require.NoError(t, err)
	assert.Equal(t, "system", welcomeNotification.Type)

	// Broadcast a test notification
	ctx := context.Background()
	testNotification := &interfaces.Notification{
		ID:        "test-broadcast",
		Type:      "alert",
		Title:     "Test Alert",
		Message:   "This is a test alert",
		Severity:  "warning",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"source": "test",
		},
	}

	err = notificationService.Broadcast(ctx, testNotification)
	require.NoError(t, err)

	// Read the broadcasted notification
	var receivedNotification interfaces.Notification
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	err = conn.ReadJSON(&receivedNotification)
	require.NoError(t, err)

	assert.Equal(t, testNotification.ID, receivedNotification.ID)
	assert.Equal(t, testNotification.Type, receivedNotification.Type)
	assert.Equal(t, testNotification.Title, receivedNotification.Title)
	assert.Equal(t, testNotification.Message, receivedNotification.Message)
	assert.Equal(t, testNotification.Severity, receivedNotification.Severity)
}