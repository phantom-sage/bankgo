package queue

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockEmailProcessor is a mock implementation of EmailProcessor
type MockEmailProcessor struct {
	mock.Mock
}

// createTestLogger creates a logger for testing
func createTestLogger() zerolog.Logger {
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}

func (m *MockEmailProcessor) ProcessWelcomeEmail(ctx context.Context, payload WelcomeEmailPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}

func TestNewQueueManager(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	logger := createTestLogger()
	qm, err := NewQueueManager(cfg, logger)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	defer qm.Close()

	require.NotNil(t, qm)
	require.NotNil(t, qm.client)
	require.NotNil(t, qm.server)
	require.NotNil(t, qm.redis)
}

func TestQueueManager_QueueWelcomeEmail(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	logger := createTestLogger()
	qm, err := NewQueueManager(cfg, logger)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	defer qm.Close()

	payload := WelcomeEmailPayload{
		UserID:    1,
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = qm.QueueWelcomeEmail(ctx, payload)
	assert.NoError(t, err)
}

func TestQueueManager_RegisterHandlers(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	logger := createTestLogger()
	qm, err := NewQueueManager(cfg, logger)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	defer qm.Close()

	mockProcessor := &MockEmailProcessor{}
	mockProcessor.On("ProcessWelcomeEmail", mock.Anything, mock.Anything).Return(nil)

	// This should not panic
	qm.RegisterHandlers(mockProcessor)
}

func TestQueueManager_HealthCheck(t *testing.T) {
	cfg := config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	logger := createTestLogger()
	qm, err := NewQueueManager(cfg, logger)
	if err != nil {
		t.Skip("Redis not available for testing")
	}
	defer qm.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = qm.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestWelcomeEmailPayload(t *testing.T) {
	payload := WelcomeEmailPayload{
		UserID:    123,
		Email:     "user@example.com",
		FirstName: "Jane",
		LastName:  "Smith",
	}

	assert.Equal(t, 123, payload.UserID)
	assert.Equal(t, "user@example.com", payload.Email)
	assert.Equal(t, "Jane", payload.FirstName)
	assert.Equal(t, "Smith", payload.LastName)
}

func TestTaskTypes(t *testing.T) {
	assert.Equal(t, "email:welcome", TypeWelcomeEmail)
}