package parser

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

// A short-lived in-memory guest identity avoids registering on every request.
var douyinGuestCache douyinGuestIdentityCache

type douyinGuestIdentityCache struct {
	mu      sync.Mutex
	cookie  string
	expires time.Time
}

func (c *douyinGuestIdentityCache) get(client *resty.Client, baseURL string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cookie != "" && time.Now().Before(c.expires) {
		return c.cookie, nil
	}
	cookie, err := registerDouyinGuest(client, baseURL)
	if err != nil {
		return "", err
	}
	c.cookie, c.expires = cookie, time.Now().Add(10*time.Minute)
	return cookie, nil
}

func (c *douyinGuestIdentityCache) invalidate(cookie string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cookie == cookie {
		c.cookie = ""
		c.expires = time.Time{}
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
