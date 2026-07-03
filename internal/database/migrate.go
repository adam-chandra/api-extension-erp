package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

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

	if _, err := sqlDB.Exec(`CREATE SCHEMA IF NOT EXISTS sync`); err != nil {
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

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		// ErrDirty means a previous migration was interrupted and left the
		// database in an inconsistent state.  Surface it clearly so it is
		// fixed before the server starts.
		var errDirty migrate.ErrDirty
		if errors.As(err, &errDirty) {
			return fmt.Errorf("run migrations: %w", err)
		}

		// If the version recorded in the database is ahead of the migration
		// files in the repository (e.g. files 17-20 were applied to the DB
		// but were never committed to the repo), golang-migrate cannot find a
		// matching file and returns an error.  Treat a clean, non-nil version
		// as "nothing to apply" so the server can still start.
		dbVersion, dirty, vErr := m.Version()
		if vErr == nil && !dirty {
			log.Printf("migrations: DB is at version %d which is ahead of the "+
				"latest migration file; skipping – no new migrations to apply", dbVersion)
			return nil
		}

		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
