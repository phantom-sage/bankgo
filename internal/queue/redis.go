package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/phantom-sage/bankgo/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// RedisClient wraps the Redis client with connection pooling
type RedisClient struct {
	client *redis.Client
	config config.RedisConfig
}

// NewRedisClient creates a new Redis client with connection pooling
func NewRedisClient(cfg config.RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Address(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: client,
		config: cfg,
	}, nil
}

// Client returns the underlying Redis client
func (r *RedisClient) Client() *redis.Client {
	return r.client
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// HealthCheck performs a health check on the Redis connection
func (r *RedisClient) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// AsyncqClient wraps the Asynq client for task queuing
type AsyncqClient struct {
	client *asynq.Client
	config config.RedisConfig
}

// NewAsyncqClient creates a new Asynq client for task queuing
func NewAsyncqClient(cfg config.RedisConfig) (*AsyncqClient, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Address(),
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	client := asynq.NewClient(redisOpt)

	return &AsyncqClient{
		client: client,
		config: cfg,
	}, nil
}

// Client returns the underlying Asynq client
func (a *AsyncqClient) Client() *asynq.Client {
	return a.client
}

// Close closes the Asynq client
func (a *AsyncqClient) Close() error {
	return a.client.Close()
}

// AsyncqServer wraps the Asynq server for task processing
type AsyncqServer struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	config config.RedisConfig
	logger zerolog.Logger
}

// NewAsyncqServer creates a new Asynq server for task processing
func NewAsyncqServer(cfg config.RedisConfig, logger zerolog.Logger) (*AsyncqServer, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Address(),
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	serverLogger := logger.With().Str("component", "asyncq_server").Logger()

	// Configure server with retry policies and error handling
	serverConfig := asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"email":    6, // High priority for email tasks
			"default":  3,
			"low":      1,
		},
		// Retry policy for failed tasks
		RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
			// Exponential backoff: 1s, 2s, 4s, 8s, 16s
			delay := time.Duration(1<<uint(n)) * time.Second
			
			serverLogger.Warn().
				Str("task_type", t.Type()).
				Int("retry_count", n).
				Err(e).
				Dur("retry_delay", delay).
				Msg("Task failed, scheduling retry")
			
			return delay
		},
		// Error handler for logging failed tasks
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			// Get correlation ID from context if available
			correlationID := ""
			if id := ctx.Value("correlation_id"); id != nil {
				if cid, ok := id.(string); ok {
					correlationID = cid
				}
			}

			serverLogger.Error().
				Str("task_type", task.Type()).
				Str("correlation_id", correlationID).
				Err(err).
				Bytes("payload", task.Payload()).
				Msg("Task processing failed")
		}),
	}

	server := asynq.NewServer(redisOpt, serverConfig)
	mux := asynq.NewServeMux()

	return &AsyncqServer{
		server: server,
		mux:    mux,
		config: cfg,
		logger: serverLogger,
	}, nil
}

// RegisterHandler registers a task handler with the server
func (a *AsyncqServer) RegisterHandler(pattern string, handler asynq.HandlerFunc) {
	a.mux.HandleFunc(pattern, handler)
}

// Start starts the Asynq server
func (a *AsyncqServer) Start() error {
	return a.server.Start(a.mux)
}

// Stop stops the Asynq server gracefully
func (a *AsyncqServer) Stop() {
	a.server.Stop()
}

// Shutdown shuts down the Asynq server with timeout
func (a *AsyncqServer) Shutdown() {
	a.server.Shutdown()
}