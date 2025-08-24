package handlers

import (
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/phantom-sage/bankgo/internal/admin/services"
)

// Container holds all admin HTTP handlers
type Container struct {
	services *services.Container

	// Handlers
	AuthHandler         interfaces.AuthHandler
	UserHandler         interfaces.UserHandler
	SystemHandler       interfaces.SystemHandler
	DatabaseHandler     interfaces.DatabaseHandler
	WebSocketHandler    interfaces.WebSocketHandler
	TransactionHandler  interfaces.TransactionHandler
	AccountHandler      interfaces.AccountHandler
}

// NewContainer creates a new handler container with service dependencies
func NewContainer(services *services.Container) *Container {
	container := &Container{
		services: services,
	}

	// Initialize handlers
	container.initHandlers()

	return container
}

// initHandlers initializes all HTTP handlers
func (c *Container) initHandlers() {
	// Initialize auth handler
	c.AuthHandler = NewAuthHandler(c.services.AuthService)

	// Initialize user handler
	c.UserHandler = NewUserHandler(c.services.UserService)

	// Initialize system handler
	c.SystemHandler = NewSystemHandler(c.services.SystemService)
	
	// Initialize WebSocket handler
	c.WebSocketHandler = NewWebSocketHandler(c.services.NotificationService)
	
	// Initialize database handler
	c.DatabaseHandler = NewDatabaseHandler(c.services.DatabaseService)
	
	// Initialize transaction handler
	c.TransactionHandler = NewTransactionHandler(c.services.TransactionService)
	
	// Initialize account handler
	c.AccountHandler = NewAccountHandler(c.services.AccountService)
}

// GetServices returns the service container
func (c *Container) GetServices() *services.Container {
	return c.services
}