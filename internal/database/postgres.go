package database

import (
	"fmt"
	"log"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewPostgres opens a pooled GORM connection.
func NewPostgres(cfg config.DBConfig) (*gorm.DB, error) {
	gormLogLevel := logger.Warn

	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN()), &gorm.Config{
		Logger:                                   logger.Default.LogMode(gormLogLevel),
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	log.Println("database: postgres connected")
	return db, nil
}
