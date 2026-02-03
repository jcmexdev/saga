package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
}

type redisCache struct {
	client *redis.Client
}

func NewRedisCache(addr string) Cache {
	return &redisCache{
		client: redis.NewClient(&redis.Options{Addr: addr}),
	}
}

func (r redisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r redisCache) Get(ctx context.Context, key string) (string, error) {
	key, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}

	if err != nil {
		return "", nil
	}

	return key, nil
}
