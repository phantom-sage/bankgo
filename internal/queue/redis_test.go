package queue

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisClient(t *testing.T) {
	tests := []struct {
		name    string
		config  config.RedisConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: config.RedisConfig{
				Host:         "localhost",
				Port:         6379,
				Password:     "",
				DB:           0,
				PoolSize:     10,
				MinIdleConns: 5,
			},
			wantErr: false,
		},
		{
			name: "invalid host",
			config: config.RedisConfig{
				Host:         "invalid-host-12345",
				Port:         6379,
				Password:     "",
				DB:           0,
				PoolSize:     10,
				MinIdleConns: 5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewRedisClient(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				if err != nil {
					t.Skip("Redis not available for testing")
				}
				assert.NoError(t, err)
				assert.NotNil(t, client)
				
				// Test health check
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				
				err = client.HealthCheck(ctx)
				assert.NoError(t, err)
				
				// Clean up
				client.Close()
			}
		})
	}
}

func TestRedisClient_HealthCheck(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	client, err := NewRedisClient(cfg)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestNewAsyncqClient(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	client, err := NewAsyncqClient(cfg)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	
	require.NotNil(t, client)
	require.NotNil(t, client.Client())
	
	// Clean up
	client.Close()
}

func TestNewAsyncqServer(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	server, err := NewAsyncqServer(cfg, logger)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	
	require.NotNil(t, server)
	require.NotNil(t, server.server)
	require.NotNil(t, server.mux)
}

func TestRedisConfig_Address(t *testing.T) {
	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
	}

	expected := "localhost:6379"
	actual := cfg.Address()
	
	assert.Equal(t, expected, actual)
}