package rediscache

import (
	"os"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
)

// EnsureRedis creates a redis pool. Only if a pool with a successful
// connection can be created then the test is continued. If a connection is not
// successful test is skipped.
// Redis connection can be set via REDIS_URL environment variable.
// It is the clients responsibility to close the pool when no longer needed.
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
