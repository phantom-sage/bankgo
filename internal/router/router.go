package router

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/phantom-sage/bankgo/internal/config"
	"github.com/phantom-sage/bankgo/internal/database"
	"github.com/phantom-sage/bankgo/internal/handlers"
	"github.com/phantom-sage/bankgo/internal/logging"
	"github.com/phantom-sage/bankgo/internal/middleware"
	"github.com/phantom-sage/bankgo/internal/queue"
	"github.com/phantom-sage/bankgo/internal/repository"
	"github.com/phantom-sage/bankgo/internal/services"
	"github.com/phantom-sage/bankgo/pkg/auth"
	"github.com/rs/zerolog"
)

// SetupRouter configures and returns the main application router
func SetupRouter(db *database.DB, queueManager *queue.QueueManager, cfg *config.Config, loggerManager *logging.LoggerManager, version string) *gin.Engine {
	// Create Gin router
	router := gin.New()

	// Add global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	router.Use(middleware.RequestID())
	// Add request logger middleware if logger manager is available
	if loggerManager != nil {
		router.Use(middleware.RequestLogger(middleware.DefaultLoggerConfig(loggerManager)))
	}

	// Create health handlers
	healthHandlers := handlers.NewHealthHandlers(db, queueManager, version)

	// Initialize services and handlers only if database and config are available
	var authHandlers *handlers.AuthHandlers
	var accountHandlers *handlers.AccountHandlers
	var transferHandlers *handlers.TransferHandlers

	if db != nil && cfg != nil {
		// Create PASETO token manager instance
		tokenManager, err := auth.NewPASETOManager(cfg.PASETO.SecretKey, cfg.PASETO.Expiration)
		if err != nil {
			log.Printf("Warning: Failed to create PASETO token manager: %v", err)
		} else {
			// Initialize repository layer with logger
			var logger zerolog.Logger
			if loggerManager != nil {
				logger = loggerManager.GetLogger()
			} else {
				// Fallback to a basic logger if LoggerManager is not available
				logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
			}
			
			repo := repository.New(db, logger)
			repos := repository.NewRepositories(repo)

			// Initialize all services with proper dependencies
			allServices := services.NewServices(repos, repo, logger)

			// Create all handler instances with services
			authHandlers = handlers.NewAuthHandlers(allServices.UserService, tokenManager, queueManager)
			accountHandlers = handlers.NewAccountHandlers(allServices.AccountService)
			transferHandlers = handlers.NewTransferHandlers(allServices.TransferService, allServices.AccountService)
		}
	}

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health check endpoint (no authentication required)
		v1.GET("/health", healthHandlers.HealthCheck)

		// Authentication routes (no authentication required)
		if authHandlers != nil {
			auth := v1.Group("/auth")
			{
				auth.POST("/register", authHandlers.Register)
				auth.POST("/login", authHandlers.Login)
				auth.POST("/logout", authHandlers.Logout)
			}

			// Protected routes (require authentication)
			protected := v1.Group("")
			protected.Use(authHandlers.AuthMiddleware())
			{
				// Account management routes
				accounts := protected.Group("/accounts")
				{
					accounts.GET("", accountHandlers.GetUserAccounts)           // GET /accounts - List user accounts
					accounts.POST("", accountHandlers.CreateAccount)           // POST /accounts - Create new account
					accounts.GET("/:id", accountHandlers.GetAccount)           // GET /accounts/:id - Get account details
					accounts.PUT("/:id", accountHandlers.UpdateAccount)        // PUT /accounts/:id - Update account
					accounts.DELETE("/:id", accountHandlers.DeleteAccount)     // DELETE /accounts/:id - Delete account
				}

				// Transfer routes
				transfers := protected.Group("/transfers")
				{
					transfers.POST("", transferHandlers.CreateTransfer)        // POST /transfers - Create money transfer
					transfers.GET("", transferHandlers.GetTransferHistory)     // GET /transfers - Get transfer history
					transfers.GET("/:id", transferHandlers.GetTransfer)        // GET /transfers/:id - Get transfer details
				}
			}
		} else {
			// If services are not available, return appropriate error responses
			v1.POST("/auth/register", serviceUnavailableHandler)
			v1.POST("/auth/login", serviceUnavailableHandler)
			v1.POST("/auth/logout", serviceUnavailableHandler)
			v1.GET("/accounts", serviceUnavailableHandler)
			v1.POST("/accounts", serviceUnavailableHandler)
			v1.GET("/accounts/:id", serviceUnavailableHandler)
			v1.PUT("/accounts/:id", serviceUnavailableHandler)
			v1.DELETE("/accounts/:id", serviceUnavailableHandler)
			v1.POST("/transfers", serviceUnavailableHandler)
			v1.GET("/transfers", serviceUnavailableHandler)
			v1.GET("/transfers/:id", serviceUnavailableHandler)
		}
	}

	return router
}

// serviceUnavailableHandler returns a service unavailable response
func serviceUnavailableHandler(c *gin.Context) {
	c.JSON(503, gin.H{
		"error":   "service_unavailable",
		"message": "Database or authentication service is currently unavailable",
		"code":    503,
	})
}