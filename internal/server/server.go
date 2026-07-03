package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/asset"
	"github.com/extension-erp/be-extension-erp/internal/auth"
	"github.com/extension-erp/be-extension-erp/internal/config"
	"github.com/extension-erp/be-extension-erp/internal/finance"
	"github.com/extension-erp/be-extension-erp/internal/hris"
	"github.com/extension-erp/be-extension-erp/internal/middleware"
	"github.com/extension-erp/be-extension-erp/pkg/cache"
	"github.com/extension-erp/be-extension-erp/pkg/jwt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Server bundles HTTP dependencies for lifecycle management.
type Server struct {
	cfg    *config.Config
	engine *gin.Engine
	http   *http.Server
	db     *gorm.DB
	rdb    *redis.Client
}

// New builds the gin engine, wires middleware, runs migrations, and registers routes.
func New(cfg *config.Config, db *gorm.DB, rdb *redis.Client) (*Server, error) {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Logger(), middleware.Recovery())
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Dependency wiring (composition root).
	jwtMgr := jwt.New(cfg.JWT)
	c := cache.New(rdb)

	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, jwtMgr, c)
	authH := auth.NewHandler(authSvc)

	financeRepo := finance.NewRepository(db)
	financeSvc := finance.NewService(financeRepo)
	financeH := finance.NewHandler(financeSvc)

	assetRepo := asset.NewRepository(db)
	assetSvc := asset.NewService(assetRepo)
	assetH := asset.NewHandler(assetSvc)

	hrisRepo := hris.NewRepository(db)
	hrisSvc := hris.NewService(hrisRepo)
	hrisH := hris.NewHandler(hrisSvc)

	registerRoutes(engine, jwtMgr, authH, financeH, assetH, hrisH)

	s := &Server{
		cfg:    cfg,
		engine: engine,
		db:     db,
		rdb:    rdb,
		http: &http.Server{
			Addr:              ":" + cfg.App.Port,
			Handler:           engine,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
	return s, nil
}

// Run starts the HTTP server (blocking).
func (s *Server) Run() error {
	log.Printf("server: listening on %s", s.http.Addr)
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server and closes resources.
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.http.Shutdown(ctx); err != nil {
		return err
	}
	if sqlDB, err := s.db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	_ = s.rdb.Close()
	return nil
}
