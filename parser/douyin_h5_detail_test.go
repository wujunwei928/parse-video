package parser

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-resty/resty/v2"
)

const (
	douyinH5TestVideoID = "7682734336332389861"
	douyinH5TestWebID   = "7687279813539268146"
	douyinH5TestToken   = "deployment-owned-token"
	douyinH5TestCipher  = "X8NCQVYkntJfw5URyA+fE4FLWq2AWWzLErM+YZEWRWM="
)

func TestEncryptDouyinH5ReflowToken(t *testing.T) {
	got, err := encryptDouyinH5ReflowToken(douyinH5TestToken, douyinH5TestWebID)
	if err != nil {
		t.Fatalf("encryptDouyinH5ReflowToken() error = %v", err)
	}
	if got != douyinH5TestCipher {
		t.Errorf("encryptDouyinH5ReflowToken() = %q, want %q", got, douyinH5TestCipher)
	}
}

func TestEncryptDouyinH5ReflowTokenRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		webID   string
		wantErr string
	}{
		{name: "empty token", webID: douyinH5TestWebID, wantErr: "token is empty"},
		{name: "short web id", token: douyinH5TestToken, webID: "1234567890", wantErr: "at least 16 bytes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptDouyinH5ReflowToken(tt.token, tt.webID)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("encryptDouyinH5ReflowToken() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseDouyinH5ReflowContext(t *testing.T) {
	pageHTML := []byte(`<!doctype html><html><body>
		<div id=douyin_reflow_token xsstoken=deployment-owned-token></div>
		<div id='douyin_reflow_webId' webId="7687279813539268146" usercip='192.0.2.1'></div>
	</body></html>`)

	got, err := parseDouyinH5ReflowContext(pageHTML)
	if err != nil {
		t.Fatalf("parseDouyinH5ReflowContext() error = %v", err)
	}
	if got.token != douyinH5TestToken {
		t.Errorf("token = %q, want %q", got.token, douyinH5TestToken)
	}
	if got.webID != douyinH5TestWebID {
		t.Errorf("webID = %q, want %q", got.webID, douyinH5TestWebID)
	}
	if got.userCIP != "192.0.2.1" {
		t.Errorf("userCIP = %q, want 192.0.2.1", got.userCIP)
	}
}

func TestParseDouyinH5ReflowContextErrors(t *testing.T) {
	tests := []struct {
		name     string
		pageHTML string
		wantErr  string
	}{
		{
			name:     "missing token",
			pageHTML: `<div id=douyin_reflow_webId webId=7687279813539268146></div>`,
			wantErr:  "did not return xsstoken",
		},
		{
			name:     "missing web id",
			pageHTML: `<div id=douyin_reflow_token xsstoken=deployment-owned-token></div>`,
			wantErr:  "did not return webId",
		},
		{
			name: "short web id",
			pageHTML: `<div id=douyin_reflow_token xsstoken=deployment-owned-token></div>
				<div id=douyin_reflow_webId webId=1234567890></div>`,
			wantErr: "invalid webId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDouyinH5ReflowContext([]byte(tt.pageHTML))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("parseDouyinH5ReflowContext() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestDouyinH5DetailFetcherFetchesOfficialItemInfo(t *testing.T) {
	pageHTML := []byte(`<div id=douyin_reflow_token xsstoken=deployment-owned-token></div>
		<div id=douyin_reflow_webId webId=7687279813539268146 usercip=192.0.2.1></div>`)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/web/api/v2/aweme/iteminfo/" {
			t.Errorf("request path = %q", request.URL.Path)
		}
		wantQuery := map[string]string{
			"reflow_source":        "reflow_page",
			"web_id":               douyinH5TestWebID,
			"device_id":            douyinH5TestWebID,
			"user_cip":             "192.0.2.1",
			"use_new_select_scope": "0",
			"item_ids":             douyinH5TestVideoID,
			"reflow_id":            douyinH5TestCipher,
		}
		for key, want := range wantQuery {
			if got := request.URL.Query().Get(key); got != want {
				t.Errorf("query %s = %q, want %q", key, got, want)
			}
		}
		if got := request.Header.Get(HttpHeaderUserAgent); got != DefaultUserAgent {
			t.Errorf("User-Agent = %q, want %q", got, DefaultUserAgent)
		}
		if got := request.Header.Get(HttpHeaderReferer); got != "https://www.iesdouyin.com/share/video/"+douyinH5TestVideoID {
			t.Errorf("Referer = %q", got)
		}
		cookie, err := request.Cookie("anonymous_session")
		if err != nil || cookie.Value != "page-cookie" {
			t.Errorf("anonymous page cookie = %v, error = %v", cookie, err)
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status_code":0,"item_list":[{"aweme_id":"7682734336332389861","desc":"native h5 detail","images":[{"url_list":["https://example.invalid/image.jpeg"]}]}]}`))
	}))
	defer server.Close()

	fetcher := newDouyinH5DetailFetcher(resty.New())
	fetcher.endpoint = server.URL + "/web/api/v2/aweme/iteminfo/"
	data, err := fetcher.fetch(douyinH5TestVideoID, pageHTML, []*http.Cookie{{Name: "anonymous_session", Value: "page-cookie"}})
	if err != nil {
		t.Fatalf("fetch() error = %v", err)
	}
	if got := data.Get("aweme_id").String(); got != douyinH5TestVideoID {
		t.Errorf("aweme_id = %q, want %q", got, douyinH5TestVideoID)
	}
	if got := data.Get("desc").String(); got != "native h5 detail" {
		t.Errorf("desc = %q, want native h5 detail", got)
	}
	if got := data.Get("images.0.url_list.0").String(); got != "https://example.invalid/image.jpeg" {
		t.Errorf("image URL = %q, want synthetic image fixture", got)
	}
}

func TestDouyinH5DetailFetcherRetriesInvalidPageContext(t *testing.T) {
	invalidPageHTML := []byte(`<div id=douyin_reflow_token xsstoken=deployment-owned-token></div>
		<div id=douyin_reflow_webId webId=undefined></div>`)
	validPageHTML := `<div id=douyin_reflow_token xsstoken=deployment-owned-token></div>
		<div id=douyin_reflow_webId webId=7687279813539268146></div>`

	var refreshCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/share/video/" + douyinH5TestVideoID:
			refreshCount.Add(1)
			http.SetCookie(writer, &http.Cookie{Name: "anonymous_session", Value: "refreshed-page-cookie"})
			_, _ = writer.Write([]byte(validPageHTML))
		case "/web/api/v2/aweme/iteminfo/":
			cookie, err := request.Cookie("anonymous_session")
			if err != nil || cookie.Value != "refreshed-page-cookie" {
				t.Errorf("refreshed page cookie = %v, error = %v", cookie, err)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"status_code":0,"item_list":[{"aweme_id":"7682734336332389861"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	fetcher := newDouyinH5DetailFetcher(resty.New())
	fetcher.endpoint = server.URL + "/web/api/v2/aweme/iteminfo/"
	data, err := fetcher.fetchWithContextRetry(
		douyinH5TestVideoID,
		server.URL+"/share/video/"+douyinH5TestVideoID,
		invalidPageHTML,
		nil,
		douyinH5ContextAttempts,
	)
	if err != nil {
		t.Fatalf("fetchWithContextRetry() error = %v", err)
	}
	if got := data.Get("aweme_id").String(); got != douyinH5TestVideoID {
		t.Errorf("aweme_id = %q, want %q", got, douyinH5TestVideoID)
	}
	if got := refreshCount.Load(); got != 1 {
		t.Errorf("share page refresh count = %d, want 1", got)
	}
}

func TestDouyinH5DetailFetcherLimitsContextRetries(t *testing.T) {
	invalidPageHTML := []byte(`<div id=douyin_reflow_token xsstoken=deployment-owned-token></div>
		<div id=douyin_reflow_webId webId=undefined></div>`)

	var refreshCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		refreshCount.Add(1)
		_, _ = writer.Write(invalidPageHTML)
	}))
	defer server.Close()

	fetcher := newDouyinH5DetailFetcher(resty.New())
	_, err := fetcher.fetchWithContextRetry(
		douyinH5TestVideoID,
		server.URL,
		invalidPageHTML,
		nil,
		douyinH5ContextAttempts,
	)
	if err == nil || !strings.Contains(err.Error(), "invalid webId length 9") {
		t.Fatalf("fetchWithContextRetry() error = %v, want invalid webId length", err)
	}
	if got := refreshCount.Load(); got != douyinH5ContextAttempts-1 {
		t.Errorf("share page refresh count = %d, want %d", got, douyinH5ContextAttempts-1)
	}
}

func TestDouyinH5DetailFetcherDoesNotRetryResponseErrors(t *testing.T) {
	pageHTML := []byte(`<div id=douyin_reflow_token xsstoken=deployment-owned-token></div>
		<div id=douyin_reflow_webId webId=7687279813539268146></div>`)

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status_code":11110,"status_msg":"encrypt_data_miss"}`))
	}))
	defer server.Close()

	fetcher := newDouyinH5DetailFetcher(resty.New())
	fetcher.endpoint = server.URL
	_, err := fetcher.fetchWithContextRetry(
		douyinH5TestVideoID,
		server.URL,
		pageHTML,
		nil,
		douyinH5ContextAttempts,
	)
	if err == nil || !strings.Contains(err.Error(), "status_code=11110") {
		t.Fatalf("fetchWithContextRetry() error = %v, want platform status error", err)
	}
	if got := requestCount.Load(); got != 1 {
		t.Errorf("request count = %d, want 1", got)
	}
}

func TestDouyinH5DetailFetcherReportsResponseErrors(t *testing.T) {
	pageHTML := []byte(`<div id=douyin_reflow_token xsstoken=deployment-owned-token></div>
		<div id=douyin_reflow_webId webId=7687279813539268146></div>`)
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{
			name:       "http error",
			statusCode: http.StatusForbidden,
			body:       `forbidden`,
			wantErr:    "returned status 403",
		},
		{
			name:       "platform status error",
			statusCode: http.StatusOK,
			body:       `{"status_code":11110,"status_msg":"encrypt_data_miss"}`,
			wantErr:    `status_code=11110`,
		},
		{
			name:       "missing item list",
			statusCode: http.StatusOK,
			body:       `{"status_code":0,"item_list":[]}`,
			wantErr:    "did not return item_list",
		},
		{
			name:       "mismatched item",
			statusCode: http.StatusOK,
			body:       `{"status_code":0,"item_list":[{"aweme_id":"7000000000000000000"}]}`,
			wantErr:    `unexpected aweme id "7000000000000000000"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(tt.statusCode)
				_, _ = writer.Write([]byte(tt.body))
			}))
			defer server.Close()

			fetcher := newDouyinH5DetailFetcher(resty.New())
			fetcher.endpoint = server.URL
			_, err := fetcher.fetch(douyinH5TestVideoID, pageHTML, nil)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("fetch() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}
