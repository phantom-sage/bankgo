package repository

import (
	"testing"

	"github.com/phantom-sage/bankgo/internal/database"
	"github.com/rs/zerolog"
)

func TestNew(t *testing.T) {
	// Test repository creation with nil database (should not panic)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Repository creation panicked: %v", r)
		}
	}()

	// This will create a repository with nil database for testing structure
	var db *database.DB
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	repo := New(db, logger)
	
	if repo == nil {
		t.Error("Expected repository to be created, got nil")
	}
}

func TestRepositoryInterfaces(t *testing.T) {
	// Test that our repository implementations satisfy the interfaces
	var db *database.DB
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	repo := New(db, logger)
	
	// Test that we can create repository instances
	userRepo := NewUserRepository(repo)
	if userRepo == nil {
		t.Error("Expected user repository to be created, got nil")
	}
	
	accountRepo := NewAccountRepository(repo)
	if accountRepo == nil {
		t.Error("Expected account repository to be created, got nil")
	}
	
	transferRepo := NewTransferRepository(repo)
	if transferRepo == nil {
		t.Error("Expected transfer repository to be created, got nil")
	}
}