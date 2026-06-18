package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/config"
	"github.com/extension-erp/be-extension-erp/internal/database"
	"github.com/extension-erp/be-extension-erp/internal/server"
)

func main() {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DB)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}

	// auth.* schema is consumed by the BE and populated by worker-erp.
	// Migrations live here so the BE codebase owns exactly one database.
	if err := database.RunMigrations(cfg.DB); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}

	srv, err := server.New(cfg, db, rdb)
	if err != nil {
		log.Fatalf("server init: %v", err)
	}

	// Run server in background so we can handle signals.
	go func() {
		if err := srv.Run(); err != nil && err.Error() != "http: Server closed" {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("server: shutting down…")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("server: stopped")
}
