package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/extension-erp/be-extension-erp/internal/config"
)

// clearEnv removes variables that would otherwise override defaults.
func clearEnv(keys ...string) {
	for _, k := range keys {
		os.Unsetenv(k)
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(
		"APP_ENV", "APP_PORT", "APP_NAME", "PORT",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
		"JWT_SECRET", "JWT_ACCESS_TTL", "JWT_REFRESH_TTL",
		"CORS_ALLOWED_ORIGINS",
	)

	cfg := config.Load()

	if cfg.App.Env != "development" {
		t.Errorf("App.Env = %q, want development", cfg.App.Env)
	}
	if cfg.App.Port != "8080" {
		t.Errorf("App.Port = %q, want 8080", cfg.App.Port)
	}
	if cfg.DB.Host != "localhost" {
		t.Errorf("DB.Host = %q, want localhost", cfg.DB.Host)
	}
	if cfg.DB.Port != "5432" {
		t.Errorf("DB.Port = %q, want 5432", cfg.DB.Port)
	}
	if cfg.DB.User != "postgres" {
		t.Errorf("DB.User = %q, want postgres", cfg.DB.User)
	}
	if cfg.DB.SSLMode != "disable" {
		t.Errorf("DB.SSLMode = %q, want disable", cfg.DB.SSLMode)
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("Redis.Addr = %q, want localhost:6379", cfg.Redis.Addr)
	}
	if cfg.Redis.DB != 0 {
		t.Errorf("Redis.DB = %d, want 0", cfg.Redis.DB)
	}
}

func TestLoad_OverridesFromEnv(t *testing.T) {
	os.Setenv("APP_ENV", "production")
	os.Setenv("APP_PORT", "9000")
	os.Setenv("APP_NAME", "my-api")
	os.Setenv("DB_HOST", "db.prod.example.com")
	os.Setenv("DB_USER", "admin")
	os.Setenv("DB_NAME", "proddb")
	os.Setenv("REDIS_ADDR", "redis.prod.example.com:6379")
	os.Setenv("REDIS_DB", "2")
	defer clearEnv("APP_ENV", "APP_PORT", "APP_NAME", "DB_HOST", "DB_USER", "DB_NAME", "REDIS_ADDR", "REDIS_DB")

	cfg := config.Load()

	if cfg.App.Env != "production" {
		t.Errorf("App.Env = %q, want production", cfg.App.Env)
	}
	if cfg.App.Port != "9000" {
		t.Errorf("App.Port = %q, want 9000", cfg.App.Port)
	}
	if cfg.App.Name != "my-api" {
		t.Errorf("App.Name = %q, want my-api", cfg.App.Name)
	}
	if cfg.DB.Host != "db.prod.example.com" {
		t.Errorf("DB.Host = %q, want db.prod.example.com", cfg.DB.Host)
	}
	if cfg.Redis.DB != 2 {
		t.Errorf("Redis.DB = %d, want 2", cfg.Redis.DB)
	}
}

func TestLoad_PORTFallback(t *testing.T) {
	clearEnv("APP_PORT")
	os.Setenv("PORT", "5000")
	defer os.Unsetenv("PORT")

	cfg := config.Load()
	if cfg.App.Port != "5000" {
		t.Errorf("App.Port via PORT fallback = %q, want 5000", cfg.App.Port)
	}
}

func TestLoad_InvalidRedisDB_FallsBackToZero(t *testing.T) {
	os.Setenv("REDIS_DB", "not-a-number")
	defer os.Unsetenv("REDIS_DB")

	cfg := config.Load()
	if cfg.Redis.DB != 0 {
		t.Errorf("Redis.DB with invalid value = %d, want 0", cfg.Redis.DB)
	}
}

func TestDBConfig_PostgresDSN(t *testing.T) {
	cfg := config.DBConfig{
		Host:     "db.example.com",
		Port:     "5432",
		User:     "admin",
		Password: "s3cr3t",
		Name:     "mydb",
		SSLMode:  "require",
	}
	dsn := cfg.PostgresDSN()
	for _, want := range []string{"db.example.com", "5432", "admin", "mydb", "require"} {
		if !strings.Contains(dsn, want) {
			t.Errorf("PostgresDSN() missing %q in %q", want, dsn)
		}
	}
}

func TestLoad_CORSOrigins(t *testing.T) {
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	cfg := config.Load()
	if len(cfg.CORS.AllowedOrigins) != 2 {
		t.Errorf("CORS.AllowedOrigins len = %d, want 2", len(cfg.CORS.AllowedOrigins))
	}
}
