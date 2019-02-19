package surf

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-surf/surf/errors"
)

func NewCookieCache(prefix string, secret []byte) (UnboundCacheService, error) {
	block, err := aes.NewCipher(adjustKeySize(secret))
	if err != nil {
		return nil, errors.Wrap(ErrInternal, "cannot create cipher block: %s", err)
	}
	cc := unboundCookieCache{
		prefix: prefix,
		secret: block,
	}
	return &cc, nil
}

// adjustKeySize trim given secret to biggest acceptable by AES implementation
// key block. If given secret is too short to be used as AES key, it is
// returned without modifications
func adjustKeySize(secret []byte) []byte {
	size := len(secret)
	if size > 32 {
		return secret[:32]
	}
	if size > 24 {
		return secret[:24]
	}
	if size > 16 {
		return secret[:16]
	}
	return secret
}

type unboundCookieCache struct {
	secret cipher.Block
	prefix string
}

func (c *unboundCookieCache) Bind(w http.ResponseWriter, r *http.Request) CacheService {
	return &cookieCache{
		prefix: c.prefix,
		secret: c.secret,
		w:      w,
		r:      r,
		staged: make(map[string]cookieCacheItem),
	}
}

type cookieCache struct {
	w      http.ResponseWriter
	r      *http.Request
	prefix string
	secret cipher.Block

	staged map[string]cookieCacheItem
}

type cookieCacheItem struct {
	payload   []byte
	validTill time.Time
}

func (s *cookieCache) Get(ctx context.Context, key string, dest interface{}) error {
	defer CurrentTrace(ctx).Begin("cookie cache get",
		"key", key,
	).Finish()

	now := time.Now()

	if item, ok := s.staged[key]; ok {
		if item.validTill.Before(now) {
			delete(s.staged, key)
		} else {
			if err := CacheUnmarshal(item.payload, dest); err != nil {
				return errors.Wrap(err, "cannot unmarshal")
			}
			return nil
		}
	}

	c, err := s.r.Cookie(s.prefix + key)
	if err != nil {
		return ErrMiss
	}

	// if cookie cannot be(decoded or signature is invalid, ErrMiss
	// is returned. User cannot deal with such issue, so no need to
	// bother with the details

	rawData, err := s.decrypt(c.Value)
	if err != nil {
		return errors.Wrap(ErrMiss, "cannot decrypt")
	}

	rawPayload := rawData[:len(rawData)-4]
	rawExp := rawData[len(rawData)-4:]
	exp := time.Unix(int64(binary.LittleEndian.Uint32(rawExp)), 0)
	if !exp.After(now) {
		s.del(key)
		return errors.Wrap(ErrMiss, "expired")
	}

	if err := CacheUnmarshal(rawPayload, dest); err != nil {
		return errors.Wrap(err, "cannot unmarshal")
	}
	return nil
}

func (s *cookieCache) Set(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	defer CurrentTrace(ctx).Begin("cookie cache set",
		"key", key,
		"exp", fmt.Sprint(exp),
	).Finish()

	return s.set(key, value, exp)
}

func (s *cookieCache) set(key string, value interface{}, exp time.Duration) error {
	rawPayload, err := CacheMarshal(value)
	if err != nil {
		return errors.Wrap(err, "cannot marshal")
	}

	expAt := time.Now().Add(exp)
	rawExp := make([]byte, 4)
	binary.LittleEndian.PutUint32(rawExp, uint32(expAt.Unix()))

	rawData := append(rawPayload, rawExp...)
	payload, err := s.encrypt(rawData)
	if err != nil {
		return errors.Wrap(err, "cannot encrypt")
	}

	http.SetCookie(s.w, &http.Cookie{
		Name:     s.prefix + key,
		Value:    payload,
		Path:     "/",
		Expires:  expAt,
		HttpOnly: true,
		//Secure:   true,
	})
	if exp > 0 {
		s.staged[key] = cookieCacheItem{
			payload:   rawPayload,
			validTill: expAt,
		}
	}
	return nil
}

func (s *cookieCache) encrypt(data []byte) (string, error) {
	cipherText := make([]byte, ivSize+len(data))

	iv := cipherText[:ivSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", errors.Wrap(ErrInternal, "cannot read: %s", err)
	}

	stream := cipher.NewCFBEncrypter(s.secret, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], data)

	return base64.URLEncoding.EncodeToString(cipherText), nil
}

const ivSize = aes.BlockSize

func (s *cookieCache) decrypt(payload string) ([]byte, error) {
	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, errors.WrapErr(ErrMalformed, err)
	}
	if len(raw) < ivSize {
		return nil, errors.Wrap(ErrValidation, "message too short")
	}

	data := raw[ivSize:]
	stream := cipher.NewCFBDecrypter(s.secret, raw[:ivSize])
	stream.XORKeyStream(data, data)
	return data, nil
}

func (s *cookieCache) SetNx(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	if _, ok := s.staged[key]; ok {
		return errors.Wrap(ErrConflict, "exists")
	}
	if _, err := s.r.Cookie(s.prefix + key); err == nil {
		// TODO check if valid and not expired
		return errors.Wrap(ErrConflict, "exists")
	}

	return s.set(key, value, exp)
}

func (s *cookieCache) Del(ctx context.Context, key string) error {
	defer CurrentTrace(ctx).Begin("cookie cache del",
		"key", key,
	).Finish()

	return s.del(key)
}

func (s *cookieCache) del(key string) error {
	existed := false
	if _, ok := s.staged[key]; ok {
		existed = true
		delete(s.staged, key)
	}

	// TODO: deleting does not remove it from the request
	if _, err := s.r.Cookie(s.prefix + key); err == nil {

		// TODO: check if cookie value is not expired

		http.SetCookie(s.w, &http.Cookie{
			Name:    s.prefix + key,
			Value:   "",
			Path:    "/",
			Expires: time.Time{},
			MaxAge:  -1,
		})
		existed = true
	}

	if !existed {
		return ErrMiss
	}
	return nil
}
