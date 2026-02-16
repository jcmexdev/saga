package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	GenerateKey(operation, key string) string
}

type redisCache struct {
	client      *redis.Client
	serviceName string
}

func NewRedisCache(addr, serviceName string) Cache {
	return &redisCache{
		client:      redis.NewClient(&redis.Options{Addr: addr}),
		serviceName: serviceName,
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
		return "", err
	}

	return key, nil
}

func (r redisCache) GenerateKey(operation, key string) string {
	return fmt.Sprintf("%s:%s:%s", r.serviceName, operation, key)

}
