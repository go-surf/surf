package surf

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCookieCache(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("cannot create a request: %s", err)
	}

	cache, err := NewCookieCache("", []byte("super-secret-test-string"))
	if err != nil {
		t.Fatalf("cannot create cookie cache: %s", err)
	}
	testCache(t, cache.Bind(w, r))
}

func TestCookieCacheBetweenRequests(t *testing.T) {
	w1 := httptest.NewRecorder()
	r1, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("cannot create a request: %s", err)
	}

	cache, err := NewCookieCache("", []byte("super-secret-test-string"))
	if err != nil {
		t.Fatalf("cannot create cookie cache: %s", err)
	}

	ctx := context.Background()

	if err := cache.Bind(w1, r1).Set(ctx, "key-abc", "abc", time.Minute); err != nil {
		t.Fatalf("cannot set: %s", err)
	}

	w2 := httptest.NewRecorder()
	r2, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("cannot create a request: %s", err)
	}
	r2.Header = http.Header{"Cookie": w1.HeaderMap["Set-Cookie"]}

	var val string
	if err := cache.Bind(w2, r2).Get(ctx, "key-abc", &val); err != nil || val != "abc" {
		t.Fatalf("cannot get value: %v, %q", err, val)
	}
}
