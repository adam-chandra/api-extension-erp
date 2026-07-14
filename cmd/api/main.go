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
	log.Printf("server: config loaded (env=%s, db=%s:%s)", cfg.App.Env, cfg.DB.Host, cfg.DB.Port)

	log.Println("server: connecting to postgres...")
	db, err := database.NewPostgres(cfg.DB)
	if err != nil {
		log.Fatalf("postgres connection failed: %v", err)
	}

	// auth.* schema is consumed by the BE and populated by worker-erp.
	// Migrations live here so the BE codebase owns exactly one database.
	log.Println("server: running migrations...")
	if err := database.RunMigrations(cfg.DB); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}
	log.Println("server: migrations completed")

	log.Println("server: connecting to redis...")
	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}
	log.Println("server: redis connected")

	log.Println("server: initializing handlers...")
	srv, err := server.New(cfg, db, rdb)
	if err != nil {
		log.Fatalf("server init failed: %v", err)
	}
	log.Println("server: handlers initialized")

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
