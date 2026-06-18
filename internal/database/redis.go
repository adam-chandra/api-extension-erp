package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewRedis opens a Redis client and verifies connectivity with PING.
func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     20,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	log.Println("database: redis connected")
	return client, nil
}
