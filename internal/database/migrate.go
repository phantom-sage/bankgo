package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}

// MigrationRunner handles database migrations
type MigrationRunner struct {
	db *DB
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *DB) *MigrationRunner {
	return &MigrationRunner{db: db}
}

// CreateMigrationsTable creates the migrations tracking table
func (mr *MigrationRunner) CreateMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT NOW()
		);
	`
	
	_, err := mr.db.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	
	return nil
}

// LoadMigrations loads all migration files from the embedded filesystem
func (mr *MigrationRunner) LoadMigrations() ([]Migration, error) {
	var migrations []Migration
	
	// Read all migration files
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}
	
	// Group files by version
	migrationMap := make(map[string]map[string]string)
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		filename := entry.Name()
		if !strings.HasSuffix(filename, ".sql") {
			continue
		}
		
		// Parse filename: 001_create_users_table.up.sql
		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			continue
		}
		
		version := parts[0]
		
		// Read file content
		content, err := fs.ReadFile(migrationFiles, filepath.Join("migrations", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}
		
		if migrationMap[version] == nil {
			migrationMap[version] = make(map[string]string)
		}
		
		if strings.Contains(filename, ".up.sql") {
			migrationMap[version]["up"] = string(content)
			migrationMap[version]["name"] = strings.Join(parts[1:len(parts)-1], "_")
		} else if strings.Contains(filename, ".down.sql") {
			migrationMap[version]["down"] = string(content)
		}
	}
	
	// Convert to Migration structs
	for versionStr, files := range migrationMap {
		var version int
		if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
			continue
		}
		
		migration := Migration{
			Version: version,
			Name:    files["name"],
			UpSQL:   files["up"],
			DownSQL: files["down"],
		}
		
		migrations = append(migrations, migration)
	}
	
	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	
	return migrations, nil
}

// GetAppliedMigrations returns list of applied migration versions
func (mr *MigrationRunner) GetAppliedMigrations(ctx context.Context) ([]int, error) {
	query := "SELECT version FROM schema_migrations ORDER BY version"
	
	rows, err := mr.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()
	
	var versions []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		versions = append(versions, version)
	}
	
	return versions, nil
}

// Up applies all pending migrations
func (mr *MigrationRunner) Up(ctx context.Context) error {
	// Ensure migrations table exists
	if err := mr.CreateMigrationsTable(ctx); err != nil {
		return err
	}
	
	// Load all migrations
	migrations, err := mr.LoadMigrations()
	if err != nil {
		return err
	}
	
	// Get applied migrations
	applied, err := mr.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}
	
	appliedMap := make(map[int]bool)
	for _, version := range applied {
		appliedMap[version] = true
	}
	
	// Apply pending migrations
	for _, migration := range migrations {
		if appliedMap[migration.Version] {
			continue
		}
		
		fmt.Printf("Applying migration %d: %s\n", migration.Version, migration.Name)
		
		// Execute migration in transaction
		err := mr.db.WithTx(ctx, func(tx pgx.Tx) error {
			// Execute migration SQL
			if _, err := mr.db.Pool.Exec(ctx, migration.UpSQL); err != nil {
				return fmt.Errorf("failed to execute migration %d: %w", migration.Version, err)
			}
			
			// Record migration as applied
			insertQuery := "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)"
			if _, err := mr.db.Pool.Exec(ctx, insertQuery, migration.Version, migration.Name); err != nil {
				return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
			}
			
			return nil
		})
		
		if err != nil {
			return err
		}
		
		fmt.Printf("Applied migration %d: %s\n", migration.Version, migration.Name)
	}
	
	return nil
}

// Down rolls back the last applied migration
func (mr *MigrationRunner) Down(ctx context.Context) error {
	// Get applied migrations
	applied, err := mr.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}
	
	if len(applied) == 0 {
		fmt.Println("No migrations to roll back")
		return nil
	}
	
	// Get the last applied migration
	lastVersion := applied[len(applied)-1]
	
	// Load migrations
	migrations, err := mr.LoadMigrations()
	if err != nil {
		return err
	}
	
	// Find the migration to roll back
	var targetMigration *Migration
	for _, migration := range migrations {
		if migration.Version == lastVersion {
			targetMigration = &migration
			break
		}
	}
	
	if targetMigration == nil {
		return fmt.Errorf("migration %d not found", lastVersion)
	}
	
	fmt.Printf("Rolling back migration %d: %s\n", targetMigration.Version, targetMigration.Name)
	
	// Execute rollback in transaction
	err = mr.db.WithTx(ctx, func(tx pgx.Tx) error {
		// Execute rollback SQL
		if _, err := mr.db.Pool.Exec(ctx, targetMigration.DownSQL); err != nil {
			return fmt.Errorf("failed to rollback migration %d: %w", targetMigration.Version, err)
		}
		
		// Remove migration record
		deleteQuery := "DELETE FROM schema_migrations WHERE version = $1"
		if _, err := mr.db.Pool.Exec(ctx, deleteQuery, targetMigration.Version); err != nil {
			return fmt.Errorf("failed to remove migration record %d: %w", targetMigration.Version, err)
		}
		
		return nil
	})
	
	if err != nil {
		return err
	}
	
	fmt.Printf("Rolled back migration %d: %s\n", targetMigration.Version, targetMigration.Name)
	return nil
}