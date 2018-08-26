package rediscache

import (
	"os"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
)

func EnsureRedis(t *testing.T) *redis.Pool {
	t.Helper()

	// https://www.iana.org/assignments/uri-schemes/prov/redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	pool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 10 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(redisURL)
		},
	}

	rc := pool.Get()
	defer rc.Close()

	if _, err := redis.String(rc.Do("PING")); err != nil {
		t.Skipf("redis not available: %s", err)
	}

	return pool
}
