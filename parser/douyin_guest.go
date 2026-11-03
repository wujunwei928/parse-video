package parser

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

// A short-lived in-memory guest identity avoids registering on every request.
var douyinGuestCache douyinGuestIdentityCache

type douyinGuestIdentityCache struct {
	mu       sync.Mutex
	cookie   string
	expires  time.Time
	file     string
	rejected string
}

// EnableDouyinGuestDiskCache allows separate CLI invocations to reuse a guest
// identity. Cache errors never prevent parsing.
func EnableDouyinGuestDiskCache() {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		return
	}
	douyinGuestCache.mu.Lock()
	douyinGuestCache.file = filepath.Join(dir, "parse-video", "douyin-guest.json")
	douyinGuestCache.mu.Unlock()
}

type douyinGuestDiskEntry struct {
	Cookie  string    `json:"cookie"`
	Expires time.Time `json:"expires"`
}

func (c *douyinGuestIdentityCache) get(client *resty.Client, baseURL string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cookie != "" && time.Now().Before(c.expires) {
		return c.cookie, nil
	}
	if c.file != "" {
		data, err := os.ReadFile(c.file)
		var entry douyinGuestDiskEntry
		if err == nil && len(data) < 8192 && json.Unmarshal(data, &entry) == nil && entry.Cookie != "" && entry.Cookie != c.rejected && time.Now().Before(entry.Expires) && entry.Expires.Before(time.Now().Add(10*time.Minute)) {
			c.cookie, c.expires = entry.Cookie, entry.Expires
			return c.cookie, nil
		}
	}
	cookie, err := registerDouyinGuest(client, baseURL)
	if err != nil {
		return "", err
	}
	c.cookie, c.expires = cookie, time.Now().Add(10*time.Minute)
	c.rejected = ""
	if c.file != "" {
		c.save()
	}
	return cookie, nil
}

// save writes a complete file atomically; failures simply leave caching disabled.
func (c *douyinGuestIdentityCache) save() {
	data, err := json.Marshal(douyinGuestDiskEntry{Cookie: c.cookie, Expires: c.expires})
	if err != nil || os.MkdirAll(filepath.Dir(c.file), 0700) != nil {
		return
	}
	f, err := os.CreateTemp(filepath.Dir(c.file), "douyin-guest-*.tmp")
	if err != nil {
		return
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return
	}
	if f.Close() == nil {
		_ = os.Rename(f.Name(), c.file)
	}
}

func (c *douyinGuestIdentityCache) invalidate(cookie string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cookie == cookie {
		c.cookie = ""
		c.expires = time.Time{}
		c.rejected = cookie
	}
}

func registerDouyinGuest(client *resty.Client, baseURL string) (string, error) {
	if client == nil {
		client = newClient()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	response, err := client.R().SetContext(ctx).
		SetHeader(HttpHeaderUserAgent, douyinWebDetailUserAgent).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]any{"aid": 6383, "service": "www.douyin.com"}).
		Post(baseURL + "/ttwid/register")
	if err != nil {
		return "", fmt.Errorf("register ttwid: %w", err)
	}
	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("register ttwid returned status %d", response.StatusCode())
	}
	var ttwid string
	for _, cookie := range response.Cookies() {
		if cookie.Name == "ttwid" {
			ttwid = cookie.Value
			break
		}
	}
	if ttwid == "" {
		return "", errors.New("register ttwid returned no cookie")
	}

	response, err = client.R().SetContext(ctx).
		SetHeader(HttpHeaderUserAgent, douyinWebDetailUserAgent).
		SetHeader(HttpHeaderReferer, "https://www.douyin.com/").
		SetCookie(&http.Cookie{Name: "ttwid", Value: ttwid}).
		Get(baseURL + "/")
	if err != nil {
		return "", fmt.Errorf("get guest visitor identifier: %w", err)
	}
	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("get guest visitor identifier returned status %d", response.StatusCode())
	}
	// The initial response's trace identifier is accepted by the detail
	// endpoint as its visitor identity; it is not a server-issued UIFID cookie.
	// See https://github.com/yt-dlp/yt-dlp/pull/17826.
	visitor := strings.TrimSpace(response.Header().Get("x-tt-trace-host"))
	if len(visitor) < 32 || len(visitor) > 512 {
		return "", errors.New("guest visitor identifier missing or invalid")
	}
	if _, err := hex.DecodeString(visitor); err != nil {
		return "", errors.New("guest visitor identifier is not hexadecimal")
	}
	return (&http.Cookie{Name: "UIFID", Value: visitor}).String() + "; " + (&http.Cookie{Name: "ttwid", Value: ttwid}).String(), nil
}
