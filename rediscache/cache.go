package rediscache

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"time"

	"github.com/go-surf/surf"
	"github.com/go-surf/surf/errors"
	"github.com/gomodule/redigo/redis"
)

// NewRedisCache returns a CacheService implementation that is using given
// redis pool as a storage backend.
func NewRedisCache(pool *redis.Pool) surf.CacheService {
	return &redisCache{
		pool: pool,
	}
}

type redisCache struct {
	pool *redis.Pool
}

func (r *redisCache) buildKey(key string) string {
	const maxKeyLength = 250

	// prevent from using long keys
	if len(key) <= maxKeyLength {
		return key
	}

	h := sha1.Sum([]byte(key))
	suffix := hex.EncodeToString(h[:])
	return key[maxKeyLength-len(suffix):] + suffix
}

func (r *redisCache) Get(ctx context.Context, key string, dest interface{}) error {
	rc, err := r.pool.GetContext(ctx)
	if err != nil {
		return errors.Wrap(ErrRedis, "cannot get connection: %s", err)
	}
	defer rc.Close()

	raw, err := redis.Bytes(rc.Do("GET", r.buildKey(key)))
	switch err {
	case nil:
		// all good
	case redis.ErrNil:
		return ErrMiss
	default:
		return errors.Wrap(ErrRedis, "cannot GET: %s", err)
	}

	if err := surf.CacheUnmarshal(raw, dest); err != nil {
		return err
	}
	return nil
}

func (r *redisCache) Set(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	raw, err := surf.CacheMarshal(value)
	if err != nil {
		return err
	}

	rc, err := r.pool.GetContext(ctx)
	if err != nil {
		return errors.Wrap(ErrRedis, "cannot get connection: %s", err)
	}
	defer rc.Close()

	if _, err := rc.Do("SET", r.buildKey(key), raw, "PX", int32(exp/time.Millisecond)); err != nil {
		return errors.Wrap(ErrRedis, "cannot SET: %s", err)
	}
	return nil
}

func (r *redisCache) SetNx(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	raw, err := surf.CacheMarshal(value)
	if err != nil {
		return err
	}

	rc, err := r.pool.GetContext(ctx)
	if err != nil {
		return errors.Wrap(ErrRedis, "cannot get connection: %s", err)
	}
	defer rc.Close()

	switch resp, err := rc.Do("SET", r.buildKey(key), raw, "PX", int32(exp/time.Millisecond), "NX"); err {
	case nil, redis.ErrNil:
		// if set was successful, resp will be OK and not nil. From
		// redis documentation http://redis.io/commands/set
		//
		// > Simple string reply: OK if SET was executed correctly.
		// > Null reply: a Null Bulk Reply is returned if the SET
		// > operation was not performed because the user specified the
		// > NX or XX option but the condition was not met.
		if resp == nil {
			return ErrConflict
		}
		return nil
	default:
		return errors.Wrap(ErrRedis, "cannot SET: %s", err)
	}
}

func (r *redisCache) Del(ctx context.Context, key string) error {
	rc, err := r.pool.GetContext(ctx)
	if err != nil {
		return errors.Wrap(ErrRedis, "cannot get connection: %s", err)
	}
	defer rc.Close()

	n, err := redis.Int(rc.Do("DEL", r.buildKey(key)))
	if err != nil {
		return errors.Wrap(ErrRedis, "cannot delete: %s", err)
	}
	if n == 0 {
		return ErrMiss
	}
	return nil
}

var (
	// ErrRedis is returned whenever there is an issue with the storage.
	// This can be for example an exhausted pool issues or a connection
	// failure.
	// Redis issues represents internal errors.
	ErrRedis = errors.Wrap(surf.ErrInternal, "redis")

	// ErrMiss is an alias.
	ErrMiss = errors.Wrap(surf.ErrMiss, "redis")

	// ErrConflict is an alias.
	ErrConflict = errors.Wrap(surf.ErrConflict, "redis")
)
