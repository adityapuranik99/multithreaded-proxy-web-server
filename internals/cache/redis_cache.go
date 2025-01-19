package cache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context // context for operations
	ttl    time.Duration   // expiration time for cache
}

func NewRedisCache(addr string, password string, db int, ttl time.Duration) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()

	// ping redis to check connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	return &RedisCache{client: rdb, ctx: ctx, ttl: ttl}
}

func (r *RedisCache) Get(key string) (string, bool) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return "", false // cache miss
	} else if err != nil {
		log.Printf("Redis GET error: %v", err)
		return "", false
	}
	return val, true // cache hit
}

func (r *RedisCache) Set(key string, value string) {
	err := r.client.Set(r.ctx, key, value, r.ttl).Err()
	if err != nil {
		log.Printf("Redis SET error: %v", err)
	}
}
