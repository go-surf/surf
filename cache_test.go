package surf

import (
	"context"
	"strings"
	"testing"
	"time"
)

func testCache(t *testing.T, c CacheService) {
	ctx := context.Background()

	// ensure basic operations are correct
	if err := c.Set(ctx, "key-1", "abc", time.Second); err != nil {
		t.Fatalf("cannot set: %s", err)
	}
	var val string
	if err := c.Get(ctx, "key-1", &val); err != nil {
		t.Fatalf("cannot get: %s", err)
	} else if val != "abc" {
		t.Fatalf("want abc value, got %q", val)
	}
	if err := c.SetNx(ctx, "key-1", "ABC", 10*time.Second); err != ErrConflict {
		t.Fatalf("want ErrConflict, got %+v", err)
	}
	if err := c.Get(ctx, "key-1", &val); err != nil {
		t.Fatalf("cannot get: %s", err)
	} else if val != "abc" {
		t.Fatalf("want abc value, got %q", val)
	}

	// wait for the value to expire and ensure it's gone
	if err := c.Set(ctx, "key-exp", "abc", time.Second); err != nil {
		t.Fatalf("cannot set: %s", err)
	}
	time.Sleep(time.Second)
	val = ""
	if err := c.Get(ctx, "key-exp", &val); err != ErrMiss {
		t.Fatalf("want ErrMiss, got: %+v (%q)", err, val)
	}

	// deleting a key works
	if err := c.Set(ctx, "key-2", "123", time.Hour); err != nil {
		t.Fatalf("Cannot set: %s", err)
	}
	if err := c.Del(ctx, "key-2"); err != nil {
		t.Fatalf("cannot delete: %s", err)
	}
	if err := c.Get(ctx, "key-2", &val); err != ErrMiss {
		t.Fatalf("want ErrMiss, got: %+v (%q)", err, val)
	}
	if err := c.Del(ctx, "key-does-not-exists"); err != ErrMiss {
		t.Fatalf("want ErrMiss, got %+v", err)
	}

	// ensure very long keys are supported
	veryLongKey := strings.Repeat("very-long-key", 1000)
	if err := c.Set(ctx, veryLongKey, "123", time.Hour); err != nil {
		t.Fatalf("cannot set: %s", err)
	}
	val = ""
	if err := c.Get(ctx, veryLongKey, &val); err != nil || val != "123" {
		t.Fatalf("want 123, got %+v, %q", err, val)
	}
}
