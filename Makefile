.PHONY: help start run migrate-up migrate-down migrate-create migrate-force migrate-version

# ==================== OS Detection ====================
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
else
    DETECTED_OS := $(shell uname -s)
endif

# ==================== Database Config ====================
-include .env
export

DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

# ==================== Help ====================
help:
	@echo "Available targets:"
	@echo "  make migrate-up                          - Run all pending migrations"
	@echo "  make migrate-down                        - Rollback last migration"
	@echo "  make migrate-create name=migration_name  - Create new migration"
	@echo "  make migrate-force version=N             - Force migration to specific version"
	@echo "  make migrate-version                     - Show current migration version"

# ==================== Migration ====================

# run all migrations
# usage : make migrate-up
migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

# rollback 1 migration
# usage : make migrate-down
migrate-down:
	migrate -path migrations -database "$(DB_URL)" down

# create new migration (timestamp-based, avoids numbering conflicts in team)
# usage : make migrate-create name=migration_name
migrate-create:
ifeq ($(DETECTED_OS),Windows)
	@if "$(name)"=="" (echo Usage: make migrate-create name=migration_name & exit 1)
else
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=migration_name"; exit 1; fi
endif
	migrate create -ext sql -dir migrations $(name)

# fix if migration error/dirty
# usage : make migrate-force version=N
migrate-force:
ifeq ($(DETECTED_OS),Windows)
	@if "$(version)"=="" (echo Usage: make migrate-force version=N & exit 1)
else
	@if [ -z "$(version)" ]; then echo "Usage: make migrate-force version=N"; exit 1; fi
endif
	migrate -path migrations -database "$(DB_URL)" force $(version)

# check current migration version
# usage : make migrate-version
migrate-version:
	migrate -path migrations -database "$(DB_URL)" version