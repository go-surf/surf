package surf

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStampedeCacheProtection(t *testing.T) {
	cache := StampedeProtect(NewLocalMemCache())
	RunCacheImplementationTest(t, cache)
}

func TestStampedeCacheProtectionMultipleReaders(t *testing.T) {
	ctx := context.Background()

	cache := StampedeProtect(NewLocalMemCache())
	exp := 250 * time.Millisecond

	for iteration := 0; iteration < 3; iteration++ {
		var cacheHitCnt, computeCnt uint64

		var wg sync.WaitGroup
		start := make(chan struct{})

		for i := 0; i < 100; i++ {
			go func() {
				wg.Add(1)
				defer wg.Done()

				<-start

				var value string
				switch err := cache.Get(ctx, "value-1", &value); err {
				case nil:
					atomic.AddUint64(&cacheHitCnt, 1)
					if value != "whatever" {
						t.Errorf("want \"whatever\", got %q", value)
					}
				case ErrMiss:
					// Pretend there is some heavy computation happening.
					time.Sleep(10 * time.Millisecond)
					atomic.AddUint64(&computeCnt, 1)

					if err := cache.Set(ctx, "value-1", "whatever", exp); err != nil {
						t.Fatalf("cannot set: %s", err)
					}
				}
			}()
		}

		close(start)
		wg.Wait()

		if cacheHitCnt != 99 {
			t.Errorf("want 99 cache hits, got %d", cacheHitCnt)
		}
		if computeCnt != 1 {
			t.Errorf("want 1 computations, got %d", computeCnt)
		}

		time.Sleep(exp)
	}
}
