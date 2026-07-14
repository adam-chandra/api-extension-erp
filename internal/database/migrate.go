package database

import (
	"database/sql"
	"fmt"

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
			return fmt.Errorf("create sync schema: %w", err)
		}
	}

	return nil
}
