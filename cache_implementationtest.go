package surf

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func RunCacheImplementationTest(t *testing.T, c CacheService) {
	ctx := context.Background()

	testCacheSimpleItemSerialization(ctx, t, c)
	testCacheCustomItemSerialization(ctx, t, c)
	testCacheOperations(ctx, t, c)
}

func testCacheOperations(ctx context.Context, t *testing.T, c CacheService) {
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
	if err := c.SetNx(ctx, "key-1", "ABC", 10*time.Second); !ErrConflict.Is(err) {
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
	time.Sleep(time.Second + 20*time.Millisecond)
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

func testCacheSimpleItemSerialization(ctx context.Context, t *testing.T, c CacheService) {
	item := testCacheItem{A: "foo", B: 42}

	if err := c.Set(ctx, t.Name(), &item, time.Minute); err != nil {
		t.Fatalf("cannot set item: %s", err)
	}

	var res testCacheItem
	if err := c.Get(ctx, t.Name(), &res); err != nil {
		t.Fatalf("cannot get item: %s", err)
	} else if !reflect.DeepEqual(item, res) {
		t.Fatalf("want %#v value, got %#v", item, res)
	}
}

type testCacheItem struct {
	A string
	B int
}

func testCacheCustomItemSerialization(ctx context.Context, t *testing.T, c CacheService) {
	item := testCacheItem2{A: "foo", B: 42}

	// Ensure that serialization is implemented correctly.
	if raw, err := item.MarshalCache(); err != nil {
		t.Fatalf("faulty marshal implementation: %s", err)
	} else {
		var res testCacheItem2
		if err := res.UnmarshalCache(raw); err != nil {
			t.Fatalf("faulty unmarshal implementation: %s", err)
		}
		if !reflect.DeepEqual(item, res) {
			t.Fatalf("faulty unmarshal implementation: want %#v, got %#v", item, res)
		}
	}

	if err := c.Set(ctx, t.Name(), &item, time.Minute); err != nil {
		t.Fatalf("cannot set item: %s", err)
	}

	var res testCacheItem2
	if err := c.Get(ctx, t.Name(), &res); err != nil {
		t.Fatalf("cannot get item: %s", err)
	} else if !reflect.DeepEqual(item, res) {
		t.Fatalf("want %#v value, got %#v", item, res)
	}
}

type testCacheItem2 struct {
	A string `json:"-"`
	B int    `json:"-"`
}

var _ CacheMarshaler = (*testCacheItem2)(nil)

func (it testCacheItem2) MarshalCache() ([]byte, error) {
	raw := fmt.Sprintf("%d\t%s", it.B, it.A)
	return []byte(raw), nil
}

func (it *testCacheItem2) UnmarshalCache(raw []byte) error {
	_, err := fmt.Sscanf(string(raw), "%d\t%s", &it.B, &it.A)
	return err
}
