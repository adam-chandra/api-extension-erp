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
// The migration-tracking table lives in the `sync` schema. The schema is
// created up-front (idempotent) so golang-migrate can place its table there.
//
// PostgreSQL privilege note: CREATE SCHEMA requires the DATABASE-level CREATE
// privilege. When the schema already exists, "CREATE SCHEMA IF NOT EXISTS" exits
// early before the privilege check (PG behaviour), so subsequent deploys work
// without the privilege. A fresh database does require it; fix with:
//
//	GRANT CREATE ON DATABASE <dbname> TO <user>;
func RunMigrations(cfg config.DBConfig) error {
	sqlDB, err := sql.Open("pgx", cfg.PostgresDSN())
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer sqlDB.Close()

	if err := ensureSyncSchema(sqlDB, cfg); err != nil {
		return err
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

// ensureSyncSchema checks whether the `sync` schema already exists and only
// attempts CREATE when it does not. This avoids a spurious permission error:
// PostgreSQL's "CREATE SCHEMA IF NOT EXISTS" skips the privilege check when
// the schema already exists, so subsequent deployments work even without the
// DATABASE-level CREATE privilege. A first-time deploy against an empty
// database still requires the privilege; if it is missing, the error message
// includes the exact SQL statement needed to fix it.
func ensureSyncSchema(db *sql.DB, cfg config.DBConfig) error {
	var exists bool
	if err := db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = 'sync')`,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check sync schema existence: %w", err)
	}
	if exists {
		return nil
	}
	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS sync`); err != nil {
		return fmt.Errorf(
			"bootstrap sync schema: user %q lacks CREATE privilege on database %q; "+
				"connect as a superuser and run: GRANT CREATE ON DATABASE %q TO %q — original error: %w",
			cfg.User, cfg.Name, cfg.Name, cfg.User, err,
		)
	}
	return nil
}
