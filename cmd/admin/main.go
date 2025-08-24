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

	"github.com/phantom-sage/bankgo/internal/admin/config"
	"github.com/phantom-sage/bankgo/internal/admin/handlers"
	"github.com/phantom-sage/bankgo/internal/admin/middleware"
	"github.com/phantom-sage/bankgo/internal/admin/router"
	"github.com/phantom-sage/bankgo/internal/admin/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize services
	serviceContainer, err := services.NewContainer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}
	defer serviceContainer.Close()

	// Initialize handlers
	handlerContainer := handlers.NewContainer(serviceContainer)

	// Initialize middleware
	middlewareContainer := middleware.NewContainer(cfg, serviceContainer)

	// Setup router
	r := router.Setup(handlerContainer, middlewareContainer)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Admin API server starting on port %d", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down admin API server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Admin API server forced to shutdown: %v", err)
	}

	log.Println("Admin API server exited")
}