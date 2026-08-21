package parser

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestBiliBiliGetBvidFromB23ShortLink(t *testing.T) {
	tests := []struct {
		name        string
		location    string
		wantBVID    string
		wantErrPart string
	}{
		{
			name:     "redirects to video URL",
			location: "https://www.bilibili.com/video/BV1xH8n6CEj1?from=share",
			wantBVID: "BV1xH8n6CEj1",
		},
		{
			name:        "redirect response without location",
			wantErrPart: "无法从b23.tv获取重定向链接",
		},
		{
			name:        "redirects to non Bilibili URL",
			location:    "https://example.com/video/BV1xH8n6CEj1",
			wantErrPart: "不是有效的B站视频链接",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProxy()
			t.Cleanup(resetProxy)

			var requestCount atomic.Int32
			proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				requestCount.Add(1)
				if request.Method != http.MethodGet {
					t.Errorf("request method = %q, want GET", request.Method)
				}
				if request.URL.Host != "b23.tv" || request.URL.Path != "/gewigdZ" {
					t.Errorf("proxy target = %q, want http://b23.tv/gewigdZ", request.URL.String())
				}
				if request.Header.Get(HttpHeaderUserAgent) != UserAgent {
					t.Errorf("User-Agent = %q, want %q", request.Header.Get(HttpHeaderUserAgent), UserAgent)
				}
				if tt.location != "" {
					writer.Header().Set("Location", tt.location)
				}
				writer.WriteHeader(http.StatusFound)
			}))
			t.Cleanup(proxy.Close)

			if err := InitProxy(proxy.URL); err != nil {
				t.Fatalf("InitProxy() error = %v", err)
			}

			got, err := (biliBili{}).getBvidFromURL("http://b23.tv/gewigdZ")
			if tt.wantErrPart != "" {
				if err == nil {
					t.Fatalf("getBvidFromURL() error = nil, want error containing %q", tt.wantErrPart)
				}
				if !strings.Contains(err.Error(), tt.wantErrPart) {
					t.Errorf("getBvidFromURL() error = %q, want substring %q", err, tt.wantErrPart)
				}
				return
			}
			if err != nil {
				t.Fatalf("getBvidFromURL() error = %v", err)
			}
			if got != tt.wantBVID {
				t.Errorf("getBvidFromURL() = %q, want %q", got, tt.wantBVID)
			}
			if got := requestCount.Load(); got != 1 {
				t.Errorf("proxy request count = %d, want 1", got)
			}
		})
	}
}

func TestBiliBiliGetBvidFromURLTransportError(t *testing.T) {
	resetProxy()
	t.Cleanup(resetProxy)

	proxy := httptest.NewServer(http.NotFoundHandler())
	proxyURL := proxy.URL
	proxy.Close()

	if err := InitProxy(proxyURL); err != nil {
		t.Fatalf("InitProxy() error = %v", err)
	}

	_, err := (biliBili{}).getBvidFromURL("http://b23.tv/gewigdZ")
	if err == nil {
		t.Fatal("getBvidFromURL() error = nil, want transport error")
	}
	if !strings.Contains(err.Error(), "请求b23.tv短链失败") {
		t.Errorf("getBvidFromURL() error = %q, want short-link request context", err)
	}
}

func TestBiliBiliGetBvidFromDirectURL(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		wantBVID string
		wantErr  bool
	}{
		{
			name:     "canonical video URL",
			rawURL:   "https://www.bilibili.com/video/BV1xH8n6CEj1",
			wantBVID: "BV1xH8n6CEj1",
		},
		{
			name:     "video URL with trailing slash and query",
			rawURL:   "https://www.bilibili.com/video/BV1xH8n6CEj1/?from=share",
			wantBVID: "BV1xH8n6CEj1",
		},
		{
			name:    "non video URL",
			rawURL:  "https://www.bilibili.com/read/cv123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (biliBili{}).getBvidFromURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("getBvidFromURL() error = %v, wantErr %t", err, tt.wantErr)
			}
			if got != tt.wantBVID {
				t.Errorf("getBvidFromURL() = %q, want %q", got, tt.wantBVID)
			}
		})
	}
}
