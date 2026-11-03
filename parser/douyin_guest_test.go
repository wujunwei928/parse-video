package parser

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-resty/resty/v2"
)

func TestDouyinGuestIdentityRegistrationAndCache(t *testing.T) {
	var registrations atomic.Int32
	visitor := strings.Repeat("ab", 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ttwid/register":
			if r.Method != http.MethodPost {
				t.Error("registration must use POST")
			}
			registrations.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "ttwid", Value: "visitor-id"})
		case "/":
			cookie, err := r.Cookie("ttwid")
			if err != nil || cookie.Value != "visitor-id" {
				t.Error("registration cookie missing from page request")
			}
			w.Header().Set("x-tt-trace-host", visitor)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	var cache douyinGuestIdentityCache
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cookie, err := cache.get(resty.New(), server.URL)
			if err != nil || cookie != "UIFID="+visitor+"; ttwid=visitor-id" {
				t.Errorf("guest registration failed: %v", err)
			}
		}()
	}
	wg.Wait()
	if registrations.Load() != 1 {
		t.Fatalf("registered %d times, want 1", registrations.Load())
	}
	cache.invalidate("unrelated-cookie")
	if _, err := cache.get(resty.New(), server.URL); err != nil || registrations.Load() != 1 {
		t.Fatalf("unrelated invalidation discarded the cache: %v", err)
	}
	cache.invalidate("UIFID=" + visitor + "; ttwid=visitor-id")
	if _, err := cache.get(resty.New(), server.URL); err != nil || registrations.Load() != 2 {
		t.Fatalf("invalidated identity was not refreshed: %v", err)
	}
}

func TestDouyinGuestIdentityRequiresVisitorHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ttwid/register" {
			http.SetCookie(w, &http.Cookie{Name: "ttwid", Value: "visitor-id"})
		}
	}))
	defer server.Close()
	if _, err := registerDouyinGuest(resty.New(), server.URL); err == nil {
		t.Fatal("expected an error when the visitor identifier is absent")
	}
}

func TestDouyinGuestRefreshesAfterForbidden(t *testing.T) {
	var registrations, details atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ttwid/register":
			registrations.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "ttwid", Value: "test"})
		case "/":
			w.Header().Set("x-tt-trace-host", strings.Repeat("ab", 32))
		case "/aweme/v1/web/aweme/detail/":
			if details.Add(1) == 1 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`{"aweme_detail":{"aweme_id":"7703117796061859110"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	var cache douyinGuestIdentityCache
	data, err := fetchDouyinDetailWithGuest(resty.New(), "7703117796061859110", &cache, server.URL)
	if err != nil || data.Get("aweme_id").String() != "7703117796061859110" {
		t.Fatalf("guest refresh failed: %v", err)
	}
	if registrations.Load() != 2 || details.Load() != 2 {
		t.Fatalf("registrations=%d, detail requests=%d; want 2 each", registrations.Load(), details.Load())
	}
}
