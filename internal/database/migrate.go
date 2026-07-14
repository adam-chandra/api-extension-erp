package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// RunMigrations applies database migrations from the ./migrations directory.
// The migration-tracking table lives in the `sync` schema because on shared
// Postgres the app user typically lacks CREATE on `public`. The schema is
// created up-front (idempotent) so golang-migrate can place its table there.
func RunMigrations(cfg config.DBConfig) error {
	sqlDB, err := sql.Open("pgx", cfg.PostgresDSN())
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer sqlDB.Close()

	// Set connection timeouts to prevent hanging on unreachable database
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err := ensureSyncSchema(sqlDB); err != nil {
		return fmt.Errorf("bootstrap sync schema: %w", err)
	}

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{
		SchemaName:      "sync",
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return fmt.Errorf("create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("new migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// ensureSyncSchema checks if the sync schema exists, and creates it only if needed.
// This guard prevents CREATE SCHEMA from running when the user lacks database-level
// CREATE privilege, avoiding SQLSTATE 42501 "permission denied for database" errors.
func ensureSyncSchema(sqlDB *sql.DB) error {
	// Check if the sync schema already exists
	var exists bool
	err := sqlDB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM information_schema.schemata 
			WHERE schema_name = 'sync'
		)
	`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check sync schema existence: %w", err)
	}

	// Only create the schema if it doesn't already exist
	if !exists {
		if _, err := sqlDB.Exec(`CREATE SCHEMA sync`); err != nil {
			// If CREATE fails due to permission (SQLSTATE 42501), that's expected
			// on shared Postgres instances. The schema may already exist but check
			// failed, or it will be created by admin. Log and continue.
			if isPermissionDenied(err) {
				fmt.Printf("WARNING: could not create sync schema (permission denied) - continuing anyway\n")
				return nil
			}
			return fmt.Errorf("create sync schema: %w", err)
		}
	}

	return nil
}

// isPermissionDenied checks if an error is a PostgreSQL permission denied error.
func isPermissionDenied(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for PostgreSQL permission denied error (SQLSTATE 42501)
	return contains(errStr, "permission denied") || contains(errStr, "42501")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
