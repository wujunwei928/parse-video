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
	"github.com/tidwall/gjson"
)

// Keep the short-lived anonymous identity in memory for the lifetime of the
// process. This avoids writing an upstream identity cookie to disk.
var douyinGuestCache douyinGuestIdentityCache

type douyinGuestIdentityCache struct {
	mu      sync.Mutex
	cookie  string
	expires time.Time
}

var errDouyinGuestRejected = errors.New("douyin detail rejected guest identity or returned no detail")

func (c *douyinGuestIdentityCache) get(ctx context.Context, client *resty.Client, baseURL string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cookie != "" && time.Now().Before(c.expires) {
		return c.cookie, nil
	}
	cookie, err := registerDouyinGuest(ctx, client, baseURL)
	if err != nil {
		return "", err
	}
	c.cookie = cookie
	c.expires = time.Now().Add(10 * time.Minute)
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

func fetchDouyinDetailWithGuest(
	ctx context.Context,
	client *resty.Client,
	videoID string,
	cache *douyinGuestIdentityCache,
	baseURL string,
) (gjson.Result, error) {
	for attempt := 0; attempt < 2; attempt++ {
		cookie, err := cache.get(ctx, client, baseURL)
		if err != nil {
			return gjson.Result{}, fmt.Errorf("get douyin guest identity: %w", err)
		}
		fetcher := newDouyinWebDetailFetcher(client, cookie)
		fetcher.endpoint = baseURL + "/aweme/v1/web/aweme/detail/"
		data, err := fetcher.fetch(ctx, videoID)
		if !errors.Is(err, errDouyinGuestRejected) {
			return data, err
		}
		cache.invalidate(cookie)
		if attempt == 1 {
			return data, err
		}
	}
	return gjson.Result{}, errDouyinGuestRejected
}

func registerDouyinGuest(ctx context.Context, client *resty.Client, baseURL string) (string, error) {
	if client == nil {
		client = newClient(SourceDouYin)
	}
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	response, err := client.R().SetContext(requestCtx).
		SetHeader(HttpHeaderUserAgent, douyinWebDetailUserAgent).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]any{"aid": 6383, "service": "www.douyin.com"}).
		Post(baseURL + "/ttwid/register")
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", errors.New("register douyin guest identity failed")
	}
	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("register douyin guest identity returned status %d", response.StatusCode())
	}
	var ttwid string
	for _, cookie := range response.Cookies() {
		if cookie.Name == "ttwid" {
			ttwid = cookie.Value
			break
		}
	}
	if ttwid == "" {
		return "", errors.New("register douyin guest identity returned no ttwid cookie")
	}

	response, err = client.R().SetContext(requestCtx).
		SetHeader(HttpHeaderUserAgent, douyinWebDetailUserAgent).
		SetHeader(HttpHeaderReferer, baseURL+"/").
		SetCookie(&http.Cookie{Name: "ttwid", Value: ttwid}).
		Get(baseURL + "/")
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", errors.New("get douyin guest visitor identifier failed")
	}
	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("get douyin guest visitor identifier returned status %d", response.StatusCode())
	}
	visitor := strings.TrimSpace(response.Header().Get("x-tt-trace-host"))
	if len(visitor) < 32 || len(visitor) > 512 {
		return "", errors.New("douyin guest visitor identifier missing or invalid")
	}
	if _, err := hex.DecodeString(visitor); err != nil {
		return "", errors.New("douyin guest visitor identifier is not hexadecimal")
	}
	return (&http.Cookie{Name: "UIFID", Value: visitor}).String() + "; " + (&http.Cookie{Name: "ttwid", Value: ttwid}).String(), nil
}
