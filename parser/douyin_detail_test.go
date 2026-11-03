package parser

import (
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

type douyinRoundTripFunc func(*http.Request) (*http.Response, error)

func (f douyinRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDouyinCookieArgumentOverridesEnvironment(t *testing.T) {
	t.Setenv(douyinCookieEnv, "UIFID=environment-cookie")
	client := resty.New().SetTransport(douyinRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("uifid") != "argument-cookie" {
			t.Errorf("request did not use the argument cookie")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"aweme_detail":{"aweme_id":"7703117796061859110"}}`)), Request: r}, nil
	}))
	data, err := (douYin{cookie: "UIFID=argument-cookie"}).fetchNativeVideoDetail(client, "7703117796061859110")
	if err != nil || data.Get("aweme_id").String() != "7703117796061859110" {
		t.Fatalf("detail fetch failed: %v", err)
	}
}

func TestDouyinSM3(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard abc vector",
			input: "abc",
			want:  "66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2297da02b8f4ba8e0",
		},
		{
			name:  "standard 64-byte vector",
			input: strings.Repeat("abcd", 16),
			want:  "debe9ff92275b8a138604889c18e5a4d6fdb70e5387e5765293dcba39c0c5732",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := douyinSM3([]byte(tt.input))
			if got := hex.EncodeToString(got[:]); got != tt.want {
				t.Fatalf("douyinSM3() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMakeDouyinABogusDeterministic(t *testing.T) {
	query := douyinWebDetailQuery("7450123456789012345")
	got := makeDouyinABogus(query, "GET", 1720000000123, 1720000000129, 1234, 5678, 9012)
	const want = "E7mhBdugDifihdWk5l/LfY3q6fuVYmQ/0SVkMD2ffaDOJL39HMOk9exobQ4vpY2NZfmv2-ujy5kSYrrMicQnA3v6HSRKl2xp-g00t-P2so0j5ZhjCfuDnzfF-vzWt-Bd-Jd3Ech/ovKSKYi0AIee-wHvyhnFwo8sNiD4"
	if got != want {
		t.Fatalf("makeDouyinABogus() = %q, want %q", got, want)
	}
}

func TestSignDouyinWebQuery(t *testing.T) {
	got := signDouyinWebQuery("a=one%2Ftwo&a_bogus=ab%2Fcd", "visitor", 1720000000)
	want := "a=one%2Ftwo&a_bogus=ab%2Fcd&uifid=visitor&timestamp=1720000000&x-secsdk-web-signature=f43c8d122c44df0895a99c40ee92475e"
	if got != want {
		t.Fatalf("signed query = %q, want %q", got, want)
	}
}

func TestDouyinWebDetailRequiresUIFID(t *testing.T) {
	fetcher := newDouyinWebDetailFetcher(resty.New(), "sessionid=test")
	_, err := fetcher.fetch("7703117796061859110")
	if err == nil || !strings.Contains(err.Error(), "UIFID") {
		t.Fatalf("error = %v, want missing UIFID error", err)
	}
}

func TestDouyinWebDetailAcceptsTemporaryUIFID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("uifid") != "temporary-visitor" || r.URL.Query().Get("uifid") != "temporary-visitor" {
			t.Error("temporary visitor identity was not used for signing")
		}
		_, _ = w.Write([]byte(`{"aweme_detail":{"aweme_id":"7703117796061859110"}}`))
	}))
	defer server.Close()
	fetcher := newDouyinWebDetailFetcher(resty.New(), "UIFID_TEMP=temporary-visitor; ttwid=test")
	fetcher.endpoint = server.URL
	if _, err := fetcher.fetch("7703117796061859110"); err != nil {
		t.Fatalf("temporary visitor identity rejected: %v", err)
	}
}

func TestDouyinWebDetailQueryPreservesSignatureOrder(t *testing.T) {
	const videoID = "7450123456789012345"
	want := strings.Join([]string{
		"device_platform=webapp",
		"aid=6383",
		"channel=channel_pc_web",
		"pc_client_type=1",
		"version_code=290100",
		"version_name=29.1.0",
		"cookie_enabled=true",
		"screen_width=1920",
		"screen_height=1080",
		"browser_language=zh-CN",
		"browser_platform=Win32",
		"browser_name=Chrome",
		"browser_version=130.0.0.0",
		"browser_online=true",
		"engine_name=Blink",
		"engine_version=130.0.0.0",
		"os_name=Windows",
		"os_version=10",
		"cpu_core_num=12",
		"device_memory=8",
		"platform=PC",
		"downlink=10",
		"effective_type=4g",
		"from_user_page=1",
		"locate_query=false",
		"need_time_list=1",
		"pc_libra_divert=Windows",
		"publish_video_strategy_type=2",
		"round_trip_time=0",
		"show_live_replay_strategy=1",
		"time_list_query=0",
		"whale_cut_token=",
		"update_version_code=170400",
		"msToken=",
		"aweme_id=" + videoID,
	}, "&")

	if got := douyinWebDetailQuery(videoID); got != want {
		t.Fatalf("douyinWebDetailQuery() = %q, want %q", got, want)
	}
}

func TestDouyinWebDetailFetcherFetchesOfficialDetail(t *testing.T) {
	const videoID = "7450123456789012345"
	query := douyinWebDetailQuery(videoID)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/aweme/v1/web/aweme/detail/" {
			t.Errorf("request path = %q", request.URL.Path)
		}
		if request.Header.Get(HttpHeaderCookie) != "sessionid=deployment-owned-cookie; UIFID=visitor" {
			t.Errorf("Cookie = %q", request.Header.Get(HttpHeaderCookie))
		}
		if request.Header.Get(HttpHeaderReferer) != "https://www.douyin.com/" {
			t.Errorf("Referer = %q", request.Header.Get(HttpHeaderReferer))
		}
		if !strings.HasPrefix(request.URL.RawQuery, query+"&a_bogus=") {
			t.Error("request did not preserve the signed query order")
		}
		if aBogus := request.URL.Query().Get("a_bogus"); aBogus == "" {
			t.Error("a_bogus is missing")
		}
		if request.Header.Get("uifid") != "visitor" || request.URL.Query().Get("uifid") != "visitor" || request.URL.Query().Get("timestamp") != "1720000000" || request.URL.Query().Get("x-secsdk-web-signature") == "" {
			t.Error("security identity, timestamp, or signature is missing")
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"aweme_detail":{"aweme_id":"7450123456789012345","desc":"native detail"}}`))
	}))
	defer server.Close()

	fetcher := newDouyinWebDetailFetcher(resty.New(), "sessionid=deployment-owned-cookie; UIFID=visitor")
	fetcher.endpoint = server.URL + "/aweme/v1/web/aweme/detail/"
	fetcher.now = func() time.Time { return time.UnixMilli(1720000000123) }
	fetcher.random = func() int { return 1234 }

	data, err := fetcher.fetch(videoID)
	if err != nil {
		t.Fatalf("fetch() error = %v", err)
	}
	if got := data.Get("aweme_id").String(); got != videoID {
		t.Errorf("aweme_id = %q, want %q", got, videoID)
	}
	if got := data.Get("desc").String(); got != "native detail" {
		t.Errorf("desc = %q, want native detail", got)
	}
}

func TestDouyinWebDetailFetcherReportsOfficialErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status_code":4,"status_msg":"detail unavailable","aweme_detail":{}}`))
	}))
	defer server.Close()

	fetcher := newDouyinWebDetailFetcher(resty.New(), "sessionid=deployment-owned-cookie; UIFID=visitor")
	fetcher.endpoint = server.URL
	fetcher.now = func() time.Time { return time.UnixMilli(1720000000123) }
	fetcher.random = func() int { return 1234 }

	_, err := fetcher.fetch("7450123456789012345")
	if err == nil {
		t.Fatal("fetch() error = nil, want official detail status error")
	}
	for _, want := range []string{`aweme id ""`, "status_code=4", `status_msg="detail unavailable"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("fetch() error = %q, want substring %q", err, want)
		}
	}
}

func TestDouyinWebDetailFetcherRequiresDeploymentCookie(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	fetcher := newDouyinWebDetailFetcher(resty.New(), "")
	fetcher.endpoint = server.URL
	_, err := fetcher.fetch("7450123456789012345")
	if err == nil || !strings.Contains(err.Error(), douyinCookieEnv+" is required") {
		t.Fatalf("fetch() error = %v, want missing cookie error", err)
	}
	if called {
		t.Fatal("fetch() made an HTTP request without a deployment cookie")
	}
}
