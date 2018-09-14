package surf

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"
)

func StampedeProtect(cache CacheService) CacheService {
	return &stampedeProtectedCache{
		cache: cache,
	}
}

type stampedeProtectedCache struct {
	cache CacheService
}

func (s *stampedeProtectedCache) Get(ctx context.Context, key string, dest interface{}) error {
	var it stampedeProtectedItem

readProtectedItem:
	for {
		switch err := s.cache.Get(ctx, key, &it); err {
		case nil:
			// All good.
		case ErrMiss:
			// Acquire lock for a short period to avoid multiple
			// clients computing the same task. If we get the lock,
			// return cache miss - we are allowed to recompute. If
			// we don't get the lock, wait and retry until value is
			// in cache again.
			if s.cache.SetNx(ctx, key+":stampedelock", 1, time.Second) == nil {
				return ErrMiss
			}
			time.Sleep(25 * time.Millisecond)
			continue readProtectedItem
		default:
			return err
		}

		if it.refreshAt.Before(time.Now()) {
			// Acquire task computation lock. If we get it, return
			// cache miss so that the client will recompute the
			// result. Otherwise return cached value - cached value
			// is still valid and another client is already
			// recomputing the task.
			if s.cache.SetNx(ctx, key+":stampedelock", 1, time.Second) == nil {
				return ErrMiss
			}
		}

		if err := CacheUnmarshal(it.value, dest); err != nil {
			return fmt.Errorf("stampede protected deserialize: %s", err)
		}
		return nil
	}
}

func (s *stampedeProtectedCache) Set(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	rawValue, err := CacheMarshal(value)
	if err != nil {
		return fmt.Errorf("cannot serialize: %s", err)
	}

	refreshMargin := 5 * time.Second
	if exp < refreshMargin {
		refreshMargin = exp / 2
	}
	it := stampedeProtectedItem{
		refreshAt: time.Now().Add(exp).Add(-refreshMargin),
		value:     rawValue,
	}
	return s.cache.Set(ctx, key, &it, exp)
}

func (s *stampedeProtectedCache) SetNx(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	rawValue, err := CacheMarshal(value)
	if err != nil {
		return fmt.Errorf("cannot serialize: %s", err)
	}

	var refreshMargin time.Duration
	if exp > 10*time.Minute {
		refreshMargin = time.Minute
	} else if exp > time.Minute {
		refreshMargin = 10 * time.Second
	} else if exp > 30*time.Second {
		refreshMargin = 3 * time.Second
	} else if exp > 10*time.Second {
		refreshMargin = time.Second
	}

	it := stampedeProtectedItem{
		refreshAt: time.Now().Add(exp).Add(-refreshMargin),
		value:     rawValue,
	}
	return s.cache.SetNx(ctx, key, &it, exp)
}

func (s *stampedeProtectedCache) Del(ctx context.Context, key string) error {
	return s.cache.Del(ctx, key)
}

type stampedeProtectedItem struct {
	refreshAt time.Time
	value     []byte
}

func (it stampedeProtectedItem) MarshalCache() ([]byte, error) {
	raw := fmt.Sprintf("%d\n%s", it.refreshAt.UnixNano(), it.value)
	return []byte(raw), nil
}

func (it *stampedeProtectedItem) UnmarshalCache(raw []byte) error {
	chunks := bytes.SplitN(raw, []byte{'\n'}, 2)
	if len(chunks) != 2 {
		return fmt.Errorf("invalid format: %s", raw)
	}
	unixNano, err := strconv.ParseInt(string(chunks[0]), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid expiration format: %s", err)
	}
	it.refreshAt = time.Unix(0, unixNano)
	it.value = chunks[1]
	return nil
}
