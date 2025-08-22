package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phantom-sage/bankgo/internal/config"
	"github.com/phantom-sage/bankgo/internal/database"
	"github.com/phantom-sage/bankgo/internal/logging"
	"github.com/phantom-sage/bankgo/internal/queue"
	"github.com/phantom-sage/bankgo/internal/router"
)

const version = "v1.0.0"

func main() {
	log.Println("Bank REST API Server starting...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger manager
	loggingConfig := logging.LogConfig{
		Level:              cfg.Logging.Level,
		Format:             cfg.Logging.Format,
		Output:             cfg.Logging.Output,
		Directory:          cfg.Logging.Directory,
		MaxAge:             cfg.Logging.MaxAge,
		MaxBackups:         cfg.Logging.MaxBackups,
		MaxSize:            cfg.Logging.MaxSize,
		Compress:           cfg.Logging.Compress,
		LocalTime:          cfg.Logging.LocalTime,
		CallerInfo:         cfg.Logging.CallerInfo,
		SamplingEnabled:    cfg.Logging.SamplingEnabled,
		SamplingInitial:    cfg.Logging.SamplingInitial,
		SamplingThereafter: cfg.Logging.SamplingThereafter,
	}
	
	loggerManager, err := logging.NewLoggerManager(loggingConfig)
	if err != nil {
		log.Fatalf("Failed to initialize logger manager: %v", err)
	}
	defer loggerManager.Close()
	
	logger := loggerManager.GetLogger()
	logger.Info().Str("version", version).Msg("Bank REST API Server starting")

	// Initialize database connection
	var db *database.DB
	if cfg.Database.Host != "" {
		db, err = database.New(cfg.Database)
		if err != nil {
			log.Printf("Warning: Failed to connect to database: %v", err)
			// Continue without database for health check testing
		} else {
			defer db.Close()
			log.Println("Database connection established")
		}
	}

	// Initialize queue manager
	var queueManager *queue.QueueManager
	if cfg.Redis.Host != "" {
		queueManager, err = queue.NewQueueManager(cfg.Redis, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to connect to Redis")
			// Continue without Redis for health check testing
		} else {
			defer queueManager.Close()
			logger.Info().Msg("Redis connection established")
			
			// Start periodic metrics logging for queue monitoring
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			queueManager.StartPeriodicMetricsLogging(ctx, 30*time.Second)
		}
	}

	// Setup router with logger manager
	r := router.SetupRouter(db, queueManager, cfg, loggerManager, version)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		logger.Info().Int("port", cfg.Server.Port).Msg("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Server exited")
}