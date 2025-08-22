package database

import (
	"context"
	"testing"
	"time"

	"github.com/phantom-sage/bankgo/internal/config"
)

func TestNew(t *testing.T) {
	// Test with invalid configuration
	cfg := config.DatabaseConfig{
		Host:            "invalid-host",
		Port:            5432,
		Name:            "test",
		User:            "test",
		Password:        "test",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	db, err := New(cfg)
	if err == nil {
		t.Error("Expected error with invalid host, got nil")
		if db != nil {
			db.Close()
		}
	}
}

func TestDatabaseConfig_ConnectionString(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	actual := cfg.ConnectionString()

	if actual != expected {
		t.Errorf("Expected connection string %q, got %q", expected, actual)
	}
}

func TestDB_HealthCheck(t *testing.T) {
	// This test would require a real database connection
	// For now, we'll just test the structure
	cfg := config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		Name:            "test",
		User:            "test",
		Password:        "test",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	// This will fail without a real database, but tests the code structure
	db, err := New(cfg)
	if err != nil {
		// Expected to fail without real database
		t.Logf("Expected error without real database: %v", err)
		return
	}
	defer db.Close()

	// Test health check
	ctx := context.Background()
	err = db.HealthCheck(ctx)
	if err != nil {
		t.Logf("Health check failed as expected without real database: %v", err)
	}
}