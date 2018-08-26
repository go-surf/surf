package rediscache

import (
	"testing"

	"github.com/go-surf/surf"
)

func TestRedisCache(t *testing.T) {
	pool := EnsureRedis(t)
	defer pool.Close()
	cache := NewRedisCache(pool)

	surf.RunCacheImplementationTest(t, cache)
}
