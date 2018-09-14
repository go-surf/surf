package surf

import "testing"

func TestLocalMemoryCache(t *testing.T) {
	cache := NewLocalMemCache()

	RunCacheImplementationTest(t, cache)
}
