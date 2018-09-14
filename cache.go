package surf

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type CacheService interface {
	// Get value stored under given key. Returns ErrMiss if key is not
	// used.
	Get(ctx context.Context, key string, dest interface{}) error

	// Set value under given key. If key is already in use, overwrite it's
	// value with given one and set new expiration time.
	Set(ctx context.Context, key string, value interface{}, exp time.Duration) error

	// SetNx set value under given key only if key is not used. It returns
	// ErrConflict if trying to set value for key that is already in use.
	SetNx(ctx context.Context, key string, value interface{}, exp time.Duration) error

	// Del deletes value under given key. It returns ErrCacheMiss if given
	// key is not used.
	Del(ctx context.Context, key string) error
}

type UnboundCacheService interface {
	Bind(http.ResponseWriter, *http.Request) CacheService
}

var (
	// ErrMiss is returned when performing operation on key is not in use.
	ErrMiss = errors.New("cache miss")

	// ErrConflict is returned when performing operation on existing key,
	// which cause conflict.
	ErrConflict = errors.New("conflict")
)

// CacheMarshal returns serialized representation of given value.
//
// Unless given destination implements CacheMarshaler interface, JSON is used
// to marshal the value.
func CacheMarshal(value interface{}) ([]byte, error) {
	if m, ok := value.(CacheMarshaler); ok {
		return m.MarshalCache()
	}
	return json.Marshal(value)

}

// CacheUnmarshal deserialize given raw data into given destination.
//
// Unless given destination implements CacheMarshaler interface, JSON is used
// to unmarshal the value.
func CacheUnmarshal(raw []byte, dest interface{}) error {
	if m, ok := dest.(CacheMarshaler); ok {
		return m.UnmarshalCache(raw)
	}
	return json.Unmarshal(raw, dest)
}

type CacheMarshaler interface {
	MarshalCache() ([]byte, error)
	UnmarshalCache([]byte) error
}
