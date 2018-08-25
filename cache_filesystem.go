package surf

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"
)

func NewFilesystemCache(rootDir string) CacheService {
	os.MkdirAll(rootDir, 0770)
	return &fscache{
		rootDir: rootDir,
	}
}

type fscache struct {
	mu      sync.RWMutex
	rootDir string
}

func (f *fscache) cachePath(key string) string {
	h := sha1.Sum([]byte(key))
	return path.Join(f.rootDir, hex.EncodeToString(h[:]))
}

func (f *fscache) Get(ctx context.Context, key string, dest interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if b, err := ioutil.ReadFile(f.cachePath(key)); err == nil {
		var item fscacheItem
		if err := json.Unmarshal(b, &item); err != nil {
			return fmt.Errorf("cannot deserialize internal representation: %s", err)
		}
		if item.ValidTill.Before(time.Now()) {
			os.Remove(f.cachePath(key))
			return ErrMiss
		}

		rawValue := []byte(item.Value)
		if err := json.Unmarshal(rawValue, &dest); err != nil {
			return fmt.Errorf("cannot deserialize: %s", err)
		}
		return nil
	}
	return ErrMiss
}

func (f *fscache) Set(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return f.set(ctx, key, value, exp)
}

func (f *fscache) set(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	rawValue, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot serialize: %s", err)
	}

	item := fscacheItem{
		ValidTill: time.Now().Add(exp),
		Value:     rawValue,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot serialize internal representation: %s", err)
	}

	if err := ioutil.WriteFile(f.cachePath(key), b, 0660); err != nil {
		return fmt.Errorf("cannot persist: %s", err)
	}
	return nil
}

func (f *fscache) SetNx(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch err := f.exists(key); err {
	case ErrMiss:
		// all good
	case nil:
		return ErrConflict
	default:
		return err
	}

	return f.set(ctx, key, value, exp)
}

func (f *fscache) Del(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.exists(key); err != nil {
		return err
	}

	os.Remove(f.cachePath(key))
	return nil
}

func (f *fscache) exists(key string) error {
	b, err := ioutil.ReadFile(f.cachePath(key))
	if err != nil {
		return ErrMiss
	}

	var item fscacheItem
	if err := json.Unmarshal(b, &item); err != nil {
		return fmt.Errorf("cannot deserialize internal representation: %s", err)
	}
	if item.ValidTill.Before(time.Now()) {
		os.Remove(f.cachePath(key))
		return ErrMiss
	}
	return nil
}

type fscacheItem struct {
	ValidTill time.Time
	Value     json.RawMessage
}
