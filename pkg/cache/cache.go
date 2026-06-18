package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrMiss is returned when a key is not present in the cache.
var ErrMiss = errors.New("cache: miss")

// Cache wraps a redis client with JSON helpers. Use this everywhere
// instead of touching the redis client directly.
type Cache struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

// Get unmarshals the JSON stored at key into dest.
func (c *Cache) Get(ctx context.Context, key string, dest any) error {
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrMiss
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

// Set marshals value as JSON and stores it with the given TTL.
func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, raw, ttl).Err()
}

// Delete removes one or more keys.
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

// Remember returns the cached value for key, or computes & stores it via loader.
func Remember[T any](ctx context.Context, c *Cache, key string, ttl time.Duration, loader func(context.Context) (T, error)) (T, error) {
	var out T
	err := c.Get(ctx, key, &out)
	if err == nil {
		return out, nil
	}
	if !errors.Is(err, ErrMiss) {
		return out, err
	}
	fresh, err := loader(ctx)
	if err != nil {
		return out, err
	}
	_ = c.Set(ctx, key, fresh, ttl)
	return fresh, nil
}
