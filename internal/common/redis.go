package common

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// InitRedisClient инициализирует клиент Redis с коротким таймаутом проверки.
// Если Redis недоступен, возвращает nil и сервис использует in-memory fallback.
func InitRedisClient(serviceName string) *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		opts = &redis.Options{
			Addr:        redisURL,
			DialTimeout: 1 * time.Second,
		}
	}

	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[%s] Redis not reachable at %s (%v). Using In-Memory fallback for locks/cache.", serviceName, redisURL, err)
		return nil
	}

	log.Printf("[%s] Connected to Redis at %s", serviceName, redisURL)
	return rdb
}

// HoldSeatInRedis блокирует место атомарной командой SETNX с TTL (15 минут).
func HoldSeatInRedis(rdb *redis.Client, sessionID int, sectorCode string, row, seat int, bookingID string, ttl time.Duration) (bool, error) {
	if rdb == nil {
		return true, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := fmt.Sprintf("hold:session:%d:seat:%s_%d_%d", sessionID, sectorCode, row, seat)
	return rdb.SetNX(ctx, key, bookingID, ttl).Result()
}

// ReleaseSeatInRedis снимает удержание места (компенсирующее действие Саги).
func ReleaseSeatInRedis(rdb *redis.Client, sessionID int, sectorCode string, row, seat int) error {
	if rdb == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := fmt.Sprintf("hold:session:%d:seat:%s_%d_%d", sessionID, sectorCode, row, seat)
	return rdb.Del(ctx, key).Err()
}
