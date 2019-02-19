package surf

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-surf/surf/errors"
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
	// This is a not found error narrowed to cache cases only.
	ErrMiss = errors.Wrap(ErrNotFound, "cache miss")

	// ErrCacheMalformed is returned whenever an operation cannot be
	// completed because value cannot be serialized or deserialized.
	ErrCacheMalformed = errors.Wrap(ErrMalformed, "cache malformed")
)

// CacheMarshal returns serialized representation of given value.
//
// Unless given destination implements CacheMarshaler interface, JSON is used
// to marshal the value.
func CacheMarshal(value interface{}) ([]byte, error) {
	if m, ok := value.(CacheMarshaler); ok {
		return m.MarshalCache()
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, errors.WrapErr(ErrCacheMalformed, err)
	}
	return raw, nil

}

// CacheUnmarshal deserialize given raw data into given destination.
//
// Unless given destination implements CacheMarshaler interface, JSON is used
// to unmarshal the value.
func CacheUnmarshal(raw []byte, dest interface{}) error {
	if m, ok := dest.(CacheMarshaler); ok {
		return m.UnmarshalCache(raw)
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return errors.WrapErr(ErrCacheMalformed, err)
	}
	return nil
}

type CacheMarshaler interface {
	MarshalCache() ([]byte, error)
	UnmarshalCache([]byte) error
}
