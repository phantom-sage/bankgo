package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/rs/zerolog/log"
)

// WebSocketHandlerImpl implements the WebSocketHandler interface
type WebSocketHandlerImpl struct {
	notificationService interfaces.NotificationService
	upgrader           websocket.Upgrader
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(notificationService interfaces.NotificationService) interfaces.WebSocketHandler {
	return &WebSocketHandlerImpl{
		notificationService: notificationService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from admin SPA
				// In production, this should be more restrictive
				return true
			},
		},
	}
}

// HandleConnection handles WebSocket connection upgrades
func (h *WebSocketHandlerImpl) HandleConnection(c *gin.Context) {
	// Get admin session from context (set by auth middleware)
	session, exists := c.Get("admin_session")
	if !exists {
		log.Error().Msg("Admin session not found in WebSocket connection")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Admin session required for WebSocket connection",
		})
		return
	}

	adminSession, ok := session.(*interfaces.AdminSession)
	if !ok {
		log.Error().Msg("Invalid admin session type in WebSocket connection")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Invalid session data",
		})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().
			Err(err).
			Str("admin_id", adminSession.ID).
			Msg("Failed to upgrade WebSocket connection")
		return
	}

	// Ensure connection is closed when function exits
	defer func() {
		if err := h.notificationService.Unsubscribe(adminSession.ID, conn); err != nil {
			log.Warn().
				Err(err).
				Str("admin_id", adminSession.ID).
				Msg("Failed to unsubscribe from notifications")
		}
		conn.Close()
	}()

	// Subscribe to notifications
	ctx := context.Background()
	if err := h.notificationService.Subscribe(ctx, conn, adminSession.ID); err != nil {
		log.Error().
			Err(err).
			Str("admin_id", adminSession.ID).
			Msg("Failed to subscribe to notifications")
		return
	}

	log.Info().
		Str("admin_id", adminSession.ID).
		Str("admin_username", adminSession.Username).
		Msg("WebSocket connection established")

	// Handle connection lifecycle
	h.handleConnectionLifecycle(conn, adminSession)
}

// HandleNotifications handles notification-related WebSocket endpoints
func (h *WebSocketHandlerImpl) HandleNotifications(c *gin.Context) {
	// This endpoint provides information about the notification system
	// without establishing a WebSocket connection
	
	connectionCount := h.notificationService.GetConnectionCount()
	
	c.JSON(http.StatusOK, gin.H{
		"status":            "active",
		"total_connections": connectionCount,
		"websocket_url":     "/api/admin/ws",
		"supported_events": []string{
			"system",
			"alert", 
			"user_activity",
			"transaction",
		},
	})
}

// handleConnectionLifecycle manages the WebSocket connection lifecycle
func (h *WebSocketHandlerImpl) handleConnectionLifecycle(conn *websocket.Conn, session *interfaces.AdminSession) {
	// Set up ping/pong to keep connection alive
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Channel to signal when to close connection
	done := make(chan struct{})

	// Goroutine to handle ping messages
	go func() {
		defer close(done)
		for {
			select {
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Warn().
						Err(err).
						Str("admin_id", session.ID).
						Msg("Failed to send ping message")
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Read messages from client (mainly for pong responses and connection management)
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().
					Err(err).
					Str("admin_id", session.ID).
					Msg("WebSocket connection closed unexpectedly")
			} else {
				log.Info().
					Str("admin_id", session.ID).
					Msg("WebSocket connection closed")
			}
			break
		}

		// Handle different message types
		switch messageType {
		case websocket.TextMessage:
			h.handleTextMessage(conn, session, message)
		case websocket.BinaryMessage:
			log.Warn().
				Str("admin_id", session.ID).
				Msg("Received unsupported binary message")
		case websocket.CloseMessage:
			log.Info().
				Str("admin_id", session.ID).
				Msg("Received close message")
			return
		}
	}
}

// handleTextMessage processes text messages from the WebSocket client
func (h *WebSocketHandlerImpl) handleTextMessage(conn *websocket.Conn, session *interfaces.AdminSession, message []byte) {
	log.Debug().
		Str("admin_id", session.ID).
		Str("message", string(message)).
		Msg("Received WebSocket text message")

	// For now, we mainly use WebSocket for server-to-client notifications
	// Client messages could be used for:
	// - Acknowledgment of notifications
	// - Subscription preferences
	// - Connection health checks
	
	// Simple echo for connection testing
	if string(message) == "ping" {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
			log.Warn().
				Err(err).
				Str("admin_id", session.ID).
				Msg("Failed to send pong response")
		}
	}
}

// RegisterRoutes registers WebSocket routes
func (h *WebSocketHandlerImpl) RegisterRoutes(router gin.IRouter) {
	// WebSocket connection endpoint (requires authentication)
	router.GET("/ws", h.HandleConnection)
	
	// Notification system info endpoint
	router.GET("/notifications", h.HandleNotifications)
}

// BroadcastSystemAlert is a helper method to broadcast system alerts
func (h *WebSocketHandlerImpl) BroadcastSystemAlert(ctx context.Context, severity, title, message, source string) error {
	notification := &interfaces.Notification{
		ID:        generateNotificationID("alert"),
		Type:      "alert",
		Title:     title,
		Message:   message,
		Severity:  severity,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"source": source,
		},
	}

	return h.notificationService.Broadcast(ctx, notification)
}

// BroadcastUserActivity is a helper method to broadcast user activity notifications
func (h *WebSocketHandlerImpl) BroadcastUserActivity(ctx context.Context, userID, action, details string) error {
	notification := &interfaces.Notification{
		ID:        generateNotificationID("user_activity"),
		Type:      "user_activity", 
		Title:     "User Activity",
		Message:   details,
		Severity:  "info",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"user_id": userID,
			"action":  action,
		},
	}

	return h.notificationService.Broadcast(ctx, notification)
}

// generateNotificationID generates a unique notification ID
func generateNotificationID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}