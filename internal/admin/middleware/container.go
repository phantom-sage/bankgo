package middleware

import (
	"github.com/phantom-sage/bankgo/internal/admin/config"
	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
	"github.com/phantom-sage/bankgo/internal/admin/services"
)

// Container holds all admin middleware
type Container struct {
	config   *config.Config
	services *services.Container

	// Middleware
	AuthMiddleware      interfaces.AuthMiddleware
	CORSMiddleware      interfaces.CORSMiddleware
	LoggingMiddleware   interfaces.LoggingMiddleware
	ErrorMiddleware     interfaces.ErrorMiddleware
	RateLimitMiddleware interfaces.RateLimitMiddleware
}

// NewContainer creates a new middleware container with dependencies
func NewContainer(cfg *config.Config, services *services.Container) *Container {
	container := &Container{
		config:   cfg,
		services: services,
	}

	// Initialize middleware
	container.initMiddleware()

	return container
}

// initMiddleware initializes all middleware
func (c *Container) initMiddleware() {
	// Initialize auth middleware
	c.AuthMiddleware = NewAuthMiddleware(c.services.AuthService)

	// Initialize CORS middleware
	c.CORSMiddleware = NewCORSMiddleware(c.config.AllowedOrigins)

	// TODO: Initialize other middleware implementations in later tasks
	c.LoggingMiddleware = nil
	c.ErrorMiddleware = nil
	c.RateLimitMiddleware = nil
}

// GetConfig returns the configuration
func (c *Container) GetConfig() *config.Config {
	return c.config
}

// GetServices returns the service container
func (c *Container) GetServices() *services.Container {
	return c.services
}