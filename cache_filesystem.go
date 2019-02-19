package surf

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/go-surf/surf/errors"
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

	b, err := ioutil.ReadFile(f.cachePath(key))
	if err != nil {
		return ErrMiss
	}
	var item fscacheItem
	if err := CacheUnmarshal(b, &item); err != nil {
		return errors.Wrap(err, "cannot unmarshal")
	}
	if item.validTill.Before(time.Now()) {
		_ = os.Remove(f.cachePath(key))
		return ErrMiss
	}

	if err := CacheUnmarshal(item.value, dest); err != nil {
		return errors.Wrap(err, "cannot unmarshal")
	}
	return nil
}

func (f *fscache) Set(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.set(ctx, key, value, exp)
}

func (f *fscache) set(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	rawValue, err := CacheMarshal(value)
	if err != nil {
		return errors.Wrap(err, "cannot marshal")
	}

	item := fscacheItem{
		validTill: time.Now().Add(exp),
		value:     rawValue,
	}
	b, err := CacheMarshal(&item)
	if err != nil {
		return errors.Wrap(err, "cannot marshal")
	}

	if err := ioutil.WriteFile(f.cachePath(key), b, 0660); err != nil {
		_ = os.Remove(f.cachePath(key))
		return errors.Wrap(ErrInternal, "cannot persist: %s", err)
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
		return errors.Wrap(ErrConflict, "exist")
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
	if err := CacheUnmarshal(b, &item); err != nil {
		return errors.Wrap(err, "cannot unmarshal")
	}
	if item.validTill.Before(time.Now()) {
		os.Remove(f.cachePath(key))
		return ErrMiss
	}
	return nil
}

type fscacheItem struct {
	validTill time.Time
	value     []byte
}

var _ CacheMarshaler = (*fscacheItem)(nil)

func (it fscacheItem) MarshalCache() ([]byte, error) {
	raw := fmt.Sprintf("%d\n%s", it.validTill.UnixNano(), it.value)
	return []byte(raw), nil
}

func (it *fscacheItem) UnmarshalCache(raw []byte) error {
	chunks := bytes.SplitN(raw, []byte{'\n'}, 2)
	if len(chunks) != 2 {
		return errors.Wrap(ErrCacheMalformed, "invalid format: %s", raw)
	}
	unixNano, err := strconv.ParseInt(string(chunks[0]), 10, 64)
	if err != nil {
		return errors.Wrap(ErrCacheMalformed, "invalid expiration format: %s", err)
	}
	it.validTill = time.Unix(0, unixNano)
	it.value = chunks[1]
	return nil
}
